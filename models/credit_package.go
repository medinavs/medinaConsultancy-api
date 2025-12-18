package models

import (
	"time"

	"gorm.io/gorm"
)

type CreditPackage struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	Name        string         `gorm:"not null" json:"name"`
	Credits     int            `gorm:"not null" json:"credits"`
	Price       string         `gorm:"not null" json:"price"`
	Description string         `json:"description"`
	Active      bool           `gorm:"default:true" json:"active"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}
