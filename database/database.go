package database

import (
	"log"
	"medina-consultancy-api/models"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func ConnectWithDatabase() {
	databaseConnection := os.Getenv("DATABASE_URL")
	if databaseConnection == "" {
		databaseConnection = "host=localhost user=postgres password=password dbname=medina_consultancy port=5432 sslmode=disable"
	}

	var err error
	DB, err = gorm.Open(postgres.New(postgres.Config{
		DSN: databaseConnection,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	sqlDB, err := DB.DB()
	if err != nil {
		log.Fatalf("Failed to get raw DB from GORM: %v", err)
	}

	sqlDB.SetMaxIdleConns(2)
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetConnMaxLifetime(time.Minute * 5)
	sqlDB.SetConnMaxIdleTime(time.Minute * 1)

	if err := sqlDB.Ping(); err != nil {
		log.Fatalf("Error pinging the database: %v", err)
	}

	if err := DB.AutoMigrate(&models.User{}); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	log.Println("Database connection established successfully.")
}
