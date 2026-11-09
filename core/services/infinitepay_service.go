package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/RodolfoBonis/spooliq/core/config"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// Typed errors so callers can map InfinitePay failures to the right HTTP status
// without inspecting raw status codes or leaking provider detail.
var (
	// ErrIPUnauthorized is a 401/403 from InfinitePay: the API key is rejected.
	ErrIPUnauthorized = errors.New("infinitepay: unauthorized")
	// ErrIPInvalid is a 4xx (other than 401/403) from InfinitePay: a malformed or
	// rejected request (e.g. amount below the minimum).
	ErrIPInvalid = errors.New("infinitepay: invalid request")
	// ErrIPUnavailable is a 5xx, timeout or transport error from InfinitePay.
	ErrIPUnavailable = errors.New("infinitepay: unavailable")
)

// infinitePayTimeout bounds every call to the Checkout API.
const infinitePayTimeout = 10 * time.Second

// IInfinitePayClient is the interface over the InfinitePay Checkout API. It is mocked
// in tests (there is no sandbox, so the real API is never called from tests).
// Credentials are passed PER CALL so a single client serves every organization.
type IInfinitePayClient interface {
	// CreateLink creates a checkout link with the given organization API key.
	CreateLink(ctx context.Context, apiKey string, req InfinitePayCreateLinkRequest) (*InfinitePayCreateLinkResponse, error)
	// PaymentCheck queries the authoritative payment state for an order/transaction.
	PaymentCheck(ctx context.Context, apiKey string, req InfinitePayPaymentCheckRequest) (*InfinitePayPaymentCheckResponse, error)
}

// InfinitePayService is the HTTP implementation of IInfinitePayClient.
type InfinitePayService struct {
	baseURL string
	logger  logger.Logger
	client  *http.Client
}

// NewInfinitePayService creates a new InfinitePayService. The HTTP transport is
// OTel-instrumented, matching the Asaas client.
func NewInfinitePayService(cfg *config.AppConfig, log logger.Logger) IInfinitePayClient {
	return &InfinitePayService{
		baseURL: cfg.InfinitePayBaseURL,
		logger:  log,
		client: &http.Client{
			Timeout:   infinitePayTimeout,
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		},
	}
}

// CreateLink creates a checkout link.
func (s *InfinitePayService) CreateLink(ctx context.Context, apiKey string, req InfinitePayCreateLinkRequest) (*InfinitePayCreateLinkResponse, error) {
	var resp InfinitePayCreateLinkResponse
	if err := s.doRequest(ctx, apiKey, "POST", "/links", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// PaymentCheck queries the authoritative state of a payment.
func (s *InfinitePayService) PaymentCheck(ctx context.Context, apiKey string, req InfinitePayPaymentCheckRequest) (*InfinitePayPaymentCheckResponse, error) {
	var resp InfinitePayPaymentCheckResponse
	if err := s.doRequest(ctx, apiKey, "POST", "/payment_check", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// doRequest performs a JSON request to the Checkout API and maps failures to the
// typed errors above. It NEVER logs the API key or the request/response bodies
// (they can carry the webhook token and customer data).
func (s *InfinitePayService) doRequest(ctx context.Context, apiKey, method, path string, body, out interface{}) error {
	url := s.baseURL + path

	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("%w: marshal: %v", ErrIPInvalid, err)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("%w: build request: %v", ErrIPUnavailable, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Api-Key", apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		s.logger.Error(ctx, "InfinitePay request failed", map[string]interface{}{
			"method": method,
			"path":   path,
			"error":  err.Error(),
		})
		return fmt.Errorf("%w: %v", ErrIPUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("%w: read body: %v", ErrIPUnavailable, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Never log the raw body (it may echo customer data). Only the provider's own
		// error fields are logged, so validation failures (422) are diagnosable.
		fields := map[string]interface{}{
			"method":      method,
			"path":        path,
			"status_code": resp.StatusCode,
		}
		var apiErr InfinitePayErrorResponse
		if json.Unmarshal(respBody, &apiErr) == nil {
			if apiErr.Message != "" {
				fields["provider_message"] = truncate(apiErr.Message, 300)
			}
			if apiErr.Error != "" {
				fields["provider_error"] = truncate(apiErr.Error, 300)
			}
			if len(apiErr.Errors) > 0 {
				keys := make([]string, 0, len(apiErr.Errors))
				for k := range apiErr.Errors {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				fields["provider_error_fields"] = keys
			}
		}
		s.logger.Error(ctx, "InfinitePay API returned error status", fields)
		switch {
		case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
			return ErrIPUnauthorized
		case resp.StatusCode >= 500:
			return ErrIPUnavailable
		default:
			return ErrIPInvalid
		}
	}

	if out != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("%w: unmarshal: %v", ErrIPUnavailable, err)
		}
	}
	return nil
}

// truncate caps a provider-supplied string before it is logged.
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
