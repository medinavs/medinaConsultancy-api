package models

import (
	"time"

	"gorm.io/gorm"
)

type IntegrationQuery struct {
	ID             uint           `gorm:"primarykey" json:"id"`
	SubscriptionID uint           `gorm:"index;not null" json:"subscription_id"`
	Subscription   Subscription   `gorm:"foreignKey:SubscriptionID" json:"-"`
	UserID         uint           `gorm:"index;not null" json:"user_id"`
	SearchID       string         `gorm:"uniqueIndex;not null" json:"search_id"`
	Query          string         `gorm:"not null" json:"query"`
	City           string         `gorm:"not null" json:"city"`
	Results        int            `gorm:"default:0" json:"results"`
	BucketURL      string         `json:"bucket_url"`
	BillingMonth   string         `gorm:"index;not null" json:"billing_month"` // "2026-03" format
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}
