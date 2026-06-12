package database

import (
	pg "P1-BASESII/internal/repository/postgres"
	"fmt"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

// Connect: Conecta a la base de datos PostgreSQL y migra las tablas.
// Entradas: Ninguna (lee variables de entorno DB_HOST, DB_USER, etc.).
// Salidas: Ninguna.
// Funcionalidad: Abre conexión GORM-AutoMigrate Postgres (no los modelos de dominio).
// Casos: Éxito, panic si no puede conectar, log si falla migración.
func Connect() {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		os.Getenv("DB_HOST"), os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"), os.Getenv("DB_PORT"))

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("Error al conectar a la base de datos")
	}

	// AutoMigrate usa las entidades internas de Postgres (con tags gorm:).
	// Los modelos de dominio (models.*) ya no tienen tags gorm y no sirven aquí.
	err = db.AutoMigrate(
		pg.ExportUserEntity(),
		pg.ExportRestaurantEntity(),
		pg.ExportMenuItemEntity(),
		pg.ExportReservationEntity(),
		pg.ExportOrderEntity(),
	)
	if err != nil {
		fmt.Println("Error migrando tablas:", err)
	}

	DB = db
}
