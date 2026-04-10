package service

import (
	"errors"

	"github.com/ValentinMarturet/is2-rojas/api/models"
	"github.com/ValentinMarturet/is2-rojas/api/repository"
)

// ProductService provides business logic for product operations
type ProductService struct {
	ProductRepository *repository.ProductRepository
}

// NewProductService creates a new instance of ProductService
func NewProductService(productRepo *repository.ProductRepository) *ProductService {
	return &ProductService{
		ProductRepository: productRepo,
	}
}

// CreateProduct validates the product data and creates a new product
func (s *ProductService) CreateProduct(p *models.Product) error {
	if p.SellerID <= 0 {
		return errors.New("sellerId is required and must be greater than 0")
	}
	if p.Title == "" {
		return errors.New("title is required")
	}
	if p.Description == "" {
		return errors.New("description is required")
	}
	if !p.Price.IsPositive() {
		return errors.New("price is required and must be greater than 0")
	}
	if !p.Price.IsAtLeast("0.01") {
		return errors.New("price must be at least 0.01")
	}
	return s.ProductRepository.CreateProduct(p)
}

// DeleteProduct deletes a product by its ID
func (s *ProductService) DeleteProduct(id int) error {
	return s.ProductRepository.DeleteProduct(id)
}

// GetAllProducts retrieves all products
func (s *ProductService) GetAllProducts() ([]*models.Product, error) {
	return s.ProductRepository.GetAllProducts()
}

// GetProductByID retrieves a product by its ID
func (s *ProductService) GetProductByID(id int) (*models.Product, error) {
	return s.ProductRepository.GetProductByID(id)
}

// UpdateProduct updates an existing product with new data
func (s *ProductService) UpdateProduct(p *models.Product) error {
	if p.Title == "" && p.Description == "" && !p.Price.IsPositive() {
		return errors.New("at least one field (title, description, price) must be provided for update")
	}
	prd, err := s.ProductRepository.GetProductByID(p.ID)
	if err != nil {
		return errors.New("product not found")
	}
	if p.Title == "" {
		p.Title = prd.Title
	}
	if p.Description == "" {
		p.Description = prd.Description
	}
	if !p.Price.IsPositive() {
		return errors.New("price must be greater than 0")
	}
	p.SellerID = prd.SellerID // Ensure SellerID is not changed
	return s.ProductRepository.UpdateProduct(p)
}
