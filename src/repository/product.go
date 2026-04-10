package repository

import (
	"database/sql"
	"errors"

	"github.com/ValentinMarturet/is2-rojas/api/models"
)

// DBTX is an interface that abstracts both *sql.DB and *sql.Tx, allowing us to use the same methods for querying and executing SQL statements in both contexts.
// This is useful for transaction management, where we can pass a *sql.Tx to the repository methods without changing their signatures.
type DBTX interface {
	QueryRow(query string, args ...interface{}) *sql.Row
	Query(query string, args ...interface{}) (*sql.Rows, error)
	Exec(query string, args ...interface{}) (sql.Result, error)
}

// ProductRepository provides methods to interact with the products table in the database
type ProductRepository struct {
	DB DBTX // acepta tanto *sql.DB como *sql.Tx
}

// NewProductRepository creates a new instance of ProductRepository with the given DB connection
func NewProductRepository(db DBTX) *ProductRepository {
	return &ProductRepository{DB: db}
}

// CreateProduct inserts a new product into the database and returns the created product with its ID and timestamps
func (r *ProductRepository) CreateProduct(p *models.Product) error {
	return r.DB.QueryRow(
		"INSERT INTO products (seller_id, title, description, price) VALUES ($1, $2, $3, $4) RETURNING id, created_at, updated_at",
		p.SellerID, p.Title, p.Description, p.Price,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
}

// DeleteProduct removes a product from the database based on its ID and returns an error if the product was not found
func (r *ProductRepository) DeleteProduct(id int) error {
	var deletedID int
	err := r.DB.QueryRow(
		"DELETE FROM products WHERE id = $1 RETURNING id",
		id,
	).Scan(&deletedID)
	if err != nil {
		return err
	}
	if deletedID == 0 {
		return errors.New("product not found")
	}
	return nil
}

// GetAllProducts retrieves all products from the database and returns them as a slice of Product pointers
func (r *ProductRepository) GetAllProducts() ([]*models.Product, error) {
	rows, err := r.DB.Query("SELECT id, seller_id, title, description, price, created_at, updated_at FROM products ORDER BY id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := []*models.Product{}
	for rows.Next() {
		var p models.Product
		if err := rows.Scan(&p.ID, &p.SellerID, &p.Title, &p.Description, &p.Price, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		products = append(products, &p)
	}
	return products, nil
}

// GetProductByID retrieves a product from the database based on its ID and returns it as a Product pointer
func (r *ProductRepository) GetProductByID(id int) (*models.Product, error) {
	var p models.Product
	err := r.DB.QueryRow(
		"SELECT id, seller_id, title, description, price, created_at, updated_at FROM products WHERE id = $1",
		id,
	).Scan(&p.ID, &p.SellerID, &p.Title, &p.Description, &p.Price, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// UpdateProduct updates an existing product in the database based on its ID and returns an error if the product was not found
func (r *ProductRepository) UpdateProduct(p *models.Product) error {
	res, err := r.DB.Exec(
		"UPDATE products SET title = $1, description = $2, price = $3, updated_at = NOW() WHERE id = $4 RETURNING id",
		p.Title, p.Description, p.Price, p.ID,
	)
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("product not found")
	}
	// get the updated timestamps
	return r.DB.QueryRow(
		"SELECT created_at, updated_at FROM products WHERE id = $1",
		p.ID,
	).Scan(&p.CreatedAt, &p.UpdatedAt)
}
