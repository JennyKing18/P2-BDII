package mongo

import (
	"P1-BASESII/internal/models"
	"context"
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// ─── GENERADOR DE IDs ─────────────────────────────────────────────────────────
// MongoDB no tiene autoincrement. Usamos un contador atómico por proceso.
// En producción real se usaría una colección counters con FindOneAndUpdate.
var idCounter uint64 = uint64(time.Now().UnixNano() / 1e6) // seed con milisegundos

// nextID: Genera un ID uint único incremental thread-safe.
// Entradas: Ninguna.
// Salidas: uint.
func nextID() uint {
	return uint(atomic.AddUint64(&idCounter, 1))
}

// parseID: Convierte un id string → uint para queries por _id.
// Entradas: id string.
// Salidas: uint, error si el formato es inválido.
func parseID(id string) (uint, error) {
	n, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("id inválido '%s': %w", id, err)
	}
	return uint(n), nil
}

// ─── USER ────────────────────────────────────────────────────────────────────

type UserRepo struct{ col *mongo.Collection }

func NewUserRepo(col *mongo.Collection) *UserRepo { return &UserRepo{col} }

// Create: Inserta un nuevo usuario en MongoDB.
// Entradas: *models.User con datos del usuario.
// Salidas: error si falla el insert.
// Funcionalidad: Genera ID único, convierte a documento, inserta, escribe ID de vuelta al dominio.
// Casos: Éxito, error de conexión/duplicado.
func (r *UserRepo) Create(u *models.User) error {
	if u.ID == 0 {
		u.ID = nextID()
	}
	doc := userDocumentFromDomain(u)
	_, err := r.col.InsertOne(context.Background(), doc)
	return err
}

// FindAll: Obtiene todos los usuarios.
// Entradas: Ninguna.
// Salidas: []models.User, error.
func (r *UserRepo) FindAll() ([]models.User, error) {
	cursor, err := r.col.Find(context.Background(), bson.M{})
	if err != nil {
		return nil, err
	}
	var docs []userDocument
	if err := cursor.All(context.Background(), &docs); err != nil {
		return nil, err
	}
	list := make([]models.User, len(docs))
	for i, d := range docs {
		list[i] = *d.toDomain()
	}
	return list, nil
}

// FindByID: Busca un usuario por ID.
// Entradas: id string (se convierte a uint para el query).
// Salidas: *models.User, error si no existe o ID inválido.
func (r *UserRepo) FindByID(id string) (*models.User, error) {
	uid, err := parseID(id)
	if err != nil {
		return nil, err
	}
	var doc userDocument
	if err := r.col.FindOne(context.Background(), bson.M{"_id": uid}).Decode(&doc); err != nil {
		return nil, err
	}
	return doc.toDomain(), nil
}

// FindByUsername: Busca un usuario por username.
// Entradas: username string.
// Salidas: *models.User, error si no existe.
func (r *UserRepo) FindByUsername(username string) (*models.User, error) {
	var doc userDocument
	if err := r.col.FindOne(context.Background(), bson.M{"username": username}).Decode(&doc); err != nil {
		return nil, err
	}
	return doc.toDomain(), nil
}

// Update: Actualiza todos los campos de un usuario.
// Entradas: *models.User con ID y campos actualizados.
// Salidas: error si falla.
func (r *UserRepo) Update(u *models.User) error {
	doc := userDocumentFromDomain(u)
	_, err := r.col.UpdateOne(context.Background(),
		bson.M{"_id": u.ID},
		bson.M{"$set": doc},
	)
	return err
}

// Delete: Elimina un usuario por ID.
// Entradas: id string.
// Salidas: error si falla.
// Nota: MongoDB no tiene soft-delete nativo; se hace hard delete.
// Para soft-delete se podría setear deleted_at en lugar de eliminar.
func (r *UserRepo) Delete(id string) error {
	uid, err := parseID(id)
	if err != nil {
		return err
	}
	_, err = r.col.DeleteOne(context.Background(), bson.M{"_id": uid})
	return err
}

// ─── RESTAURANT ───────────────────────────────────────────────────────────────

type RestaurantRepo struct{ col *mongo.Collection }

func NewRestaurantRepo(col *mongo.Collection) *RestaurantRepo { return &RestaurantRepo{col} }

// Create: Inserta un nuevo restaurante en MongoDB.
// Entradas: *models.Restaurant.
// Salidas: error si falla.
// Funcionalidad: Genera ID, inserta documento, escribe ID y timestamps de vuelta al dominio.
// Casos: Éxito, error DB.
func (r *RestaurantRepo) Create(rest *models.Restaurant) error {
	if rest.ID == 0 {
		rest.ID = nextID()
	}
	doc := restaurantDocumentFromDomain(rest)
	_, err := r.col.InsertOne(context.Background(), doc)
	if err != nil {
		return err
	}
	// Escribir timestamps generados de vuelta al dominio
	rest.CreatedAt = doc.CreatedAt
	rest.UpdatedAt = doc.UpdatedAt
	return nil
}

// FindAll: Obtiene todos los restaurantes.
// Entradas: Ninguna.
// Salidas: []models.Restaurant, error.
func (r *RestaurantRepo) FindAll() ([]models.Restaurant, error) {
	cursor, err := r.col.Find(context.Background(), bson.M{})
	if err != nil {
		return nil, err
	}
	var docs []restaurantDocument
	if err := cursor.All(context.Background(), &docs); err != nil {
		return nil, err
	}
	list := make([]models.Restaurant, len(docs))
	for i, d := range docs {
		list[i] = *d.toDomain()
	}
	return list, nil
}

// FindByID: Busca un restaurante por ID.
// Entradas: id string.
// Salidas: *models.Restaurant, error si no existe.
func (r *RestaurantRepo) FindByID(id string) (*models.Restaurant, error) {
	uid, err := parseID(id)
	if err != nil {
		return nil, err
	}
	var doc restaurantDocument
	if err := r.col.FindOne(context.Background(), bson.M{"_id": uid}).Decode(&doc); err != nil {
		return nil, err
	}
	return doc.toDomain(), nil
}

// FindByAdminID: Busca restaurantes por ID de administrador.
// Entradas: adminID string.
// Salidas: []models.Restaurant, error.
func (r *RestaurantRepo) FindByAdminID(adminID string) ([]models.Restaurant, error) {
	uid, err := parseID(adminID)
	if err != nil {
		return nil, err
	}
	cursor, err := r.col.Find(context.Background(), bson.M{"admin_id": uid})
	if err != nil {
		return nil, err
	}
	var docs []restaurantDocument
	if err := cursor.All(context.Background(), &docs); err != nil {
		return nil, err
	}
	list := make([]models.Restaurant, len(docs))
	for i, d := range docs {
		list[i] = *d.toDomain()
	}
	return list, nil
}

// Update: Actualiza todos los campos de un restaurante.
// Entradas: *models.Restaurant con ID y campos actualizados.
// Salidas: error si falla.
func (r *RestaurantRepo) Update(rest *models.Restaurant) error {
	doc := restaurantDocumentFromDomain(rest)
	_, err := r.col.UpdateOne(context.Background(),
		bson.M{"_id": rest.ID},
		bson.M{"$set": doc},
	)
	return err
}

// Delete: Elimina un restaurante por ID.
// Entradas: id string.
// Salidas: error si falla.
func (r *RestaurantRepo) Delete(id string) error {
	uid, err := parseID(id)
	if err != nil {
		return err
	}
	_, err = r.col.DeleteOne(context.Background(), bson.M{"_id": uid})
	return err
}

// GetIDsByAdminID: Obtiene IDs de restaurantes de un admin como strings.
// Entradas: adminID string.
// Salidas: []string de IDs, error.
// Funcionalidad: Usado para cascading deletes desde el handler.
func (r *RestaurantRepo) GetIDsByAdminID(adminID string) ([]string, error) {
	uid, err := parseID(adminID)
	if err != nil {
		return nil, err
	}
	cursor, err := r.col.Find(context.Background(), bson.M{"admin_id": uid})
	if err != nil {
		return nil, err
	}
	var docs []restaurantDocument
	if err := cursor.All(context.Background(), &docs); err != nil {
		return nil, err
	}
	ids := make([]string, len(docs))
	for i, d := range docs {
		ids[i] = fmt.Sprintf("%d", d.ID)
	}
	return ids, nil
}

// DeleteMenusByRestaurantIDs: Elimina menús de una lista de restaurantes.
// Entradas: []string de IDs de restaurantes.
// Salidas: error si falla.
// Nota: En Mongo esta lógica se delega al MenuItemRepo; aquí es no-op intencional.
// El handler debe llamar database.MenuRepo.DeleteByRestaurantIDs() directamente.
func (r *RestaurantRepo) DeleteMenusByRestaurantIDs(ids []string) error {
	uids := parseIDs(ids)
	_, err := r.col.Database().Collection("menu_items").DeleteMany(
		context.Background(),
		bson.M{"restaurant_id": bson.M{"$in": uids}},
	)
	return err
}

// DeleteOrdersByRestaurantIDs: Elimina orders de los restaurantes dados.
func (r *RestaurantRepo) DeleteOrdersByRestaurantIDs(ids []string) error {
	uids := parseIDs(ids)
	_, err := r.col.Database().Collection("orders").DeleteMany(
		context.Background(),
		bson.M{"restaurant_id": bson.M{"$in": uids}},
	)
	return err
}

// DeleteReservationsByRestaurantIDs: Elimina reservations de los restaurantes dados.
func (r *RestaurantRepo) DeleteReservationsByRestaurantIDs(ids []string) error {
	uids := parseIDs(ids)
	_, err := r.col.Database().Collection("reservations").DeleteMany(
		context.Background(),
		bson.M{"restaurant_id": bson.M{"$in": uids}},
	)
	return err
}

// parseIDs: Helper para convertir []string → []uint ignorando errores.
func parseIDs(ids []string) []uint {
	uids := make([]uint, 0, len(ids))
	for _, id := range ids {
		if uid, err := parseID(id); err == nil {
			uids = append(uids, uid)
		}
	}
	return uids
}

// ─── MENU ITEM ────────────────────────────────────────────────────────────────

type MenuItemRepo struct{ col *mongo.Collection }

func NewMenuItemRepo(col *mongo.Collection) *MenuItemRepo { return &MenuItemRepo{col} }

// Create: Inserta un ítem de menú en MongoDB.
// Entradas: *models.MenuItem.
// Salidas: error si falla.
func (r *MenuItemRepo) Create(m *models.MenuItem) error {
	if m.ID == 0 {
		m.ID = nextID()
	}
	doc := menuItemDocumentFromDomain(m)
	_, err := r.col.InsertOne(context.Background(), doc)
	if err != nil {
		return err
	}
	m.CreatedAt = doc.CreatedAt
	m.UpdatedAt = doc.UpdatedAt
	return nil
}

// FindAll: Obtiene todos los ítems de menú.
// Entradas: Ninguna.
// Salidas: []models.MenuItem, error.
func (r *MenuItemRepo) FindAll() ([]models.MenuItem, error) {
	cursor, err := r.col.Find(context.Background(), bson.M{})
	if err != nil {
		return nil, err
	}
	var docs []menuItemDocument
	if err := cursor.All(context.Background(), &docs); err != nil {
		return nil, err
	}
	list := make([]models.MenuItem, len(docs))
	for i, d := range docs {
		list[i] = *d.toDomain()
	}
	return list, nil
}

// FindByID: Busca un ítem de menú por ID.
// Entradas: id string.
// Salidas: *models.MenuItem, error si no existe.
func (r *MenuItemRepo) FindByID(id string) (*models.MenuItem, error) {
	uid, err := parseID(id)
	if err != nil {
		return nil, err
	}
	var doc menuItemDocument
	if err := r.col.FindOne(context.Background(), bson.M{"_id": uid}).Decode(&doc); err != nil {
		return nil, err
	}
	return doc.toDomain(), nil
}

// FindByRestaurantID: Busca ítems de menú por restaurante.
// Entradas: restaurantID string.
// Salidas: []models.MenuItem, error.
func (r *MenuItemRepo) FindByRestaurantID(restaurantID string) ([]models.MenuItem, error) {
	uid, err := parseID(restaurantID)
	if err != nil {
		return nil, err
	}
	cursor, err := r.col.Find(context.Background(), bson.M{"restaurant_id": uid})
	if err != nil {
		return nil, err
	}
	var docs []menuItemDocument
	if err := cursor.All(context.Background(), &docs); err != nil {
		return nil, err
	}
	list := make([]models.MenuItem, len(docs))
	for i, d := range docs {
		list[i] = *d.toDomain()
	}
	return list, nil
}

// Update: Actualiza un ítem de menú.
// Entradas: *models.MenuItem con ID y campos actualizados.
// Salidas: error si falla.
func (r *MenuItemRepo) Update(m *models.MenuItem) error {
	doc := menuItemDocumentFromDomain(m)
	_, err := r.col.UpdateOne(context.Background(),
		bson.M{"_id": m.ID},
		bson.M{"$set": doc},
	)
	return err
}

// Delete: Elimina un ítem de menú por ID.
// Entradas: id string.
// Salidas: error si falla.
func (r *MenuItemRepo) Delete(id string) error {
	uid, err := parseID(id)
	if err != nil {
		return err
	}
	_, err = r.col.DeleteOne(context.Background(), bson.M{"_id": uid})
	return err
}

// NullifyOrderMenuItemID: Pone menu_item_id en 0 en órdenes que referencian el ítem.
// Entradas: menuItemID string.
// Salidas: error si falla.
// Funcionalidad: Evita referencias huérfanas al borrar un ítem de menú.
func (r *MenuItemRepo) NullifyOrderMenuItemID(menuItemID string) error {
	uid, err := parseID(menuItemID)
	if err != nil {
		return err
	}
	_, err = r.col.Database().Collection("orders").UpdateMany(
		context.Background(),
		bson.M{"menu_item_id": uid},
		bson.M{"$set": bson.M{"menu_item_id": 0}},
	)
	return err
}

// ─── RESERVATION ─────────────────────────────────────────────────────────────

type ReservationRepo struct{ col *mongo.Collection }

func NewReservationRepo(col *mongo.Collection) *ReservationRepo { return &ReservationRepo{col} }

// Create: Inserta una reserva en MongoDB.
// Entradas: *models.Reservation.
// Salidas: error si falla.
func (r *ReservationRepo) Create(res *models.Reservation) error {
	if res.ID == 0 {
		res.ID = nextID()
	}
	doc := reservationDocumentFromDomain(res)
	_, err := r.col.InsertOne(context.Background(), doc)
	if err != nil {
		return err
	}
	res.CreatedAt = doc.CreatedAt
	return nil
}

// FindByID: Busca una reserva por ID.
// Entradas: id string.
// Salidas: *models.Reservation, error si no existe.
func (r *ReservationRepo) FindByID(id string) (*models.Reservation, error) {
	uid, err := parseID(id)
	if err != nil {
		return nil, err
	}
	var doc reservationDocument
	if err := r.col.FindOne(context.Background(), bson.M{"_id": uid}).Decode(&doc); err != nil {
		return nil, err
	}
	return doc.toDomain(), nil
}

// FindByUserID: Busca reservas por usuario.
// Entradas: userID string.
// Salidas: []models.Reservation, error.
func (r *ReservationRepo) FindByUserID(userID string) ([]models.Reservation, error) {
	uid, err := parseID(userID)
	if err != nil {
		return nil, err
	}
	cursor, err := r.col.Find(context.Background(), bson.M{"user_id": uid})
	if err != nil {
		return nil, err
	}
	var docs []reservationDocument
	if err := cursor.All(context.Background(), &docs); err != nil {
		return nil, err
	}
	list := make([]models.Reservation, len(docs))
	for i, d := range docs {
		list[i] = *d.toDomain()
	}
	return list, nil
}

// FindByRestaurantIDs: Busca reservas de una lista de restaurantes.
// Entradas: []string de IDs de restaurantes.
// Salidas: []models.Reservation, error.
func (r *ReservationRepo) FindByRestaurantIDs(restaurantIDs []string) ([]models.Reservation, error) {
	uids := make([]uint, 0, len(restaurantIDs))
	for _, id := range restaurantIDs {
		uid, err := parseID(id)
		if err != nil {
			return nil, err
		}
		uids = append(uids, uid)
	}
	cursor, err := r.col.Find(context.Background(), bson.M{"restaurant_id": bson.M{"$in": uids}})
	if err != nil {
		return nil, err
	}
	var docs []reservationDocument
	if err := cursor.All(context.Background(), &docs); err != nil {
		return nil, err
	}
	list := make([]models.Reservation, len(docs))
	for i, d := range docs {
		list[i] = *d.toDomain()
	}
	return list, nil
}

// UpdateDate: Actualiza la fecha de una reserva.
// Entradas: id string, date time.Time.
// Salidas: error si falla.
func (r *ReservationRepo) UpdateDate(id string, date time.Time) error {
	uid, err := parseID(id)
	if err != nil {
		return err
	}
	_, err = r.col.UpdateOne(context.Background(),
		bson.M{"_id": uid},
		bson.M{"$set": bson.M{"date": date, "updated_at": time.Now()}},
	)
	return err
}

// UpdateStatus: Actualiza el estado de una reserva.
// Entradas: id string, status string.
// Salidas: error si falla.
func (r *ReservationRepo) UpdateStatus(id string, status string) error {
	uid, err := parseID(id)
	if err != nil {
		return err
	}
	_, err = r.col.UpdateOne(context.Background(),
		bson.M{"_id": uid},
		bson.M{"$set": bson.M{"status": status, "updated_at": time.Now()}},
	)
	return err
}

// Delete: Elimina una reserva por ID.
// Entradas: id string.
// Salidas: error si falla.
func (r *ReservationRepo) Delete(id string) error {
	uid, err := parseID(id)
	if err != nil {
		return err
	}
	_, err = r.col.DeleteOne(context.Background(), bson.M{"_id": uid})
	return err
}

// DeleteByUserID: Elimina todas las reservas de un usuario.
// Entradas: userID string.
// Salidas: error si falla.
func (r *ReservationRepo) DeleteByUserID(userID string) error {
	uid, err := parseID(userID)
	if err != nil {
		return err
	}
	_, err = r.col.DeleteMany(context.Background(), bson.M{"user_id": uid})
	return err
}

// DeleteByRestaurantIDs: Elimina reservas de una lista de restaurantes.
// Entradas: []string de IDs de restaurantes.
// Salidas: error si falla.
func (r *ReservationRepo) DeleteByRestaurantIDs(ids []string) error {
	uids := make([]uint, 0, len(ids))
	for _, id := range ids {
		uid, err := parseID(id)
		if err != nil {
			return err
		}
		uids = append(uids, uid)
	}
	_, err := r.col.DeleteMany(context.Background(), bson.M{"restaurant_id": bson.M{"$in": uids}})
	return err
}

// ─── ORDER ────────────────────────────────────────────────────────────────────

type OrderRepo struct {
	col     *mongo.Collection
	menuCol *mongo.Collection
}

func NewOrderRepo(col, menuCol *mongo.Collection) *OrderRepo { return &OrderRepo{col, menuCol} }

// Create: Inserta una orden en MongoDB.
// Entradas: *models.Order.
// Salidas: error si falla.
// Funcionalidad: Genera ID, inserta, puebla MenuItem manualmente via lookup.
// Casos: Éxito, MenuItem no encontrado (orden se crea igual, MenuItem queda vacío).
func (r *OrderRepo) Create(o *models.Order) error {
	if o.ID == 0 {
		o.ID = nextID()
	}
	doc := orderDocumentFromDomain(o)
	_, err := r.col.InsertOne(context.Background(), doc)
	if err != nil {
		return err
	}
	o.CreatedAt = doc.CreatedAt
	// Poblar MenuItem manualmente (no hay Preload en Mongo)
	var menuDoc menuItemDocument
	if err := r.menuCol.FindOne(context.Background(), bson.M{"_id": o.MenuItemID}).Decode(&menuDoc); err == nil {
		o.MenuItem = *menuDoc.toDomain()
	}
	return nil
}

// FindByID: Busca una orden por ID, incluyendo MenuItem.
// Entradas: id string.
// Salidas: *models.Order, error si no existe.
func (r *OrderRepo) FindByID(id string) (*models.Order, error) {
	uid, err := parseID(id)
	if err != nil {
		return nil, err
	}
	var doc orderDocument
	if err := r.col.FindOne(context.Background(), bson.M{"_id": uid}).Decode(&doc); err != nil {
		return nil, err
	}
	order := doc.toDomain()
	var menuDoc menuItemDocument
	if err := r.menuCol.FindOne(context.Background(), bson.M{"_id": doc.MenuItemID}).Decode(&menuDoc); err == nil {
		order.MenuItem = *menuDoc.toDomain()
	}
	return order, nil
}

// FindByUserID: Busca órdenes por usuario.
// Entradas: userID string.
// Salidas: []models.Order, error.
func (r *OrderRepo) FindByUserID(userID string) ([]models.Order, error) {
	uid, err := parseID(userID)
	if err != nil {
		return nil, err
	}
	cursor, err := r.col.Find(context.Background(), bson.M{"user_id": uid})
	if err != nil {
		return nil, err
	}
	var docs []orderDocument
	if err := cursor.All(context.Background(), &docs); err != nil {
		return nil, err
	}
	return r.populateMenuItems(docs), nil
}

// FindByUserOrRestaurants: Busca órdenes de un usuario o de sus restaurantes.
// Entradas: userID string, restaurantIDs []string.
// Salidas: []models.Order, error.
func (r *OrderRepo) FindByUserOrRestaurants(userID string, restaurantIDs []string) ([]models.Order, error) {
	uid, err := parseID(userID)
	if err != nil {
		return nil, err
	}
	uids := make([]uint, 0, len(restaurantIDs))
	for _, id := range restaurantIDs {
		rid, err := parseID(id)
		if err != nil {
			return nil, err
		}
		uids = append(uids, rid)
	}
	filter := bson.M{
		"$or": []bson.M{
			{"user_id": uid},
			{"restaurant_id": bson.M{"$in": uids}},
		},
	}
	cursor, err := r.col.Find(context.Background(), filter)
	if err != nil {
		return nil, err
	}
	var docs []orderDocument
	if err := cursor.All(context.Background(), &docs); err != nil {
		return nil, err
	}
	return r.populateMenuItems(docs), nil
}

// UpdateStatus: Actualiza el estado de una orden.
// Entradas: id string, status string.
// Salidas: error si falla.
func (r *OrderRepo) UpdateStatus(id string, status string) error {
	uid, err := parseID(id)
	if err != nil {
		return err
	}
	_, err = r.col.UpdateOne(context.Background(),
		bson.M{"_id": uid},
		bson.M{"$set": bson.M{"status": status, "updated_at": time.Now()}},
	)
	return err
}

// DeleteByRestaurantIDs: Elimina órdenes de una lista de restaurantes.
// Entradas: []string de IDs de restaurantes.
// Salidas: error si falla.
func (r *OrderRepo) DeleteByRestaurantIDs(ids []string) error {
	uids := make([]uint, 0, len(ids))
	for _, id := range ids {
		uid, err := parseID(id)
		if err != nil {
			return err
		}
		uids = append(uids, uid)
	}
	_, err := r.col.DeleteMany(context.Background(), bson.M{"restaurant_id": bson.M{"$in": uids}})
	return err
}

// populateMenuItems: Helper interno que agrega MenuItem a una lista de órdenes.
// Entradas: []orderDocument.
// Salidas: []models.Order con MenuItem poblado donde sea posible.
// Funcionalidad: Hace un lookup por cada orden; si el MenuItem no existe, la orden se devuelve igual.
func (r *OrderRepo) populateMenuItems(docs []orderDocument) []models.Order {
	list := make([]models.Order, len(docs))
	for i, d := range docs {
		order := d.toDomain()
		var menuDoc menuItemDocument
		if err := r.menuCol.FindOne(context.Background(), bson.M{"_id": d.MenuItemID}).Decode(&menuDoc); err == nil {
			order.MenuItem = *menuDoc.toDomain()
		}
		list[i] = *order
	}
	return list
}
