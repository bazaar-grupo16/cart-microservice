package models

// Cart represents a shopping cart for a user, containing multiple items and the total price.
type Cart struct {
	UserID     int        `json:"userId"`
	Items      []CartItem `json:"items"`
	TotalPrice Price      `json:"totalPrice"`
}

// CartItem represents an individual item in the shopping cart, including product details and price. Note: The price here is a snapshot of the product price at the time it was added to the cart, not a reference to the current product price.
type CartItem struct {
	ID        int    `json:"id"`
	ProductID int    `json:"productId"`
	Title     string `json:"title"`
	UnitPrice Price  `json:"unitPrice"`
	AddedAt   string `json:"addedAt"`
}

// NewCart creates a new Cart instance for a given user with the provided items and calculates the total price.
func NewCart(userID int, items []CartItem) *Cart {
	total := Price("0.00")
	for _, item := range items {
		total = total.Add(item.UnitPrice)
	}

	Items := []CartItem{}
	Items = append(Items, items...)

	return &Cart{
		UserID:     userID,
		Items:      Items,
		TotalPrice: total,
	}
}
