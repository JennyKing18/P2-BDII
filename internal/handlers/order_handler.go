package handlers

import (
	"P1-BASESII/internal/cache"
	"P1-BASESII/internal/database"
	"P1-BASESII/internal/models"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// CreateOrder: Crea una nueva orden.
// Entradas: Contexto Gin con JSON {menu_item_id} y usuario autenticado.
// Salidas: 201 Created + orden, 400/401/404/500 con error.
// Funcionalidad: Valida usuario y plato, crea orden en DB e invalida caché del usuario.
// Casos: Éxito, usuario no encontrado, plato inexistente, error DB.
func CreateOrder(c *gin.Context) {
	username := c.GetString("user")
	user, err := database.UserRepo.FindByUsername(username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuario no encontrado"})
		return
	}

	var input struct {
		MenuItemID uint `json:"menu_item_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Debe proporcionar un menu_item_id válido"})
		return
	}

	item, err := database.MenuRepo.FindByID(fmt.Sprintf("%d", input.MenuItemID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "El plato seleccionado no existe"})
		return
	}

	order := &models.Order{
		UserID:       user.ID,
		MenuItemID:   item.ID,
		RestaurantID: item.RestaurantID,
		Total:        item.Price,
		Status:       "pending",
	}
	if err := database.OrderRepo.Create(order); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al procesar el pedido"})
		return
	}

	cache.Delete(fmt.Sprintf("orders:user:%d", user.ID))
	c.JSON(http.StatusCreated, order)
}

// GetOrder: Lista órdenes del usuario autenticado.
// Entradas: Contexto Gin con usuario autenticado.
// Salidas: 200 OK + lista de órdenes, 401 si usuario no existe.
// Funcionalidad: Cache hit → devuelve órdenes; miss → consulta DB según rol y cachea.
// Casos: Cache hit, cache miss admin (ve sus órdenes + las de sus restaurantes), cache miss cliente (solo las suyas).
func GetOrder(c *gin.Context) {
	username := c.GetString("user")
	user, err := database.UserRepo.FindByUsername(username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuario no encontrado"})
		return
	}

	key := fmt.Sprintf("orders:user:%d", user.ID)
	var orders []models.Order

	if err := cache.Get(key, &orders); err == nil {
		c.JSON(http.StatusOK, orders)
		return
	}

	userIDStr := fmt.Sprintf("%d", user.ID)
	if user.Role == "admin" {
		restaurantIDs, _ := database.RestaurantRepo.GetIDsByAdminID(userIDStr)
		orders, err = database.OrderRepo.FindByUserOrRestaurants(userIDStr, restaurantIDs)
	} else {
		orders, err = database.OrderRepo.FindByUserID(userIDStr)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener órdenes"})
		return
	}

	cache.Set(key, orders, cache.TTLOrders)
	c.JSON(http.StatusOK, orders)
}

// UpdateOrderStatus: Cambia el estado de una orden.
// Entradas: Contexto Gin con :id, JSON {status} y usuario autenticado.
// Salidas: 200 OK + mensaje, 400/403/404 con error.
// Funcionalidad: Valida permisos, actualiza estado en DB e invalida caché del usuario y del admin del restaurante.
// Casos: Éxito, sin permisos, orden no encontrada, estado inválido.
func UpdateOrderStatus(c *gin.Context) {
	id := c.Param("id")
	username := c.GetString("user")

	var input struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cuerpo de petición inválido"})
		return
	}

	order, err := database.OrderRepo.FindByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Orden no encontrada"})
		return
	}

	currentUser, _ := database.UserRepo.FindByUsername(username)

	isAdminOfThisRestaurant := false
	if currentUser.Role == "admin" {
		restaurantID := fmt.Sprintf("%d", order.RestaurantID)
		rest, err := database.RestaurantRepo.FindByID(restaurantID)
		if err == nil && rest.AdminID == currentUser.ID {
			isAdminOfThisRestaurant = true
		}
	}

	isOwner := order.UserID == currentUser.ID

	if input.Status == "cancelled" {
		if isOwner && order.Status != "pending" && !isAdminOfThisRestaurant {
			c.JSON(http.StatusForbidden, gin.H{"error": "No puedes cancelar una orden que ya está en preparación"})
			return
		}
	} else if (input.Status == "ready_for_pickup" || input.Status == "completed") && !isAdminOfThisRestaurant {
		c.JSON(http.StatusForbidden, gin.H{"error": "Solo el restaurante puede marcar pedidos como listos o completados"})
		return
	}

	if err := database.OrderRepo.UpdateStatus(id, input.Status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error actualizando estado"})
		return
	}

	cache.Delete(
		fmt.Sprintf("orders:user:%d", order.UserID),
		fmt.Sprintf("orders:user:%d", currentUser.ID),
	)

	c.JSON(http.StatusOK, gin.H{"message": "Estado actualizado con éxito", "new_status": input.Status})
}
