package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/ValentinMarturet/is2-rojas/src/models"
	"go.uber.org/zap"
)

// Sentinel errors returned by CatalogClientI implementations.
var (
	// ErrProductNotFound is returned when the Catalog service responds with 404.
	ErrProductNotFound = errors.New("product not found in catalog")

	// ErrCatalogUnavailable is returned on timeout, circuit-open, or any non-200/404 response.
	ErrCatalogUnavailable = errors.New("catalog service is unavailable")
)

// CatalogClientI defines the contract for fetching product data from the Catalog service.
// Using an interface here allows swapping a real HTTP client for a mock in tests.
type CatalogClientI interface {
	GetProduct(productID int) (*models.Product, error)
}

// CatalogClient is the production implementation of CatalogClientI.
// It incorporates:
//   - 2-second HTTP timeout
//   - Simple Circuit Breaker: after 3 consecutive failures the circuit opens for 30 seconds
//     (subsequent calls return ErrCatalogUnavailable immediately without hitting the network).
//     After 30 s the circuit enters half-open: one probe is allowed; success closes it, failure
//     re-opens it for another 30 s.
type CatalogClient struct {
	BaseURL     string
	HTTPClient  *http.Client
	mu          sync.Mutex
	failures    int
	circuitOpen bool
	openUntil   time.Time
}

// NewCatalogClient creates a production CatalogClient with the given base URL.
func NewCatalogClient(baseURL string) *CatalogClient {
	return &CatalogClient{
		BaseURL:    baseURL,
		HTTPClient: &http.Client{Timeout: 2 * time.Second},
	}
}

// GetProduct fetches a product by ID from the Catalog microservice.
// Returns ErrProductNotFound when the catalog responds 404.
// Returns ErrCatalogUnavailable on any other failure (timeout, circuit open, 5xx, etc.).
func (c *CatalogClient) GetProduct(productID int) (*models.Product, error) {
	// --- Circuit Breaker: check state ---
	c.mu.Lock()
	if c.circuitOpen {
		if time.Now().Before(c.openUntil) {
			// Circuit is open — fail fast without attempting the request.
			c.mu.Unlock()
			zap.S().Warnw("Catalog circuit breaker is OPEN, rejecting request", "productID", productID)
			return nil, ErrCatalogUnavailable
		}
		// Half-open: allow one probe attempt.
		c.circuitOpen = false
		zap.S().Infow("Catalog circuit breaker entering HALF-OPEN, allowing probe request")
	}
	c.mu.Unlock()

	// --- Execute HTTP request ---
	url := fmt.Sprintf("%s/products/%d", c.BaseURL, productID)
	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		c.recordFailure()
		zap.S().Errorw("Catalog HTTP request failed", "url", url, "error", err)
		return nil, ErrCatalogUnavailable
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNotFound:
		// 404 is a valid business response — counts as a success for circuit purposes.
		c.recordSuccess()
		return nil, ErrProductNotFound

	case http.StatusOK:
		var wrapper struct {
			Data models.Product `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
			c.recordFailure()
			zap.S().Errorw("Failed to decode catalog response", "error", err)
			return nil, ErrCatalogUnavailable
		}
		c.recordSuccess()
		return &wrapper.Data, nil

	default:
		c.recordFailure()
		zap.S().Warnw("Catalog returned unexpected status", "status", resp.StatusCode, "url", url)
		return nil, ErrCatalogUnavailable
	}
}

// recordFailure increments the failure counter and opens the circuit after 3 consecutive failures.
func (c *CatalogClient) recordFailure() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.failures++
	if c.failures >= 3 && !c.circuitOpen {
		c.circuitOpen = true
		c.openUntil = time.Now().Add(30 * time.Second)
		zap.S().Warnw("Catalog circuit breaker OPENED",
			"failures", c.failures,
			"openUntil", c.openUntil.Format(time.RFC3339),
		)
	}
}

// recordSuccess resets the failure counter and closes the circuit.
func (c *CatalogClient) recordSuccess() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.failures > 0 || c.circuitOpen {
		zap.S().Infow("Catalog circuit breaker CLOSED after successful response")
	}
	c.failures = 0
	c.circuitOpen = false
}
