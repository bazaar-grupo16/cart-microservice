package service

import (
	"errors"

	"github.com/ValentinMarturet/is2-rojas/api/models"
	"github.com/ValentinMarturet/is2-rojas/api/repository"
)

// CartService provides methods to manage shopping carts
type CartService struct {
	CartRepository    *repository.CartRepository
	ProductRepository *repository.ProductRepository
}

// NewCartService creates a new instance of CartService
func NewCartService(cartRepo *repository.CartRepository, productRepo *repository.ProductRepository) *CartService {
	return &CartService{
		CartRepository:    cartRepo,
		ProductRepository: productRepo,
	}
}

// GetCartByUserID retrieves the cart for a specific user
func (s *CartService) GetCartByUserID(userID int) (*models.Cart, error) {
	items, err := s.CartRepository.GetCartItemsByUserID(userID)
	if err != nil {
		return nil, err
	}

	cart := models.NewCart(userID, items)

	return cart, nil
}

// ClearCart removes all items from the user's cart
func (s *CartService) ClearCart(userID int) error {
	return s.CartRepository.DeleteAllCartItemsWithUserID(userID)
}

func (s *CartService) AddToCart(userID int, item *models.CartItem) error {
	product, err := s.ProductRepository.GetProductByID(item.ProductID)
	if err != nil {
		return err
	}

	item.UnitPrice = product.Price
	item.Title = product.Title

	return s.CartRepository.CreateCartItem(userID, item)
}

// RemoveFromCart removes a specific item from the user's cart
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
