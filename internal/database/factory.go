package database

import (
	"os"

	"P1-BASESII/internal/repository"
	mg "P1-BASESII/internal/repository/mongo"
	pg "P1-BASESII/internal/repository/postgres"
)

// Repositorios globales — inicializados una vez en main via InitRepositories().
// Los handlers acceden a estos directamente con database.UserRepo, etc.
var (
	UserRepo        repository.UserRepository
	RestaurantRepo  repository.RestaurantRepository
	MenuRepo        repository.MenuItemRepository
	ReservationRepo repository.ReservationRepository
	OrderRepo       repository.OrderRepository
)

// InitRepositories: Inicializa todos los repositorios según DB_DRIVER.
// Entradas: Ninguna (lee DB_DRIVER del entorno).
// Salidas: Ninguna.
// Funcionalidad: Asigna la implementación correcta (Postgres o Mongo) a cada repo.
// Casos: DB_DRIVER=postgres → GORM, DB_DRIVER=mongo → mongo-driver via router.
func InitRepositories() {
	UserRepo = NewUserRepository()
	RestaurantRepo = NewRestaurantRepository()
	MenuRepo = NewMenuItemRepository()
	ReservationRepo = NewReservationRepository()
	OrderRepo = NewOrderRepository()
}

// isMongo: Helper que determina si el driver activo es Mongo.
func isMongo() bool {
	return os.Getenv("DB_DRIVER") == "mongo"
}

// NewUserRepository: Devuelve la implementación de UserRepository según DB_DRIVER.
func NewUserRepository() repository.UserRepository {
	if isMongo() {
		return mg.NewUserRepo(MongoClient.Database(os.Getenv("DB_NAME")).Collection("users"))
	}
	return pg.NewUserRepo(DB)
}

// NewRestaurantRepository: Devuelve la implementación de RestaurantRepository según DB_DRIVER.
func NewRestaurantRepository() repository.RestaurantRepository {
	if isMongo() {
		return mg.NewRestaurantRepo(MongoClient.Database(os.Getenv("DB_NAME")).Collection("restaurants"))
	}
	return pg.NewRestaurantRepo(DB)
}

// NewMenuItemRepository: Devuelve la implementación de MenuItemRepository según DB_DRIVER.
func NewMenuItemRepository() repository.MenuItemRepository {
	if isMongo() {
		return mg.NewMenuItemRepo(MongoClient.Database(os.Getenv("DB_NAME")).Collection("menu_items"))
	}
	return pg.NewMenuItemRepo(DB)
}

// NewReservationRepository: Devuelve la implementación de ReservationRepository según DB_DRIVER.
func NewReservationRepository() repository.ReservationRepository {
	if isMongo() {
		return mg.NewReservationRepo(MongoClient.Database(os.Getenv("DB_NAME")).Collection("reservations"))
	}
	return pg.NewReservationRepo(DB)
}

// NewOrderRepository: Devuelve la implementación de OrderRepository según DB_DRIVER.
func NewOrderRepository() repository.OrderRepository {
	if isMongo() {
		dbName := os.Getenv("DB_NAME")
		return mg.NewOrderRepo(
			MongoClient.Database(dbName).Collection("orders"),
			MongoClient.Database(dbName).Collection("menu_items"),
		)
	}
	return pg.NewOrderRepo(DB)
}
