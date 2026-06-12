package handlers

import (
	"P1-BASESII/internal/cache"
	"P1-BASESII/internal/database"
	"P1-BASESII/internal/models"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// searchProduct: Struct local que coincide con Product del search-service.
// Solo los 4 campos que ES indexa — no expone precio ni restaurant_id.
type searchProduct struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	Description string `json:"description"`
}

// indexInSearch: Indexa o reindexa un MenuItem en ElasticSearch via el search-service.
// Entradas: models.MenuItem con los datos actualizados.
// Salidas: Ninguna (fire-and-forget, nunca bloquea ni rompe el flujo principal).
// Funcionalidad: POST a /search/reindex con array de 1 producto.
// Casos: Éxito silencioso, error logueado.
func indexInSearch(item models.MenuItem) {
	payload, err := json.Marshal([]searchProduct{{
		ID:          item.ID,
		Name:        item.Name,
		Category:    item.Category,
		Description: item.Description,
	}})
	if err != nil {
		fmt.Printf("search: error serializando plato %d: %v\n", item.ID, err)
		return
	}

	url := os.Getenv("SEARCH_SERVICE_URL") + "/search/reindex"
	resp, err := http.Post(url, "application/json", bytes.NewReader(payload))
	if err != nil {
		fmt.Printf("search: no se pudo indexar plato %d: %v\n", item.ID, err)
		return
	}
	defer resp.Body.Close()
	fmt.Printf("search: plato %d indexado correctamente\n", item.ID)
}

// deleteFromSearch: Elimina un MenuItem del índice ES via el search-service.
// Entradas: id string del MenuItem eliminado.
// Salidas: Ninguna (fire-and-forget).
// Funcionalidad: DELETE a /search/products/:id.
// Casos: Éxito silencioso, error logueado.
func deleteFromSearch(id string) {
	url := os.Getenv("SEARCH_SERVICE_URL") + "/search/products/" + id
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		fmt.Printf("search: error creando request delete para %s: %v\n", id, err)
		return
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("search: no se pudo eliminar plato %s del índice: %v\n", id, err)
		return
	}
	defer resp.Body.Close()
	fmt.Printf("search: plato %s eliminado del índice\n", id)
}

// GetMenuItems: Obtiene todos los platos disponibles.
// Entradas: Contexto Gin.
// Salidas: 200 OK + lista de platos.
// Funcionalidad: Intenta leer de Redis; si miss, consulta DB y cachea.
// Casos: Cache hit, cache miss → DB, error DB.
func GetMenuItems(c *gin.Context) {
	const key = "menus:all"
	var items []models.MenuItem

	if err := cache.Get(key, &items); err == nil {
		c.JSON(http.StatusOK, items)
		return
	}

	items, err := database.MenuRepo.FindAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener el menú"})
		return
	}

	cache.Set(key, items, cache.TTLMenus)
	c.JSON(http.StatusOK, items)
}

// GetMenuItem: Obtiene un plato por ID.
// Entradas: Contexto Gin con parámetro :id.
// Salidas: 200 OK + plato, 404 si no existe.
// Funcionalidad: Busca en Redis; si miss, consulta DB y cachea.
// Casos: Cache hit, cache miss → DB, no encontrado.
func GetMenuItem(c *gin.Context) {
	id := c.Param("id")
	key := fmt.Sprintf("menus:%s", id)
	var item models.MenuItem

	if err := cache.Get(key, &item); err == nil {
		c.JSON(http.StatusOK, item)
		return
	}

	result, err := database.MenuRepo.FindByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plato no encontrado"})
		return
	}

	cache.Set(key, result, cache.TTLMenus)
	c.JSON(http.StatusOK, result)
}

// CreateMenuItem: Crea un plato para un restaurante.
// Entradas: Contexto Gin con JSON {restaurant_id, name, category?, description?, price}.
// Salidas: 201 Created + plato, 400/403/404/500 con error.
// Funcionalidad: Valida dueño del restaurante, crea en DB, invalida caché e indexa en ES.
// Casos: Éxito, no propietario, restaurante no encontrado, datos inválidos.
func CreateMenuItem(c *gin.Context) {
	username := c.GetString("user")
	currentUser, err := database.UserRepo.FindByUsername(username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuario no encontrado en la DB local"})
		return
	}

	var input struct {
		RestaurantID uint    `json:"restaurant_id" binding:"required"`
		Name         string  `json:"name" binding:"required"`
		Category     string  `json:"category"`
		Description  string  `json:"description"`
		Price        float64 `json:"price" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	restaurant, err := database.RestaurantRepo.FindByID(fmt.Sprintf("%d", input.RestaurantID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Restaurante no encontrado"})
		return
	}

	if restaurant.AdminID != currentUser.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "No eres el dueño de este restaurante"})
		return
	}

	description := input.Description
	if description == "" {
		description = "Producto sin descripción"
	}

	item := &models.MenuItem{
		RestaurantID: input.RestaurantID,
		Name:         input.Name,
		Category:     input.Category,
		Description:  description,
		Price:        input.Price,
	}
	if err := database.MenuRepo.Create(item); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo crear el plato"})
		return
	}

	cache.Delete("menus:all")
	cache.DeleteByPattern(fmt.Sprintf("menus:restaurant:%d:*", input.RestaurantID))
	go indexInSearch(*item) // indexar en ES de forma asíncrona

	c.JSON(http.StatusCreated, item)
}

// UpdateMenuItem: Actualiza un plato existente.
// Entradas: Contexto Gin con :id y JSON de campos a cambiar.
// Salidas: 200 OK + plato actualizado, 400/403/404 con error.
// Funcionalidad: Verifica propietario, actualiza en DB, invalida caché y reindexa en ES.
// Casos: Éxito, no propietario, plato no encontrado.
func UpdateMenuItem(c *gin.Context) {
	id := c.Param("id")
	username := c.GetString("user")

	item, err := database.MenuRepo.FindByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plato no encontrado"})
		return
	}

	restaurant, _ := database.RestaurantRepo.FindByID(fmt.Sprintf("%d", item.RestaurantID))
	currentUser, _ := database.UserRepo.FindByUsername(username)

	if restaurant.AdminID != currentUser.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "No eres el dueño de este restaurante"})
		return
	}

	var input struct {
		Name        string  `json:"name"`
		Category    string  `json:"category"`
		Description string  `json:"description"`
		Price       float64 `json:"price"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Name != "" {
		item.Name = input.Name
	}
	if input.Category != "" {
		item.Category = input.Category
	}
	if input.Description != "" {
		item.Description = input.Description
	}
	if input.Price != 0 {
		item.Price = input.Price
	}

	if err := database.MenuRepo.Update(item); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar plato"})
		return
	}

	cache.Delete(fmt.Sprintf("menus:%s", id), "menus:all")
	go indexInSearch(*item) // reindexar en ES con datos actualizados

	c.JSON(http.StatusOK, item)
}

// DeleteMenuItem: Elimina un plato, desasocia órdenes y lo borra del índice ES.
// Entradas: Contexto Gin con :id.
// Salidas: 200 OK + mensaje, 401/403/404/500 con error.
// Funcionalidad: Verifica propietario, desvincula órdenes, borra de DB, caché y ES.
// Casos: Éxito, no propietario, plato no encontrado.
func DeleteMenuItem(c *gin.Context) {
	id := c.Param("id")
	username := c.GetString("user")

	currentUser, err := database.UserRepo.FindByUsername(username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuario no identificado"})
		return
	}

	item, err := database.MenuRepo.FindByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plato no encontrado"})
		return
	}

	restaurant, err := database.RestaurantRepo.FindByID(fmt.Sprintf("%d", item.RestaurantID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al validar el restaurante"})
		return
	}

	if restaurant.AdminID != currentUser.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "No eres el dueño del restaurante asociado a este plato"})
		return
	}

	database.MenuRepo.NullifyOrderMenuItemID(id)
	database.MenuRepo.Delete(id)

	cache.Delete(fmt.Sprintf("menus:%s", id), "menus:all")
	go deleteFromSearch(id) // eliminar del índice ES de forma asíncrona

	c.JSON(http.StatusOK, gin.H{"message": "Plato eliminado y órdenes desvinculadas"})
}
