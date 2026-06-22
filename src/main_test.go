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
	mux.HandleFunc("/products/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{
			"data": {
				"id": "%s",
				"sellerId": "%s",
				"title": "Test Product",
				"category": "Test Category",
				"price": "9.99",
				"stock": 42,
				"status": "active"
			}
		}`, testProductID, testSellerID)
	})
	return httptest.NewServer(mux)
}

// newCatalogNotFoundServer returns a server that always responds 404.
func newCatalogNotFoundServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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
	col := config.GetCartCollection()
	if col == nil {
		zap.S().Warn("Failed to clean test collection: collection is nil")
		return
	}
	if _, err := col.DeleteMany(ctx, bson.D{}); err != nil {
		zap.S().Warnw("Failed to clean test collection", "error", err)
	}
}

// cleanUser removes the cart document for a specific user (used in t.Cleanup).
func cleanUser(userID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	col := config.GetCartCollection()
	if col == nil {
		zap.S().Warnw("Failed to clean user cart: collection is nil", "userID", userID)
		return
	}
	if _, err := col.DeleteMany(ctx, bson.D{{Key: "userId", Value: userID}}); err != nil {
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
	cartRepo := repository.NewCartRepository(config.GetCartCollection)
	catalogClient := service.NewCatalogClient(catalogURL)
	cartService := service.NewCartService(cartRepo, catalogClient)
	cartController := controller.NewCartController(cartService)
	return router.SetupRouter(cartController)
}

// helperAddItemToCart adds testProductID to the given user's cart via the HTTP API.
func helperAddItemToCart(r *gin.Engine, t *testing.T, userID string) {
	t.Helper()

	body := testProductBody
	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/cart/%s/items", userID),
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
	uidGetEmpty        = "11111111-0000-0000-0000-000000000001"
	uidGetCart         = "11111111-0000-0000-0000-000000000002"
	uidAdd             = "11111111-0000-0000-0000-000000000003"
	uidNotFound        = "11111111-0000-0000-0000-000000000004"
	uidUnavailable     = "11111111-0000-0000-0000-000000000005"
	uidClear           = "11111111-0000-0000-0000-000000000006"
	uidRemove          = "11111111-0000-0000-0000-000000000007"
	uidRemoveWrong     = "11111111-0000-0000-0000-000000000008"
	uidRemoveWrongAlt  = "11111111-0000-0000-0000-000000000009"
	uidTotalPrice      = "11111111-0000-0000-0000-000000000010"
	uidContentType     = "11111111-0000-0000-0000-000000000011"
	uidLivez           = "11111111-0000-0000-0000-000000000012"
	uidReadyz          = "11111111-0000-0000-0000-000000000013"
	uidInvalidUser     = "11111111-0000-0000-0000-000000000014"
	uidMissingProd     = "11111111-0000-0000-0000-000000000015"
	uidInvalidType     = "11111111-0000-0000-0000-000000000016"
	uidIncrement       = "11111111-0000-0000-0000-000000000017"
	uidDecrement       = "11111111-0000-0000-0000-000000000018"
	uidDecrementToZero = "11111111-0000-0000-0000-000000000019"
	uidStockSynced     = "11111111-0000-0000-0000-000000000020"
	uidAddCatalogDown  = "11111111-0000-0000-0000-000000000021"
	uidUpdateSnapshot  = "11111111-0000-0000-0000-000000000022"
	uidStockAware      = "11111111-0000-0000-0000-000000000023"
	uidAddSameItem     = "11111111-0000-0000-0000-000000000024"
	uidAddWithQty      = "11111111-0000-0000-0000-000000000025"
)

const (
	testProductID         = "fcbcd5d2-7842-40c2-9ee4-c6ff6d99d670"
	testSellerID          = "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
	testNotFoundProductID = "00000000-0000-0000-0000-000000000000"
	testProductBody       = `{"productId":"` + testProductID + `"}`
	testNotFoundBody      = `{"productId":"` + testNotFoundProductID + `"}`
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

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/cart/%s", uidGetEmpty), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !bytes.Contains(w.Body.Bytes(), []byte(`"items":[]`)) {
		t.Errorf(`expected "items":[], got: %s`, body)
	}
	// Check for the new struct format
	if !bytes.Contains(w.Body.Bytes(), []byte(`"totalPrice":{"value":"0.00"`)) {
		t.Errorf(`expected "totalPrice" with value "0.00", got: %s`, body)
	}
}

// TestGetCart verifies that added items appear in GET /cart/{userId}.
func TestGetCart(t *testing.T) {
	t.Cleanup(func() { cleanUser(uidGetCart) })

	r, srv := setupRouter(t)
	defer srv.Close()
	helperAddItemToCart(r, t, uidGetCart)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/cart/%s", uidGetCart), nil)
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
				ProductID string `json:"productId"`
				Quantity  int    `json:"quantity"`
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

	body := testProductBody
	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/cart/%s/items", uidAdd),
		bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d — body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data struct {
			ProductID string `json:"productId"`
			SellerID  string `json:"sellerId"`
			Title     string `json:"title"`
			UnitPrice struct {
				Value     string `json:"value"`
				UpdatedAt string `json:"updatedAt"`
			} `json:"unitPrice"`
			Quantity int    `json:"quantity"`
			Stock    int    `json:"stock"`
			Status   string `json:"status"`
			AddedAt  string `json:"addedAt"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v — body: %s", err, w.Body.String())
	}

	item := resp.Data
	if item.ProductID != testProductID {
		t.Errorf("expected productId %s, got %s", testProductID, item.ProductID)
	}
	if item.Title != "Test Product" {
		t.Errorf("expected title 'Test Product', got '%s'", item.Title)
	}
	if item.UnitPrice.Value != "9.99" {
		t.Errorf("expected unitPrice value '9.99', got %v", item.UnitPrice.Value)
	}
	if item.UnitPrice.UpdatedAt == "" {
		t.Error("expected unitPrice.updatedAt to be populated")
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
	if item.SellerID != testSellerID {
		t.Errorf("expected sellerId %s, got %s", testSellerID, item.SellerID)
	}
}

// TestSellerIDStoredAndReturned verifies that sellerID captured from the catalog
// on POST /cart/{userId}/items is persisted and returned by GET /cart/{userId}.
func TestSellerIDStoredAndReturned(t *testing.T) {
	uid := "22222222-0000-0000-0000-000000000001"
	t.Cleanup(func() { cleanUser(uid) })

	r, srv := setupRouter(t)
	defer srv.Close()

	// Add item — mock catalog returns testSellerID.
	helperAddItemToCart(r, t, uid)

	// GET cart and assert sellerId is present on the embedded item.
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/cart/%s", uid), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var cartResp struct {
		Data struct {
			Items []struct {
				ProductID string `json:"productId"`
				SellerID  string `json:"sellerId"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &cartResp); err != nil {
		t.Fatalf("failed to parse cart response: %v", err)
	}
	if len(cartResp.Data.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(cartResp.Data.Items))
	}
	if cartResp.Data.Items[0].SellerID != testSellerID {
		t.Errorf("GET cart: expected sellerId %s, got %s", testSellerID, cartResp.Data.Items[0].SellerID)
	}
}

// TestAddItemToCart_WithQuantity verifies that POST /cart/{userId}/items supports
// adding multiple units in a single request using the "quantity" field.
func TestAddItemToCart_WithQuantity(t *testing.T) {
	uid := uidAddWithQty
	t.Cleanup(func() { cleanUser(uid) })

	r, srv := setupRouter(t)
	defer srv.Close()

	body := fmt.Sprintf(`{"productId":"%s","quantity":3}`, testProductID)
	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/cart/%s/items", uid),
		bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d — body: %s", w.Code, w.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/cart/%s", uid), nil)
	wGet := httptest.NewRecorder()
	r.ServeHTTP(wGet, getReq)
	if wGet.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", wGet.Code, wGet.Body.String())
	}

	var cartResp struct {
		Data struct {
			Items []struct {
				ProductID string `json:"productId"`
				Quantity  int    `json:"quantity"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(wGet.Body.Bytes(), &cartResp); err != nil {
		t.Fatalf("failed to parse cart response: %v", err)
	}
	if len(cartResp.Data.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(cartResp.Data.Items))
	}
	if cartResp.Data.Items[0].ProductID != testProductID {
		t.Fatalf("expected productId %s, got %s", testProductID, cartResp.Data.Items[0].ProductID)
	}
	if cartResp.Data.Items[0].Quantity != 3 {
		t.Fatalf("expected quantity 3, got %d", cartResp.Data.Items[0].Quantity)
	}
}

// TestAddSameItemIncrementsQuantity verifies that adding the same productId twice
// increments the quantity to 2 instead of creating a duplicate entry.
func TestAddSameItemIncrementsQuantity(t *testing.T) {
	t.Cleanup(func() { cleanUser(uidAddSameItem) })

	r, srv := setupRouter(t)
	defer srv.Close()

	helperAddItemToCart(r, t, uidAddSameItem)
	helperAddItemToCart(r, t, uidAddSameItem)

	getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/cart/%s", uidAddSameItem), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, getReq)

	var cartResp struct {
		Data struct {
			Items []struct {
				ProductID string `json:"productId"`
				Quantity  int    `json:"quantity"`
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

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/cart/%s/items", uidNotFound),
		bytes.NewBufferString(testNotFoundBody))
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
	downSrv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
	downSrv.Close()

	r := setupRouterWithCatalogURL(downSrv.URL)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/cart/%s/items", uidUnavailable),
		bytes.NewBufferString(testProductBody))
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
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/cart/%s", uidClear), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}

	// Verify cart is empty.
	getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/cart/%s", uidClear), nil)
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
	req2 := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/cart/%s", uidClear), nil)
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
		fmt.Sprintf("/cart/%s/items/%s", uidRemove, testProductID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, delReq)
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d — body: %s", w.Code, w.Body.String())
	}

	// Verify cart is now empty.
	getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/cart/%s", uidRemove), nil)
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
	t.Cleanup(func() { cleanUser(uidRemoveWrongAlt) })

	r, srv := setupRouter(t)
	defer srv.Close()

	// Add item for uidRemoveWrong.
	helperAddItemToCart(r, t, uidRemoveWrong)

	// Try to delete it as uidRemoveWrongAlt (a different user who has no cart).
	delReq := httptest.NewRequest(http.MethodDelete,
		fmt.Sprintf("/cart/%s/items/%s", uidRemoveWrongAlt, testProductID), nil)
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

	getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/cart/%s", uidTotalPrice), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, getReq)

	body := w.Body.String()
	// 3 × 9.99 = 29.97
	if !bytes.Contains(w.Body.Bytes(), []byte(`"totalPrice":{"value":"29.97"`)) {
		t.Errorf("expected totalPrice value 29.97, got: %s", body)
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
		fmt.Sprintf("/cart/%s/%s/increment", uidIncrement, testProductID),
		bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d — body: %s", w.Code, w.Body.String())
	}

	// Verify quantity is now 4.
	getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/cart/%s", uidIncrement), nil)
	wGet := httptest.NewRecorder()
	r.ServeHTTP(wGet, getReq)

	var cartResp struct {
		Data struct {
			Items []struct {
				ProductID string `json:"productId"`
				Quantity  int    `json:"quantity"`
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
		fmt.Sprintf("/cart/%s/%s/decrement", uidDecrement, testProductID),
		bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d — body: %s", w.Code, w.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/cart/%s", uidDecrement), nil)
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

// TestDecrementItemStopsAtOne verifies that decrementing the last unit
// of an item keeps it in the cart with quantity 1.
func TestDecrementItemStopsAtOne(t *testing.T) {
	t.Cleanup(func() { cleanUser(uidDecrementToZero) })

	r, srv := setupRouter(t)
	defer srv.Close()

	// Add once → quantity = 1.
	helperAddItemToCart(r, t, uidDecrementToZero)

	// Decrement by 1 → quantity should stay at 1.
	body := `{"quantity": 1}`
	req := httptest.NewRequest(http.MethodPut,
		fmt.Sprintf("/cart/%s/%s/decrement", uidDecrementToZero, testProductID),
		bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d — body: %s", w.Code, w.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/cart/%s", uidDecrementToZero), nil)
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
		t.Errorf("expected 1 item after decrementing at 1, got %d", len(cartResp.Data.Items))
	} else if cartResp.Data.Items[0].Quantity != 1 {
		t.Errorf("expected quantity 1, got %d", cartResp.Data.Items[0].Quantity)
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
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/cart/%s", uidStockSynced), nil)
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
	downSrv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
	downSrv.Close()

	r := setupRouterWithCatalogURL(downSrv.URL)

	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/cart/%s/items", uidAddCatalogDown),
		bytes.NewBufferString(testProductBody))
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

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/cart/%s/items", uidContentType),
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

// TestInvalidUserID verifies that a non-UUID userId returns 400.
func TestInvalidUserID(t *testing.T) {
	r, srv := setupRouter(t)
	defer srv.Close()

	cases := []string{"/cart/not-a-uuid", "/cart/12345", "/cart/0", "/cart/abc"}
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

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/cart/%s/items", uidMissingProd),
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

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/cart/%s/items", uidInvalidType),
		bytes.NewBufferString(`{"productId": true}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for boolean productId, got %d", w.Code)
	}
}

// TestUpdateProductSnapshot verifies that GET /cart/{userId} synchronizes
// stock and status every time, but only updates the price if it's stale (>24h).
func TestUpdateProductSnapshot(t *testing.T) {
	t.Cleanup(func() { cleanUser(uidUpdateSnapshot) })

	// 1. Setup dynamic catalog mock.
	var (
		priceVal  = "10.00"
		stockVal  = 100
		statusVal = "active"
	)
	mux := http.NewServeMux()
	mux.HandleFunc("/products/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{
			"data": {
				"id": "%s",
				"title": "Snapshot Product",
				"category": "Test Category",
				"price": "%s",
				"stock": %d,
				"status": "%s"
			}
		}`, testProductID, priceVal, stockVal, statusVal)
	})
	catalogSrv := httptest.NewServer(mux)
	defer catalogSrv.Close()

	r := buildRouter(catalogSrv.URL)

	// 2. Initial add — captures "10.00", 100, "active".
	helperAddItemToCart(r, t, uidUpdateSnapshot)

	// 3. Scenario: Fresh price (Catalog changes, but cart snapshot is <24h old).
	priceVal = "15.00"
	stockVal = 50
	statusVal = "limited_stock"

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/cart/%s", uidUpdateSnapshot), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Data struct {
			Items []struct {
				UnitPrice struct {
					Value string `json:"value"`
				} `json:"unitPrice"`
				Stock  int    `json:"stock"`
				Status string `json:"status"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	item := resp.Data.Items[0]
	if item.UnitPrice.Value != "10.00" {
		t.Errorf("fresh price: expected captured price '10.00' to remain, got '%s'", item.UnitPrice.Value)
	}
	if item.Stock != 50 {
		t.Errorf("fresh price: expected stock to sync to 50, got %d", item.Stock)
	}
	if item.Status != "limited_stock" {
		t.Errorf("fresh price: expected status to sync to 'limited_stock', got '%s'", item.Status)
	}

	// 4. Scenario: Stale price (Manually age the snapshot in DB to >24h).
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	staleTime := time.Now().UTC().Add(-25 * time.Hour)
	col := config.GetCartCollection()
	if col == nil {
		t.Fatalf("collection is nil")
	}
	_, err := col.UpdateOne(ctx,
		bson.D{{Key: "userId", Value: uidUpdateSnapshot}, {Key: "items.productId", Value: testProductID}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "items.$.unitPrice.updatedAt", Value: staleTime}}}},
	)
	if err != nil {
		t.Fatalf("failed to manually age price snapshot: %v", err)
	}

	// New catalog state (Price changed again).
	priceVal = "20.00"
	stockVal = 5
	statusVal = "out_of_stock"

	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/cart/%s", uidUpdateSnapshot), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	item = resp.Data.Items[0]
	if item.UnitPrice.Value != "20.00" {
		t.Errorf("stale price: expected price to sync to '20.00', got '%s'", item.UnitPrice.Value)
	}
	if item.Stock != 5 {
		t.Errorf("stale price: expected stock to sync to 5, got %d", item.Stock)
	}
	if item.Status != "out_of_stock" {
		t.Errorf("stale price: expected status to sync to 'out_of_stock', got '%s'", item.Status)
	}
}

// TestStockAwareQuantity verifies that:
// 1. Adding/incrementing beyond catalog stock fails with 422.
// 2. Decrementing stops at quantity 1 (doesn't remove item).
// 3. Every quantity change syncs price, stock and status from catalog.
func TestStockAwareQuantity(t *testing.T) {
	t.Cleanup(func() { cleanUser(uidStockAware) })

	var (
		priceVal  = "10.00"
		stockVal  = 5
		statusVal = "active"
	)
	mux := http.NewServeMux()
	mux.HandleFunc("/products/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{
			"data": {
				"id": "%s",
				"title": "Stock Product",
				"category": "Test Category",
				"price": "%s",
				"stock": %d,
				"status": "%s"
			}
		}`, testProductID, priceVal, stockVal, statusVal)
	})
	catalogSrv := httptest.NewServer(mux)
	defer catalogSrv.Close()

	r := buildRouter(catalogSrv.URL)

	// 1. Initial add - stock is 5.
	helperAddItemToCart(r, t, uidStockAware)

	// 2. Add units until stock is reached (total 5).
	for i := 0; i < 4; i++ {
		helperAddItemToCart(r, t, uidStockAware)
	}

	// 3. Try one more unit beyond stock (total 6 > 5) -> 422.
	body := testProductBody
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/cart/%s/items", uidStockAware), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 for stock exceeded on add, got %d — body: %s", w.Code, w.Body.String())
	}

	// 4. Try to increment beyond stock -> 422.
	incrBody := `{"quantity": 1}`
	req = httptest.NewRequest(http.MethodPut, fmt.Sprintf("/cart/%s/%s/increment", uidStockAware, testProductID), bytes.NewBufferString(incrBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 for increment beyond stock, got %d", w.Code)
	}

	// 5. Increase stock in catalog and test sync on increment.
	priceVal = "12.00"
	stockVal = 10
	statusVal = "active"

	// Current quantity is 5. Stock is now 10. Increment by 1.
	req = httptest.NewRequest(http.MethodPut, fmt.Sprintf("/cart/%s/%s/increment", uidStockAware, testProductID), bytes.NewBufferString(incrBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204 for valid increment, got %d", w.Code)
	}

	// Verify sync.
	getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/cart/%s", uidStockAware), nil)
	wGet := httptest.NewRecorder()
	r.ServeHTTP(wGet, getReq)
	if !bytes.Contains(wGet.Body.Bytes(), []byte(`"value":"12.00"`)) {
		t.Errorf("expected price sync to 12.00, got: %s", wGet.Body.String())
	}
	if !bytes.Contains(wGet.Body.Bytes(), []byte(`"stock":10`)) {
		t.Errorf("expected stock sync to 10, got: %s", wGet.Body.String())
	}

	// 6. Sync on decrement and minimum quantity 1.
	priceVal = "8.00"
	statusVal = "on_sale"

	// Decrement by 10 (current is 6). Result should be 1.
	decrBody := `{"quantity": 10}`
	req = httptest.NewRequest(http.MethodPut, fmt.Sprintf("/cart/%s/%s/decrement", uidStockAware, testProductID), bytes.NewBufferString(decrBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204 for decrement, got %d", w.Code)
	}

	// Verify quantity is 1 and price/status synced.
	wSync := httptest.NewRecorder()
	r.ServeHTTP(wSync, getReq)
	if !bytes.Contains(wSync.Body.Bytes(), []byte(`"quantity":1`)) {
		t.Errorf("expected quantity to be 1, got: %s", wSync.Body.String())
	}
	if !bytes.Contains(wSync.Body.Bytes(), []byte(`"value":"8.00"`)) {
		t.Errorf("expected price sync to 8.00, got: %s", wSync.Body.String())
	}
	if !bytes.Contains(wSync.Body.Bytes(), []byte(`"status":"on_sale"`)) {
		t.Errorf("expected status sync to 'on_sale', got: %s", wSync.Body.String())
	}
}

// ensure time is imported (used implicitly by cleanUser/cleanCollection timeouts).
var _ = time.Second
