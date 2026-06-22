package models

import (
	"testing"
)

func TestNewCart(t *testing.T) {
	userID := "user123"

	// 1. Empty items
	cart := NewCart(userID, nil)
	if cart.UserID != userID {
		t.Errorf("expected userId %s, got %s", userID, cart.UserID)
	}
	if len(cart.Items) != 0 {
		t.Errorf("expected empty items, got %d", len(cart.Items))
	}
	if cart.Items == nil {
		t.Errorf("expected items to be initialized slice, got nil")
	}
	if cart.TotalPrice.Float64() != 0.0 {
		t.Errorf("expected total 0.0, got %f", cart.TotalPrice.Float64())
	}

	// 2. With items
	items := []CartItem{
		{ProductID: "prod1", Quantity: 2, UnitPrice: NewPrice("10.50")},
		{ProductID: "prod2", Quantity: 1, UnitPrice: NewPrice("5.00")},
	}
	cart = NewCart(userID, items)
	if len(cart.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(cart.Items))
	}
	if cart.TotalPrice.Float64() != 26.0 {
		t.Errorf("expected total 26.0, got %f", cart.TotalPrice.Float64())
	}
}
