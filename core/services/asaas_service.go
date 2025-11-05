package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/RodolfoBonis/spooliq/core/config"
	"github.com/RodolfoBonis/spooliq/core/logger"
)

// IAsaasService defines the interface for Asaas API interactions.
type IAsaasService interface {
	CreateCustomer(ctx context.Context, req AsaasCustomerRequest) (*AsaasCustomerResponse, error)
	CreateSubscription(ctx context.Context, req AsaasSubscriptionRequest) (*AsaasSubscriptionResponse, error)
	GetSubscription(ctx context.Context, subscriptionID string) (*AsaasSubscriptionResponse, error)
	CancelSubscription(ctx context.Context, subscriptionID string) error
	UpdateSubscription(ctx context.Context, subscriptionID string, req AsaasSubscriptionUpdateRequest) (*AsaasSubscriptionResponse, error)
	GetPayment(ctx context.Context, paymentID string) (*AsaasPaymentResponse, error)
	ListPayments(ctx context.Context, customerID string, offset, limit int) (*AsaasPaymentListResponse, error)
	UpdateCustomer(ctx context.Context, customerID string, req AsaasCustomerRequest) (*AsaasCustomerResponse, error)
	GetCustomer(ctx context.Context, customerID string) (*AsaasCustomerResponse, error)
	TokenizeCreditCard(ctx context.Context, req AsaasTokenizeCreditCardRequest) (*AsaasTokenizeCreditCardResponse, error)
	CreatePayment(ctx context.Context, req AsaasPaymentRequest) (*AsaasPaymentCreateResponse, error)
}

// AsaasService handles integration with Asaas payment gateway
type AsaasService struct {
	apiKey  string
	baseURL string
	logger  logger.Logger
	client  *http.Client
}

// NewAsaasService creates a new instance of AsaasService
func NewAsaasService(cfg *config.AppConfig, logger logger.Logger) IAsaasService {
	return &AsaasService{
		apiKey:  cfg.AsaasAPIKey,
		baseURL: cfg.AsaasBaseURL,
		logger:  logger,
		client:  &http.Client{},
	}
}

// CreateCustomer creates a customer in Asaas
func (s *AsaasService) CreateCustomer(ctx context.Context, request AsaasCustomerRequest) (*AsaasCustomerResponse, error) {
	url := fmt.Sprintf("%s/customers", s.baseURL)

	s.logger.Info(ctx, "Calling Asaas CreateCustomer", map[string]interface{}{
		"url":      url,
		"base_url": s.baseURL,
	})

	body, err := json.Marshal(request)
	if err != nil {
		s.logger.Error(ctx, "Failed to marshal Asaas customer request", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, err
	}

	resp, err := s.doRequest(ctx, "POST", url, body)
	if err != nil {
		return nil, err
	}

	var customer AsaasCustomerResponse
	if err := json.Unmarshal(resp, &customer); err != nil {
		s.logger.Error(ctx, "Failed to unmarshal Asaas customer response", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, err
	}

	s.logger.Info(ctx, "Customer created successfully in Asaas", map[string]interface{}{
		"customer_id": customer.ID,
		"email":       customer.Email,
	})

	return &customer, nil
}

// CreateSubscription creates a subscription in Asaas
func (s *AsaasService) CreateSubscription(ctx context.Context, request AsaasSubscriptionRequest) (*AsaasSubscriptionResponse, error) {
	url := fmt.Sprintf("%s/subscriptions", s.baseURL)

	body, err := json.Marshal(request)
	if err != nil {
		s.logger.Error(ctx, "Failed to marshal Asaas subscription request", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, err
	}

	resp, err := s.doRequest(ctx, "POST", url, body)
	if err != nil {
		return nil, err
	}

	var subscription AsaasSubscriptionResponse
	if err := json.Unmarshal(resp, &subscription); err != nil {
		s.logger.Error(ctx, "Failed to unmarshal Asaas subscription response", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, err
	}

	s.logger.Info(ctx, "Subscription created successfully in Asaas", map[string]interface{}{
		"subscription_id": subscription.ID,
		"customer_id":     subscription.Customer,
		"value":           subscription.Value,
	})

	return &subscription, nil
}

// GetSubscription retrieves a subscription from Asaas
func (s *AsaasService) GetSubscription(ctx context.Context, subscriptionID string) (*AsaasSubscriptionResponse, error) {
	url := fmt.Sprintf("%s/subscriptions/%s", s.baseURL, subscriptionID)

	resp, err := s.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	var subscription AsaasSubscriptionResponse
	if err := json.Unmarshal(resp, &subscription); err != nil {
		s.logger.Error(ctx, "Failed to unmarshal Asaas subscription response", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, err
	}

	return &subscription, nil
}

// CancelSubscription cancels a subscription in Asaas
func (s *AsaasService) CancelSubscription(ctx context.Context, subscriptionID string) error {
	url := fmt.Sprintf("%s/subscriptions/%s", s.baseURL, subscriptionID)

	_, err := s.doRequest(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}

	s.logger.Info(ctx, "Subscription cancelled successfully in Asaas", map[string]interface{}{
		"subscription_id": subscriptionID,
	})

	return nil
}

// GetPayment retrieves a payment from Asaas
func (s *AsaasService) GetPayment(ctx context.Context, paymentID string) (*AsaasPaymentResponse, error) {
	url := fmt.Sprintf("%s/payments/%s", s.baseURL, paymentID)

	resp, err := s.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	var payment AsaasPaymentResponse
	if err := json.Unmarshal(resp, &payment); err != nil {
		s.logger.Error(ctx, "Failed to unmarshal Asaas payment response", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, err
	}

	return &payment, nil
}

// ListPayments retrieves a list of payments from Asaas
func (s *AsaasService) ListPayments(ctx context.Context, customerID string, offset, limit int) (*AsaasPaymentListResponse, error) {
	url := fmt.Sprintf("%s/payments?customer=%s&offset=%d&limit=%d", s.baseURL, customerID, offset, limit)

	resp, err := s.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	var payments AsaasPaymentListResponse
	if err := json.Unmarshal(resp, &payments); err != nil {
		s.logger.Error(ctx, "Failed to unmarshal Asaas payments response", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, err
	}

	return &payments, nil
}

// doRequest performs an HTTP request to Asaas API
func (s *AsaasService) doRequest(ctx context.Context, method, url string, body []byte) ([]byte, error) {
	var req *http.Request
	var err error

	if body != nil {
		req, err = http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(body))
	} else {
		req, err = http.NewRequestWithContext(ctx, method, url, nil)
	}

	if err != nil {
		s.logger.Error(ctx, "Failed to create Asaas HTTP request", map[string]interface{}{
			"error":  err.Error(),
			"method": method,
			"url":    url,
		})
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("access_token", s.apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		s.logger.Error(ctx, "Failed to execute Asaas HTTP request", map[string]interface{}{
			"error":  err.Error(),
			"method": method,
			"url":    url,
		})
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		s.logger.Error(ctx, "Failed to read Asaas HTTP response body", map[string]interface{}{
			"error":  err.Error(),
			"method": method,
			"url":    url,
		})
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var asaasError AsaasErrorResponse
		if err := json.Unmarshal(respBody, &asaasError); err == nil && len(asaasError.Errors) > 0 {
			s.logger.Error(ctx, "Asaas API error", map[string]interface{}{
				"status_code": resp.StatusCode,
				"error_code":  asaasError.Errors[0].Code,
				"description": asaasError.Errors[0].Description,
				"method":      method,
				"url":         url,
			})
			return nil, fmt.Errorf("asaas API error: %s - %s", asaasError.Errors[0].Code, asaasError.Errors[0].Description)
		}

		s.logger.Error(ctx, "Asaas API returned error status", map[string]interface{}{
			"status_code": resp.StatusCode,
			"response":    string(respBody),
			"method":      method,
			"url":         url,
		})
		return nil, fmt.Errorf("asaas API error: status %d", resp.StatusCode)
	}

	return respBody, nil
}

// UpdateSubscription updates a subscription in Asaas
func (s *AsaasService) UpdateSubscription(ctx context.Context, subscriptionID string, request AsaasSubscriptionUpdateRequest) (*AsaasSubscriptionResponse, error) {
	url := fmt.Sprintf("%s/subscriptions/%s", s.baseURL, subscriptionID)

	body, err := json.Marshal(request)
	if err != nil {
		s.logger.Error(ctx, "Failed to marshal Asaas subscription update request", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, err
	}

	resp, err := s.doRequest(ctx, "POST", url, body)
	if err != nil {
		return nil, err
	}

	var subscription AsaasSubscriptionResponse
	if err := json.Unmarshal(resp, &subscription); err != nil {
		s.logger.Error(ctx, "Failed to unmarshal Asaas subscription response", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, err
	}

	s.logger.Info(ctx, "Subscription updated successfully in Asaas", map[string]interface{}{
		"subscription_id": subscription.ID,
	})

	return &subscription, nil
}

// UpdateCustomer updates a customer in Asaas
func (s *AsaasService) UpdateCustomer(ctx context.Context, customerID string, request AsaasCustomerRequest) (*AsaasCustomerResponse, error) {
	url := fmt.Sprintf("%s/customers/%s", s.baseURL, customerID)

	body, err := json.Marshal(request)
	if err != nil {
		s.logger.Error(ctx, "Failed to marshal Asaas customer update request", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, err
	}

	resp, err := s.doRequest(ctx, "POST", url, body)
	if err != nil {
		return nil, err
	}

	var customer AsaasCustomerResponse
	if err := json.Unmarshal(resp, &customer); err != nil {
		s.logger.Error(ctx, "Failed to unmarshal Asaas customer response", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, err
	}

	s.logger.Info(ctx, "Customer updated successfully in Asaas", map[string]interface{}{
		"customer_id": customer.ID,
	})

	return &customer, nil
}

// GetCustomer retrieves a customer from Asaas
func (s *AsaasService) GetCustomer(ctx context.Context, customerID string) (*AsaasCustomerResponse, error) {
	url := fmt.Sprintf("%s/customers/%s", s.baseURL, customerID)

	resp, err := s.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	var customer AsaasCustomerResponse
	if err := json.Unmarshal(resp, &customer); err != nil {
		s.logger.Error(ctx, "Failed to unmarshal Asaas customer response", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, err
	}

	return &customer, nil
}

// TokenizeCreditCard tokenizes a credit card for future use
func (s *AsaasService) TokenizeCreditCard(ctx context.Context, request AsaasTokenizeCreditCardRequest) (*AsaasTokenizeCreditCardResponse, error) {
	url := fmt.Sprintf("%s/creditCard/tokenize", s.baseURL)

	body, err := json.Marshal(request)
	if err != nil {
		s.logger.Error(ctx, "Failed to marshal Asaas tokenize credit card request", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, err
	}

	resp, err := s.doRequest(ctx, "POST", url, body)
	if err != nil {
		return nil, err
	}

	var tokenResponse AsaasTokenizeCreditCardResponse
	if err := json.Unmarshal(resp, &tokenResponse); err != nil {
		s.logger.Error(ctx, "Failed to unmarshal Asaas tokenize credit card response", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, err
	}

	s.logger.Info(ctx, "Credit card tokenized successfully in Asaas", map[string]interface{}{
		"customer_id": request.Customer,
		"card_brand":  tokenResponse.CreditCardBrand,
		"last_4":      tokenResponse.CreditCardNumber,
	})

	return &tokenResponse, nil
}

// CreatePayment creates a single payment (charge) in Asaas
func (s *AsaasService) CreatePayment(ctx context.Context, request AsaasPaymentRequest) (*AsaasPaymentCreateResponse, error) {
	url := fmt.Sprintf("%s/payments", s.baseURL)

	body, err := json.Marshal(request)
	if err != nil {
		s.logger.Error(ctx, "Failed to marshal Asaas payment request", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, err
	}

	resp, err := s.doRequest(ctx, "POST", url, body)
	if err != nil {
		return nil, err
	}

	var payment AsaasPaymentCreateResponse
	if err := json.Unmarshal(resp, &payment); err != nil {
		s.logger.Error(ctx, "Failed to unmarshal Asaas payment response", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, err
	}

	s.logger.Info(ctx, "Payment created successfully in Asaas", map[string]interface{}{
		"payment_id":   payment.ID,
		"customer_id":  payment.Customer,
		"value":        payment.Value,
		"billing_type": payment.BillingType,
	})

	return &payment, nil
}
