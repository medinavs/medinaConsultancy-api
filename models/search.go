package models

import (
	"time"

	"gorm.io/gorm"
)

type Search struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	UserID    uint           `gorm:"not null" json:"user_id"`
	User      User           `gorm:"foreignKey:UserID" json:"-"`
	SearchID  string         `gorm:"uniqueIndex;not null" json:"search_id"`
	Query     string         `gorm:"not null" json:"query"`
	City      string         `gorm:"not null" json:"city"`
	BucketURL string         `gorm:"not null" json:"bucket_url"`
	FileName  string         `gorm:"not null" json:"file_name"`
	Results   int            `gorm:"default:0" json:"results"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
