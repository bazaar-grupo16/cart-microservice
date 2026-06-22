package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCatalogClient_GetProduct(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse int
		serverBody     string
		expectErr      error
		expectProduct  bool
	}{
		{
			name:           "Success",
			serverResponse: http.StatusOK,
			serverBody:     `{"data": {"_id": "123", "title": "Test", "price": "100.00", "stock": 5, "status": "active"}}`,
			expectErr:      nil,
			expectProduct:  true,
		},
		{
			name:           "Not Found",
			serverResponse: http.StatusNotFound,
			serverBody:     `{}`,
			expectErr:      ErrProductNotFound,
			expectProduct:  false,
		},
		{
			name:           "Invalid JSON",
			serverResponse: http.StatusOK,
			serverBody:     `{bad json}`,
			expectErr:      ErrCatalogUnavailable,
			expectProduct:  false,
		},
		{
			name:           "Server Error",
			serverResponse: http.StatusInternalServerError,
			serverBody:     `{}`,
			expectErr:      ErrCatalogUnavailable,
			expectProduct:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.serverResponse)
				_, _ = w.Write([]byte(tt.serverBody))
			}))
			defer server.Close()

			client := NewCatalogClient(server.URL)
			product, err := client.GetProduct(context.Background(), "123")

			if !errors.Is(err, tt.expectErr) {
				t.Errorf("expected error %v, got %v", tt.expectErr, err)
			}

			if tt.expectProduct && product == nil {
				t.Errorf("expected product, got nil")
			}
			if !tt.expectProduct && product != nil {
				t.Errorf("expected no product, got %v", product)
			}
		})
	}
}

func TestCatalogClient_CircuitBreaker(t *testing.T) {
	// Create a server that always fails (500)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewCatalogClient(server.URL)
	ctx := context.Background()

	// 1st failure
	_, err := client.GetProduct(ctx, "123")
	if !errors.Is(err, ErrCatalogUnavailable) {
		t.Errorf("expected ErrCatalogUnavailable, got %v", err)
	}

	// 2nd failure
	_, _ = client.GetProduct(ctx, "123")

	// 3rd failure - should open the circuit
	_, _ = client.GetProduct(ctx, "123")

	if !client.circuitOpen {
		t.Errorf("expected circuit to be open")
	}

	// 4th request - should fail immediately without network call (mocking network by stopping server won't matter, but we just check state)
	_, err = client.GetProduct(ctx, "123")
	if !errors.Is(err, ErrCatalogUnavailable) {
		t.Errorf("expected ErrCatalogUnavailable, got %v", err)
	}

	// Test half-open by forcing openUntil to the past
	client.mu.Lock()
	client.openUntil = time.Now().Add(-1 * time.Second)
	client.mu.Unlock()

	// Now it should try again, fail, and re-open
	_, _ = client.GetProduct(ctx, "123")

	client.mu.Lock()
	isOpen := client.circuitOpen
	client.mu.Unlock()

	if !isOpen {
		t.Errorf("expected circuit to be open again after half-open failure")
	}
}

func TestCatalogClient_NetworkError(t *testing.T) {
	client := NewCatalogClient("http://invalid-url-that-does-not-exist")
	// Set a very short timeout
	client.HTTPClient.Timeout = 1 * time.Millisecond

	_, err := client.GetProduct(context.Background(), "123")
	if !errors.Is(err, ErrCatalogUnavailable) {
		t.Errorf("expected ErrCatalogUnavailable on network error, got %v", err)
	}
}
