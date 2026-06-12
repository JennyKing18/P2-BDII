package postgres

import (
	"P1-BASESII/internal/models"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// ─── USER ────────────────────────────────────────────────────────────────────

type UserRepo struct{ db *gorm.DB }

func NewUserRepo(db *gorm.DB) *UserRepo { return &UserRepo{db} }

// Create: Inserta un nuevo usuario en Postgres.
// Entradas: *models.User con datos del usuario.
// Salidas: error si falla el insert.
// Funcionalidad: Convierte dominio → entity, inserta, escribe ID generado de vuelta al dominio.
// Casos: Éxito, username/email duplicado.
func (r *UserRepo) Create(u *models.User) error {
	entity := userEntityFromDomain(u)
	if err := r.db.Create(&entity).Error; err != nil {
		return err
	}
	// Escribir ID y timestamps autogenerados de vuelta al dominio
	u.ID = entity.ID
	u.CreatedAt = entity.CreatedAt
	u.UpdatedAt = entity.UpdatedAt
	return nil
}

// FindAll: Obtiene todos los usuarios.
// Entradas: Ninguna.
// Salidas: []models.User, error.
// Funcionalidad: Consulta todos los registros y convierte a dominio.
// Casos: Lista vacía, error DB.
func (r *UserRepo) FindAll() ([]models.User, error) {
	var entities []userEntity
	if err := r.db.Find(&entities).Error; err != nil {
		return nil, err
	}
	list := make([]models.User, len(entities))
	for i, e := range entities {
		list[i] = *e.toDomain()
	}
	return list, nil
}

// FindByID: Busca un usuario por ID.
// Entradas: id string (se convierte internamente a uint para GORM).
// Salidas: *models.User, error si no existe.
// Funcionalidad: First por PK, convierte a dominio.
// Casos: Encontrado, no encontrado.
func (r *UserRepo) FindByID(id string) (*models.User, error) {
	var entity userEntity
	if err := r.db.First(&entity, id).Error; err != nil {
		return nil, err
	}
	return entity.toDomain(), nil
}

// FindByUsername: Busca un usuario por username único.
// Entradas: username string.
// Salidas: *models.User, error si no existe.
func (r *UserRepo) FindByUsername(username string) (*models.User, error) {
	var entity userEntity
	if err := r.db.Where("username = ?", username).First(&entity).Error; err != nil {
		return nil, err
	}
	return entity.toDomain(), nil
}

// Update: Actualiza todos los campos de un usuario.
// Entradas: *models.User con ID y campos actualizados.
// Salidas: error si falla.
func (r *UserRepo) Update(u *models.User) error {
	entity := userEntityFromDomain(u)
	return r.db.Save(&entity).Error
}

// Delete: Soft-delete de un usuario por ID.
// Entradas: id string.
// Salidas: error si falla.
func (r *UserRepo) Delete(id string) error {
	return r.db.Delete(&userEntity{}, "id = ?", id).Error
}

// ─── RESTAURANT ───────────────────────────────────────────────────────────────

type RestaurantRepo struct{ db *gorm.DB }

func NewRestaurantRepo(db *gorm.DB) *RestaurantRepo { return &RestaurantRepo{db} }

// Create: Inserta un nuevo restaurante en Postgres.
// Entradas: *models.Restaurant con datos del restaurante.
// Salidas: error si falla.
// Funcionalidad: Convierte dominio → entity, inserta, escribe ID generado de vuelta.
// Casos: Éxito, error DB.
func (r *RestaurantRepo) Create(rest *models.Restaurant) error {
	entity := restaurantEntityFromDomain(rest)
	if err := r.db.Create(&entity).Error; err != nil {
		return err
	}
	rest.ID = entity.ID
	rest.CreatedAt = entity.CreatedAt
	rest.UpdatedAt = entity.UpdatedAt
	return nil
}

// FindAll: Obtiene todos los restaurantes.
// Entradas: Ninguna.
// Salidas: []models.Restaurant, error.
func (r *RestaurantRepo) FindAll() ([]models.Restaurant, error) {
	var entities []restaurantEntity
	if err := r.db.Find(&entities).Error; err != nil {
		return nil, err
	}
	list := make([]models.Restaurant, len(entities))
	for i, e := range entities {
		list[i] = *e.toDomain()
	}
	return list, nil
}

// FindByID: Busca un restaurante por ID.
// Entradas: id string.
// Salidas: *models.Restaurant, error si no existe.
func (r *RestaurantRepo) FindByID(id string) (*models.Restaurant, error) {
	var entity restaurantEntity
	if err := r.db.First(&entity, id).Error; err != nil {
		return nil, err
	}
	return entity.toDomain(), nil
}

// FindByAdminID: Busca restaurantes por ID de administrador.
// Entradas: adminID string.
// Salidas: []models.Restaurant, error.
func (r *RestaurantRepo) FindByAdminID(adminID string) ([]models.Restaurant, error) {
	var entities []restaurantEntity
	if err := r.db.Where("admin_id = ?", adminID).Find(&entities).Error; err != nil {
		return nil, err
	}
	list := make([]models.Restaurant, len(entities))
	for i, e := range entities {
		list[i] = *e.toDomain()
	}
	return list, nil
}

// Update: Actualiza todos los campos de un restaurante.
// Entradas: *models.Restaurant con ID y campos actualizados.
// Salidas: error si falla.
func (r *RestaurantRepo) Update(rest *models.Restaurant) error {
	entity := restaurantEntityFromDomain(rest)
	return r.db.Save(&entity).Error
}

// Delete: Soft-delete de un restaurante por ID.
// Entradas: id string.
// Salidas: error si falla.
func (r *RestaurantRepo) Delete(id string) error {
	return r.db.Delete(&restaurantEntity{}, "id = ?", id).Error
}

// GetIDsByAdminID: Obtiene IDs de restaurantes de un admin como strings.
// Entradas: adminID string.
// Salidas: []string de IDs, error.
// Funcionalidad: Usado para cascading deletes en handler.
func (r *RestaurantRepo) GetIDsByAdminID(adminID string) ([]string, error) {
	var ids []string
	err := r.db.Model(&restaurantEntity{}).Where("admin_id = ?", adminID).Pluck("id", &ids).Error
	return ids, err
}

// DeleteMenusByRestaurantIDs: Elimina menús de una lista de restaurantes.
// Entradas: []string de IDs de restaurantes.
// Salidas: error si falla.
func (r *RestaurantRepo) DeleteMenusByRestaurantIDs(ids []string) error {
	return r.db.Where("restaurant_id IN ?", ids).Delete(&menuItemEntity{}).Error
}

// DeleteOrdersByRestaurantIDs: Elimina órdenes de una lista de restaurantes.
// Entradas: []string de IDs de restaurantes.
// Salidas: error si falla.
func (r *RestaurantRepo) DeleteOrdersByRestaurantIDs(ids []string) error {
	return r.db.Where("restaurant_id IN ?", ids).Delete(&orderEntity{}).Error
}

// DeleteReservationsByRestaurantIDs: Elimina reservas de una lista de restaurantes.
// Entradas: []string de IDs de restaurantes.
// Salidas: error si falla.
func (r *RestaurantRepo) DeleteReservationsByRestaurantIDs(ids []string) error {
	return r.db.Where("restaurant_id IN ?", ids).Delete(&reservationEntity{}).Error
}

// ─── MENU ITEM ────────────────────────────────────────────────────────────────

type MenuItemRepo struct{ db *gorm.DB }

func NewMenuItemRepo(db *gorm.DB) *MenuItemRepo { return &MenuItemRepo{db} }

// Create: Inserta un nuevo ítem de menú.
// Entradas: *models.MenuItem.
// Salidas: error si falla.
// Funcionalidad: Convierte dominio → entity, inserta, escribe ID y timestamps de vuelta al dominio.
func (r *MenuItemRepo) Create(m *models.MenuItem) error {
	entity := menuItemEntityFromDomain(m)
	if err := r.db.Create(&entity).Error; err != nil {
		return err
	}
	m.ID = entity.ID
	m.CreatedAt = entity.CreatedAt
	m.UpdatedAt = entity.UpdatedAt
	return nil
}

// FindAll: Obtiene todos los ítems de menú.
// Entradas: Ninguna.
// Salidas: []models.MenuItem, error.
func (r *MenuItemRepo) FindAll() ([]models.MenuItem, error) {
	var entities []menuItemEntity
	if err := r.db.Find(&entities).Error; err != nil {
		return nil, err
	}
	list := make([]models.MenuItem, len(entities))
	for i, e := range entities {
		list[i] = *e.toDomain()
	}
	return list, nil
}

// FindByID: Busca un ítem de menú por ID.
// Entradas: id string.
// Salidas: *models.MenuItem, error si no existe.
func (r *MenuItemRepo) FindByID(id string) (*models.MenuItem, error) {
	var entity menuItemEntity
	if err := r.db.First(&entity, id).Error; err != nil {
		return nil, err
	}
	return entity.toDomain(), nil
}

// FindByRestaurantID: Busca ítems de menú por restaurante.
// Entradas: restaurantID string.
// Salidas: []models.MenuItem, error.
func (r *MenuItemRepo) FindByRestaurantID(restaurantID string) ([]models.MenuItem, error) {
	var entities []menuItemEntity
	if err := r.db.Where("restaurant_id = ?", restaurantID).Find(&entities).Error; err != nil {
		return nil, err
	}
	list := make([]models.MenuItem, len(entities))
	for i, e := range entities {
		list[i] = *e.toDomain()
	}
	return list, nil
}

// Update: Actualiza un ítem de menú.
// Entradas: *models.MenuItem con ID y campos actualizados.
// Salidas: error si falla.
func (r *MenuItemRepo) Update(m *models.MenuItem) error {
	entity := menuItemEntityFromDomain(m)
	return r.db.Save(&entity).Error
}

// Delete: Soft-delete de un ítem de menú.
// Entradas: id string.
// Salidas: error si falla.
func (r *MenuItemRepo) Delete(id string) error {
	return r.db.Delete(&menuItemEntity{}, "id = ?", id).Error
}

// NullifyOrderMenuItemID: Pone menu_item_id en NULL en órdenes que referencian el ítem.
// Entradas: menuItemID string.
// Salidas: error si falla.
// Funcionalidad: Evita FK violations al borrar un ítem de menú.
func (r *MenuItemRepo) NullifyOrderMenuItemID(menuItemID string) error {
	return r.db.Model(&orderEntity{}).Where("menu_item_id = ?", menuItemID).Update("menu_item_id", nil).Error
}

// ─── RESERVATION ─────────────────────────────────────────────────────────────

type ReservationRepo struct{ db *gorm.DB }

func NewReservationRepo(db *gorm.DB) *ReservationRepo { return &ReservationRepo{db} }

// Create: Inserta una nueva reserva.
// Entradas: *models.Reservation.
// Salidas: error si falla.
// Funcionalidad: Convierte dominio → entity, inserta, escribe ID y timestamps de vuelta al dominio.
func (r *ReservationRepo) Create(res *models.Reservation) error {
	entity := reservationEntityFromDomain(res)
	if err := r.db.Create(&entity).Error; err != nil {
		return err
	}
	res.ID = entity.ID
	res.CreatedAt = entity.CreatedAt
	res.UpdatedAt = entity.UpdatedAt
	return nil
}

// FindByID: Busca una reserva por ID.
// Entradas: id string.
// Salidas: *models.Reservation, error si no existe.
func (r *ReservationRepo) FindByID(id string) (*models.Reservation, error) {
	var entity reservationEntity
	if err := r.db.First(&entity, id).Error; err != nil {
		return nil, err
	}
	return entity.toDomain(), nil
}

// FindByUserID: Busca reservas por usuario.
// Entradas: userID string.
// Salidas: []models.Reservation, error.
func (r *ReservationRepo) FindByUserID(userID string) ([]models.Reservation, error) {
	var entities []reservationEntity
	if err := r.db.Where("user_id = ?", userID).Find(&entities).Error; err != nil {
		return nil, err
	}
	list := make([]models.Reservation, len(entities))
	for i, e := range entities {
		list[i] = *e.toDomain()
	}
	return list, nil
}

// FindByRestaurantIDs: Busca reservas de una lista de restaurantes.
// Entradas: []string de IDs de restaurantes.
// Salidas: []models.Reservation, error.
func (r *ReservationRepo) FindByRestaurantIDs(restaurantIDs []string) ([]models.Reservation, error) {
	var entities []reservationEntity
	if err := r.db.Where("restaurant_id IN ?", restaurantIDs).Find(&entities).Error; err != nil {
		return nil, err
	}
	list := make([]models.Reservation, len(entities))
	for i, e := range entities {
		list[i] = *e.toDomain()
	}
	return list, nil
}

// UpdateDate: Actualiza la fecha de una reserva.
// Entradas: id string, date time.Time.
// Salidas: error si falla.
func (r *ReservationRepo) UpdateDate(id string, date time.Time) error {
	return r.db.Model(&reservationEntity{}).Where("id = ?", id).Update("date", date).Error
}

// UpdateStatus: Actualiza el estado de una reserva.
// Entradas: id string, status string.
// Salidas: error si falla.
func (r *ReservationRepo) UpdateStatus(id string, status string) error {
	return r.db.Model(&reservationEntity{}).Where("id = ?", id).Update("status", status).Error
}

// Delete: Soft-delete de una reserva.
// Entradas: id string.
// Salidas: error si falla.
func (r *ReservationRepo) Delete(id string) error {
	return r.db.Delete(&reservationEntity{}, "id = ?", id).Error
}

// DeleteByUserID: Elimina todas las reservas de un usuario.
// Entradas: userID string.
// Salidas: error si falla.
func (r *ReservationRepo) DeleteByUserID(userID string) error {
	return r.db.Where("user_id = ?", userID).Delete(&reservationEntity{}).Error
}

// DeleteByRestaurantIDs: Elimina reservas de una lista de restaurantes.
// Entradas: []string de IDs de restaurantes.
// Salidas: error si falla.
func (r *ReservationRepo) DeleteByRestaurantIDs(ids []string) error {
	return r.db.Where("restaurant_id IN ?", ids).Delete(&reservationEntity{}).Error
}

// ─── ORDER ────────────────────────────────────────────────────────────────────

type OrderRepo struct{ db *gorm.DB }

func NewOrderRepo(db *gorm.DB) *OrderRepo { return &OrderRepo{db} }

// Create: Inserta una nueva orden y carga el MenuItem relacionado.
// Entradas: *models.Order.
// Salidas: error si falla.
// Funcionalidad: Crea la orden y hace Preload de MenuItem para devolver el objeto completo.
func (r *OrderRepo) Create(o *models.Order) error {
	entity := orderEntityFromDomain(o)
	if err := r.db.Create(&entity).Error; err != nil {
		return err
	}
	// Recargar con Preload para poblar MenuItem
	if err := r.db.Preload("MenuItem").First(&entity, entity.ID).Error; err != nil {
		return err
	}
	o.ID = entity.ID
	o.CreatedAt = entity.CreatedAt
	o.UpdatedAt = entity.UpdatedAt
	o.MenuItem = *entity.MenuItem.toDomain()
	return nil
}

// FindByID: Busca una orden por ID, incluyendo MenuItem.
// Entradas: id string.
// Salidas: *models.Order, error si no existe.
func (r *OrderRepo) FindByID(id string) (*models.Order, error) {
	var entity orderEntity
	if err := r.db.Preload("MenuItem").First(&entity, id).Error; err != nil {
		return nil, err
	}
	return entity.toDomain(), nil
}

// FindByUserID: Busca órdenes por usuario, incluyendo MenuItem.
// Entradas: userID string.
// Salidas: []models.Order, error.
func (r *OrderRepo) FindByUserID(userID string) ([]models.Order, error) {
	var entities []orderEntity
	if err := r.db.Where("user_id = ?", userID).Preload("MenuItem").Find(&entities).Error; err != nil {
		return nil, err
	}
	list := make([]models.Order, len(entities))
	for i, e := range entities {
		list[i] = *e.toDomain()
	}
	return list, nil
}

// FindByUserOrRestaurants: Busca órdenes de un usuario o de sus restaurantes.
// Entradas: userID string, restaurantIDs []string.
// Salidas: []models.Order, error.
// Funcionalidad: Usado por admins para ver órdenes de sus restaurantes y sus propias órdenes.
func (r *OrderRepo) FindByUserOrRestaurants(userID string, restaurantIDs []string) ([]models.Order, error) {
	var entities []orderEntity
	query := fmt.Sprintf("user_id = '%s'", userID)
	var err error
	if len(restaurantIDs) > 0 {
		query += " OR restaurant_id IN ?"
		err = r.db.Where(query, restaurantIDs).Preload("MenuItem").Find(&entities).Error
	} else {
		err = r.db.Where(query).Preload("MenuItem").Find(&entities).Error
	}
	if err != nil {
		return nil, err
	}
	list := make([]models.Order, len(entities))
	for i, e := range entities {
		list[i] = *e.toDomain()
	}
	return list, nil
}

// UpdateStatus: Actualiza el estado de una orden.
// Entradas: id string, status string.
// Salidas: error si falla.
func (r *OrderRepo) UpdateStatus(id string, status string) error {
	return r.db.Model(&orderEntity{}).Where("id = ?", id).Update("status", status).Error
}

// DeleteByRestaurantIDs: Elimina órdenes de una lista de restaurantes.
// Entradas: []string de IDs de restaurantes.
// Salidas: error si falla.
func (r *OrderRepo) DeleteByRestaurantIDs(ids []string) error {
	return r.db.Where("restaurant_id IN ?", ids).Delete(&orderEntity{}).Error
}
