package services

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// NewInstrumentedHTTPClient creates an HTTP client with OpenTelemetry instrumentation
func NewInstrumentedHTTPClient() *http.Client {
	// Create HTTP client with OpenTelemetry instrumentation
	return &http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}
}

// InstrumentHTTPClient wraps an existing HTTP client with OpenTelemetry instrumentation
func InstrumentHTTPClient(client *http.Client) *http.Client {
	if client == nil {
		return NewInstrumentedHTTPClient()
	}

	// Wrap the existing transport with OpenTelemetry
	client.Transport = otelhttp.NewTransport(client.Transport)
	return client
}