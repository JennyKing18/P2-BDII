package mongo

import (
	"P1-BASESII/internal/models"
	"time"
)

// ─── DOCUMENTOS INTERNOS (solo para MongoDB) ──────────────────────────────────
// Estas structs NUNCA salen del paquete mongo.
// Los handlers y la lógica de negocio solo ven models.*
// _id se almacena como uint para mantener compatibilidad con los IDs del dominio.
// En Mongo, el campo _id puede ser cualquier tipo; usamos uint para consistencia.

// ─── USER ────────────────────────────────────────────────────────────────────

// userDocument: Representación interna de User para MongoDB.
// Contiene todos los tags bson necesarios para el driver.
type userDocument struct {
	ID        uint       `bson:"_id"`
	CreatedAt time.Time  `bson:"created_at"`
	UpdatedAt time.Time  `bson:"updated_at"`
	DeletedAt *time.Time `bson:"deleted_at,omitempty"`
	Username  string     `bson:"username"`
	Email     string     `bson:"email"`
	Password  string     `bson:"password"`
	Role      string     `bson:"role"`
}

// toDomain: Convierte userDocument → models.User (dominio puro).
func (d *userDocument) toDomain() *models.User {
	return &models.User{
		BaseModel: models.BaseModel{
			ID:        d.ID,
			CreatedAt: d.CreatedAt,
			UpdatedAt: d.UpdatedAt,
			DeletedAt: d.DeletedAt,
		},
		Username: d.Username,
		Email:    d.Email,
		Password: d.Password,
		Role:     d.Role,
	}
}

// userDocumentFromDomain: Convierte models.User → userDocument para Insert/Update.
// Entradas: *models.User.
// Salidas: userDocument con timestamps inicializados si son cero.
func userDocumentFromDomain(u *models.User) userDocument {
	now := time.Now()
	createdAt := u.CreatedAt
	if createdAt.IsZero() {
		createdAt = now
	}
	return userDocument{
		ID:        u.ID,
		CreatedAt: createdAt,
		UpdatedAt: now,
		DeletedAt: u.DeletedAt,
		Username:  u.Username,
		Email:     u.Email,
		Password:  u.Password,
		Role:      u.Role,
	}
}

// ─── RESTAURANT ───────────────────────────────────────────────────────────────

// restaurantDocument: Representación interna de Restaurant para MongoDB.
type restaurantDocument struct {
	ID          uint       `bson:"_id"`
	CreatedAt   time.Time  `bson:"created_at"`
	UpdatedAt   time.Time  `bson:"updated_at"`
	DeletedAt   *time.Time `bson:"deleted_at,omitempty"`
	Name        string     `bson:"name"`
	Address     string     `bson:"address"`
	Description string     `bson:"description"`
	AdminID     uint       `bson:"admin_id"`
}

// toDomain: Convierte restaurantDocument → models.Restaurant.
func (d *restaurantDocument) toDomain() *models.Restaurant {
	return &models.Restaurant{
		BaseModel: models.BaseModel{
			ID:        d.ID,
			CreatedAt: d.CreatedAt,
			UpdatedAt: d.UpdatedAt,
			DeletedAt: d.DeletedAt,
		},
		Name:        d.Name,
		Address:     d.Address,
		Description: d.Description,
		AdminID:     d.AdminID,
	}
}

// restaurantDocumentFromDomain: Convierte models.Restaurant → restaurantDocument.
func restaurantDocumentFromDomain(r *models.Restaurant) restaurantDocument {
	now := time.Now()
	createdAt := r.CreatedAt
	if createdAt.IsZero() {
		createdAt = now
	}
	return restaurantDocument{
		ID:          r.ID,
		CreatedAt:   createdAt,
		UpdatedAt:   now,
		DeletedAt:   r.DeletedAt,
		Name:        r.Name,
		Address:     r.Address,
		Description: r.Description,
		AdminID:     r.AdminID,
	}
}

// ─── MENU ITEM ────────────────────────────────────────────────────────────────

// menuItemDocument: Representación interna de MenuItem para MongoDB.
type menuItemDocument struct {
	ID           uint       `bson:"_id"`
	CreatedAt    time.Time  `bson:"created_at"`
	UpdatedAt    time.Time  `bson:"updated_at"`
	DeletedAt    *time.Time `bson:"deleted_at,omitempty"`
	RestaurantID uint       `bson:"restaurant_id"`
	Name         string     `bson:"name"`
	Category     string     `bson:"category"`
	Description  string     `bson:"description"`
	Price        float64    `bson:"price"`
}

// toDomain: Convierte menuItemDocument → models.MenuItem.
func (d *menuItemDocument) toDomain() *models.MenuItem {
	return &models.MenuItem{
		BaseModel: models.BaseModel{
			ID:        d.ID,
			CreatedAt: d.CreatedAt,
			UpdatedAt: d.UpdatedAt,
			DeletedAt: d.DeletedAt,
		},
		RestaurantID: d.RestaurantID,
		Name:         d.Name,
		Category:     d.Category,
		Description:  d.Description,
		Price:        d.Price,
	}
}

// menuItemDocumentFromDomain: Convierte models.MenuItem → menuItemDocument.
func menuItemDocumentFromDomain(m *models.MenuItem) menuItemDocument {
	now := time.Now()
	createdAt := m.CreatedAt
	if createdAt.IsZero() {
		createdAt = now
	}
	return menuItemDocument{
		ID:           m.ID,
		CreatedAt:    createdAt,
		UpdatedAt:    now,
		DeletedAt:    m.DeletedAt,
		RestaurantID: m.RestaurantID,
		Name:         m.Name,
		Category:     m.Category,
		Description:  m.Description,
		Price:        m.Price,
	}
}

// ─── RESERVATION ─────────────────────────────────────────────────────────────

// reservationDocument: Representación interna de Reservation para MongoDB.
type reservationDocument struct {
	ID           uint       `bson:"_id"`
	CreatedAt    time.Time  `bson:"created_at"`
	UpdatedAt    time.Time  `bson:"updated_at"`
	DeletedAt    *time.Time `bson:"deleted_at,omitempty"`
	UserID       uint       `bson:"user_id"`
	RestaurantID uint       `bson:"restaurant_id"`
	Date         time.Time  `bson:"date"`
	Status       string     `bson:"status"`
}

// toDomain: Convierte reservationDocument → models.Reservation.
func (d *reservationDocument) toDomain() *models.Reservation {
	return &models.Reservation{
		BaseModel: models.BaseModel{
			ID:        d.ID,
			CreatedAt: d.CreatedAt,
			UpdatedAt: d.UpdatedAt,
			DeletedAt: d.DeletedAt,
		},
		UserID:       d.UserID,
		RestaurantID: d.RestaurantID,
		Date:         d.Date,
		Status:       d.Status,
	}
}

// reservationDocumentFromDomain: Convierte models.Reservation → reservationDocument.
func reservationDocumentFromDomain(r *models.Reservation) reservationDocument {
	now := time.Now()
	createdAt := r.CreatedAt
	if createdAt.IsZero() {
		createdAt = now
	}
	return reservationDocument{
		ID:           r.ID,
		CreatedAt:    createdAt,
		UpdatedAt:    now,
		DeletedAt:    r.DeletedAt,
		UserID:       r.UserID,
		RestaurantID: r.RestaurantID,
		Date:         r.Date,
		Status:       r.Status,
	}
}

// ─── ORDER ────────────────────────────────────────────────────────────────────

// orderDocument: Representación interna de Order para MongoDB.
// MenuItem NO se embebe en el documento; se puebla manualmente en el repo via lookup.
type orderDocument struct {
	ID           uint       `bson:"_id"`
	CreatedAt    time.Time  `bson:"created_at"`
	UpdatedAt    time.Time  `bson:"updated_at"`
	DeletedAt    *time.Time `bson:"deleted_at,omitempty"`
	UserID       uint       `bson:"user_id"`
	MenuItemID   uint       `bson:"menu_item_id"`
	RestaurantID uint       `bson:"restaurant_id"`
	Total        float64    `bson:"total"`
	Status       string     `bson:"status"`
}

// toDomain: Convierte orderDocument → models.Order (sin MenuItem; se añade externamente).
func (d *orderDocument) toDomain() *models.Order {
	return &models.Order{
		BaseModel: models.BaseModel{
			ID:        d.ID,
			CreatedAt: d.CreatedAt,
			UpdatedAt: d.UpdatedAt,
			DeletedAt: d.DeletedAt,
		},
		UserID:       d.UserID,
		MenuItemID:   d.MenuItemID,
		RestaurantID: d.RestaurantID,
		Total:        d.Total,
		Status:       d.Status,
	}
}

// orderDocumentFromDomain: Convierte models.Order → orderDocument.
func orderDocumentFromDomain(o *models.Order) orderDocument {
	now := time.Now()
	createdAt := o.CreatedAt
	if createdAt.IsZero() {
		createdAt = now
	}
	return orderDocument{
		ID:           o.ID,
		CreatedAt:    createdAt,
		UpdatedAt:    now,
		DeletedAt:    o.DeletedAt,
		UserID:       o.UserID,
		MenuItemID:   o.MenuItemID,
		RestaurantID: o.RestaurantID,
		Total:        o.Total,
		Status:       o.Status,
	}
}
