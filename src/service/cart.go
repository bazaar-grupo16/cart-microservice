package service

import (
	"github.com/bazaar-grupo16/cart-microservice/src/models"
)

// CartRepositoryI defines the persistence contract for the cart service.
// Using an interface here makes the service layer testable without a real MongoDB
// instance when only unit-level tests are needed.
type CartRepositoryI interface {
	GetCartByUserID(userID int) (*models.Cart, error)
	AddItem(userID int, item *models.CartItem) error
	RemoveItem(userID int, productID int) error
	ClearCart(userID int) error
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

// GetCartByUserID retrieves the cart for a given user.
// An unknown userId returns an empty cart (no error), as per the OpenAPI spec.
func (s *CartService) GetCartByUserID(userID int) (*models.Cart, error) {
	return s.CartRepository.GetCartByUserID(userID)
}

// ClearCart removes all items from the user's cart.
// Idempotent — clearing an empty or non-existent cart is not an error.
func (s *CartService) ClearCart(userID int) error {
	return s.CartRepository.ClearCart(userID)
}

// AddToCart adds a product to the user's cart (or increments its quantity).
// It fetches the current price and title from the Catalog service and persists them
// as an immutable snapshot on first add; subsequent adds simply increment quantity.
//
// Returns:
//   - ErrProductNotFound if the catalog returns 404.
//   - ErrCatalogUnavailable if the catalog is unreachable or the circuit is open.
func (s *CartService) AddToCart(userID int, item *models.CartItem) error {
	product, err := s.CatalogClient.GetProduct(item.ProductID)
	if err != nil {
		return err
	}

	// Persist immutable snapshots of the product's title and price at this moment.
	item.Title = product.Title
	item.UnitPrice = product.Price

	return s.CartRepository.AddItem(userID, item)
}

// RemoveFromCart removes the product entry from the user's cart.
// Returns an error (propagated from the repository) if the product is not in the cart
// or does not belong to the given user.
func (s *CartService) RemoveFromCart(userID int, productID int) error {
	return s.CartRepository.RemoveItem(userID, productID)
}
