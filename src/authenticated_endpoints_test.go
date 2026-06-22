package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// helperAddItemToOwnCart adds testProductID to the given user's cart via the
// authenticated (own-cart) HTTP API by setting the X-User-ID header.
func helperAddItemToOwnCart(r *gin.Engine, t *testing.T, userID string) {
	t.Helper()

	body := testProductBody
	req := httptest.NewRequest(http.MethodPost,
		"/cart/items",
		bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("helperAddItemToOwnCart: expected 201, got %d — body: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// GET /cart (own)
// ---------------------------------------------------------------------------

func TestGetOwnCart(t *testing.T) {
	t.Cleanup(func() { cleanUser(uidGetCart) })

	r, srv := setupRouter(t)
	defer srv.Close()

	helperAddItemToOwnCart(r, t, uidGetCart)

	req := httptest.NewRequest(http.MethodGet, "/cart", nil)
	req.Header.Set("X-User-ID", uidGetCart)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("Test Product")) {
		t.Errorf("expected cart to contain 'Test Product', got: %s", w.Body.String())
	}
}

func TestGetOwnCart_MissingHeader(t *testing.T) {
	r, srv := setupRouter(t)
	defer srv.Close()

	req := httptest.NewRequest(http.MethodGet, "/cart", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 when X-User-ID missing, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Errorf("expected Content-Type application/problem+json, got %s", ct)
	}

	var resp struct {
		Title string `json:"title"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse problem response: %v", err)
	}
	if resp.Title != "Missing User ID" {
		t.Errorf("expected problem title 'Missing User ID', got '%s'", resp.Title)
	}
}

func TestGetOwnCart_InvalidHeader(t *testing.T) {
	r, srv := setupRouter(t)
	defer srv.Close()

	req := httptest.NewRequest(http.MethodGet, "/cart", nil)
	req.Header.Set("X-User-ID", "not-a-uuid")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 when X-User-ID invalid, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// POST /cart/items (own)
// ---------------------------------------------------------------------------

func TestAddToOwnCart(t *testing.T) {
	t.Cleanup(func() { cleanUser(uidAdd) })

	r, srv := setupRouter(t)
	defer srv.Close()

	body := testProductBody
	req := httptest.NewRequest(http.MethodPost, "/cart/items", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", uidAdd)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d — body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data struct {
			ProductID string `json:"productId"`
			Title     string `json:"title"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v — body: %s", err, w.Body.String())
	}
	if resp.Data.ProductID != testProductID {
		t.Errorf("expected productId %s, got %s", testProductID, resp.Data.ProductID)
	}
	if resp.Data.Title != "Test Product" {
		t.Errorf("expected title 'Test Product', got '%s'", resp.Data.Title)
	}
}

func TestAddToOwnCart_MissingHeader(t *testing.T) {
	r, srv := setupRouter(t)
	defer srv.Close()

	body := testProductBody
	req := httptest.NewRequest(http.MethodPost, "/cart/items", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 when X-User-ID missing, got %d", w.Code)
	}
}

func TestAddToOwnCart_InvalidHeader(t *testing.T) {
	r, srv := setupRouter(t)
	defer srv.Close()

	body := testProductBody
	req := httptest.NewRequest(http.MethodPost, "/cart/items", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "invalid-uid")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 when X-User-ID invalid, got %d", w.Code)
	}
}

func TestAddToOwnCart_ProductNotFound(t *testing.T) {
	t.Cleanup(func() { cleanUser(uidNotFound) })

	notFoundSrv := newCatalogNotFoundServer(t)
	defer notFoundSrv.Close()
	r := setupRouterWithCatalogURL(notFoundSrv.URL)

	req := httptest.NewRequest(http.MethodPost, "/cart/items", bytes.NewBufferString(testNotFoundBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", uidNotFound)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d — body: %s", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Errorf("expected Content-Type application/problem+json, got %s", ct)
	}
}

func TestAddToOwnCart_CatalogUnavailable(t *testing.T) {
	t.Cleanup(func() { cleanUser(uidUnavailable) })

	downSrv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
	downSrv.Close()

	r := setupRouterWithCatalogURL(downSrv.URL)

	req := httptest.NewRequest(http.MethodPost, "/cart/items", bytes.NewBufferString(testProductBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", uidUnavailable)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d — body: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// DELETE /cart/items/:cart_item_id (own)
// ---------------------------------------------------------------------------

func TestRemoveFromOwnCart(t *testing.T) {
	t.Cleanup(func() { cleanUser(uidRemove) })

	r, srv := setupRouter(t)
	defer srv.Close()

	helperAddItemToOwnCart(r, t, uidRemove)

	delReq := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/cart/items/%s", testProductID), nil)
	delReq.Header.Set("X-User-ID", uidRemove)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, delReq)
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d — body: %s", w.Code, w.Body.String())
	}

	// Verify cart is empty.
	getReq := httptest.NewRequest(http.MethodGet, "/cart", nil)
	getReq.Header.Set("X-User-ID", uidRemove)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, getReq)

	var cartResp struct {
		Data struct {
			Items []interface{} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w2.Body.Bytes(), &cartResp); err != nil {
		t.Fatalf("failed to parse cart response: %v", err)
	}
	if len(cartResp.Data.Items) != 0 {
		t.Errorf("expected 0 items after remove, got %d", len(cartResp.Data.Items))
	}
}

func TestRemoveFromOwnCart_MissingHeader(t *testing.T) {
	r, srv := setupRouter(t)
	defer srv.Close()

	delReq := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/cart/items/%s", testProductID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, delReq)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 when X-User-ID missing, got %d", w.Code)
	}
}

func TestRemoveFromOwnCart_NotFound(t *testing.T) {
	t.Cleanup(func() { cleanUser(uidRemoveWrong) })

	r, srv := setupRouter(t)
	defer srv.Close()

	// Do not add item for this user. Attempt to remove.
	delReq := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/cart/items/%s", testProductID), nil)
	delReq.Header.Set("X-User-ID", uidRemoveWrong)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, delReq)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 when removing non-existent item, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// PUT /cart/increment/:product_id (own)
// ---------------------------------------------------------------------------

func TestIncrementOwnItem(t *testing.T) {
	t.Cleanup(func() { cleanUser(uidIncrement) })

	r, srv := setupRouter(t)
	defer srv.Close()

	// Start with quantity = 1.
	helperAddItemToOwnCart(r, t, uidIncrement)

	// Increment by 3.
	body := `{"quantity": 3}`
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/cart/increment/%s", testProductID), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", uidIncrement)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d — body: %s", w.Code, w.Body.String())
	}

	// Verify quantity is now 4.
	getReq := httptest.NewRequest(http.MethodGet, "/cart", nil)
	getReq.Header.Set("X-User-ID", uidIncrement)
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

func TestIncrementOwnItem_MissingHeader(t *testing.T) {
	r, srv := setupRouter(t)
	defer srv.Close()

	body := `{"quantity": 1}`
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/cart/increment/%s", testProductID), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 when X-User-ID missing on increment, got %d", w.Code)
	}
}

func TestIncrementOwnItem_InsufficientStock(t *testing.T) {
	t.Cleanup(func() { cleanUser(uidStockAware) })

	var (
		priceVal  = "10.00"
		stockVal  = 1
		statusVal = "active"
	)
	mux := http.NewServeMux()
	mux.HandleFunc("/products/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{
			"data": {
				"id": "%s",
				"title": "Stock Product",
				"price": "%s",
				"stock": %d,
				"status": "%s"
			}
		}`, testProductID, priceVal, stockVal, statusVal)
	})
	catalogSrv := httptest.NewServer(mux)
	defer catalogSrv.Close()

	r := setupRouterWithCatalogURL(catalogSrv.URL)

	// Initial add - stock is 1.
	helperAddItemToOwnCart(r, t, uidStockAware)

	// Try to increment by 1 -> would exceed stock -> expect 422.
	incrBody := `{"quantity": 1}`
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/cart/increment/%s", testProductID), bytes.NewBufferString(incrBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", uidStockAware)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 for increment beyond stock, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// PUT /cart/decrement/:product_id (own)
// ---------------------------------------------------------------------------

func TestDecrementOwnItem(t *testing.T) {
	t.Cleanup(func() { cleanUser(uidDecrement) })

	r, srv := setupRouter(t)
	defer srv.Close()

	// Add twice so quantity = 2.
	helperAddItemToOwnCart(r, t, uidDecrement)
	helperAddItemToOwnCart(r, t, uidDecrement)

	// Decrement by 1 → expected quantity = 1.
	body := `{"quantity": 1}`
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/cart/decrement/%s", testProductID), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", uidDecrement)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d — body: %s", w.Code, w.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/cart", nil)
	getReq.Header.Set("X-User-ID", uidDecrement)
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

func TestDecrementOwnItem_MissingHeader(t *testing.T) {
	r, srv := setupRouter(t)
	defer srv.Close()

	body := `{"quantity": 1}`
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/cart/decrement/%s", testProductID), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 when X-User-ID missing on decrement, got %d", w.Code)
	}
}
