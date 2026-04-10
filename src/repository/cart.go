package repository

import "github.com/ValentinMarturet/is2-rojas/src/models"

// CartRepository provides methods to interact with the cart_items table in the database
type CartRepository struct {
	DB DBTX
}

// NewCartRepository creates a new instance of CartRepository with the given DB connection
func NewCartRepository(db DBTX) *CartRepository {
	return &CartRepository{DB: db}
}

// GetCartItemsByUserID retrieves all cart items for a specific user ID
func (r *CartRepository) GetCartItemsByUserID(userID int) ([]models.CartItem, error) {
	rows, err := r.DB.Query("SELECT id, product_id, title, unit_price, added_at FROM cart_items WHERE user_id = $1 ORDER BY added_at DESC, id DESC", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.CartItem
	for rows.Next() {
		var item models.CartItem
		if err := rows.Scan(&item.ID, &item.ProductID, &item.Title, &item.UnitPrice, &item.AddedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, nil
}

// DeleteAllCartItemsWithUserID removes all cart items associated with a specific user ID
func (r *CartRepository) DeleteAllCartItemsWithUserID(userID int) error {
	_, err := r.DB.Exec("DELETE FROM cart_items WHERE user_id = $1", userID)
	return err
}

// CreateCartItem adds a new item to the cart for a specific user ID
func (r *CartRepository) CreateCartItem(userID int, item *models.CartItem) error {
	err := r.DB.QueryRow("INSERT INTO cart_items (user_id, product_id, title, unit_price) VALUES ($1, $2, $3, $4) RETURNING id, added_at", userID, item.ProductID, item.Title, item.UnitPrice).Scan(&item.ID, &item.AddedAt)
	return err
}

// DeleteCartItem removes a specific item from the cart based on its ID
func (r *CartRepository) DeleteCartItem(cartItemID int) error {
	_, err := r.DB.Exec("DELETE FROM cart_items WHERE id = $1", cartItemID)
	return err
}
