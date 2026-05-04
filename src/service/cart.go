package service

import (
	"errors"

	"github.com/bazaar-grupo16/cart-microservice/src/models"
	"github.com/bazaar-grupo16/cart-microservice/src/repository"
	"go.uber.org/zap"
)

var ErrInsufficientStock = errors.New("insufficient stock")

// CartRepositoryI defines the persistence contract for the cart service.
// Using an interface here makes the service layer testable without a real MongoDB
// instance when only unit-level tests are needed.
type CartRepositoryI interface {
	GetCartByUserID(userID string) (*models.Cart, error)
	AddItem(userID string, item *models.CartItem) error
	IncrementItem(userID string, productID string, delta int, update repository.ProductUpdate) error
	DecrementItem(userID string, productID string, delta int, update repository.ProductUpdate) error
	UpdateProductSnapshot(userID string, items []models.CartItem, updates []repository.ProductUpdate) error
	RemoveItem(userID string, productID string) error
	ClearCart(userID string) error
}

// CartService provides business logic for shopping cart operations.
type CartService struct {
	CartRepository CartRepositoryI
	CatalogClient  CatalogClientI // interface — allows mocking in tests
}

// NewCartService creates a new CartService instance.
func NewCartService(cartRepo CartRepositoryI, catalogClient CatalogClientI) *CartService {
	return &CartService{
		CartRepository: cartRepo,
		CatalogClient:  catalogClient,
	}
}

// GetCartByUserID retrieves the cart for a given user and synchronizes the
// stock and status of each item with the Catalog service.
// If the Catalog is unavailable, the cart is returned with the last cached values
// (no error is propagated — the failure is logged and silently skipped).
func (s *CartService) GetCartByUserID(userID string) (*models.Cart, error) {
	cart, err := s.CartRepository.GetCartByUserID(userID)
	if err != nil {
		return nil, err
	}

	// Sync stock/status for each item. Best-effort: skip on catalog failure.
	if len(cart.Items) > 0 {
		var updates []repository.ProductUpdate
		for _, item := range cart.Items {
			product, err := s.CatalogClient.GetProduct(item.ProductID)
			if err != nil {
				zap.S().Warnw("Catalog unavailable during stock sync; using cached values",
					"userID", userID, "productID", item.ProductID, "error", err)
				// Skip this item — keep cached value.
				continue
			}
			updates = append(updates, repository.ProductUpdate{
				ProductID: item.ProductID,
				Stock:     product.Stock,
				Status:    product.Status,
				UnitPrice: product.Price,
			})
		}
		if len(updates) > 0 {
			if err := s.CartRepository.UpdateProductSnapshot(userID, cart.Items, updates); err != nil {
				zap.S().Errorw("Failed to persist stock/status update", "userID", userID, "error", err)
			} else {
				// Re-fetch to return the freshly updated cart.
				cart, err = s.CartRepository.GetCartByUserID(userID)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	return cart, nil
}

// ClearCart removes all items from the user's cart.
// Idempotent — clearing an empty or non-existent cart is not an error.
func (s *CartService) ClearCart(userID string) error {
	return s.CartRepository.ClearCart(userID)
}

// AddToCart adds a product to the user's cart (or increments its quantity).
// It fetches the current price, title, stock and status from the Catalog service
// and stores them in the CartItem.
//
// Returns:
//   - ErrProductNotFound if the catalog returns 404.
//   - ErrCatalogUnavailable if the catalog is unreachable or the circuit is open.
func (s *CartService) AddToCart(userID string, item *models.CartItem) error {
	product, err := s.CatalogClient.GetProduct(item.ProductID)
	if err != nil {
		return err
	}

	qty := item.Quantity
	if qty < 1 {
		qty = 1
	}

	if product.Stock < qty {
		return ErrInsufficientStock
	}

	cart, err := s.CartRepository.GetCartByUserID(userID)
	if err != nil {
		return err
	}

	currentQty := 0
	for _, ci := range cart.Items {
		if ci.ProductID == item.ProductID {
			currentQty = ci.Quantity
			break
		}
	}

	if currentQty+qty > product.Stock {
		return ErrInsufficientStock
	}

	// Persist snapshots of title and price, plus live stock/status.
	item.Title = product.Title
	item.SellerID = product.SellerID
	item.UnitPrice = product.Price
	item.Stock = product.Stock
	item.Status = product.Status
	item.Quantity = qty

	return s.CartRepository.AddItem(userID, item)
}

// IncrementItem increases the quantity of an existing cart item by qty.
// Returns ErrCartItemNotFound (from the repository) if the product is not in the cart.
func (s *CartService) IncrementItem(userID string, productID string, qty int) error {
	product, err := s.CatalogClient.GetProduct(productID)
	if err != nil {
		return err
	}

	cart, err := s.CartRepository.GetCartByUserID(userID)
	if err != nil {
		return err
	}

	currentQty := 0
	found := false
	for _, ci := range cart.Items {
		if ci.ProductID == productID {
			currentQty = ci.Quantity
			found = true
			break
		}
	}

	if !found {
		return repository.ErrCartItemNotFound
	}

	if currentQty+qty > product.Stock {
		return ErrInsufficientStock
	}

	update := repository.ProductUpdate{
		ProductID: productID,
		Stock:     product.Stock,
		Status:    product.Status,
		UnitPrice: product.Price,
	}

	if err := s.CartRepository.IncrementItem(userID, productID, qty, update); err != nil {
		if errors.Is(err, repository.ErrCartItemNotFound) {
			return repository.ErrCartItemNotFound
		}
		return err
	}
	return nil
}

// DecrementItem decreases the quantity of an existing cart item by qty.
// If the resulting quantity would be <= 0, the item is removed from the cart.
// Returns ErrCartItemNotFound (from the repository) if the product is not in the cart.
func (s *CartService) DecrementItem(userID string, productID string, qty int) error {
	product, err := s.CatalogClient.GetProduct(productID)
	if err != nil {
		return err
	}

	update := repository.ProductUpdate{
		ProductID: productID,
		Stock:     product.Stock,
		Status:    product.Status,
		UnitPrice: product.Price,
	}

	if err := s.CartRepository.DecrementItem(userID, productID, qty, update); err != nil {
		if errors.Is(err, repository.ErrCartItemNotFound) {
			return repository.ErrCartItemNotFound
		}
		return err
	}
	return nil
}

// RemoveFromCart removes the product entry from the user's cart.
// Returns an error (propagated from the repository) if the product is not in the cart
// or does not belong to the given user.
func (s *CartService) RemoveFromCart(userID string, productID string) error {
	return s.CartRepository.RemoveItem(userID, productID)
}
