package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/bazaar-grupo16/cart-microservice/src/config"
	"github.com/bazaar-grupo16/cart-microservice/src/controller"
	"github.com/bazaar-grupo16/cart-microservice/src/models"
	"github.com/bazaar-grupo16/cart-microservice/src/repository"
	"github.com/bazaar-grupo16/cart-microservice/src/router"
	"github.com/bazaar-grupo16/cart-microservice/src/service"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// MockCatalogClient — implements service.CatalogClientI for tests.
// It returns a fixed product without making any HTTP calls, allowing tests to
// run without a real Catalog service.
// ---------------------------------------------------------------------------

type MockCatalogClient struct {
	Product *models.Product
	Err     error
}

// GetProduct satisfies CatalogClientI.
func (m *MockCatalogClient) GetProduct(_ int) (*models.Product, error) {
	return m.Product, m.Err
}

// defaultMockCatalog is a convenience mock that always returns a valid product.
func defaultMockCatalog() *MockCatalogClient {
	return &MockCatalogClient{
		Product: &models.Product{
			ID:    1,
			Title: "Test Product",
			Price: models.Price("9.99"),
		},
	}
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// setupRouter creates a Gin router wired against a real test-DB transaction
// and a MockCatalogClient. The transaction is rolled back in each test so
// tests are isolated and don't leave data behind.
func setupRouter() (*gin.Engine, *sql.Tx) {
	logger, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(logger)

	if err := godotenv.Load("../.env", ".env"); err != nil {
		zap.S().Info("No .env file found; using system environment variables")
	}

	// Override DB name to the dedicated test database.
	os.Setenv("DATABASE_NAME", "ecommerce_test")

	config.ConnectDatabase()
	config.RunMigrations()

	tx, err := config.DB.Begin()
	if err != nil {
		zap.S().Fatalf("Failed to begin transaction: %v", err)
	}

	cartRepo := repository.NewCartRepository(tx)
	cartService := service.NewCartService(cartRepo, defaultMockCatalog())
	cartController := controller.NewCartController(cartService)

	return router.SetupRouter(cartController), tx
}

// helperAddItemToCart adds a cart item using the mock catalog (productId=1).
// It uses Fatalf so the test fails immediately if the setup call fails.
func helperAddItemToCart(r *gin.Engine, t *testing.T, userID int) int {
	t.Helper()

	body := `{"productId": 1}`
	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/cart/%d/items", userID),
		bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("helperAddItemToCart: expected 201, got %d — body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data struct {
			ID int `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("helperAddItemToCart: failed to parse response: %v", err)
	}
	return resp.Data.ID
}

// ---------------------------------------------------------------------------
// Cart tests
// ---------------------------------------------------------------------------

// TestGetEmptyCart verifies that GET /cart/{userId} for a new user returns
// an empty items array and a totalPrice of 0.00.
func TestGetEmptyCart(t *testing.T) {
	r, tx := setupRouter()
	defer tx.Rollback()

	req := httptest.NewRequest(http.MethodGet, "/cart/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !bytes.Contains(w.Body.Bytes(), []byte(`"items":[]`)) {
		t.Errorf(`expected "items":[], got: %s`, body)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"totalPrice":0.00`)) {
		t.Errorf(`expected "totalPrice":0.00, got: %s`, body)
	}
}

// TestGetCart verifies that added items appear in GET /cart/{userId}.
func TestGetCart(t *testing.T) {
	r, tx := setupRouter()
	defer tx.Rollback()

	userID := 1
	helperAddItemToCart(r, t, userID)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/cart/%d", userID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("Test Product")) {
		t.Errorf("expected cart to contain 'Test Product', got: %s", w.Body.String())
	}

	var cartResp struct {
		Data struct {
			Items []struct {
				ID int `json:"id"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &cartResp); err != nil {
		t.Fatalf("failed to parse cart response: %v", err)
	}
	if len(cartResp.Data.Items) != 1 {
		t.Errorf("expected 1 cart item, got %d", len(cartResp.Data.Items))
	}
}

// TestAddItemToCart verifies POST /cart/{userId}/items creates a CartItem with
// the snapshot price and title from the mock catalog.
func TestAddItemToCart(t *testing.T) {
	r, tx := setupRouter()
	defer tx.Rollback()

	userID := 1
	body := `{"productId": 1}`
	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/cart/%d/items", userID),
		bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d — body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data struct {
			ID        int     `json:"id"`
			ProductID int     `json:"productId"`
			Title     string  `json:"title"`
			UnitPrice float64 `json:"unitPrice"`
			AddedAt   string  `json:"addedAt"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	item := resp.Data
	if item.ProductID != 1 {
		t.Errorf("expected productId 1, got %d", item.ProductID)
	}
	if item.Title != "Test Product" {
		t.Errorf("expected title 'Test Product', got '%s'", item.Title)
	}
	if item.UnitPrice != 9.99 {
		t.Errorf("expected unitPrice 9.99, got %v", item.UnitPrice)
	}
	if item.AddedAt == "" {
		t.Error("expected addedAt to be populated")
	}
}

// TestAddItemToCart_ProductNotFound verifies that a mock returning ErrProductNotFound
// results in a 404 problem+json response.
func TestAddItemToCart_ProductNotFound(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(logger)

	if err := godotenv.Load("../.env", ".env"); err != nil {
		zap.S().Info("No .env file found")
	}
	os.Setenv("DATABASE_NAME", "ecommerce_test")
	config.ConnectDatabase()
	config.RunMigrations()
	tx, _ := config.DB.Begin()
	defer tx.Rollback()

	notFoundMock := &MockCatalogClient{Err: service.ErrProductNotFound}
	cartRepo := repository.NewCartRepository(tx)
	cartService := service.NewCartService(cartRepo, notFoundMock)
	cartController := controller.NewCartController(cartService)
	r := router.SetupRouter(cartController)

	req := httptest.NewRequest(http.MethodPost, "/cart/1/items",
		bytes.NewBufferString(`{"productId": 9999}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d — body: %s", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Errorf("expected Content-Type application/problem+json, got %s", ct)
	}
}

// TestAddItemToCart_CatalogUnavailable verifies that ErrCatalogUnavailable maps to 503.
func TestAddItemToCart_CatalogUnavailable(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(logger)

	if err := godotenv.Load("../.env", ".env"); err != nil {
		zap.S().Info("No .env file found")
	}
	os.Setenv("DATABASE_NAME", "ecommerce_test")
	config.ConnectDatabase()
	config.RunMigrations()
	tx, _ := config.DB.Begin()
	defer tx.Rollback()

	unavailableMock := &MockCatalogClient{Err: service.ErrCatalogUnavailable}
	cartRepo := repository.NewCartRepository(tx)
	cartService := service.NewCartService(cartRepo, unavailableMock)
	cartController := controller.NewCartController(cartService)
	r := router.SetupRouter(cartController)

	req := httptest.NewRequest(http.MethodPost, "/cart/1/items",
		bytes.NewBufferString(`{"productId": 1}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d — body: %s", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Errorf("expected Content-Type application/problem+json, got %s", ct)
	}
}

// TestClearCart verifies that DELETE /cart/{userId} empties the cart and is idempotent.
func TestClearCart(t *testing.T) {
	r, tx := setupRouter()
	defer tx.Rollback()

	userID := 1
	helperAddItemToCart(r, t, userID)

	// First clear
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/cart/%d", userID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}

	// Verify cart is empty
	getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/cart/%d", userID), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, getReq)

	var cartResp struct {
		Data struct {
			Items []interface{} `json:"items"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &cartResp)
	if len(cartResp.Data.Items) != 0 {
		t.Errorf("expected empty cart after clear, got %d items", len(cartResp.Data.Items))
	}

	// Second clear on empty cart must still return 204 (idempotent)
	req2 := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/cart/%d", userID), nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusNoContent {
		t.Errorf("expected idempotent 204 on empty cart, got %d", w2.Code)
	}
}

// TestRemoveItemFromCart verifies DELETE /cart/{userId}/items/{cartItemId}.
func TestRemoveItemFromCart(t *testing.T) {
	r, tx := setupRouter()
	defer tx.Rollback()

	userID := 1
	// Add two items
	helperAddItemToCart(r, t, userID)
	helperAddItemToCart(r, t, userID)

	// Get current cart to find a cartItemId
	getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/cart/%d", userID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, getReq)

	var cartResp struct {
		Data struct {
			Items []struct {
				ID int `json:"id"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &cartResp); err != nil {
		t.Fatalf("failed to parse cart: %v", err)
	}
	if len(cartResp.Data.Items) < 2 {
		t.Fatalf("expected at least 2 items, got %d", len(cartResp.Data.Items))
	}

	cartItemID := cartResp.Data.Items[0].ID

	// Remove first item
	delReq := httptest.NewRequest(http.MethodDelete,
		fmt.Sprintf("/cart/%d/items/%d", userID, cartItemID), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, delReq)
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}

	// Verify only 1 item remains and the deleted one is gone
	getReq2 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/cart/%d", userID), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, getReq2)
	json.Unmarshal(w.Body.Bytes(), &cartResp)

	if len(cartResp.Data.Items) != 1 {
		t.Errorf("expected 1 item remaining, got %d", len(cartResp.Data.Items))
	}
	for _, item := range cartResp.Data.Items {
		if item.ID == cartItemID {
			t.Errorf("item %d should have been deleted", cartItemID)
		}
	}
}

// TestRemoveItemFromCart_WrongUser verifies that deleting another user's item returns 404.
func TestRemoveItemFromCart_WrongUser(t *testing.T) {
	r, tx := setupRouter()
	defer tx.Rollback()

	// Add item for user 1
	cartItemID := helperAddItemToCart(r, t, 1)

	// Try to delete it as user 2
	delReq := httptest.NewRequest(http.MethodDelete,
		fmt.Sprintf("/cart/%d/items/%d", 2, cartItemID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, delReq)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 when deleting another user's item, got %d", w.Code)
	}
}

// TestTotalPriceCalculation verifies that totalPrice is the sum of all item unitPrices.
func TestTotalPriceCalculation(t *testing.T) {
	r, tx := setupRouter()
	defer tx.Rollback()

	userID := 1
	// Add 3 items — each mocked at 9.99
	helperAddItemToCart(r, t, userID)
	helperAddItemToCart(r, t, userID)
	helperAddItemToCart(r, t, userID)

	getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/cart/%d", userID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, getReq)

	body := w.Body.String()
	// 3 × 9.99 = 29.97
	if !bytes.Contains(w.Body.Bytes(), []byte(`"totalPrice":29.97`)) {
		t.Errorf("expected totalPrice 29.97, got: %s", body)
	}
}

// ---------------------------------------------------------------------------
// Middleware tests
// ---------------------------------------------------------------------------

// TestContentTypeMiddleware verifies that POST without application/json returns
// 400 with a problem+json body.
func TestContentTypeMiddleware(t *testing.T) {
	r, tx := setupRouter()
	defer tx.Rollback()

	req := httptest.NewRequest(http.MethodPost, "/cart/1/items",
		bytes.NewBufferString(`{"productId":1}`))
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Errorf("expected application/problem+json, got %s", ct)
	}

	var response struct {
		Type   string `json:"type"`
		Title  string `json:"title"`
		Status int    `json:"status"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal problem response: %v", err)
	}
	if response.Title != "Invalid Content-Type" {
		t.Errorf("expected title 'Invalid Content-Type', got '%s'", response.Title)
	}
	if response.Type != "about:blank" {
		t.Errorf("expected type 'about:blank', got '%s'", response.Type)
	}
}

// ---------------------------------------------------------------------------
// Health probe tests
// ---------------------------------------------------------------------------

// TestLivezAlwaysOK verifies /livez always returns 200 with {"status":"ok"}.
func TestLivezAlwaysOK(t *testing.T) {
	r, tx := setupRouter()
	defer tx.Rollback()

	req := httptest.NewRequest(http.MethodGet, "/livez", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"status":"ok"`)) {
		t.Errorf("expected status ok, got: %s", w.Body.String())
	}
}

// TestReadyzWithDB verifies /readyz returns 200 when the DB is connected.
func TestReadyzWithDB(t *testing.T) {
	r, tx := setupRouter()
	defer tx.Rollback()

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// Input validation tests
// ---------------------------------------------------------------------------

// TestInvalidUserID verifies that non-integer or zero userId returns 400.
func TestInvalidUserID(t *testing.T) {
	r, tx := setupRouter()
	defer tx.Rollback()

	cases := []string{"/cart/abc", "/cart/0", "/cart/-1"}
	for _, path := range cases {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("path %s: expected 400, got %d", path, w.Code)
		}
	}
}

// TestAddItem_MissingProductId verifies that omitting productId returns 400.
func TestAddItem_MissingProductId(t *testing.T) {
	r, tx := setupRouter()
	defer tx.Rollback()

	req := httptest.NewRequest(http.MethodPost, "/cart/1/items",
		bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing productId, got %d", w.Code)
	}
}

// TestAddItem_InvalidProductIdType verifies that sending a boolean for productId returns 400.
func TestAddItem_InvalidProductIdType(t *testing.T) {
	r, tx := setupRouter()
	defer tx.Rollback()

	req := httptest.NewRequest(http.MethodPost, "/cart/1/items",
		bytes.NewBufferString(`{"productId": true}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for boolean productId, got %d", w.Code)
	}
}
