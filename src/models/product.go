package models

// Product represents the data fetched from the Catalog microservice
// when resolving a price snapshot for a new CartItem.
// This struct is used only for deserialization — the cart service does not
// own or persist products; it reads them transiently to capture snapshots.
type Product struct {
	ID        string `json:"id"`
	SellerID  string `json:"sellerId"`
	Title     string `json:"title"`
	Category  string `json:"category"`
	Price     Price  `json:"price"`
	Stock     int    `json:"stock"`
	Status    string `json:"status"`
	IsBlocked bool   `json:"adminDisabled"`
}
