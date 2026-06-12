package handlers

import (
	"P1-BASESII/internal/cache"
	"P1-BASESII/internal/database"
	"P1-BASESII/internal/models"
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

var FailMode string
var errSimulated = errors.New("db error")

func setupRouter() (*gin.Engine, *miniredis.Miniredis) {
	gin.SetMode(gin.TestMode)
	mr, _ := miniredis.Run()
	cache.Client = redis.NewClient(&redis.Options{Addr: mr.Addr()})
	r := gin.Default()
	r.Use(func(c *gin.Context) {
		u := c.GetHeader("X-User")
		if u != "" {
			c.Set("user", u)
		}
		c.Next()
	})
	return r, mr
}

// ==========================================
// MOCKS ADAPTADOS AL COMPORTAMIENTO REAL DEL CÓDIGO
// ==========================================
type MockUserRepo struct{}

func (m *MockUserRepo) Create(u *models.User) error {
	if FailMode == "create" {
		return errSimulated
	}
	return nil
}
func (m *MockUserRepo) FindAll() ([]models.User, error) {
	if FailMode == "findall" {
		return nil, errSimulated
	}
	return []models.User{{BaseModel: models.BaseModel{ID: 1}}}, nil
}
func (m *MockUserRepo) FindByID(id string) (*models.User, error) {
	if FailMode == "findbyid" {
		return nil, errSimulated
	}
	if id == "99" {
		return nil, errors.New("not found")
	}
	if id == "2" {
		return &models.User{BaseModel: models.BaseModel{ID: 2}, Username: "client_user", Role: "client"}, nil
	}
	return &models.User{BaseModel: models.BaseModel{ID: 1}, Username: "admin_user", Role: "admin"}, nil
}
func (m *MockUserRepo) FindByUsername(username string) (*models.User, error) {
	if FailMode == "user_lookup" {
		return nil, errSimulated
	}
	if username == "invalid" {
		return nil, errors.New("not found")
	}
	if username == "client_user" {
		return &models.User{BaseModel: models.BaseModel{ID: 2}, Username: "client_user", Role: "client"}, nil
	}
	return &models.User{BaseModel: models.BaseModel{ID: 1}, Username: "admin_user", Role: "admin"}, nil
}
func (m *MockUserRepo) Update(u *models.User) error {
	if FailMode == "update" {
		return errSimulated
	}
	return nil
}
func (m *MockUserRepo) Delete(id string) error {
	if FailMode == "delete" {
		return errSimulated
	}
	return nil
}

type MockRestaurantRepo struct{}

func (m *MockRestaurantRepo) Create(r *models.Restaurant) error {
	if FailMode == "create" {
		return errSimulated
	}
	return nil
}
func (m *MockRestaurantRepo) FindAll() ([]models.Restaurant, error) {
	if FailMode == "findall" {
		return nil, errSimulated
	}
	return []models.Restaurant{{BaseModel: models.BaseModel{ID: 1}, Name: "Burger", AdminID: 1}}, nil
}
func (m *MockRestaurantRepo) FindByID(id string) (*models.Restaurant, error) {
	if FailMode == "findbyid" {
		return nil, errSimulated
	}
	if id == "99" {
		return nil, errors.New("not found")
	}
	if id == "3" {
		return &models.Restaurant{BaseModel: models.BaseModel{ID: 3}, AdminID: 999}, nil
	}
	return &models.Restaurant{BaseModel: models.BaseModel{ID: 1}, AdminID: 1}, nil
}
func (m *MockRestaurantRepo) FindByAdminID(adminID string) ([]models.Restaurant, error) {
	return nil, nil
}
func (m *MockRestaurantRepo) Update(r *models.Restaurant) error {
	if FailMode == "update" {
		return errSimulated
	}
	return nil
}
func (m *MockRestaurantRepo) Delete(id string) error                               { return nil } // Código ignora error
func (m *MockRestaurantRepo) DeleteMenusByRestaurantIDs(ids []string) error        { return nil }
func (m *MockRestaurantRepo) DeleteOrdersByRestaurantIDs(ids []string) error       { return nil }
func (m *MockRestaurantRepo) DeleteReservationsByRestaurantIDs(ids []string) error { return nil }
func (m *MockRestaurantRepo) GetIDsByAdminID(adminID string) ([]string, error) {
	if FailMode == "cascade_rest" {
		return nil, errSimulated
	}
	if adminID == "1" {
		return []string{"1"}, nil
	}
	return []string{}, nil
}

type MockMenuRepo struct{}

func (m *MockMenuRepo) Create(menu *models.MenuItem) error {
	if FailMode == "create" {
		return errSimulated
	}
	return nil
}
func (m *MockMenuRepo) FindAll() ([]models.MenuItem, error) {
	if FailMode == "findall" {
		return nil, errSimulated
	}
	return []models.MenuItem{{BaseModel: models.BaseModel{ID: 1}}}, nil
}
func (m *MockMenuRepo) FindByID(id string) (*models.MenuItem, error) {
	if FailMode == "findbyid" {
		return nil, errSimulated
	}
	if id == "99" {
		return nil, errors.New("not found")
	}
	if id == "3" {
		return &models.MenuItem{BaseModel: models.BaseModel{ID: 3}, RestaurantID: 3, Price: 15.0}, nil
	}
	return &models.MenuItem{BaseModel: models.BaseModel{ID: 1}, RestaurantID: 1, Price: 15.0}, nil
}
func (m *MockMenuRepo) FindByRestaurantID(id string) ([]models.MenuItem, error) { return nil, nil }
func (m *MockMenuRepo) Update(menu *models.MenuItem) error {
	if FailMode == "update" {
		return errSimulated
	}
	return nil
}
func (m *MockMenuRepo) Delete(id string) error                 { return nil } // Código ignora error
func (m *MockMenuRepo) NullifyOrderMenuItemID(id string) error { return nil }

type MockOrderRepo struct{}

func (m *MockOrderRepo) Create(o *models.Order) error {
	if FailMode == "create" {
		return errSimulated
	}
	return nil
}
func (m *MockOrderRepo) FindByID(id string) (*models.Order, error) {
	if FailMode == "findbyid" {
		return nil, errSimulated
	}
	if id == "99" {
		return nil, errors.New("not found")
	}
	if id == "4" {
		return &models.Order{BaseModel: models.BaseModel{ID: 4}, UserID: 2, RestaurantID: 1, Status: "in_preparation"}, nil
	}
	if id == "3" {
		return &models.Order{BaseModel: models.BaseModel{ID: 3}, UserID: 999, RestaurantID: 3, Status: "pending"}, nil
	}
	return &models.Order{BaseModel: models.BaseModel{ID: 1}, UserID: 2, RestaurantID: 1, Status: "pending"}, nil
}
func (m *MockOrderRepo) FindByUserID(userID string) ([]models.Order, error) {
	if FailMode == "findall" {
		return nil, errSimulated
	}
	return []models.Order{{BaseModel: models.BaseModel{ID: 1}}}, nil
}
func (m *MockOrderRepo) FindByUserOrRestaurants(userID string, restaurantIDs []string) ([]models.Order, error) {
	if FailMode == "findall_admin" {
		return nil, errSimulated
	}
	return []models.Order{{BaseModel: models.BaseModel{ID: 1}}}, nil
}
func (m *MockOrderRepo) UpdateStatus(id string, status string) error {
	if FailMode == "update" {
		return errSimulated
	}
	return nil
}
func (m *MockOrderRepo) DeleteByRestaurantIDs(ids []string) error { return nil }

type MockReservationRepo struct{}

func (m *MockReservationRepo) Create(r *models.Reservation) error {
	if FailMode == "create" {
		return errSimulated
	}
	return nil
}
func (m *MockReservationRepo) FindByID(id string) (*models.Reservation, error) {
	if FailMode == "findbyid" {
		return nil, errSimulated
	}
	if id == "99" {
		return nil, errors.New("not found")
	}
	if id == "3" {
		return &models.Reservation{BaseModel: models.BaseModel{ID: 3}, UserID: 999, RestaurantID: 3}, nil
	}
	return &models.Reservation{BaseModel: models.BaseModel{ID: 1}, UserID: 2, RestaurantID: 1}, nil
}
func (m *MockReservationRepo) FindByUserID(userID string) ([]models.Reservation, error) {
	if FailMode == "findall" {
		return nil, errSimulated
	}
	return []models.Reservation{{BaseModel: models.BaseModel{ID: 1}}}, nil
}
func (m *MockReservationRepo) FindByRestaurantIDs(ids []string) ([]models.Reservation, error) {
	return []models.Reservation{{BaseModel: models.BaseModel{ID: 1}}}, nil
}                                                                          // Ignora error
func (m *MockReservationRepo) UpdateDate(id string, date time.Time) error  { return nil } // Ignora error
func (m *MockReservationRepo) UpdateStatus(id string, status string) error { return nil } // Ignora error
func (m *MockReservationRepo) Delete(id string) error {
	if FailMode == "delete" {
		return errSimulated
	}
	return nil
}
func (m *MockReservationRepo) DeleteByUserID(userID string) error {
	if FailMode == "cascade_res" {
		return errSimulated
	}
	return nil
}
func (m *MockReservationRepo) DeleteByRestaurantIDs(ids []string) error { return nil }

// ==========================================
// EL TEST MAESTRO DEFINITIVO
// ==========================================
func TestAllHandlersCoverage(t *testing.T) {
	database.UserRepo = &MockUserRepo{}
	database.RestaurantRepo = &MockRestaurantRepo{}
	database.MenuRepo = &MockMenuRepo{}
	database.OrderRepo = &MockOrderRepo{}
	database.ReservationRepo = &MockReservationRepo{}

	// Servidor FALSO para que Elasticsearch responda y las Goroutines avancen
	fakeSearch := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer fakeSearch.Close()
	os.Setenv("SEARCH_SERVICE_URL", fakeSearch.URL)

	r, mr := setupRouter()
	defer mr.Close()

	r.GET("/users", GetUsers)
	r.GET("/users/:id", GetUser)
	r.PUT("/users/:id", UpdateUser)
	r.DELETE("/users/:id", DeleteUser)

	r.GET("/restaurants", GetRestaurants)
	r.GET("/restaurants/:id", GetRestaurant)
	r.POST("/restaurants", CreateRestaurant)
	r.PUT("/restaurants/:id", UpdateRestaurant)
	r.DELETE("/restaurants/:id", DeleteRestaurant)
	r.GET("/search", GetRestaurantsBySearch)

	r.GET("/menus", GetMenuItems)
	r.GET("/menus/:id", GetMenuItem)
	r.POST("/menus", CreateMenuItem)
	r.PUT("/menus/:id", UpdateMenuItem)
	r.DELETE("/menus/:id", DeleteMenuItem)

	r.POST("/orders", CreateOrder)
	r.GET("/orders", GetOrder)
	r.GET("/orders/:id", GetOrder)
	r.PUT("/orders/:id", UpdateOrderStatus)

	r.POST("/reservations", CreateReservation)
	r.GET("/reservations", GetReservations)
	r.GET("/reservations/:id", GetReservation)
	r.PUT("/reservations/:id", UpdateReservation)
	r.DELETE("/reservations/:id", DeleteReservation)

	tests := []struct {
		user       string
		method     string
		url        string
		body       string
		code       int
		failMode   string
		clearCache bool
	}{
		// --- USERS ---
		{"admin_user", "GET", "/users", "", 200, "", true},
		{"admin_user", "GET", "/users", "", 500, "findall", true},
		{"admin_user", "GET", "/users/1", "", 200, "", true},
		{"admin_user", "GET", "/users/99", "", 404, "findbyid", true},
		{"admin_user", "PUT", "/users/1", `{"username":"x", "email":"y", "role":"z"}`, 200, "", true},
		{"admin_user", "PUT", "/users/1", `{mal}`, 400, "", true},
		{"admin_user", "PUT", "/users/1", `{"username":"x"}`, 500, "update", true},
		{"admin_user", "DELETE", "/users/1", "", 200, "", true},
		{"admin_user", "DELETE", "/users/1", "", 500, "cascade_res", true},
		{"admin_user", "DELETE", "/users/1", "", 500, "cascade_rest", true},
		{"admin_user", "DELETE", "/users/2", "", 200, "", true},
		{"admin_user", "DELETE", "/users/1", "", 500, "delete", true},

		// --- RESTAURANTS ---
		{"admin_user", "GET", "/restaurants", "", 200, "", true},
		{"admin_user", "GET", "/restaurants", "", 200, "", false},
		{"admin_user", "GET", "/restaurants", "", 500, "findall", true},
		{"admin_user", "GET", "/restaurants/1", "", 200, "", true},
		{"admin_user", "GET", "/restaurants/1", "", 200, "", false},
		{"admin_user", "GET", "/restaurants/1", "", 404, "findbyid", true},
		{"admin_user", "GET", "/search?q=Burger", "", 200, "", true},
		{"admin_user", "GET", "/search?q=NoMatch", "", 200, "", true}, // Entra al else de busqueda
		{"admin_user", "GET", "/search?q=", "", 200, "", true},
		{"admin_user", "GET", "/search?q=Burger", "", 500, "findall", true},
		{"admin_user", "POST", "/restaurants", `{"name":"x", "address":"y", "description":"z"}`, 201, "", true},
		{"admin_user", "POST", "/restaurants", `{"name":"x"}`, 500, "create", true},
		{"admin_user", "PUT", "/restaurants/99", `{"name":"x"}`, 404, "findbyid", true},
		{"admin_user", "PUT", "/restaurants/1", `{"name":"x", "address":"y", "description":"z"}`, 200, "", true},
		{"admin_user", "PUT", "/restaurants/1", `{"name":"x"}`, 500, "update", true},
		{"admin_user", "DELETE", "/restaurants/1", "", 200, "", true},
		{"admin_user", "DELETE", "/restaurants/99", "", 404, "findbyid", true},

		// --- MENUS ---
		{"admin_user", "GET", "/menus", "", 200, "", true},
		{"admin_user", "GET", "/menus", "", 500, "findall", true},
		{"admin_user", "GET", "/menus/1", "", 200, "", true},
		{"admin_user", "POST", "/menus", `{"restaurant_id":1, "name":"x", "category":"c", "description":"d", "price":10}`, 201, "", true},
		{"admin_user", "POST", "/menus", `{"restaurant_id":1, "name":"x", "price":10}`, 201, "", true}, // Sin descripción
		{"admin_user", "POST", "/menus", `{"restaurant_id":1, "name":"x", "price":10}`, 500, "create", true},
		{"admin_user", "PUT", "/menus/1", `{"name":"x", "category":"c", "description":"d", "price":10}`, 200, "", true},
		{"admin_user", "PUT", "/menus/1", `{"name":"x"}`, 500, "update", true},
		{"admin_user", "DELETE", "/menus/1", "", 200, "", true},
		{"invalid", "DELETE", "/menus/1", "", 401, "user_lookup", true},

		// --- ORDERS ---
		{"client_user", "POST", "/orders", `{"menu_item_id":1}`, 201, "", true},
		{"client_user", "POST", "/orders", `{"menu_item_id":1}`, 500, "create", true},
		{"client_user", "GET", "/orders", "", 200, "", true},
		{"client_user", "GET", "/orders", "", 500, "findall", true},
		{"admin_user", "GET", "/orders", "", 200, "", true},
		{"admin_user", "GET", "/orders", "", 500, "findall_admin", true},
		{"client_user", "PUT", "/orders/1", `{"status":"cancelled"}`, 200, "", true},
		{"client_user", "PUT", "/orders/1", `{"status":"cancelled"}`, 500, "update", true},
		{"client_user", "PUT", "/orders/1", `{"status":"completed"}`, 403, "", true},
		{"admin_user", "PUT", "/orders/1", `{"status":"completed"}`, 200, "", true},

		// --- RESERVATIONS ---
		{"client_user", "POST", "/reservations", `{"restaurant_id":1, "date":"2024-12-25T20:00:00Z"}`, 201, "", true},
		{"client_user", "POST", "/reservations", `{"restaurant_id":1, "date":"2024-12-25T20:00:00Z"}`, 500, "create", true},
		{"client_user", "GET", "/reservations", "", 200, "", true},
		{"client_user", "GET", "/reservations", "", 500, "findall", true},
		{"admin_user", "GET", "/reservations", "", 200, "", true},
		{"client_user", "GET", "/reservations/1", "", 200, "", true},
		{"client_user", "PUT", "/reservations/1", `{"status":"confirmed", "date":"2024-12-25T20:00:00Z"}`, 200, "", true},
		{"client_user", "DELETE", "/reservations/1", "", 200, "", true},
		{"client_user", "DELETE", "/reservations/1", "", 500, "delete", true},

		// --- EDGE CASES Y VALIDACIONES RESTANTES ---

		// Users
		{"admin_user", "PUT", "/users/99", `{"username":"x"}`, 404, "", true},
		{"admin_user", "DELETE", "/users/99", "", 404, "", true},

		// Restaurants
		{"invalid", "POST", "/restaurants", `{"name":"x"}`, 401, "", true},
		{"admin_user", "POST", "/restaurants", `{json_invalido}`, 400, "", true},
		{"client_user", "PUT", "/restaurants/1", `{"name":"x"}`, 403, "", true},
		{"admin_user", "PUT", "/restaurants/1", `{json_invalido}`, 400, "", true},
		{"client_user", "DELETE", "/restaurants/1", "", 403, "", true},

		// Menus
		{"invalid", "POST", "/menus", `{"restaurant_id":1, "name":"x", "price":10}`, 401, "", true},
		{"admin_user", "POST", "/menus", `{json_invalido}`, 400, "", true},
		{"admin_user", "POST", "/menus", `{"restaurant_id":99, "name":"x", "price":10}`, 404, "", true},
		{"client_user", "POST", "/menus", `{"restaurant_id":1, "name":"x", "price":10}`, 403, "", true},
		{"admin_user", "PUT", "/menus/99", `{"name":"x"}`, 404, "", true},
		{"client_user", "PUT", "/menus/1", `{"name":"x"}`, 403, "", true},
		{"admin_user", "PUT", "/menus/1", `{json_invalido}`, 400, "", true},
		{"admin_user", "DELETE", "/menus/99", "", 404, "", true},
		{"client_user", "DELETE", "/menus/1", "", 403, "", true},

		// Orders
		{"invalid", "POST", "/orders", `{"menu_item_id":1}`, 401, "", true},
		{"client_user", "POST", "/orders", `{json_invalido}`, 400, "", true},
		{"client_user", "POST", "/orders", `{"menu_item_id":99}`, 404, "", true},
		{"invalid", "GET", "/orders", "", 401, "", true},
		{"client_user", "PUT", "/orders/1", `{json_invalido}`, 400, "", true},
		{"client_user", "PUT", "/orders/99", `{"status":"cancelled"}`, 404, "", true},
		{"client_user", "PUT", "/orders/4", `{"status":"cancelled"}`, 403, "", true}, // Order 4 está in_preparation

		// Reservations
		{"invalid", "POST", "/reservations", `{"restaurant_id":1, "date":"2024-12-25T20:00:00Z"}`, 401, "", true},
		{"client_user", "POST", "/reservations", `{json_invalido}`, 400, "", true},
		{"client_user", "POST", "/reservations", `{"restaurant_id":99, "date":"2024-12-25T20:00:00Z"}`, 404, "", true},
		{"client_user", "POST", "/reservations", `{"restaurant_id":1, "date":"fecha_mala"}`, 400, "", true},
		{"admin_user", "GET", "/reservations/99", "", 404, "", true},
		{"invalid", "GET", "/reservations/1", "", 401, "", true},
		{"admin_user", "GET", "/reservations/3", "", 403, "", true}, // Reserva 3 pertenece al usuario 999
		{"invalid", "GET", "/reservations", "", 401, "", true},
		{"admin_user", "PUT", "/reservations/99", `{"status":"confirmed"}`, 404, "", true},
		{"invalid", "PUT", "/reservations/1", `{"status":"confirmed"}`, 401, "", true},
		{"admin_user", "PUT", "/reservations/1", `{"status":"confirmed"}`, 403, "", true}, // Admin no puede editar reservas ajenas
		{"client_user", "PUT", "/reservations/1", `{json_invalido}`, 400, "", true},
		{"client_user", "PUT", "/reservations/1", `{"date":"fecha_mala"}`, 400, "", true},
		{"admin_user", "DELETE", "/reservations/99", "", 404, "", true},
		{"invalid", "DELETE", "/reservations/1", "", 401, "", true},
		{"admin_user", "DELETE", "/reservations/1", "", 403, "", true},
	}

	for _, tt := range tests {
		FailMode = tt.failMode
		if tt.clearCache {
			cache.DeleteByPattern("*")
		}

		req, _ := http.NewRequest(tt.method, tt.url, bytes.NewBuffer([]byte(tt.body)))
		req.Header.Set("Content-Type", "application/json")
		if tt.user != "" {
			req.Header.Set("X-User", tt.user)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != tt.code {
			t.Errorf("Error en %s %s (User: %s, FailMode: %s): esperaba %d, obtuvo %d", tt.method, tt.url, tt.user, tt.failMode, tt.code, w.Code)
		}
	}

	// Dormimos la prueba MEDIO SEGUNDO para que las Goroutines de ElasticSearch
	// terminen de ejecutarse y sumen a la cobertura total del código.
	time.Sleep(500 * time.Millisecond)
}
