package mercadopago

import (
	"context"
	"fmt"
	"os"

	"github.com/mercadopago/sdk-go/pkg/config"
	"github.com/mercadopago/sdk-go/pkg/customer"
	"github.com/mercadopago/sdk-go/pkg/customercard"
	"github.com/mercadopago/sdk-go/pkg/payment"
)

type CardPaymentRequest struct {
	Amount      float64
	Description string
	CustomerID  string
	CardID      string
	PayerEmail  string
	ExternalRef string
}

type CardPaymentResponse struct {
	ID     string
	Status string
}

func newConfig() (*config.Config, error) {
	accessToken := os.Getenv("MERCADO_PAGO_ACCESS_TOKEN")
	if accessToken == "" {
		return nil, fmt.Errorf("MERCADO_PAGO_ACCESS_TOKEN is not set")
	}
	return config.New(accessToken)
}

func GetOrCreateCustomer(ctx context.Context, email string) (string, error) {
	cfg, err := newConfig()
	if err != nil {
		return "", fmt.Errorf("failed to create mercado pago config: %w", err)
	}

	client := customer.NewClient(cfg)

	searchResp, err := client.Search(ctx, customer.SearchRequest{
		Filters: map[string]string{"email": email},
	})
	if err == nil && len(searchResp.Results) > 0 {
		return searchResp.Results[0].ID, nil
	}

	resp, err := client.Create(ctx, customer.Request{
		Email: email,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create customer: %w", err)
	}

	return resp.ID, nil
}

func SaveCardToCustomer(ctx context.Context, customerID string, cardToken string) (string, error) {
	cfg, err := newConfig()
	if err != nil {
		return "", fmt.Errorf("failed to create mercado pago config: %w", err)
	}

	client := customercard.NewClient(cfg)

	resp, err := client.Create(ctx, customerID, customercard.Request{
		Token: cardToken,
	})
	if err != nil {
		return "", fmt.Errorf("failed to save card to customer: %w", err)
	}

	return resp.ID, nil
}

func ChargeCard(ctx context.Context, req CardPaymentRequest) (*CardPaymentResponse, error) {
	cfg, err := newConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create mercado pago config: %w", err)
	}

	client := payment.NewClient(cfg)

	installments := 1
	paymentReq := payment.Request{
		Token:             req.CardID,
		TransactionAmount: req.Amount,
		Description:       req.Description,
		ExternalReference: req.ExternalRef,
		Installments:      installments,
		Payer: &payment.PayerRequest{
			ID:    req.CustomerID,
			Email: req.PayerEmail,
		},
	}

	resource, err := client.Create(ctx, paymentReq)
	if err != nil {
		return nil, fmt.Errorf("failed to charge card: %w", err)
	}

	return &CardPaymentResponse{
		ID:     fmt.Sprintf("%d", resource.ID),
		Status: resource.Status,
	}, nil
}
