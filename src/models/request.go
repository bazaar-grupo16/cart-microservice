package models

// AddCartRequest is the request body for POST /cart/{userId}/items.
// productId is a UUID string.
// quantity is optional; when omitted it defaults to 1.
type AddCartRequest struct {
	ProductID string `json:"productId" binding:"required"`
	Quantity  *int   `json:"quantity"  binding:"omitempty,min=1"`
}

// QuantityRequest is the request body for PUT /cart/{userId}/{productId}/increment
// and PUT /cart/{userId}/{productId}/decrement.
// Quantity must be a positive integer.
type QuantityRequest struct {
	Quantity int `json:"quantity" binding:"required,min=1"`
}
