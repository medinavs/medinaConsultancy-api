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

	if err := DB.AutoMigrate(&models.User{}, &models.CreditPackage{}, &models.Order{}, &models.Search{}); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	seedCreditPackages()

	log.Println("Database connection established successfully.")
}

func seedCreditPackages() {
	packages := []models.CreditPackage{
		{Name: "Starter", Credits: 10, Price: "7.90", Description: "10 credits for basic usage", Active: true},
		{Name: "Standard", Credits: 50, Price: "29.90", Description: "50 credits for basic usage", Active: true},
		{Name: "Advanced", Credits: 150, Price: "79.90", Description: "150 credits for regular usage", Active: true},
		{Name: "Professional", Credits: 300, Price: "149.90", Description: "300 credits for heavy usage", Active: true},
	}

	for _, pkg := range packages {
		var existing models.CreditPackage
		if err := DB.Where("name = ?", pkg.Name).First(&existing).Error; err != nil {
			DB.Create(&pkg)
		}
	}
}
