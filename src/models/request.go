package models

type CreateProductRequest struct {
	SellerID    int    `json:"sellerId" binding:"required,min=1"`
	Title       string `json:"title" binding:"required"`
	Description string `json:"description" binding:"required"`
	Price       Price  `json:"price" binding:"required"`
}

// UpdateProductRequest strictly enforces the OpenAPI request schema for updating a product entirely
type UpdateProductRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description" binding:"required"`
	Price       Price  `json:"price" binding:"required,gt=0"`
}

// AddCartRequest strictly enforces the OpenAPI request schema for adding a product to a cart
type AddCartRequest struct {
	ProductID int `json:"productId" binding:"required,min=1"`
}
