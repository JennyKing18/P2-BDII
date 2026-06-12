package models

import "time"

// BaseModel: Campos comunes a todas las entidades del dominio.
// No contiene tags de infraestructura (ni gorm ni bson).
// Cada implementación de repositorio mapea estos campos a su propio formato interno.
type BaseModel struct {
	ID        uint
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time // puntero para distinguir "no borrado" (nil) de valor cero
}

// User: Representa un usuario del sistema.
// Roles posibles: "admin", "client".
type User struct {
	BaseModel
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"-"` // nunca se serializa al cliente
	Role     string `json:"role"`
}

// Restaurant: Representa un restaurante registrado en el sistema.
// AdminID referencia al User dueño del restaurante.
type Restaurant struct {
	BaseModel
	Name        string `json:"name"`
	Address     string `json:"address"`
	Description string `json:"description"`
	AdminID     uint   `json:"admin_id"`
}

// MenuItem: Representa un ítem del menú de un restaurante.
// RestaurantID referencia al Restaurant al que pertenece.
type MenuItem struct {
	BaseModel
	RestaurantID uint    `json:"restaurant_id"`
	Name         string  `json:"name"`
	Category     string  `json:"category"` // "vegan", "meat", "drink", etc.
	Description  string  `json:"description"`
	Price        float64 `json:"price"`
}

// Reservation: Representa una reserva de mesa.
// Status posibles: "confirmed", "cancelled", "pending".
type Reservation struct {
	BaseModel
	UserID       uint      `json:"user_id"`
	RestaurantID uint      `json:"restaurant_id"`
	Date         time.Time `json:"date"`
	Status       string    `json:"status"`
}

// Order: Representa un pedido realizado por un usuario.
// Status posibles: "pending", "ready_for_pickup", "completed", "cancelled".
// MenuItem se puebla manualmente en el repositorio (no hay ORM que lo haga en Mongo).
type Order struct {
	BaseModel
	UserID       uint     `json:"user_id"`
	MenuItemID   uint     `json:"menu_item_id"`
	RestaurantID uint     `json:"restaurant_id"`
	Total        float64  `json:"total"`
	Status       string   `json:"status"`
	MenuItem     MenuItem `json:"menu_item"` // poblado por el repo, no por el cliente
}
