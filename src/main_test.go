package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/bazaar-grupo16/cart-microservice/src/config"
	"github.com/bazaar-grupo16/cart-microservice/src/controller"
	"github.com/bazaar-grupo16/cart-microservice/src/repository"
	"github.com/bazaar-grupo16/cart-microservice/src/router"
	"github.com/bazaar-grupo16/cart-microservice/src/service"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// catalogMockServer — wraps an httptest.Server that simulates the Catalog API.
// It allows tests to control whether the catalog is healthy, returns a 404,
// or is completely unavailable (server closed).
// ---------------------------------------------------------------------------

// newCatalogMockServer starts an httptest.Server that returns a fixed product
// for any GET /products/{id} request.
// The caller is responsible for calling server.Close() in a t.Cleanup call.
func newCatalogMockServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/products/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{
			"data": {
				"id": 1,
				"title": "Test Product",
				"price": "9.99",
				"stock": 42,
				"status": "active"
			}
		}`)
	})
	return httptest.NewServer(mux)
}

// newCatalogNotFoundServer returns a server that always responds 404.
func newCatalogNotFoundServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"type":"about:blank","title":"Not Found","status":404,"detail":"Product not found","instance":"/products/9999"}`)
	}))
}

// ---------------------------------------------------------------------------
// TestMain — suite-level setup and teardown.
// ---------------------------------------------------------------------------

func TestMain(m *testing.M) {
	logger, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(logger)

	// Load .env (optional — CI injects vars directly).
	if err := godotenv.Load("../.env", ".env"); err != nil {
		zap.S().Info("No .env file found; using system environment variables")
	}

	// Ensure we target the test database, not the production one.
	if os.Getenv("MONGO_DB_NAME") == "" {
		os.Setenv("MONGO_DB_NAME", "cart_test")
	}

	config.ConnectDatabase()

	// Clean the collection before and after all tests run.
	cleanCollection()
	code := m.Run()
	cleanCollection()

	os.Exit(code)
}

// cleanCollection deletes all documents from the carts collection.
func cleanCollection() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := config.CartCollection.DeleteMany(ctx, bson.D{}); err != nil {
		zap.S().Warnw("Failed to clean test collection", "error", err)
	}
}

// cleanUser removes the cart document for a specific user (used in t.Cleanup).
func cleanUser(userID int) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := config.CartCollection.DeleteMany(ctx, bson.D{{Key: "userId", Value: userID}}); err != nil {
		zap.S().Warnw("Failed to clean user cart", "userID", userID, "error", err)
	}
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// setupRouter creates a Gin router backed by the real MongoDB collection
// and a real CatalogClient pointing at a local httptest server.
func setupRouter(t *testing.T) (*gin.Engine, *httptest.Server) {
	t.Helper()
	catalogSrv := newCatalogMockServer(t)
	r := buildRouter(catalogSrv.URL)
	return r, catalogSrv
}

// setupRouterWithCatalogURL creates a router using the given catalog base URL.
func setupRouterWithCatalogURL(catalogURL string) *gin.Engine {
	return buildRouter(catalogURL)
}

func buildRouter(catalogURL string) *gin.Engine {
	cartRepo := repository.NewCartRepository(config.CartCollection)
	catalogClient := service.NewCatalogClient(catalogURL)
	cartService := service.NewCartService(cartRepo, catalogClient)
	cartController := controller.NewCartController(cartService)
	return router.SetupRouter(cartController)
}

// helperAddItemToCart adds productId=1 to the given user's cart via the HTTP API.
func helperAddItemToCart(r *gin.Engine, t *testing.T, userID int) {
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
}

// ---------------------------------------------------------------------------
// User ID constants — each test uses a unique one to avoid collisions.
// ---------------------------------------------------------------------------

const (
	uidGetEmpty           = 1001
	uidGetCart            = 1002
	uidAdd                = 1003
	uidNotFound           = 1004
	uidUnavailable        = 1005
	uidClear              = 1006
	uidRemove             = 1007
	uidRemoveWrong        = 1008
	uidTotalPrice         = 1009
	uidContentType        = 1010
	uidLivez              = 1011
	uidReadyz             = 1012
	uidInvalidUser        = 1013
	uidMissingProd        = 1014
	uidInvalidType        = 1015
	uidIncrement          = 1016
	uidDecrement          = 1017
	uidDecrementToZero    = 1018
	uidStockSynced        = 1019
	uidAddCatalogDown     = 1020
)

// ---------------------------------------------------------------------------
// Cart tests
// ---------------------------------------------------------------------------

// TestGetEmptyCart verifies that GET /cart/{userId} for a new user returns
// an empty items array and a totalPrice of 0.00.
func TestGetEmptyCart(t *testing.T) {
	t.Cleanup(func() { cleanUser(uidGetEmpty) })

	r, srv := setupRouter(t)
	defer srv.Close()

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/cart/%d", uidGetEmpty), nil)
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
	t.Cleanup(func() { cleanUser(uidGetCart) })

	r, srv := setupRouter(t)
	defer srv.Close()
	helperAddItemToCart(r, t, uidGetCart)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/cart/%d", uidGetCart), nil)
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
				ProductID int `json:"productId"`
				Quantity  int `json:"quantity"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &cartResp); err != nil {
		t.Fatalf("failed to parse cart response: %v", err)
	}
	if len(cartResp.Data.Items) != 1 {
		t.Errorf("expected 1 distinct cart item, got %d", len(cartResp.Data.Items))
	}
	if cartResp.Data.Items[0].Quantity != 1 {
		t.Errorf("expected quantity 1, got %d", cartResp.Data.Items[0].Quantity)
	}
}

// TestAddItemToCart verifies POST /cart/{userId}/items creates a CartItem with
// the snapshot price, title, quantity=1, stock and status from the mock catalog.
func TestAddItemToCart(t *testing.T) {
	t.Cleanup(func() { cleanUser(uidAdd) })

	r, srv := setupRouter(t)
	defer srv.Close()

	body := `{"productId": 1}`
	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/cart/%d/items", uidAdd),
		bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d — body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data struct {
			ProductID int     `json:"productId"`
			Title     string  `json:"title"`
			UnitPrice float64 `json:"unitPrice"`
			Quantity  int     `json:"quantity"`
			Stock     int     `json:"stock"`
			Status    string  `json:"status"`
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
	if item.Quantity != 1 {
		t.Errorf("expected quantity 1 on first add, got %d", item.Quantity)
	}
	if item.Stock != 42 {
		t.Errorf("expected stock 42, got %d", item.Stock)
	}
	if item.Status != "active" {
		t.Errorf("expected status 'active', got '%s'", item.Status)
	}
	if item.AddedAt == "" {
		t.Error("expected addedAt to be populated")
	}
}

// TestAddSameItemIncrementsQuantity verifies that adding the same productId twice
// increments the quantity to 2 instead of creating a duplicate entry.
func TestAddSameItemIncrementsQuantity(t *testing.T) {
	t.Cleanup(func() { cleanUser(uidAdd + 100) })

	r, srv := setupRouter(t)
	defer srv.Close()

	helperAddItemToCart(r, t, uidAdd+100)
	helperAddItemToCart(r, t, uidAdd+100)

	getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/cart/%d", uidAdd+100), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, getReq)

	var cartResp struct {
		Data struct {
			Items []struct {
				ProductID int `json:"productId"`
				Quantity  int `json:"quantity"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &cartResp); err != nil {
		t.Fatalf("failed to parse cart response: %v", err)
	}
	if len(cartResp.Data.Items) != 1 {
		t.Errorf("expected 1 distinct item, got %d", len(cartResp.Data.Items))
	}
	if cartResp.Data.Items[0].Quantity != 2 {
		t.Errorf("expected quantity 2 after two adds, got %d", cartResp.Data.Items[0].Quantity)
	}
}

// TestAddItemToCart_ProductNotFound verifies that a 404 from the catalog
// results in a 404 problem+json response.
func TestAddItemToCart_ProductNotFound(t *testing.T) {
	t.Cleanup(func() { cleanUser(uidNotFound) })

	notFoundSrv := newCatalogNotFoundServer(t)
	defer notFoundSrv.Close()
	r := setupRouterWithCatalogURL(notFoundSrv.URL)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/cart/%d/items", uidNotFound),
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

// TestAddItemToCart_CatalogUnavailable verifies that a closed catalog server
// (simulating network failure) results in a 503 response.
func TestAddItemToCart_CatalogUnavailable(t *testing.T) {
	t.Cleanup(func() { cleanUser(uidUnavailable) })

	// Start and immediately close the server — simulates catalog being down.
	downSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	downSrv.Close()

	r := setupRouterWithCatalogURL(downSrv.URL)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/cart/%d/items", uidUnavailable),
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
	t.Cleanup(func() { cleanUser(uidClear) })

	r, srv := setupRouter(t)
	defer srv.Close()
	helperAddItemToCart(r, t, uidClear)

	// First clear.
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/cart/%d", uidClear), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}

	// Verify cart is empty.
	getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/cart/%d", uidClear), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, getReq)

	var cartResp struct {
		Data struct {
			Items []interface{} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &cartResp); err != nil {
		t.Fatalf("failed to parse cart response: %v", err)
	}
	if len(cartResp.Data.Items) != 0 {
		t.Errorf("expected empty cart after clear, got %d items", len(cartResp.Data.Items))
	}

	// Second clear on empty cart must still return 204 (idempotent).
	req2 := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/cart/%d", uidClear), nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusNoContent {
		t.Errorf("expected idempotent 204 on empty cart, got %d", w2.Code)
	}
}

// TestRemoveItemFromCart verifies DELETE /cart/{userId}/items/{productId}.
func TestRemoveItemFromCart(t *testing.T) {
	t.Cleanup(func() { cleanUser(uidRemove) })

	r, srv := setupRouter(t)
	defer srv.Close()

	// Add productId=1 so it exists in the cart.
	helperAddItemToCart(r, t, uidRemove)

	// Remove by productId.
	delReq := httptest.NewRequest(http.MethodDelete,
		fmt.Sprintf("/cart/%d/items/%d", uidRemove, 1), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, delReq)
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d — body: %s", w.Code, w.Body.String())
	}

	// Verify cart is now empty.
	getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/cart/%d", uidRemove), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, getReq)

	var cartResp struct {
		Data struct {
			Items []interface{} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &cartResp); err != nil {
		t.Fatalf("failed to parse cart response: %v", err)
	}
	if len(cartResp.Data.Items) != 0 {
		t.Errorf("expected 0 items after remove, got %d", len(cartResp.Data.Items))
	}
}

// TestRemoveItemFromCart_WrongUser verifies that deleting a product from a user
// who does not have it in their cart returns 404.
func TestRemoveItemFromCart_WrongUser(t *testing.T) {
	t.Cleanup(func() { cleanUser(uidRemoveWrong) })
	t.Cleanup(func() { cleanUser(uidRemoveWrong + 1) })

	r, srv := setupRouter(t)
	defer srv.Close()

	// Add item for uidRemoveWrong.
	helperAddItemToCart(r, t, uidRemoveWrong)

	// Try to delete it as uidRemoveWrong+1 (a different user who has no cart).
	delReq := httptest.NewRequest(http.MethodDelete,
		fmt.Sprintf("/cart/%d/items/%d", uidRemoveWrong+1, 1), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, delReq)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 when deleting another user's item, got %d", w.Code)
	}
}

// TestTotalPriceCalculation verifies that totalPrice is the sum of (unitPrice × quantity).
// Adds the same product 3 times — each call increments quantity — so
// totalPrice should be 3 × 9.99 = 29.97.
func TestTotalPriceCalculation(t *testing.T) {
	t.Cleanup(func() { cleanUser(uidTotalPrice) })

	r, srv := setupRouter(t)
	defer srv.Close()

	helperAddItemToCart(r, t, uidTotalPrice)
	helperAddItemToCart(r, t, uidTotalPrice)
	helperAddItemToCart(r, t, uidTotalPrice)

	getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/cart/%d", uidTotalPrice), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, getReq)

	body := w.Body.String()
	// 3 × 9.99 = 29.97
	if !bytes.Contains(w.Body.Bytes(), []byte(`"totalPrice":29.97`)) {
		t.Errorf("expected totalPrice 29.97, got: %s", body)
	}
}

// ---------------------------------------------------------------------------
// Increment / Decrement tests
// ---------------------------------------------------------------------------

// TestIncrementItem verifies that PUT /cart/{userId}/{productId}/increment
// increases the item's quantity by the requested amount.
func TestIncrementItem(t *testing.T) {
	t.Cleanup(func() { cleanUser(uidIncrement) })

	r, srv := setupRouter(t)
	defer srv.Close()

	// Start with quantity = 1.
	helperAddItemToCart(r, t, uidIncrement)

	// Increment by 3.
	body := `{"quantity": 3}`
	req := httptest.NewRequest(http.MethodPut,
		fmt.Sprintf("/cart/%d/1/increment", uidIncrement),
		bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d — body: %s", w.Code, w.Body.String())
	}

	// Verify quantity is now 4.
	getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/cart/%d", uidIncrement), nil)
	wGet := httptest.NewRecorder()
	r.ServeHTTP(wGet, getReq)

	var cartResp struct {
		Data struct {
			Items []struct {
				ProductID int `json:"productId"`
				Quantity  int `json:"quantity"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(wGet.Body.Bytes(), &cartResp); err != nil {
		t.Fatalf("failed to parse cart response: %v", err)
	}
	if len(cartResp.Data.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(cartResp.Data.Items))
	}
	if cartResp.Data.Items[0].Quantity != 4 {
		t.Errorf("expected quantity 4 after increment by 3, got %d", cartResp.Data.Items[0].Quantity)
	}
}

// TestDecrementItem verifies that PUT /cart/{userId}/{productId}/decrement
// decreases the item's quantity when the result remains > 0.
func TestDecrementItem(t *testing.T) {
	t.Cleanup(func() { cleanUser(uidDecrement) })

	r, srv := setupRouter(t)
	defer srv.Close()

	// Add twice so quantity = 2.
	helperAddItemToCart(r, t, uidDecrement)
	helperAddItemToCart(r, t, uidDecrement)

	// Decrement by 1 → expected quantity = 1.
	body := `{"quantity": 1}`
	req := httptest.NewRequest(http.MethodPut,
		fmt.Sprintf("/cart/%d/1/decrement", uidDecrement),
		bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d — body: %s", w.Code, w.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/cart/%d", uidDecrement), nil)
	wGet := httptest.NewRecorder()
	r.ServeHTTP(wGet, getReq)

	var cartResp struct {
		Data struct {
			Items []struct {
				Quantity int `json:"quantity"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(wGet.Body.Bytes(), &cartResp); err != nil {
		t.Fatalf("failed to parse cart response: %v", err)
	}
	if len(cartResp.Data.Items) != 1 {
		t.Fatalf("expected 1 item to remain, got %d", len(cartResp.Data.Items))
	}
	if cartResp.Data.Items[0].Quantity != 1 {
		t.Errorf("expected quantity 1, got %d", cartResp.Data.Items[0].Quantity)
	}
}

// TestDecrementItemRemovesWhenZero verifies that decrementing the last unit
// of an item removes it entirely from the cart.
func TestDecrementItemRemovesWhenZero(t *testing.T) {
	t.Cleanup(func() { cleanUser(uidDecrementToZero) })

	r, srv := setupRouter(t)
	defer srv.Close()

	// Add once → quantity = 1.
	helperAddItemToCart(r, t, uidDecrementToZero)

	// Decrement by 1 → quantity becomes 0 → item should be removed.
	body := `{"quantity": 1}`
	req := httptest.NewRequest(http.MethodPut,
		fmt.Sprintf("/cart/%d/1/decrement", uidDecrementToZero),
		bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d — body: %s", w.Code, w.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/cart/%d", uidDecrementToZero), nil)
	wGet := httptest.NewRecorder()
	r.ServeHTTP(wGet, getReq)

	var cartResp struct {
		Data struct {
			Items []interface{} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(wGet.Body.Bytes(), &cartResp); err != nil {
		t.Fatalf("failed to parse cart response: %v", err)
	}
	if len(cartResp.Data.Items) != 0 {
		t.Errorf("expected 0 items after decrement to zero, got %d", len(cartResp.Data.Items))
	}
}

// ---------------------------------------------------------------------------
// Stock synchronization tests
// ---------------------------------------------------------------------------

// TestGetCart_StockSynced verifies that GET /cart/{userId} returns items with
// up-to-date stock and status values fetched from the Catalog service.
func TestGetCart_StockSynced(t *testing.T) {
	t.Cleanup(func() { cleanUser(uidStockSynced) })

	r, srv := setupRouter(t)
	defer srv.Close()

	// Add item first — stock should be captured from catalog.
	helperAddItemToCart(r, t, uidStockSynced)

	// GET should re-sync stock/status.
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/cart/%d", uidStockSynced), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var cartResp struct {
		Data struct {
			Items []struct {
				Stock  int    `json:"stock"`
				Status string `json:"status"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &cartResp); err != nil {
		t.Fatalf("failed to parse cart response: %v", err)
	}
	if len(cartResp.Data.Items) == 0 {
		t.Fatal("expected at least 1 item")
	}
	if cartResp.Data.Items[0].Stock != 42 {
		t.Errorf("expected stock 42 from catalog mock, got %d", cartResp.Data.Items[0].Stock)
	}
	if cartResp.Data.Items[0].Status != "active" {
		t.Errorf("expected status 'active' from catalog mock, got '%s'", cartResp.Data.Items[0].Status)
	}
}

// TestAddToCart_CatalogDown verifies that a POST /cart/{userId}/items returns 503
// when the catalog service is unreachable.
func TestAddToCart_CatalogDown(t *testing.T) {
	t.Cleanup(func() { cleanUser(uidAddCatalogDown) })

	// Start and immediately close the server — simulates catalog being down.
	downSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	downSrv.Close()

	r := setupRouterWithCatalogURL(downSrv.URL)

	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/cart/%d/items", uidAddCatalogDown),
		bytes.NewBufferString(`{"productId": 1}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 when catalog is down, got %d — body: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// Middleware tests
// ---------------------------------------------------------------------------

// TestContentTypeMiddleware verifies that POST without application/json returns
// 400 with a problem+json body.
func TestContentTypeMiddleware(t *testing.T) {
	t.Cleanup(func() { cleanUser(uidContentType) })

	r, srv := setupRouter(t)
	defer srv.Close()

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/cart/%d/items", uidContentType),
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
	r, srv := setupRouter(t)
	defer srv.Close()

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

// TestReadyzWithDB verifies /readyz returns 200 when MongoDB is connected.
func TestReadyzWithDB(t *testing.T) {
	r, srv := setupRouter(t)
	defer srv.Close()

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
	r, srv := setupRouter(t)
	defer srv.Close()

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
	t.Cleanup(func() { cleanUser(uidMissingProd) })

	r, srv := setupRouter(t)
	defer srv.Close()

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/cart/%d/items", uidMissingProd),
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
	t.Cleanup(func() { cleanUser(uidInvalidType) })

	r, srv := setupRouter(t)
	defer srv.Close()

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/cart/%d/items", uidInvalidType),
		bytes.NewBufferString(`{"productId": true}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for boolean productId, got %d", w.Code)
	}
}

// ensure time is imported (used implicitly by cleanUser/cleanCollection timeouts).
var _ = time.Second
