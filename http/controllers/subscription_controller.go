package controllers

import (
	"context"
	"fmt"
	"medina-consultancy-api/database"
	"medina-consultancy-api/models"
	"medina-consultancy-api/pkg/jwt"
	mercadopago "medina-consultancy-api/pkg/mercado_pago"
	"medina-consultancy-api/pkg/response"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type SubscriptionRequest struct {
	CardToken string `json:"card_token" binding:"required"`
}

func CreateSubscription(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		response.SendGinResponse(c, http.StatusUnauthorized, nil, nil, "User not authenticated")
		return
	}

	email, _ := c.Get("email")

	var existing models.Subscription
	if err := database.DB.Where("user_id = ? AND status = ?", userID, "active").First(&existing).Error; err == nil {
		response.SendGinResponse(c, http.StatusConflict, nil, nil, "User already has an active subscription")
		return
	}

	var req SubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendGinResponse(c, http.StatusBadRequest, nil, nil, err.Error())
		return
	}

	ctx := context.Background()

	customerID, err := mercadopago.GetOrCreateCustomer(ctx, email.(string))
	if err != nil {
		response.SendGinResponse(c, http.StatusInternalServerError, nil, nil, fmt.Sprintf("Failed to create payment customer: %v", err))
		return
	}

	cardID, err := mercadopago.SaveCardToCustomer(ctx, customerID, req.CardToken)
	if err != nil {
		response.SendGinResponse(c, http.StatusInternalServerError, nil, nil, fmt.Sprintf("Failed to save card: %v", err))
		return
	}

	now := time.Now()
	periodEnd := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC)

	subscription := models.Subscription{
		UserID:             userID.(uint),
		Status:             "active",
		MPCustomerID:       customerID,
		MPCardID:           cardID,
		IntegrationToken:   "pending", // temporary, updated below
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   periodEnd,
	}

	if err := database.DB.Create(&subscription).Error; err != nil {
		response.SendGinResponse(c, http.StatusInternalServerError, nil, nil, "Failed to create subscription")
		return
	}

	token, err := jwt.GenerateIntegrationToken(userID.(uint), email.(string), subscription.ID)
	if err != nil {
		database.DB.Delete(&subscription)
		response.SendGinResponse(c, http.StatusInternalServerError, nil, nil, "Failed to generate integration token")
		return
	}

	subscription.IntegrationToken = token
	database.DB.Save(&subscription)

	response.SendGinResponse(c, http.StatusCreated, gin.H{
		"subscription_id":      subscription.ID,
		"status":               subscription.Status,
		"integration_token":    token,
		"current_period_start": subscription.CurrentPeriodStart,
		"current_period_end":   subscription.CurrentPeriodEnd,
	}, nil, "")
}

func GetSubscriptionStatus(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		response.SendGinResponse(c, http.StatusUnauthorized, nil, nil, "User not authenticated")
		return
	}

	var subscription models.Subscription
	if err := database.DB.Where("user_id = ? AND status IN ?", userID, []string{"active", "past_due"}).First(&subscription).Error; err != nil {
		response.SendGinResponse(c, http.StatusNotFound, nil, nil, "No active subscription found")
		return
	}

	billingMonth := time.Now().Format("2006-01")
	var queryCount int64
	database.DB.Model(&models.IntegrationQuery{}).
		Where("subscription_id = ? AND billing_month = ?", subscription.ID, billingMonth).
		Count(&queryCount)

	unitPrice := CalculateUnitPrice(int(queryCount))
	estimatedTotal := unitPrice * float64(queryCount)

	tier := "0-99"
	if queryCount >= 200 {
		tier = "200+"
	} else if queryCount >= 100 {
		tier = "100-199"
	}

	response.SendGinResponse(c, http.StatusOK, gin.H{
		"subscription_id":      subscription.ID,
		"status":               subscription.Status,
		"current_period_start": subscription.CurrentPeriodStart,
		"current_period_end":   subscription.CurrentPeriodEnd,
		"queries_this_month":   queryCount,
		"current_tier":         tier,
		"unit_price":           fmt.Sprintf("%.2f", unitPrice),
		"estimated_total":      fmt.Sprintf("%.2f", estimatedTotal),
	}, nil, "")
}

func CancelSubscription(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		response.SendGinResponse(c, http.StatusUnauthorized, nil, nil, "User not authenticated")
		return
	}

	var subscription models.Subscription
	if err := database.DB.Where("user_id = ? AND status = ?", userID, "active").First(&subscription).Error; err != nil {
		response.SendGinResponse(c, http.StatusNotFound, nil, nil, "No active subscription found")
		return
	}

	now := time.Now()
	subscription.Status = "cancelled"
	subscription.CancelledAt = &now
	subscription.IntegrationToken = "revoked"

	if err := database.DB.Save(&subscription).Error; err != nil {
		response.SendGinResponse(c, http.StatusInternalServerError, nil, nil, "Failed to cancel subscription")
		return
	}

	response.SendGinResponse(c, http.StatusOK, gin.H{
		"subscription_id": subscription.ID,
		"status":          subscription.Status,
		"cancelled_at":    subscription.CancelledAt,
	}, nil, "")
}

func GetInvoices(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		response.SendGinResponse(c, http.StatusUnauthorized, nil, nil, "User not authenticated")
		return
	}

	var invoices []models.Invoice
	if err := database.DB.Where("user_id = ?", userID).Order("created_at DESC").Find(&invoices).Error; err != nil {
		response.SendGinResponse(c, http.StatusInternalServerError, nil, nil, "Failed to fetch invoices")
		return
	}

	response.SendGinResponse(c, http.StatusOK, invoices, nil, "")
}

func RegenerateToken(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		response.SendGinResponse(c, http.StatusUnauthorized, nil, nil, "User not authenticated")
		return
	}

	email, _ := c.Get("email")

	var subscription models.Subscription
	if err := database.DB.Where("user_id = ? AND status = ?", userID, "active").First(&subscription).Error; err != nil {
		response.SendGinResponse(c, http.StatusNotFound, nil, nil, "No active subscription found")
		return
	}

	token, err := jwt.GenerateIntegrationToken(userID.(uint), email.(string), subscription.ID)
	if err != nil {
		response.SendGinResponse(c, http.StatusInternalServerError, nil, nil, "Failed to generate new token")
		return
	}

	subscription.IntegrationToken = token
	if err := database.DB.Save(&subscription).Error; err != nil {
		response.SendGinResponse(c, http.StatusInternalServerError, nil, nil, "Failed to save new token")
		return
	}

	response.SendGinResponse(c, http.StatusOK, gin.H{
		"integration_token": token,
	}, nil, "")
}

func CalculateUnitPrice(queryCount int) float64 {
	switch {
	case queryCount >= 200:
		return 7.00
	case queryCount >= 100:
		return 8.50
	default:
		return 15.90
	}
}
