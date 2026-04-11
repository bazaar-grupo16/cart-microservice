package service

import (
	"errors"

	"github.com/bazaar-grupo16/cart-microservice/src/models"
	"github.com/bazaar-grupo16/cart-microservice/src/repository"
)

// CartService provides business logic for shopping cart operations.
type CartService struct {
	CartRepository *repository.CartRepository
	CatalogClient  CatalogClientI // interface — allows mocking in tests
}

// NewCartService creates a new CartService instance.
func NewCartService(cartRepo *repository.CartRepository, catalogClient CatalogClientI) *CartService {
	return &CartService{
		CartRepository: cartRepo,
		CatalogClient:  catalogClient,
	}
}

// GetCartByUserID retrieves the cart for a given user.
// An unknown userId returns an empty cart (no error), as per the OpenAPI spec.
func (s *CartService) GetCartByUserID(userID int) (*models.Cart, error) {
	items, err := s.CartRepository.GetCartItemsByUserID(userID)
	if err != nil {
		return nil, err
	}
	return models.NewCart(userID, items), nil
}

// ClearCart removes all items from the user's cart.
// Idempotent — clearing an empty cart is not an error.
func (s *CartService) ClearCart(userID int) error {
	return s.CartRepository.DeleteAllCartItemsWithUserID(userID)
}

// AddToCart adds a new item to the user's cart.
// It fetches the current price and title from the Catalog service and persists them
// as an immutable snapshot so future price changes in the catalog don't affect the cart.
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

	return s.CartRepository.CreateCartItem(userID, item)
}

// RemoveFromCart removes a specific item from the user's cart.
// Returns an error if the item does not exist or does not belong to the given userId.
func (s *CartService) RemoveFromCart(userID int, cartItemID int) error {
	items, err := s.CartRepository.GetCartItemsByUserID(userID)
	if err != nil {
		return err
	}

	var found bool
	for _, item := range items {
		if item.ID == cartItemID {
			found = true
			break
		}
	}
	if !found {
		return errors.New("cart item not found")
	}

	return s.CartRepository.DeleteCartItem(cartItemID)
}
