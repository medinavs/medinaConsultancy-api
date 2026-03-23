package models

import (
	"time"

	"gorm.io/gorm"
)

type Subscription struct {
	ID                 uint           `gorm:"primarykey" json:"id"`
	UserID             uint           `gorm:"uniqueIndex;not null" json:"user_id"`
	User               User           `gorm:"foreignKey:UserID" json:"-"`
	Status             string         `gorm:"default:active;not null" json:"status"` // active, cancelled, suspended, past_due
	MPCustomerID       string         `gorm:"not null" json:"mp_customer_id"`
	MPCardID           string         `gorm:"not null" json:"mp_card_id"`
	IntegrationToken   string         `gorm:"uniqueIndex;not null" json:"-"`
	CurrentPeriodStart time.Time      `json:"current_period_start"`
	CurrentPeriodEnd   time.Time      `json:"current_period_end"`
	CancelledAt        *time.Time     `json:"cancelled_at"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"-"`
}
