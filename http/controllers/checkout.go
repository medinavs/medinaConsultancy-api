package controllers

import (
	"context"
	"fmt"
	"medina-consultancy-api/database"
	"medina-consultancy-api/models"
	mercadopago "medina-consultancy-api/pkg/mercado_pago"
	"medina-consultancy-api/pkg/response"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type CheckoutRequest struct {
	PackageID  uint   `json:"package_id" binding:"required"`
	PayerEmail string `json:"payer_email" binding:"required,email"`
}

type PackageResponse struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Credits     int    `json:"credits"`
	Price       string `json:"price"`
	Description string `json:"description"`
}

func GetCreditPackages(c *gin.Context) {
	var packages []models.CreditPackage
	if err := database.DB.Where("active = ?", true).Find(&packages).Error; err != nil {
		response.SendGinResponse(c, http.StatusInternalServerError, nil, nil, "Failed to fetch packages")
		return
	}

	var packageResponses []PackageResponse
	for _, pkg := range packages {
		packageResponses = append(packageResponses, PackageResponse{
			ID:          pkg.ID,
			Name:        pkg.Name,
			Credits:     pkg.Credits,
			Price:       pkg.Price,
			Description: pkg.Description,
		})
	}

	response.SendGinResponse(c, http.StatusOK, packageResponses, nil, "")
}

func GetCreditPackageByID(c *gin.Context) {
	packageID := c.Param("id")

	var pkg models.CreditPackage
	if err := database.DB.Where("id = ? AND active = ?", packageID, true).First(&pkg).Error; err != nil {
		response.SendGinResponse(c, http.StatusNotFound, nil, nil, "Package not found")
		return
	}

	packageResponse := PackageResponse{
		ID:          pkg.ID,
		Name:        pkg.Name,
		Credits:     pkg.Credits,
		Price:       pkg.Price,
		Description: pkg.Description,
	}

	response.SendGinResponse(c, http.StatusOK, packageResponse, nil, "")
}

func CreateCheckout(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		response.SendGinResponse(c, http.StatusUnauthorized, nil, nil, "User not authenticated")
		return
	}

	var req CheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendGinResponse(c, http.StatusBadRequest, nil, nil, err.Error())
		return
	}

	var creditPackage models.CreditPackage
	if err := database.DB.Where("id = ? AND active = ?", req.PackageID, true).First(&creditPackage).Error; err != nil {
		response.SendGinResponse(c, http.StatusNotFound, nil, nil, "Credit package not found")
		return
	}

	price, err := strconv.ParseFloat(creditPackage.Price, 64)
	if err != nil {
		response.SendGinResponse(c, http.StatusInternalServerError, nil, nil, "Invalid package price")
		return
	}

	// (idempotency key)
	externalRef := fmt.Sprintf("order_%s_%d", uuid.New().String()[:8], time.Now().Unix())

	// create order in db
	order := models.Order{
		UserID:            userID.(uint),
		CreditPackageID:   creditPackage.ID,
		Amount:            creditPackage.Price,
		Status:            "pending",
		ExternalReference: externalRef,
	}

	if err := database.DB.Create(&order).Error; err != nil {
		response.SendGinResponse(c, http.StatusInternalServerError, nil, nil, "Failed to create order")
		return
	}

	// create pix payment
	mpRequest := mercadopago.PixPaymentRequest{
		Amount:      price,
		Description: fmt.Sprintf("Pacote %s - %d créditos", creditPackage.Name, creditPackage.Credits),
		PayerEmail:  req.PayerEmail,
		ExternalRef: externalRef,
	}

	ctx := context.Background()
	mpResponse, err := mercadopago.CreatePixPayment(ctx, mpRequest)
	if err != nil {
		database.DB.Model(&order).Update("status", "failed")
		response.SendGinResponse(c, http.StatusInternalServerError, nil, nil, fmt.Sprintf("Payment failed: %v", err))
		return
	}

	// mercado pago response
	order.MercadoPagoID = mpResponse.ID
	order.Status = mpResponse.Status
	order.QRCode = mpResponse.QRCode
	order.QRCodeBase64 = mpResponse.QRCodeBase64
	order.TicketURL = mpResponse.TicketURL
	database.DB.Save(&order)

	checkoutResponse := gin.H{
		"order_id":           order.ID,
		"mercado_pago_id":    mpResponse.ID,
		"status":             mpResponse.Status,
		"external_reference": externalRef,
		"amount":             creditPackage.Price,
		"credits":            creditPackage.Credits,
		"pix": gin.H{
			"qr_code":        mpResponse.QRCode,
			"qr_code_base64": mpResponse.QRCodeBase64,
			"ticket_url":     mpResponse.TicketURL,
		},
	}

	response.SendGinResponse(c, http.StatusCreated, checkoutResponse, nil, "")
}

func CheckPaymentStatus(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		response.SendGinResponse(c, http.StatusUnauthorized, nil, nil, "User not authenticated")
		return
	}

	orderID := c.Param("id")

	var order models.Order
	if err := database.DB.Preload("CreditPackage").Where("id = ? AND user_id = ?", orderID, userID).First(&order).Error; err != nil {
		response.SendGinResponse(c, http.StatusNotFound, nil, nil, "Order not found")
		return
	}

	// just return current status if approved and added credits
	if order.Status == "approved" && order.CreditsAdded {
		response.SendGinResponse(c, http.StatusOK, gin.H{
			"order_id":      order.ID,
			"status":        order.Status,
			"credits_added": order.CreditsAdded,
		}, nil, "")
		return
	}

	// poll for current status
	if order.MercadoPagoID != "" {
		paymentID, err := strconv.ParseInt(order.MercadoPagoID, 10, 64)
		if err == nil {
			ctx := context.Background()
			status, err := mercadopago.GetPaymentStatus(ctx, int(paymentID))
			if err == nil && status != order.Status {
				order.Status = status
				database.DB.Model(&order).Update("status", status)
			}
		}
	}

	if order.Status == "approved" && !order.CreditsAdded {
		var user models.User
		if err := database.DB.First(&user, userID).Error; err != nil {
			response.SendGinResponse(c, http.StatusInternalServerError, nil, nil, "Failed to find user")
			return
		}

		user.Credits += order.CreditPackage.Credits
		if err := database.DB.Save(&user).Error; err != nil {
			response.SendGinResponse(c, http.StatusInternalServerError, nil, nil, "Failed to add credits")
			return
		}

		order.CreditsAdded = true
		database.DB.Model(&order).Update("credits_added", true)
	}

	response.SendGinResponse(c, http.StatusOK, gin.H{
		"order_id":      order.ID,
		"status":        order.Status,
		"credits_added": order.CreditsAdded,
		"credits":       order.CreditPackage.Credits,
	}, nil, "")
}

func GetOrderStatus(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		response.SendGinResponse(c, http.StatusUnauthorized, nil, nil, "User not authenticated")
		return
	}

	orderID := c.Param("id")

	var order models.Order
	if err := database.DB.Preload("CreditPackage").Where("id = ? AND user_id = ?", orderID, userID).First(&order).Error; err != nil {
		response.SendGinResponse(c, http.StatusNotFound, nil, nil, "Order not found")
		return
	}

	orderResponse := gin.H{
		"id":                 order.ID,
		"status":             order.Status,
		"amount":             order.Amount,
		"mercado_pago_id":    order.MercadoPagoID,
		"external_reference": order.ExternalReference,
		"credits_added":      order.CreditsAdded,
		"package":            order.CreditPackage,
		"pix": gin.H{
			"qr_code":        order.QRCode,
			"qr_code_base64": order.QRCodeBase64,
			"ticket_url":     order.TicketURL,
		},
		"created_at": order.CreatedAt,
	}

	response.SendGinResponse(c, http.StatusOK, orderResponse, nil, "")
}

func GetUserOrders(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		response.SendGinResponse(c, http.StatusUnauthorized, nil, nil, "User not authenticated")
		return
	}

	var orders []models.Order
	if err := database.DB.Preload("CreditPackage").Where("user_id = ?", userID).Order("created_at DESC").Find(&orders).Error; err != nil {
		response.SendGinResponse(c, http.StatusInternalServerError, nil, nil, "Failed to fetch orders")
		return
	}

	response.SendGinResponse(c, http.StatusOK, orders, nil, "")
}

func GetUserCredits(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		response.SendGinResponse(c, http.StatusUnauthorized, nil, nil, "User not authenticated")
		return
	}

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		response.SendGinResponse(c, http.StatusNotFound, nil, nil, "User not found")
		return
	}

	response.SendGinResponse(c, http.StatusOK, gin.H{
		"credits": user.Credits,
	}, nil, "")
}
