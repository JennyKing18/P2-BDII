package repository

import (
	"P1-BASESII/internal/models"
	"time"
)

type UserRepository interface {
	Create(u *models.User) error
	FindAll() ([]models.User, error)
	FindByID(id string) (*models.User, error)
	FindByUsername(username string) (*models.User, error)
	Update(u *models.User) error
	Delete(id string) error
}

type RestaurantRepository interface {
	Create(r *models.Restaurant) error
	FindAll() ([]models.Restaurant, error)
	FindByID(id string) (*models.Restaurant, error)
	FindByAdminID(adminID string) ([]models.Restaurant, error)
	Update(r *models.Restaurant) error
	Delete(id string) error
	DeleteMenusByRestaurantIDs(ids []string) error
	DeleteOrdersByRestaurantIDs(ids []string) error
	DeleteReservationsByRestaurantIDs(ids []string) error
	GetIDsByAdminID(adminID string) ([]string, error)
}

type MenuItemRepository interface {
	Create(m *models.MenuItem) error
	FindAll() ([]models.MenuItem, error)
	FindByID(id string) (*models.MenuItem, error)
	FindByRestaurantID(restaurantID string) ([]models.MenuItem, error)
	Update(m *models.MenuItem) error
	Delete(id string) error
	NullifyOrderMenuItemID(menuItemID string) error
}

type ReservationRepository interface {
	Create(r *models.Reservation) error
	FindByID(id string) (*models.Reservation, error)
	FindByUserID(userID string) ([]models.Reservation, error)
	FindByRestaurantIDs(restaurantIDs []string) ([]models.Reservation, error)
	UpdateDate(id string, date time.Time) error
	UpdateStatus(id string, status string) error
	Delete(id string) error
	DeleteByUserID(userID string) error
	DeleteByRestaurantIDs(ids []string) error
}

type OrderRepository interface {
	Create(o *models.Order) error
	FindByID(id string) (*models.Order, error)
	FindByUserID(userID string) ([]models.Order, error)
	FindByUserOrRestaurants(userID string, restaurantIDs []string) ([]models.Order, error)
	UpdateStatus(id string, status string) error
	DeleteByRestaurantIDs(ids []string) error
}
