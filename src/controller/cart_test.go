package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bazaar-grupo16/cart-microservice/src/models"
	"github.com/bazaar-grupo16/cart-microservice/src/repository"
	"github.com/bazaar-grupo16/cart-microservice/src/service"
	"github.com/gin-gonic/gin"
)

type MockCartService struct {
	GetCartByUserIDFn func(ctx context.Context, userID string) (*models.Cart, error)
	ClearCartFn       func(ctx context.Context, userID string) error
	AddToCartFn       func(ctx context.Context, userID string, item *models.CartItem) error
	RemoveFromCartFn  func(ctx context.Context, userID string, productID string) error
	IncrementItemFn   func(ctx context.Context, userID string, productID string, qty int) error
	DecrementItemFn   func(ctx context.Context, userID string, productID string, qty int) error
}

func (m *MockCartService) GetCartByUserID(ctx context.Context, userID string) (*models.Cart, error) {
	if m.GetCartByUserIDFn != nil {
		return m.GetCartByUserIDFn(ctx, userID)
	}
	return nil, nil
}
func (m *MockCartService) ClearCart(ctx context.Context, userID string) error {
	if m.ClearCartFn != nil {
		return m.ClearCartFn(ctx, userID)
	}
	return nil
}
func (m *MockCartService) AddToCart(ctx context.Context, userID string, item *models.CartItem) error {
	if m.AddToCartFn != nil {
		return m.AddToCartFn(ctx, userID, item)
	}
	return nil
}
func (m *MockCartService) RemoveFromCart(ctx context.Context, userID string, productID string) error {
	if m.RemoveFromCartFn != nil {
		return m.RemoveFromCartFn(ctx, userID, productID)
	}
	return nil
}
func (m *MockCartService) IncrementItem(ctx context.Context, userID string, productID string, qty int) error {
	if m.IncrementItemFn != nil {
		return m.IncrementItemFn(ctx, userID, productID, qty)
	}
	return nil
}
func (m *MockCartService) DecrementItem(ctx context.Context, userID string, productID string, qty int) error {
	if m.DecrementItemFn != nil {
		return m.DecrementItemFn(ctx, userID, productID, qty)
	}
	return nil
}

func setupTestRouter(c *CartController) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Unauthenticated routes
	r.GET("/cart/:user_id", c.GetCart)
	r.DELETE("/cart/:user_id", c.ClearCart)
	r.POST("/cart/:user_id/items", c.AddToCart)
	r.DELETE("/cart/:user_id/items/:cart_item_id", c.RemoveFromCart)
	r.PUT("/cart/:user_id/:product_id/increment", c.IncrementItem)
	r.PUT("/cart/:user_id/:product_id/decrement", c.DecrementItem)

	// Authenticated routes
	r.GET("/cart", c.GetOwnCart)
	r.DELETE("/cart", c.ClearOwnCart)
	r.POST("/cart/items", c.AddToOwnCart)
	r.DELETE("/cart/items/:cart_item_id", c.RemoveFromOwnCart)
	r.PUT("/cart/increment/:product_id", c.IncrementOwnItem)
	r.PUT("/cart/decrement/:product_id", c.DecrementOwnItem)

	return r
}

func TestGetCart(t *testing.T) {
	validUUID := "550e8400-e29b-41d4-a716-446655440000"
	invalidUUID := "invalid-uuid"

	tests := []struct {
		name           string
		userID         string
		mockReturnCart *models.Cart
		mockReturnErr  error
		expectedStatus int
	}{
		{
			name:           "Valid user ID - success",
			userID:         validUUID,
			mockReturnCart: models.NewCart(validUUID, nil),
			mockReturnErr:  nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid user ID",
			userID:         invalidUUID,
			mockReturnCart: nil,
			mockReturnErr:  nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Service error",
			userID:         validUUID,
			mockReturnCart: nil,
			mockReturnErr:  errors.New("internal err"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &MockCartService{
				GetCartByUserIDFn: func(_ context.Context, _ string) (*models.Cart, error) {
					return tt.mockReturnCart, tt.mockReturnErr
				},
			}
			ctrl := NewCartController(mockSvc)
			r := setupTestRouter(ctrl)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/cart/"+tt.userID, nil)
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestClearCart(t *testing.T) {
	validUUID := "550e8400-e29b-41d4-a716-446655440000"

	tests := []struct {
		name           string
		userID         string
		mockReturnErr  error
		expectedStatus int
	}{
		{
			name:           "Valid user ID - success",
			userID:         validUUID,
			mockReturnErr:  nil,
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "Invalid user ID",
			userID:         "invalid",
			mockReturnErr:  nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Service error",
			userID:         validUUID,
			mockReturnErr:  errors.New("db err"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &MockCartService{
				ClearCartFn: func(_ context.Context, _ string) error {
					return tt.mockReturnErr
				},
			}
			ctrl := NewCartController(mockSvc)
			r := setupTestRouter(ctrl)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodDelete, "/cart/"+tt.userID, nil)
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestAddToCart(t *testing.T) {
	validUUID := "550e8400-e29b-41d4-a716-446655440000"
	validProdUUID := "770e8400-e29b-41d4-a716-446655440000"
	qty := 2

	tests := []struct {
		name           string
		userID         string
		body           interface{}
		mockReturnErr  error
		expectedStatus int
	}{
		{
			name:           "Valid request",
			userID:         validUUID,
			body:           models.AddCartRequest{ProductID: validProdUUID, Quantity: &qty},
			mockReturnErr:  nil,
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "Invalid User ID",
			userID:         "invalid",
			body:           models.AddCartRequest{ProductID: validProdUUID},
			mockReturnErr:  nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid JSON",
			userID:         validUUID,
			body:           "not a json",
			mockReturnErr:  nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid Product ID",
			userID:         validUUID,
			body:           models.AddCartRequest{ProductID: "invalid-prod"},
			mockReturnErr:  nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Product Not Found",
			userID:         validUUID,
			body:           models.AddCartRequest{ProductID: validProdUUID},
			mockReturnErr:  service.ErrProductNotFound,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Catalog Unavailable",
			userID:         validUUID,
			body:           models.AddCartRequest{ProductID: validProdUUID},
			mockReturnErr:  service.ErrCatalogUnavailable,
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name:           "Insufficient Stock",
			userID:         validUUID,
			body:           models.AddCartRequest{ProductID: validProdUUID},
			mockReturnErr:  service.ErrInsufficientStock,
			expectedStatus: http.StatusUnprocessableEntity,
		},
		{
			name:           "Internal Error",
			userID:         validUUID,
			body:           models.AddCartRequest{ProductID: validProdUUID},
			mockReturnErr:  errors.New("some error"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &MockCartService{
				AddToCartFn: func(_ context.Context, _ string, _ *models.CartItem) error {
					return tt.mockReturnErr
				},
			}
			ctrl := NewCartController(mockSvc)
			r := setupTestRouter(ctrl)

			b, _ := json.Marshal(tt.body)
			if str, ok := tt.body.(string); ok {
				b = []byte(str)
			}
			req, _ := http.NewRequest(http.MethodPost, "/cart/"+tt.userID+"/items", bytes.NewBuffer(b))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestRemoveFromCart(t *testing.T) {
	validUUID := "550e8400-e29b-41d4-a716-446655440000"
	validProdUUID := "770e8400-e29b-41d4-a716-446655440000"

	tests := []struct {
		name           string
		userID         string
		productID      string
		mockReturnErr  error
		expectedStatus int
	}{
		{
			name:           "Valid request",
			userID:         validUUID,
			productID:      validProdUUID,
			mockReturnErr:  nil,
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "Invalid user ID",
			userID:         "invalid",
			productID:      validProdUUID,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid product ID",
			userID:         validUUID,
			productID:      "invalid",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Item Not Found",
			userID:         validUUID,
			productID:      validProdUUID,
			mockReturnErr:  repository.ErrCartItemNotFound,
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &MockCartService{
				RemoveFromCartFn: func(_ context.Context, _ string, _ string) error {
					return tt.mockReturnErr
				},
			}
			ctrl := NewCartController(mockSvc)
			r := setupTestRouter(ctrl)

			req, _ := http.NewRequest(http.MethodDelete, "/cart/"+tt.userID+"/items/"+tt.productID, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestIncrementItem(t *testing.T) {
	validUUID := "550e8400-e29b-41d4-a716-446655440000"
	validProdUUID := "770e8400-e29b-41d4-a716-446655440000"

	tests := []struct {
		name           string
		userID         string
		productID      string
		body           interface{}
		mockReturnErr  error
		expectedStatus int
	}{
		{
			name:           "Valid request",
			userID:         validUUID,
			productID:      validProdUUID,
			body:           models.QuantityRequest{Quantity: 2},
			mockReturnErr:  nil,
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "Invalid user ID",
			userID:         "invalid",
			productID:      validProdUUID,
			body:           models.QuantityRequest{Quantity: 2},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid Payload",
			userID:         validUUID,
			productID:      validProdUUID,
			body:           "not json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Item Not Found",
			userID:         validUUID,
			productID:      validProdUUID,
			body:           models.QuantityRequest{Quantity: 1},
			mockReturnErr:  repository.ErrCartItemNotFound,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Insufficient Stock",
			userID:         validUUID,
			productID:      validProdUUID,
			body:           models.QuantityRequest{Quantity: 1},
			mockReturnErr:  service.ErrInsufficientStock,
			expectedStatus: http.StatusUnprocessableEntity,
		},
		{
			name:           "Internal Error",
			userID:         validUUID,
			productID:      validProdUUID,
			body:           models.QuantityRequest{Quantity: 1},
			mockReturnErr:  errors.New("db error"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &MockCartService{
				IncrementItemFn: func(_ context.Context, _ string, _ string, _ int) error {
					return tt.mockReturnErr
				},
			}
			ctrl := NewCartController(mockSvc)
			r := setupTestRouter(ctrl)

			b, _ := json.Marshal(tt.body)
			if str, ok := tt.body.(string); ok {
				b = []byte(str)
			}
			req, _ := http.NewRequest(http.MethodPut, "/cart/"+tt.userID+"/"+tt.productID+"/increment", bytes.NewBuffer(b))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestDecrementItem(t *testing.T) {
	validUUID := "550e8400-e29b-41d4-a716-446655440000"
	validProdUUID := "770e8400-e29b-41d4-a716-446655440000"

	tests := []struct {
		name           string
		userID         string
		productID      string
		body           interface{}
		mockReturnErr  error
		expectedStatus int
	}{
		{
			name:           "Valid request",
			userID:         validUUID,
			productID:      validProdUUID,
			body:           models.QuantityRequest{Quantity: 2},
			mockReturnErr:  nil,
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "Invalid user ID",
			userID:         "invalid",
			productID:      validProdUUID,
			body:           models.QuantityRequest{Quantity: 2},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid Payload",
			userID:         validUUID,
			productID:      validProdUUID,
			body:           "not json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Item Not Found",
			userID:         validUUID,
			productID:      validProdUUID,
			body:           models.QuantityRequest{Quantity: 1},
			mockReturnErr:  repository.ErrCartItemNotFound,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Internal Error",
			userID:         validUUID,
			productID:      validProdUUID,
			body:           models.QuantityRequest{Quantity: 1},
			mockReturnErr:  errors.New("db error"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &MockCartService{
				DecrementItemFn: func(_ context.Context, _ string, _ string, _ int) error {
					return tt.mockReturnErr
				},
			}
			ctrl := NewCartController(mockSvc)
			r := setupTestRouter(ctrl)

			b, _ := json.Marshal(tt.body)
			if str, ok := tt.body.(string); ok {
				b = []byte(str)
			}
			req, _ := http.NewRequest(http.MethodPut, "/cart/"+tt.userID+"/"+tt.productID+"/decrement", bytes.NewBuffer(b))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

// Tests for the authenticated "Own Cart" variants

func TestGetOwnCart(t *testing.T) {
	validUUID := "550e8400-e29b-41d4-a716-446655440000"
	tests := []struct {
		name           string
		headerUserID   string
		mockReturnCart *models.Cart
		mockReturnErr  error
		expectedStatus int
	}{
		{
			name:           "Valid Header",
			headerUserID:   validUUID,
			mockReturnCart: models.NewCart(validUUID, nil),
			mockReturnErr:  nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Missing Header",
			headerUserID:   "",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Service error",
			headerUserID:   validUUID,
			mockReturnErr:  errors.New("internal"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &MockCartService{
				GetCartByUserIDFn: func(_ context.Context, _ string) (*models.Cart, error) {
					return tt.mockReturnCart, tt.mockReturnErr
				},
			}
			ctrl := NewCartController(mockSvc)
			r := setupTestRouter(ctrl)

			req, _ := http.NewRequest(http.MethodGet, "/cart", nil)
			if tt.headerUserID != "" {
				req.Header.Set("X-User-ID", tt.headerUserID)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestClearOwnCart(t *testing.T) {
	validUUID := "550e8400-e29b-41d4-a716-446655440000"
	tests := []struct {
		name           string
		headerUserID   string
		mockReturnErr  error
		expectedStatus int
	}{
		{
			name:           "Valid Header",
			headerUserID:   validUUID,
			mockReturnErr:  nil,
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "Missing Header",
			headerUserID:   "",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Service error",
			headerUserID:   validUUID,
			mockReturnErr:  errors.New("internal"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &MockCartService{
				ClearCartFn: func(_ context.Context, _ string) error {
					return tt.mockReturnErr
				},
			}
			ctrl := NewCartController(mockSvc)
			r := setupTestRouter(ctrl)

			req, _ := http.NewRequest(http.MethodDelete, "/cart", nil)
			if tt.headerUserID != "" {
				req.Header.Set("X-User-ID", tt.headerUserID)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestAddToOwnCart(t *testing.T) {
	validUUID := "550e8400-e29b-41d4-a716-446655440000"
	validProdUUID := "770e8400-e29b-41d4-a716-446655440000"

	tests := []struct {
		name           string
		headerUserID   string
		body           interface{}
		mockReturnErr  error
		expectedStatus int
	}{
		{
			name:           "Valid request",
			headerUserID:   validUUID,
			body:           models.AddCartRequest{ProductID: validProdUUID},
			mockReturnErr:  nil,
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "Missing Header",
			headerUserID:   "",
			body:           models.AddCartRequest{ProductID: validProdUUID},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid Payload",
			headerUserID:   validUUID,
			body:           "not json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid Product ID",
			headerUserID:   validUUID,
			body:           models.AddCartRequest{ProductID: "invalid"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Service error",
			headerUserID:   validUUID,
			body:           models.AddCartRequest{ProductID: validProdUUID},
			mockReturnErr:  service.ErrInsufficientStock,
			expectedStatus: http.StatusUnprocessableEntity,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &MockCartService{
				AddToCartFn: func(_ context.Context, _ string, _ *models.CartItem) error {
					return tt.mockReturnErr
				},
			}
			ctrl := NewCartController(mockSvc)
			r := setupTestRouter(ctrl)

			b, _ := json.Marshal(tt.body)
			if str, ok := tt.body.(string); ok {
				b = []byte(str)
			}
			req, _ := http.NewRequest(http.MethodPost, "/cart/items", bytes.NewBuffer(b))
			if tt.headerUserID != "" {
				req.Header.Set("X-User-ID", tt.headerUserID)
			}
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestRemoveFromOwnCart(t *testing.T) {
	validUUID := "550e8400-e29b-41d4-a716-446655440000"
	validProdUUID := "770e8400-e29b-41d4-a716-446655440000"

	tests := []struct {
		name           string
		headerUserID   string
		productID      string
		mockReturnErr  error
		expectedStatus int
	}{
		{
			name:           "Valid request",
			headerUserID:   validUUID,
			productID:      validProdUUID,
			mockReturnErr:  nil,
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "Missing Header",
			headerUserID:   "",
			productID:      validProdUUID,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid product ID",
			headerUserID:   validUUID,
			productID:      "invalid",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Service error",
			headerUserID:   validUUID,
			productID:      validProdUUID,
			mockReturnErr:  repository.ErrCartItemNotFound,
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &MockCartService{
				RemoveFromCartFn: func(_ context.Context, _ string, _ string) error {
					return tt.mockReturnErr
				},
			}
			ctrl := NewCartController(mockSvc)
			r := setupTestRouter(ctrl)

			req, _ := http.NewRequest(http.MethodDelete, "/cart/items/"+tt.productID, nil)
			if tt.headerUserID != "" {
				req.Header.Set("X-User-ID", tt.headerUserID)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestIncrementOwnItem(t *testing.T) {
	validUUID := "550e8400-e29b-41d4-a716-446655440000"
	validProdUUID := "770e8400-e29b-41d4-a716-446655440000"

	tests := []struct {
		name           string
		headerUserID   string
		productID      string
		body           interface{}
		mockReturnErr  error
		expectedStatus int
	}{
		{
			name:           "Valid request",
			headerUserID:   validUUID,
			productID:      validProdUUID,
			body:           models.QuantityRequest{Quantity: 2},
			mockReturnErr:  nil,
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "Missing header",
			headerUserID:   "",
			productID:      validProdUUID,
			body:           models.QuantityRequest{Quantity: 2},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid payload",
			headerUserID:   validUUID,
			productID:      validProdUUID,
			body:           "not json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Service error",
			headerUserID:   validUUID,
			productID:      validProdUUID,
			body:           models.QuantityRequest{Quantity: 1},
			mockReturnErr:  repository.ErrCartItemNotFound,
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &MockCartService{
				IncrementItemFn: func(_ context.Context, _ string, _ string, _ int) error {
					return tt.mockReturnErr
				},
			}
			ctrl := NewCartController(mockSvc)
			r := setupTestRouter(ctrl)

			b, _ := json.Marshal(tt.body)
			if str, ok := tt.body.(string); ok {
				b = []byte(str)
			}
			req, _ := http.NewRequest(http.MethodPut, "/cart/increment/"+tt.productID, bytes.NewBuffer(b))
			if tt.headerUserID != "" {
				req.Header.Set("X-User-ID", tt.headerUserID)
			}
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestDecrementOwnItem(t *testing.T) {
	validUUID := "550e8400-e29b-41d4-a716-446655440000"
	validProdUUID := "770e8400-e29b-41d4-a716-446655440000"

	tests := []struct {
		name           string
		headerUserID   string
		productID      string
		body           interface{}
		mockReturnErr  error
		expectedStatus int
	}{
		{
			name:           "Valid request",
			headerUserID:   validUUID,
			productID:      validProdUUID,
			body:           models.QuantityRequest{Quantity: 2},
			mockReturnErr:  nil,
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "Missing header",
			headerUserID:   "",
			productID:      validProdUUID,
			body:           models.QuantityRequest{Quantity: 2},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid payload",
			headerUserID:   validUUID,
			productID:      validProdUUID,
			body:           "not json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Service error",
			headerUserID:   validUUID,
			productID:      validProdUUID,
			body:           models.QuantityRequest{Quantity: 1},
			mockReturnErr:  repository.ErrCartItemNotFound,
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &MockCartService{
				DecrementItemFn: func(_ context.Context, _ string, _ string, _ int) error {
					return tt.mockReturnErr
				},
			}
			ctrl := NewCartController(mockSvc)
			r := setupTestRouter(ctrl)

			b, _ := json.Marshal(tt.body)
			if str, ok := tt.body.(string); ok {
				b = []byte(str)
			}
			req, _ := http.NewRequest(http.MethodPut, "/cart/decrement/"+tt.productID, bytes.NewBuffer(b))
			if tt.headerUserID != "" {
				req.Header.Set("X-User-ID", tt.headerUserID)
			}
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}
