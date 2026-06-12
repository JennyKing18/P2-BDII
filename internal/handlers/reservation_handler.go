package handlers

import (
	"P1-BASESII/internal/database"
	"P1-BASESII/internal/models"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// POST /reservations — cualquier usuario autenticado puede crear
func CreateReservation(c *gin.Context) {
	username := c.GetString("user")
	user, err := database.UserRepo.FindByUsername(username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuario no encontrado"})
		return
	}

	var input struct {
		RestaurantID uint   `json:"restaurant_id" binding:"required"`
		Date         string `json:"date" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	restaurantID := fmt.Sprintf("%d", input.RestaurantID)
	if _, err := database.RestaurantRepo.FindByID(restaurantID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Restaurante no encontrado"})
		return
	}

	reservation := &models.Reservation{
		UserID:       user.ID,
		RestaurantID: input.RestaurantID,
		Status:       "pending",
	}
	if err := reservation.Date.UnmarshalJSON([]byte(`"` + input.Date + `"`)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Formato de fecha inválido, usa: 2024-12-25T20:00:00Z"})
		return
	}

	if err := database.ReservationRepo.Create(reservation); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al crear reserva"})
		return
	}
	c.JSON(http.StatusCreated, reservation)
}

// GET /reservations/:id — solo el creador o el dueño del restaurante
func GetReservation(c *gin.Context) {
	reservation, err := database.ReservationRepo.FindByID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Reserva no encontrada"})
		return
	}

	username := c.GetString("user")
	user, err := database.UserRepo.FindByUsername(username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuario no encontrado"})
		return
	}

	restaurantID := fmt.Sprintf("%d", reservation.RestaurantID)
	restaurant, _ := database.RestaurantRepo.FindByID(restaurantID)

	if reservation.UserID != user.ID && restaurant.AdminID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "No tienes permiso para ver esta reserva"})
		return
	}

	c.JSON(http.StatusOK, reservation)
}

// GET /reservations — el usuario ve sus propias, el dueño ve las de su restaurante
func GetReservations(c *gin.Context) {
	username := c.GetString("user")
	user, err := database.UserRepo.FindByUsername(username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuario no encontrado"})
		return
	}

	userIDStr := fmt.Sprintf("%d", user.ID)

	// Reservas donde el usuario es el cliente (aplica a todos los roles)
	personal, err := database.ReservationRepo.FindByUserID(userIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error obteniendo reservas"})
		return
	}

	// Cliente normal: solo ve sus propias reservas
	if user.Role != "admin" {
		c.JSON(http.StatusOK, personal)
		return
	}

	// Admin: combina sus reservas personales + reservas hechas EN sus restaurantes.
	// Se deduplica por ID porque si el admin reservó en su propio restaurante,
	// aparecería en ambas queries.
	seen := make(map[uint]struct{})
	result := make([]models.Reservation, 0)

	// 1. Sus reservas como cliente (en cualquier restaurante)
	for _, r := range personal {
		seen[r.ID] = struct{}{}
		result = append(result, r)
	}

	// 2. Reservas de clientes en los restaurantes que este admin posee
	restaurantIDs, _ := database.RestaurantRepo.GetIDsByAdminID(userIDStr)
	if len(restaurantIDs) > 0 {
		fromRestaurants, _ := database.ReservationRepo.FindByRestaurantIDs(restaurantIDs)
		for _, r := range fromRestaurants {
			if _, exists := seen[r.ID]; !exists {
				result = append(result, r)
			}
		}
	}

	c.JSON(http.StatusOK, result)
}

// PUT /reservations/:id — solo el creador puede editar
func UpdateReservation(c *gin.Context) {
	id := c.Param("id")

	reservation, err := database.ReservationRepo.FindByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Reserva no encontrada"})
		return
	}

	username := c.GetString("user")
	user, err := database.UserRepo.FindByUsername(username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuario no encontrado"})
		return
	}

	if reservation.UserID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Solo el creador puede editar esta reserva"})
		return
	}

	var input struct {
		Date   string `json:"date"`
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Date != "" {
		var t models.Reservation
		if err := t.Date.UnmarshalJSON([]byte(`"` + input.Date + `"`)); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Formato de fecha inválido"})
			return
		}
		database.ReservationRepo.UpdateDate(id, t.Date)
		reservation.Date = t.Date
	}
	if input.Status != "" {
		database.ReservationRepo.UpdateStatus(id, input.Status)
		reservation.Status = input.Status
	}

	c.JSON(http.StatusOK, reservation)
}

// DELETE /reservations/:id — solo el creador puede eliminar
func DeleteReservation(c *gin.Context) {
	id := c.Param("id")

	reservation, err := database.ReservationRepo.FindByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Reserva no encontrada"})
		return
	}

	username := c.GetString("user")
	user, err := database.UserRepo.FindByUsername(username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuario no encontrado"})
		return
	}

	if reservation.UserID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Solo el creador puede eliminar esta reserva"})
		return
	}

	if err := database.ReservationRepo.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error eliminando reserva"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Reserva eliminada"})
}
