package models

import (
	"time"

	"gorm.io/gorm"
)

type Order struct {
	ID                uint           `gorm:"primarykey" json:"id"`
	UserID            uint           `gorm:"not null" json:"user_id"`
	User              User           `gorm:"foreignKey:UserID" json:"-"`
	CreditPackageID   uint           `gorm:"not null" json:"credit_package_id"`
	CreditPackage     CreditPackage  `gorm:"foreignKey:CreditPackageID" json:"credit_package"`
	Amount            string         `gorm:"not null" json:"amount"`
	Status            string         `gorm:"default:pending" json:"status"` // pending, approved, rejected, cancelled, in_process
	MercadoPagoID     string         `json:"mercado_pago_id"`
	ExternalReference string         `gorm:"uniqueIndex" json:"external_reference"`
	QRCode            string         `gorm:"type:text" json:"qr_code"`
	QRCodeBase64      string         `gorm:"type:text" json:"qr_code_base64"`
	TicketURL         string         `json:"ticket_url"`
	CreditsAdded      bool           `gorm:"default:false" json:"credits_added"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
}
