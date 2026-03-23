package models

import (
	"time"

	"gorm.io/gorm"
)

type Invoice struct {
	ID             uint           `gorm:"primarykey" json:"id"`
	SubscriptionID uint           `gorm:"index;not null" json:"subscription_id"`
	Subscription   Subscription   `gorm:"foreignKey:SubscriptionID" json:"-"`
	UserID         uint           `gorm:"index;not null" json:"user_id"`
	BillingMonth   string         `gorm:"not null" json:"billing_month"` // "2026-03" format
	QueryCount     int            `gorm:"not null" json:"query_count"`
	UnitPrice      string         `gorm:"not null" json:"unit_price"`
	TotalAmount    string         `gorm:"not null" json:"total_amount"`
	Status         string         `gorm:"default:pending;not null" json:"status"` // pending, paid, failed, void
	MercadoPagoID  string         `json:"mercado_pago_id"`
	PaidAt         *time.Time     `json:"paid_at"`
	Attempts       int            `gorm:"default:0" json:"attempts"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}
