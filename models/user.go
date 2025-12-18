package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	Email     string         `gorm:"uniqueIndex;not null" json:"email"`
	Password  string         `gorm:"not null" json:"-"`
	Credits   int            `gorm:"default:0" json:"credits"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
