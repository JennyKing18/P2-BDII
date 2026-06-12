package handlers

import (
	"P1-BASESII/internal/database"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetUsers: Obtiene la lista de todos los usuarios.
// Entradas: Contexto Gin.
// Salidas: Respuesta JSON con lista de usuarios.
// Funcionalidad: Consulta la base de datos para obtener todos los usuarios.
// Casos: Éxito con lista, error de base de datos.
func GetUsers(c *gin.Context) {
	users, err := database.UserRepo.FindAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener usuarios"})
		return
	}
	c.JSON(http.StatusOK, users)
}

// GetUser: Obtiene un usuario por ID.
// Entradas: Contexto Gin con parámetro ID.
// Salidas: Respuesta JSON con usuario o error.
// Funcionalidad: Busca el usuario en la base de datos por ID.
// Casos: Usuario encontrado, usuario no encontrado.
func GetUser(c *gin.Context) {
	user, err := database.UserRepo.FindByID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Usuario no encontrado"})
		return
	}
	c.JSON(http.StatusOK, user)
}

// UpdateUser: Actualiza un usuario por ID.
// Entradas: Contexto Gin con ID y datos JSON.
// Salidas: Respuesta JSON con usuario actualizado o error.
// Funcionalidad: Actualiza los campos del usuario en la base de datos.
// Casos: Éxito, usuario no encontrado, datos inválidos.
func UpdateUser(c *gin.Context) {
	user, err := database.UserRepo.FindByID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Usuario no encontrado"})
		return
	}

	var input struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Role     string `json:"role"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Username != "" {
		user.Username = input.Username
	}
	if input.Email != "" {
		user.Email = input.Email
	}
	if input.Role != "" {
		user.Role = input.Role
	}

	if err := database.UserRepo.Update(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar usuario"})
		return
	}
	c.JSON(http.StatusOK, user)
}

// DeleteUser: Elimina un usuario por ID.
// Entradas: Contexto Gin con parámetro ID.
// Salidas: Respuesta JSON con mensaje de éxito o error.
// Funcionalidad: Elimina el usuario de la base de datos.
// Casos: Éxito, usuario no encontrado.
func DeleteUser(c *gin.Context) {
	id := c.Param("id")

	user, err := database.UserRepo.FindByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Usuario no encontrado"})
		return
	}

	// 1. Borrar reservas del usuario como cliente
	if err := database.ReservationRepo.DeleteByUserID(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error limpiando reservas"})
		return
	}

	// 2. Cascada si es admin
	if user.Role == "admin" {
		restaurantIDs, err := database.RestaurantRepo.GetIDsByAdminID(id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error obteniendo restaurantes"})
			return
		}
		if len(restaurantIDs) > 0 {
			database.RestaurantRepo.DeleteMenusByRestaurantIDs(restaurantIDs)
			database.RestaurantRepo.DeleteOrdersByRestaurantIDs(restaurantIDs)
			database.RestaurantRepo.DeleteReservationsByRestaurantIDs(restaurantIDs)
			for _, rid := range restaurantIDs {
				database.RestaurantRepo.Delete(rid)
			}
		}
	}

	// 3. Borrar usuario
	if err := database.UserRepo.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error eliminando usuario"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Usuario, sus restaurantes, órdenes y todas sus reservaciones han sido eliminados"})
}
