package service

import (
	"context"
	"errors"
	"testing"

	"github.com/bazaar-grupo16/cart-microservice/src/models"
	"github.com/bazaar-grupo16/cart-microservice/src/repository"
)

type MockCartRepository struct {
	GetCartByUserIDFn       func(userID string) (*models.Cart, error)
	AddItemFn               func(userID string, item *models.CartItem) error
	IncrementItemFn         func(userID string, productID string, delta int, update repository.ProductUpdate) error
	DecrementItemFn         func(userID string, productID string, delta int, update repository.ProductUpdate) error
	UpdateProductSnapshotFn func(userID string, items []models.CartItem, updates []repository.ProductUpdate) error
	RemoveItemFn            func(userID string, productID string) error
	ClearCartFn             func(userID string) error
}

func (m *MockCartRepository) GetCartByUserID(userID string) (*models.Cart, error) {
	if m.GetCartByUserIDFn != nil {
		return m.GetCartByUserIDFn(userID)
	}
	return nil, nil
}
func (m *MockCartRepository) AddItem(userID string, item *models.CartItem) error {
	if m.AddItemFn != nil {
		return m.AddItemFn(userID, item)
	}
	return nil
}
func (m *MockCartRepository) IncrementItem(userID string, productID string, delta int, update repository.ProductUpdate) error {
	if m.IncrementItemFn != nil {
		return m.IncrementItemFn(userID, productID, delta, update)
	}
	return nil
}
func (m *MockCartRepository) DecrementItem(userID string, productID string, delta int, update repository.ProductUpdate) error {
	if m.DecrementItemFn != nil {
		return m.DecrementItemFn(userID, productID, delta, update)
	}
	return nil
}
func (m *MockCartRepository) UpdateProductSnapshot(userID string, items []models.CartItem, updates []repository.ProductUpdate) error {
	if m.UpdateProductSnapshotFn != nil {
		return m.UpdateProductSnapshotFn(userID, items, updates)
	}
	return nil
}
func (m *MockCartRepository) RemoveItem(userID string, productID string) error {
	if m.RemoveItemFn != nil {
		return m.RemoveItemFn(userID, productID)
	}
	return nil
}
func (m *MockCartRepository) ClearCart(userID string) error {
	if m.ClearCartFn != nil {
		return m.ClearCartFn(userID)
	}
	return nil
}

type MockCatalogClient struct {
	GetProductFn func(ctx context.Context, productID string) (*models.Product, error)
}

func (m *MockCatalogClient) GetProduct(ctx context.Context, productID string) (*models.Product, error) {
	if m.GetProductFn != nil {
		return m.GetProductFn(ctx, productID)
	}
	return nil, nil
}

func TestGetCartByUserID(t *testing.T) {
	validUUID := "550e8400-e29b-41d4-a716-446655440000"

	tests := []struct {
		name           string
		mockRepoCart   *models.Cart
		mockRepoErr    error
		mockCatalogErr error
		expectErr      bool
	}{
		{
			name:         "Success Empty Cart",
			mockRepoCart: models.NewCart(validUUID, nil),
		},
		{
			name:        "Repo Error",
			mockRepoErr: errors.New("db error"),
			expectErr:   true,
		},
		{
			name: "Success with items - catalog success",
			mockRepoCart: &models.Cart{
				UserID: validUUID,
				Items: []models.CartItem{
					{ProductID: "prod1", Quantity: 1},
				},
			},
		},
		{
			name: "Success with items - catalog failure (fallback)",
			mockRepoCart: &models.Cart{
				UserID: validUUID,
				Items: []models.CartItem{
					{ProductID: "prod1", Quantity: 1},
				},
			},
			mockCatalogErr: ErrCatalogUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockCartRepository{
				GetCartByUserIDFn: func(userID string) (*models.Cart, error) {
					return tt.mockRepoCart, tt.mockRepoErr
				},
				UpdateProductSnapshotFn: func(userID string, items []models.CartItem, updates []repository.ProductUpdate) error {
					return nil
				},
			}
			mockCatalog := &MockCatalogClient{
				GetProductFn: func(ctx context.Context, productID string) (*models.Product, error) {
					return &models.Product{Stock: 10, Status: "active"}, tt.mockCatalogErr
				},
			}

			svc := NewCartService(mockRepo, mockCatalog)
			_, err := svc.GetCartByUserID(context.Background(), validUUID)

			if (err != nil) != tt.expectErr {
				t.Errorf("expected err: %v, got: %v", tt.expectErr, err)
			}
		})
	}
}

func TestClearCart(t *testing.T) {
	mockRepo := &MockCartRepository{
		ClearCartFn: func(userID string) error { return nil },
	}
	svc := NewCartService(mockRepo, nil)
	err := svc.ClearCart(context.Background(), "123")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestAddToCart(t *testing.T) {
	tests := []struct {
		name           string
		itemToAdd      *models.CartItem
		mockCatalogErr error
		mockCatalogPrd *models.Product
		mockRepoCart   *models.Cart
		mockRepoErr    error
		expectErr      error
	}{
		{
			name:           "Success new item",
			itemToAdd:      &models.CartItem{ProductID: "prod1", Quantity: 1},
			mockCatalogPrd: &models.Product{Stock: 10, Status: "active"},
			mockRepoCart:   &models.Cart{Items: []models.CartItem{}},
			expectErr:      nil,
		},
		{
			name:           "Cannot add own product",
			itemToAdd:      &models.CartItem{ProductID: "prod1", Quantity: 1},
			mockCatalogPrd: &models.Product{Stock: 10, Status: "active", SellerID: "user1"},
			expectErr:      ErrCannotAddOwnProduct,
		},
		{
			name:           "Cannot add blocked product",
			itemToAdd:      &models.CartItem{ProductID: "prod1", Quantity: 1},
			mockCatalogPrd: &models.Product{Stock: 10, Status: "active", IsBlocked: true},
			expectErr:      ErrProductBlocked,
		},
		{
			name:           "Catalog err",
			itemToAdd:      &models.CartItem{ProductID: "prod1", Quantity: 1},
			mockCatalogErr: ErrProductNotFound,
			expectErr:      ErrProductNotFound,
		},
		{
			name:           "Insufficient stock - catalog",
			itemToAdd:      &models.CartItem{ProductID: "prod1", Quantity: 5},
			mockCatalogPrd: &models.Product{Stock: 2, Status: "active"},
			expectErr:      ErrInsufficientStock,
		},
		{
			name:           "Repo get cart error",
			itemToAdd:      &models.CartItem{ProductID: "prod1", Quantity: 1},
			mockCatalogPrd: &models.Product{Stock: 10, Status: "active"},
			mockRepoErr:    errors.New("db err"),
			expectErr:      errors.New("db err"), // Simplified check
		},
		{
			name:           "Insufficient stock - combined",
			itemToAdd:      &models.CartItem{ProductID: "prod1", Quantity: 5},
			mockCatalogPrd: &models.Product{Stock: 6, Status: "active"},
			mockRepoCart:   &models.Cart{Items: []models.CartItem{{ProductID: "prod1", Quantity: 2}}},
			expectErr:      ErrInsufficientStock,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockCartRepository{
				GetCartByUserIDFn: func(userID string) (*models.Cart, error) {
					return tt.mockRepoCart, tt.mockRepoErr
				},
				AddItemFn: func(userID string, item *models.CartItem) error { return nil },
			}
			mockCatalog := &MockCatalogClient{
				GetProductFn: func(ctx context.Context, productID string) (*models.Product, error) {
					return tt.mockCatalogPrd, tt.mockCatalogErr
				},
			}

			svc := NewCartService(mockRepo, mockCatalog)
			err := svc.AddToCart(context.Background(), "user1", tt.itemToAdd)

			if err != nil && tt.expectErr != nil {
				if err.Error() != tt.expectErr.Error() && !errors.Is(err, tt.expectErr) {
					t.Errorf("expected %v, got %v", tt.expectErr, err)
				}
			} else if (err != nil) != (tt.expectErr != nil) {
				t.Errorf("expected err %v, got %v", tt.expectErr, err)
			}
		})
	}
}

func TestIncrementItem(t *testing.T) {
	tests := []struct {
		name           string
		mockCatalogErr error
		mockCatalogPrd *models.Product
		mockRepoCart   *models.Cart
		mockRepoErr    error
		incErr         error
		expectErr      error
	}{
		{
			name:           "Success",
			mockCatalogPrd: &models.Product{Stock: 10},
			mockRepoCart:   &models.Cart{Items: []models.CartItem{{ProductID: "prod1", Quantity: 1}}},
			expectErr:      nil,
		},
		{
			name:           "Cannot increment own product",
			mockCatalogPrd: &models.Product{Stock: 10, SellerID: "user1"},
			expectErr:      ErrCannotAddOwnProduct,
		},
		{
			name:           "Cannot increment blocked product",
			mockCatalogPrd: &models.Product{Stock: 10, IsBlocked: true},
			expectErr:      ErrProductBlocked,
		},
		{
			name:           "Item not in cart",
			mockCatalogPrd: &models.Product{Stock: 10},
			mockRepoCart:   &models.Cart{Items: []models.CartItem{}},
			expectErr:      repository.ErrCartItemNotFound,
		},
		{
			name:           "Insufficient stock",
			mockCatalogPrd: &models.Product{Stock: 2},
			mockRepoCart:   &models.Cart{Items: []models.CartItem{{ProductID: "prod1", Quantity: 2}}},
			expectErr:      ErrInsufficientStock,
		},
		{
			name:           "Repo inc err",
			mockCatalogPrd: &models.Product{Stock: 10},
			mockRepoCart:   &models.Cart{Items: []models.CartItem{{ProductID: "prod1", Quantity: 1}}},
			incErr:         repository.ErrCartItemNotFound,
			expectErr:      repository.ErrCartItemNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockCartRepository{
				GetCartByUserIDFn: func(userID string) (*models.Cart, error) {
					return tt.mockRepoCart, tt.mockRepoErr
				},
				IncrementItemFn: func(userID string, productID string, delta int, update repository.ProductUpdate) error {
					return tt.incErr
				},
			}
			mockCatalog := &MockCatalogClient{
				GetProductFn: func(ctx context.Context, productID string) (*models.Product, error) {
					return tt.mockCatalogPrd, tt.mockCatalogErr
				},
			}

			svc := NewCartService(mockRepo, mockCatalog)
			err := svc.IncrementItem(context.Background(), "user1", "prod1", 1)

			if err != nil && tt.expectErr != nil {
				if err.Error() != tt.expectErr.Error() && !errors.Is(err, tt.expectErr) {
					t.Errorf("expected %v, got %v", tt.expectErr, err)
				}
			} else if (err != nil) != (tt.expectErr != nil) {
				t.Errorf("expected err %v, got %v", tt.expectErr, err)
			}
		})
	}
}

func TestDecrementItem(t *testing.T) {
	tests := []struct {
		name           string
		mockCatalogErr error
		mockCatalogPrd *models.Product
		decErr         error
		expectErr      error
	}{
		{
			name:           "Success",
			mockCatalogPrd: &models.Product{Stock: 10},
			expectErr:      nil,
		},
		{
			name:           "Catalog Err",
			mockCatalogErr: ErrProductNotFound,
			expectErr:      ErrProductNotFound,
		},
		{
			name:           "Repo dec err",
			mockCatalogPrd: &models.Product{Stock: 10},
			decErr:         repository.ErrCartItemNotFound,
			expectErr:      repository.ErrCartItemNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockCartRepository{
				DecrementItemFn: func(userID string, productID string, delta int, update repository.ProductUpdate) error {
					return tt.decErr
				},
			}
			mockCatalog := &MockCatalogClient{
				GetProductFn: func(ctx context.Context, productID string) (*models.Product, error) {
					return tt.mockCatalogPrd, tt.mockCatalogErr
				},
			}

			svc := NewCartService(mockRepo, mockCatalog)
			err := svc.DecrementItem(context.Background(), "user1", "prod1", 1)

			if err != nil && tt.expectErr != nil {
				if err.Error() != tt.expectErr.Error() && !errors.Is(err, tt.expectErr) {
					t.Errorf("expected %v, got %v", tt.expectErr, err)
				}
			} else if (err != nil) != (tt.expectErr != nil) {
				t.Errorf("expected err %v, got %v", tt.expectErr, err)
			}
		})
	}
}

func TestRemoveFromCart(t *testing.T) {
	mockRepo := &MockCartRepository{
		RemoveItemFn: func(userID string, productID string) error { return nil },
	}
	svc := NewCartService(mockRepo, nil)
	err := svc.RemoveFromCart(context.Background(), "user1", "prod1")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}
