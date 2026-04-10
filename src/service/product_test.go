package service

import (
	"testing"

	"github.com/ValentinMarturet/is2-rojas/api/models"
)

func TestCreateProduct_ReturnsErrorWhenTitleIsMissing(t *testing.T) {
	svc := NewProductService(nil)
	product := &models.Product{
		SellerID:    1,
		Description: "A simple description",
		Price:       models.Price("10.00"),
	}

	err := svc.CreateProduct(product)

	if err == nil {
		t.Fatalf("expected an error when title is missing")
	}
	if err.Error() != "title is required" {
		t.Fatalf("expected 'title is required', got %q", err.Error())
	}
}

func TestCreateProduct_ReturnsErrorWhenDescriptionIsMissing(t *testing.T) {
	svc := NewProductService(nil)
	product := &models.Product{
		SellerID: 1,
		Title:    "Phone",
		Price:    models.Price("10.00"),
	}

	err := svc.CreateProduct(product)

	if err == nil {
		t.Fatalf("expected an error when description is missing")
	}
	if err.Error() != "description is required" {
		t.Fatalf("expected 'description is required', got %q", err.Error())
	}
}

func TestCreateProduct_ReturnsErrorWhenPriceIsInvalid(t *testing.T) {
	svc := NewProductService(nil)
	product := &models.Product{
		SellerID:    1,
		Title:       "Phone",
		Description: "A simple description",
		Price:       models.Price("0.00"),
	}

	err := svc.CreateProduct(product)

	if err == nil {
		t.Fatalf("expected an error when price is invalid")
	}
	if err.Error() != "price is required and must be greater than 0" {
		t.Fatalf("expected price validation error, got %q", err.Error())
	}
}

func TestUpdateProduct_ReturnsErrorWhenNoFieldsAreProvided(t *testing.T) {
	svc := NewProductService(nil)
	product := &models.Product{ID: 1}

	err := svc.UpdateProduct(product)

	if err == nil {
		t.Fatalf("expected an error when no update fields are provided")
	}
	if err.Error() != "at least one field (title, description, price) must be provided for update" {
		t.Fatalf("expected update validation error, got %q", err.Error())
	}
}
