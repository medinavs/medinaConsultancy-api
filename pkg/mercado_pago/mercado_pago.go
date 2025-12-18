package mercadopago

import (
	"context"
	"fmt"
	"os"

	"github.com/mercadopago/sdk-go/pkg/config"
	"github.com/mercadopago/sdk-go/pkg/payment"
)

type PixPaymentRequest struct {
	Amount      float64 `json:"amount"`
	Description string  `json:"description"`
	PayerEmail  string  `json:"payer_email" binding:"required,email"`
	ExternalRef string  `json:"external_reference"`
}

type PixPaymentResponse struct {
	ID                string `json:"id"`
	Status            string `json:"status"`
	ExternalReference string `json:"external_reference"`
	QRCode            string `json:"qr_code"`
	QRCodeBase64      string `json:"qr_code_base64"`
	TicketURL         string `json:"ticket_url"`
}

func CreatePixPayment(ctx context.Context, req PixPaymentRequest) (*PixPaymentResponse, error) {
	accessToken := os.Getenv("MERCADO_PAGO_ACCESS_TOKEN")
	if accessToken == "" {
		return nil, fmt.Errorf("MERCADO_PAGO_ACCESS_TOKEN is not set")
	}

	cfg, err := config.New(accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create mercado pago config: %w", err)
	}

	client := payment.NewClient(cfg)

	paymentRequest := payment.Request{
		TransactionAmount: req.Amount,
		Description:       req.Description,
		PaymentMethodID:   "pix",
		Payer: &payment.PayerRequest{
			Email: req.PayerEmail,
		},
		ExternalReference: req.ExternalRef,
	}

	resource, err := client.Create(ctx, paymentRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to create pix payment: %w", err)
	}

	response := &PixPaymentResponse{
		ID:                fmt.Sprintf("%d", resource.ID),
		Status:            resource.Status,
		ExternalReference: resource.ExternalReference,
	}

	// Extract PIX data from point of interaction
	// Access nested struct fields directly
	response.QRCode = resource.PointOfInteraction.TransactionData.QRCode
	response.QRCodeBase64 = resource.PointOfInteraction.TransactionData.QRCodeBase64
	response.TicketURL = resource.PointOfInteraction.TransactionData.TicketURL

	return response, nil
}

func GetPaymentStatus(ctx context.Context, paymentID int) (string, error) {
	accessToken := os.Getenv("MERCADO_PAGO_ACCESS_TOKEN")
	if accessToken == "" {
		return "", fmt.Errorf("MERCADO_PAGO_ACCESS_TOKEN is not set")
	}

	cfg, err := config.New(accessToken)
	if err != nil {
		return "", fmt.Errorf("failed to create mercado pago config: %w", err)
	}

	client := payment.NewClient(cfg)

	resource, err := client.Get(ctx, paymentID)
	if err != nil {
		return "", fmt.Errorf("failed to get payment: %w", err)
	}

	return resource.Status, nil
}
