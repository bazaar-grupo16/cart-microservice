package models

// Product represents an item that can be bought or sold in the marketplace
type Product struct {
	ID          int    `json:"id"`
	SellerID    int    `json:"sellerId"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Price       Price  `json:"price"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}
