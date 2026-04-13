package models

// AddCartRequest is the request body for POST /cart/{userId}/items.
// Only the productId is required — the title and unitPrice are fetched from the
// Catalog service at request time and stored as an immutable snapshot in the CartItem.
type AddCartRequest struct {
	ProductID int `json:"productId" binding:"required,min=1"`
}

// QuantityRequest is the request body for PUT /cart/{userId}/{productId}/increment
// and PUT /cart/{userId}/{productId}/decrement.
// Quantity must be a positive integer.
type QuantityRequest struct {
	Quantity int `json:"quantity" binding:"required,min=1"`
}
