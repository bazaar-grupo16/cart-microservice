package repository

import (
	"context"
	"errors"
	"time"

	"github.com/bazaar-grupo16/cart-microservice/src/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// ErrCartItemNotFound is returned when the requested product is not in the user's cart.
var ErrCartItemNotFound = errors.New("cart item not found for this user")

// CartRepository encapsulates all MongoDB operations for cart documents.
// Each document represents one user's cart, with items embedded as an array.
type CartRepository struct {
	col *mongo.Collection
}

// NewCartRepository creates a CartRepository backed by the given collection.
func NewCartRepository(col *mongo.Collection) *CartRepository {
	return &CartRepository{col: col}
}

// GetCartByUserID retrieves the cart document for the given user.
// If no document exists yet, an empty cart is returned (no error).
func (r *CartRepository) GetCartByUserID(userID int) (*models.Cart, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var cart models.Cart
	err := r.col.FindOne(ctx, bson.D{{Key: "userId", Value: userID}}).Decode(&cart)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return models.NewCart(userID, nil), nil
	}
	if err != nil {
		return nil, err
	}
	return &cart, nil
}

// AddItem adds a product to the user's cart.
// If the product already exists in the cart, its quantity is incremented by 1.
// If the cart document does not exist yet, it is created (upsert).
// The title and unitPrice are snapshot values passed in from the service layer.
// totalPrice is recalculated and persisted atomically on every write.
func (r *CartRepository) AddItem(userID int, item *models.CartItem) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	item.AddedAt = time.Now().UTC().Format(time.RFC3339)

	// Attempt to increment quantity when the product is already in the cart.
	incrResult, err := r.col.UpdateOne(
		ctx,
		bson.D{
			{Key: "userId", Value: userID},
			{Key: "items.productId", Value: item.ProductID},
		},
		bson.D{
			{Key: "$inc", Value: bson.D{{Key: "items.$.quantity", Value: 1}}},
		},
	)
	if err != nil {
		return err
	}

	if incrResult.ModifiedCount == 0 {
		// Product not found in the cart — push a new item entry with quantity = 1.
		item.Quantity = 1
		_, err = r.col.UpdateOne(
			ctx,
			bson.D{{Key: "userId", Value: userID}},
			bson.D{{Key: "$push", Value: bson.D{{Key: "items", Value: item}}}},
			options.UpdateOne().SetUpsert(true),
		)
		if err != nil {
			return err
		}
	}

	// Recalculate and persist totalPrice.
	return r.recalcTotalPrice(ctx, userID)
}

// RemoveItem removes the cart entry for the given productId from the user's cart.
// Returns ErrCartItemNotFound if no such product exists in that user's cart.
// totalPrice is recalculated and persisted after the removal.
func (r *CartRepository) RemoveItem(userID int, productID int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := r.col.UpdateOne(
		ctx,
		bson.D{{Key: "userId", Value: userID}},
		bson.D{{Key: "$pull", Value: bson.D{
			{Key: "items", Value: bson.D{{Key: "productId", Value: productID}}},
		}}},
	)
	if err != nil {
		return err
	}
	if result.ModifiedCount == 0 {
		return ErrCartItemNotFound
	}

	return r.recalcTotalPrice(ctx, userID)
}

// ClearCart removes all items from the user's cart and resets totalPrice to 0.00.
// This operation is idempotent — clearing an already-empty cart succeeds silently.
func (r *CartRepository) ClearCart(userID int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := r.col.UpdateOne(
		ctx,
		bson.D{{Key: "userId", Value: userID}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "items", Value: bson.A{}},
			{Key: "totalPrice", Value: "0.00"},
		}}},
		options.UpdateOne().SetUpsert(true),
	)
	return err
}

// recalcTotalPrice reads the current items from the user's cart and writes the
// correct totalPrice back to the document in a single targeted $set.
func (r *CartRepository) recalcTotalPrice(ctx context.Context, userID int) error {
	var cart models.Cart
	err := r.col.FindOne(ctx, bson.D{{Key: "userId", Value: userID}}).Decode(&cart)
	if err != nil {
		return err
	}

	total := models.Price("0.00")
	for _, item := range cart.Items {
		for range item.Quantity {
			total = total.Add(item.UnitPrice)
		}
	}

	// Marshal to the canonical "X.YY" string before persisting.
	totalBytes, err := total.MarshalJSON()
	if err != nil {
		return err
	}

	_, err = r.col.UpdateOne(
		ctx,
		bson.D{{Key: "userId", Value: userID}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "totalPrice", Value: string(totalBytes)}}}},
	)
	return err
}
