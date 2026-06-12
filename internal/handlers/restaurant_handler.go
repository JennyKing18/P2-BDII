package handlers

import (
	"P1-BASESII/internal/cache"
	"P1-BASESII/internal/database"
	"P1-BASESII/internal/models"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// GetRestaurants: Obtiene todos los restaurantes.
// Entradas: Contexto Gin.
// Salidas: 200 OK + lista de restaurantes.
// Funcionalidad: Intenta leer de Redis; si miss, consulta DB via repositorio y cachea resultado.
// Casos: Cache hit, cache miss → DB, error DB.
func GetRestaurants(c *gin.Context) {
	const key = "restaurants:all"
	var restaurants []models.Restaurant

	// Cache hit — devuelve sin tocar la BD
	if err := cache.Get(key, &restaurants); err == nil {
		c.JSON(http.StatusOK, restaurants)
		return
	}

	// Cache miss → repositorio (Postgres o Mongo según DB_DRIVER)
	var err error
	restaurants, err = database.RestaurantRepo.FindAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error obteniendo restaurantes"})
		return
	}

	cache.Set(key, restaurants, cache.TTLRestaurants)
	c.JSON(http.StatusOK, restaurants)
}

// GetRestaurant: Obtiene un restaurante por ID.
// Entradas: Contexto Gin con parámetro :id.
// Salidas: 200 OK + restaurante, 404 si no existe.
// Funcionalidad: Busca en Redis primero; si miss, consulta DB via repositorio y cachea.
// Casos: Cache hit, cache miss → DB, no encontrado.
func GetRestaurant(c *gin.Context) {
	id := c.Param("id")
	key := fmt.Sprintf("restaurants:%s", id)
	var restaurant models.Restaurant

	// Cache hit
	if err := cache.Get(key, &restaurant); err == nil {
		c.JSON(http.StatusOK, restaurant)
		return
	}

	// Cache miss → repositorio
	result, err := database.RestaurantRepo.FindByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Restaurante no encontrado"})
		return
	}

	cache.Set(key, result, cache.TTLRestaurants)
	c.JSON(http.StatusOK, result)
}

// CreateRestaurant: Crea un nuevo restaurante.
// Entradas: Contexto Gin con JSON {name, address?, description?}.
// Salidas: 201 Created + restaurante, 400/401/500 con error.
// Funcionalidad: Busca usuario autenticado via repositorio, crea en DB e invalida caché.
// Casos: Éxito, usuario no encontrado, datos inválidos, error DB.
func CreateRestaurant(c *gin.Context) {
	username := c.GetString("user")
	user, err := database.UserRepo.FindByUsername(username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuario no encontrado en la DB", "buscado": username})
		return
	}

	var input struct {
		Name        string `json:"name" binding:"required"`
		Address     string `json:"address"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	restaurant := models.Restaurant{
		Name:        input.Name,
		Address:     input.Address,
		Description: input.Description,
		AdminID:     user.ID,
	}

	if err := database.RestaurantRepo.Create(&restaurant); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creando restaurante"})
		return
	}

	cache.Delete("restaurants:all")
	cache.DeleteByPattern("search:restaurants:*")

	c.JSON(http.StatusCreated, restaurant)
}

// UpdateRestaurant: Actualiza un restaurante si el usuario es el dueño.
// Entradas: Contexto Gin con :id y JSON de campos a actualizar.
// Salidas: 200 OK + restaurante actualizado, 400/403/404 con error.
// Funcionalidad: Verifica propiedad via repositorio, actualiza en DB e invalida caché.
// Casos: Éxito, no propietario, no encontrado, error DB.
func UpdateRestaurant(c *gin.Context) {
	restaurant, ok := getOwnedRestaurant(c)
	if !ok {
		return
	}

	var input struct {
		Name        string `json:"name"`
		Address     string `json:"address"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Solo actualizar campos enviados
	if input.Name != "" {
		restaurant.Name = input.Name
	}
	if input.Address != "" {
		restaurant.Address = input.Address
	}
	if input.Description != "" {
		restaurant.Description = input.Description
	}

	if err := database.RestaurantRepo.Update(restaurant); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error actualizando restaurante"})
		return
	}

	// Invalidar caché individual y lista
	cache.Delete(
		fmt.Sprintf("restaurants:%s", c.Param("id")),
		"restaurants:all",
	)
	cache.DeleteByPattern("search:restaurants:*")

	c.JSON(http.StatusOK, restaurant)
}

// DeleteRestaurant: Elimina un restaurante y sus datos relacionados.
// Entradas: Contexto Gin con :id.
// Salidas: 200 OK + mensaje, 403/404 con error.
// Funcionalidad: Verifica propiedad, borra menús/órdenes/restaurante via repositorio e invalida caché.
// Casos: Éxito, no propietario, no encontrado.
func DeleteRestaurant(c *gin.Context) {
	restaurant, ok := getOwnedRestaurant(c)
	if !ok {
		return
	}

	id := fmt.Sprintf("%v", restaurant.ID)

	// Borrar menús y órdenes relacionadas antes de borrar el restaurante
	database.RestaurantRepo.DeleteMenusByRestaurantIDs([]string{id})
	database.RestaurantRepo.DeleteOrdersByRestaurantIDs([]string{id})
	database.ReservationRepo.DeleteByRestaurantIDs([]string{id})
	database.RestaurantRepo.Delete(id)

	// Invalidar caché individual, lista y menús relacionados
	cache.Delete(
		fmt.Sprintf("restaurants:%s", c.Param("id")),
		"restaurants:all",
	)
	cache.DeleteByPattern(fmt.Sprintf("menus:restaurant:%s:*", id))
	cache.DeleteByPattern("search:restaurants:*")

	c.JSON(http.StatusOK, gin.H{"message": "Restaurante, menús y órdenes asociadas eliminados con éxito"})
}

// getOwnedRestaurant: Verifica que el restaurante pertenece al usuario autenticado.
// Entradas: Contexto Gin con :id.
// Salidas: Puntero a restaurante y booleano de éxito.
// Funcionalidad: Busca restaurante y usuario via repositorio, verifica propiedad.
// Casos: Propietario, no propietario, no encontrado.
func getOwnedRestaurant(c *gin.Context) (*models.Restaurant, bool) {
	id := c.Param("id")

	// Buscar restaurante via repositorio (Postgres o Mongo)
	restaurant, err := database.RestaurantRepo.FindByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Restaurante no encontrado"})
		return nil, false
	}

	// Buscar usuario autenticado via repositorio
	username := c.GetString("user")
	user, err := database.UserRepo.FindByUsername(username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuario no encontrado"})
		return nil, false
	}

	// Verificar que el usuario es el dueño del restaurante
	if restaurant.AdminID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "No eres el admin de este restaurante"})
		return nil, false
	}

	return restaurant, true
}

// GetRestaurantsBySearch: Busca restaurantes por nombre o descripción.
// Entradas: Contexto Gin con query param ?q=texto.
// Salidas: 200 OK + lista filtrada.
// Funcionalidad: Cachea búsquedas frecuentes con TTL de 1h; cache miss → repositorio.
// Nota: Para búsqueda avanzada usar el search-service con ElasticSearch.
// Casos: Cache hit, cache miss → DB, sin resultados.
func GetRestaurantsBySearch(c *gin.Context) {
	q := c.Query("q")
	key := fmt.Sprintf("search:restaurants:%s", q)
	var restaurants []models.Restaurant

	// Cache hit
	if err := cache.Get(key, &restaurants); err == nil {
		c.JSON(http.StatusOK, restaurants)
		return
	}

	// Cache miss → repositorio
	all, err := database.RestaurantRepo.FindAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error en búsqueda"})
		return
	}

	// Filtrar en memoria (búsqueda básica — ElasticSearch maneja la avanzada)
	for _, r := range all {
		if containsInsensitive(r.Name, q) || containsInsensitive(r.Description, q) {
			restaurants = append(restaurants, r)
		}
	}

	cache.Set(key, restaurants, cache.TTLSearch)
	c.JSON(http.StatusOK, restaurants)
}

// invalidateSearchCache: Limpia búsquedas cacheadas de restaurantes.
// Entradas: Ninguna.
// Salidas: Ninguna.
// Funcionalidad: Elimina todas las claves de búsqueda al modificar datos.
// Casos: Sin claves, claves borradas.
func invalidateSearchCache() {
	cache.DeleteByPattern("search:restaurants:*")
}

// containsInsensitive: helper para búsqueda case-insensitive en memoria.
// Entradas: texto (string), query (string).
// Salidas: booleano.
func containsInsensitive(text, query string) bool {
	if query == "" {
		return true
	}
	return strings.Contains(strings.ToLower(text), strings.ToLower(query))
}
