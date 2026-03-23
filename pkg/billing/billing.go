package billing

import (
	"context"
	"fmt"
	"log"
	"medina-consultancy-api/database"
	"medina-consultancy-api/models"
	mercadopago "medina-consultancy-api/pkg/mercado_pago"
	"time"

	"github.com/google/uuid"
)

func CalculateUnitPrice(queryCount int) float64 {
	switch {
	case queryCount >= 200:
		return 7.00
	case queryCount >= 100:
		return 8.50
	default:
		return 9.90
	}
}

func ProcessMonthlyBilling() error {
	billingMonth := time.Now().AddDate(0, -1, 0).Format("2006-01")
	log.Printf("Processing billing for month: %s", billingMonth)

	var subscriptions []models.Subscription
	if err := database.DB.Where("status IN ?", []string{"active", "past_due"}).Find(&subscriptions).Error; err != nil {
		return fmt.Errorf("failed to fetch subscriptions: %w", err)
	}

	log.Printf("Found %d subscriptions to process", len(subscriptions))

	for _, sub := range subscriptions {
		if err := processSubscriptionBilling(sub, billingMonth); err != nil {
			log.Printf("Failed to process billing for subscription %d: %v", sub.ID, err)
		}
	}

	return nil
}

func processSubscriptionBilling(sub models.Subscription, billingMonth string) error {
	var existingInvoice models.Invoice
	if err := database.DB.Where("subscription_id = ? AND billing_month = ? AND status = ?", sub.ID, billingMonth, "paid").First(&existingInvoice).Error; err == nil {
		log.Printf("Subscription %d already billed for %s, skipping", sub.ID, billingMonth)
		return nil
	}

	var queryCount int64
	if err := database.DB.Model(&models.IntegrationQuery{}).
		Where("subscription_id = ? AND billing_month = ?", sub.ID, billingMonth).
		Count(&queryCount).Error; err != nil {
		return fmt.Errorf("failed to count queries: %w", err)
	}

	if queryCount == 0 {
		log.Printf("Subscription %d has no queries for %s, skipping", sub.ID, billingMonth)
		return nil
	}

	unitPrice := CalculateUnitPrice(int(queryCount))
	totalAmount := unitPrice * float64(queryCount)

	log.Printf("Subscription %d: %d queries x R$%.2f = R$%.2f", sub.ID, queryCount, unitPrice, totalAmount)

	var invoice models.Invoice
	if err := database.DB.Where("subscription_id = ? AND billing_month = ? AND status IN ?", sub.ID, billingMonth, []string{"pending", "failed"}).First(&invoice).Error; err != nil {
		invoice = models.Invoice{
			SubscriptionID: sub.ID,
			UserID:         sub.UserID,
			BillingMonth:   billingMonth,
			QueryCount:     int(queryCount),
			UnitPrice:      fmt.Sprintf("%.2f", unitPrice),
			TotalAmount:    fmt.Sprintf("%.2f", totalAmount),
			Status:         "pending",
		}
		if err := database.DB.Create(&invoice).Error; err != nil {
			return fmt.Errorf("failed to create invoice: %w", err)
		}
	}

	var user models.User
	if err := database.DB.First(&user, sub.UserID).Error; err != nil {
		return fmt.Errorf("failed to fetch user: %w", err)
	}

	externalRef := fmt.Sprintf("invoice_%s_%d", uuid.New().String()[:8], time.Now().Unix())
	ctx := context.Background()

	paymentResp, err := mercadopago.ChargeCard(ctx, mercadopago.CardPaymentRequest{
		Amount:      totalAmount,
		Description: fmt.Sprintf("Medina Consultancy - %s (%d consultas)", billingMonth, queryCount),
		CustomerID:  sub.MPCustomerID,
		CardID:      sub.MPCardID,
		PayerEmail:  user.Email,
		ExternalRef: externalRef,
	})

	invoice.Attempts++

	if err != nil {
		log.Printf("Payment failed for subscription %d: %v", sub.ID, err)
		invoice.Status = "failed"
		database.DB.Save(&invoice)

		if invoice.Attempts >= 3 {
			log.Printf("Suspending subscription %d after %d failed attempts", sub.ID, invoice.Attempts)
			database.DB.Model(&sub).Update("status", "past_due")
		}

		return fmt.Errorf("payment failed: %w", err)
	}

	if paymentResp.Status == "approved" {
		now := time.Now()
		invoice.Status = "paid"
		invoice.MercadoPagoID = paymentResp.ID
		invoice.PaidAt = &now
	} else {
		invoice.Status = "failed"
		invoice.MercadoPagoID = paymentResp.ID
	}

	database.DB.Save(&invoice)

	log.Printf("Subscription %d billed successfully: R$%.2f (status: %s)", sub.ID, totalAmount, invoice.Status)
	return nil
}
