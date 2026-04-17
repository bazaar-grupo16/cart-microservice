package models

// Cart represents a shopping cart document stored in MongoDB.
// One document per user — items are embedded in the "items" array.
type Cart struct {
	UserID     int        `bson:"userId"     json:"userId"`
	Items      []CartItem `bson:"items"      json:"items"`
	TotalPrice Price      `bson:"totalPrice" json:"totalPrice"`
}

// CartItem represents a single product entry in the cart.
// There is at most one CartItem per productId per user.
// Quantity tracks how many units of that product are in the cart.
// UnitPrice is an immutable snapshot captured when the item was first added.
type CartItem struct {
	ProductID      string `bson:"productId" json:"productId"`
	Title          string `bson:"title"     json:"title"`
	UnitPrice      Price  `bson:"unitPrice" json:"unitPrice"`
	Quantity       int    `bson:"quantity"  json:"quantity"`
	Stock          int    `bson:"stock"     json:"stock"`
	Status         string `bson:"status"    json:"status"`
	AddedAt        string `bson:"addedAt"   json:"addedAt"`
}

// NewCart creates a Cart instance for a given user with the provided items and
// calculates the total price as the sum of (unitPrice × quantity) for every item.
func NewCart(userID int, items []CartItem) *Cart {
	total := NewPrice("0.00")
	for _, item := range items {
		total = total.Add(item.UnitPrice.Multiply(item.Quantity))
	}

	// Always return an initialised slice so the JSON output is [] not null.
	result := make([]CartItem, len(items))
	copy(result, items)

	return &Cart{
		UserID:     userID,
		Items:      result,
		TotalPrice: total,
	}
}
