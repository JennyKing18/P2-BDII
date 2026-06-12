package postgres

import (
	"P1-BASESII/internal/models"
	"time"

	"gorm.io/gorm"
)

// ─── ENTIDADES INTERNAS (solo para Postgres / GORM) ──────────────────────────
// Estas structs NUNCA salen del paquete postgres.
// Los handlers y la lógica de negocio solo ven models.*

// ─── USER ────────────────────────────────────────────────────────────────────

// userEntity: Representación interna de User para GORM.
// Contiene todos los tags de infraestructura de Postgres.
type userEntity struct {
	gorm.Model
	Username string `gorm:"unique;not null"`
	Email    string `gorm:"unique;not null"`
	Password string
	Role     string
}

func (userEntity) TableName() string { return "users" }

// toDomain: Convierte userEntity → models.User (dominio puro).
// Entradas: userEntity.
// Salidas: *models.User sin referencias a GORM.
func (e *userEntity) toDomain() *models.User {
	var deletedAt *time.Time
	if e.DeletedAt.Valid {
		t := e.DeletedAt.Time
		deletedAt = &t
	}
	return &models.User{
		BaseModel: models.BaseModel{
			ID:        e.ID,
			CreatedAt: e.CreatedAt,
			UpdatedAt: e.UpdatedAt,
			DeletedAt: deletedAt,
		},
		Username: e.Username,
		Email:    e.Email,
		Password: e.Password,
		Role:     e.Role,
	}
}

// userEntityFromDomain: Convierte models.User → userEntity para operaciones GORM.
// Entradas: *models.User.
// Salidas: userEntity lista para Insert/Update.
func userEntityFromDomain(u *models.User) userEntity {
	e := userEntity{
		Username: u.Username,
		Email:    u.Email,
		Password: u.Password,
		Role:     u.Role,
	}
	if u.ID != 0 {
		e.Model.ID = u.ID
	}
	return e
}

// ─── RESTAURANT ───────────────────────────────────────────────────────────────

// restaurantEntity: Representación interna de Restaurant para GORM.
type restaurantEntity struct {
	gorm.Model
	Name        string `gorm:"not null"`
	Address     string
	Description string
	AdminID     uint
}

func (restaurantEntity) TableName() string { return "restaurants" }

// toDomain: Convierte restaurantEntity → models.Restaurant.
func (e *restaurantEntity) toDomain() *models.Restaurant {
	var deletedAt *time.Time
	if e.DeletedAt.Valid {
		t := e.DeletedAt.Time
		deletedAt = &t
	}
	return &models.Restaurant{
		BaseModel: models.BaseModel{
			ID:        e.ID,
			CreatedAt: e.CreatedAt,
			UpdatedAt: e.UpdatedAt,
			DeletedAt: deletedAt,
		},
		Name:        e.Name,
		Address:     e.Address,
		Description: e.Description,
		AdminID:     e.AdminID,
	}
}

// restaurantEntityFromDomain: Convierte models.Restaurant → restaurantEntity.
func restaurantEntityFromDomain(r *models.Restaurant) restaurantEntity {
	e := restaurantEntity{
		Name:        r.Name,
		Address:     r.Address,
		Description: r.Description,
		AdminID:     r.AdminID,
	}
	if r.ID != 0 {
		e.Model.ID = r.ID
	}
	return e
}

// ─── MENU ITEM ────────────────────────────────────────────────────────────────

// menuItemEntity: Representación interna de MenuItem para GORM.
type menuItemEntity struct {
	gorm.Model
	RestaurantID uint
	Name         string `gorm:"not null"`
	Category     string
	Description  string
	Price        float64 `gorm:"not null"`
}

func (menuItemEntity) TableName() string { return "menu_items" }

// toDomain: Convierte menuItemEntity → models.MenuItem.
func (e *menuItemEntity) toDomain() *models.MenuItem {
	var deletedAt *time.Time
	if e.DeletedAt.Valid {
		t := e.DeletedAt.Time
		deletedAt = &t
	}
	return &models.MenuItem{
		BaseModel: models.BaseModel{
			ID:        e.ID,
			CreatedAt: e.CreatedAt,
			UpdatedAt: e.UpdatedAt,
			DeletedAt: deletedAt,
		},
		RestaurantID: e.RestaurantID,
		Name:         e.Name,
		Category:     e.Category,
		Description:  e.Description,
		Price:        e.Price,
	}
}

// menuItemEntityFromDomain: Convierte models.MenuItem → menuItemEntity.
func menuItemEntityFromDomain(m *models.MenuItem) menuItemEntity {
	e := menuItemEntity{
		RestaurantID: m.RestaurantID,
		Name:         m.Name,
		Category:     m.Category,
		Description:  m.Description,
		Price:        m.Price,
	}
	if m.ID != 0 {
		e.Model.ID = m.ID
	}
	return e
}

// ─── RESERVATION ─────────────────────────────────────────────────────────────

// reservationEntity: Representación interna de Reservation para GORM.
type reservationEntity struct {
	gorm.Model
	UserID       uint
	RestaurantID uint
	Date         time.Time
	Status       string
}

func (reservationEntity) TableName() string { return "reservations" }

// toDomain: Convierte reservationEntity → models.Reservation.
func (e *reservationEntity) toDomain() *models.Reservation {
	var deletedAt *time.Time
	if e.DeletedAt.Valid {
		t := e.DeletedAt.Time
		deletedAt = &t
	}
	return &models.Reservation{
		BaseModel: models.BaseModel{
			ID:        e.ID,
			CreatedAt: e.CreatedAt,
			UpdatedAt: e.UpdatedAt,
			DeletedAt: deletedAt,
		},
		UserID:       e.UserID,
		RestaurantID: e.RestaurantID,
		Date:         e.Date,
		Status:       e.Status,
	}
}

// reservationEntityFromDomain: Convierte models.Reservation → reservationEntity.
func reservationEntityFromDomain(r *models.Reservation) reservationEntity {
	e := reservationEntity{
		UserID:       r.UserID,
		RestaurantID: r.RestaurantID,
		Date:         r.Date,
		Status:       r.Status,
	}
	if r.ID != 0 {
		e.Model.ID = r.ID
	}
	return e
}

// ─── ORDER ────────────────────────────────────────────────────────────────────

// orderEntity: Representación interna de Order para GORM.
// MenuItem se carga via Preload, no es una FK real en la entidad.
type orderEntity struct {
	gorm.Model
	UserID       uint
	MenuItemID   uint
	RestaurantID uint
	Total        float64
	Status       string
	MenuItem     menuItemEntity `gorm:"foreignKey:MenuItemID"`
}

func (orderEntity) TableName() string { return "orders" }

// toDomain: Convierte orderEntity → models.Order, incluyendo MenuItem anidado.
func (e *orderEntity) toDomain() *models.Order {
	var deletedAt *time.Time
	if e.DeletedAt.Valid {
		t := e.DeletedAt.Time
		deletedAt = &t
	}
	return &models.Order{
		BaseModel: models.BaseModel{
			ID:        e.ID,
			CreatedAt: e.CreatedAt,
			UpdatedAt: e.UpdatedAt,
			DeletedAt: deletedAt,
		},
		UserID:       e.UserID,
		MenuItemID:   e.MenuItemID,
		RestaurantID: e.RestaurantID,
		Total:        e.Total,
		Status:       e.Status,
		MenuItem:     *e.MenuItem.toDomain(),
	}
}

// orderEntityFromDomain: Convierte models.Order → orderEntity.
func orderEntityFromDomain(o *models.Order) orderEntity {
	e := orderEntity{
		UserID:       o.UserID,
		MenuItemID:   o.MenuItemID,
		RestaurantID: o.RestaurantID,
		Total:        o.Total,
		Status:       o.Status,
	}
	if o.ID != 0 {
		e.Model.ID = o.ID
	}
	return e
}

// ─── EXPORTS PARA AUTOMIGRATE ─────────────────────────────────────────────────
// Las entidades son privadas al paquete postgres (lowercase).
// Estas funciones exponen punteros vacíos para que database.Connect()
// pueda pasarlos a GORM AutoMigrate sin romper el encapsulamiento.
// Solo database.go debe llamar estas funciones.

func ExportUserEntity() interface{} { return &userEntity{} }

func ExportRestaurantEntity() interface{} { return &restaurantEntity{} }

func ExportMenuItemEntity() interface{} { return &menuItemEntity{} }

func ExportReservationEntity() interface{} { return &reservationEntity{} }

func ExportOrderEntity() interface{} { return &orderEntity{} }
