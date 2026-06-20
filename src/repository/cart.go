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

// ErrDatabaseUnavailable is returned when the repository has no MongoDB collection
// available (e.g., during startup or while a reconnect attempt is in progress).
var ErrDatabaseUnavailable = errors.New("database unavailable")

// ProductUpdate carries catalog-fresh stock and status for a single product,
// used by UpdateStockStatus to patch cart items atomically.
type ProductUpdate struct {
	ProductID string
	Stock     int
	Status    string
	UnitPrice models.Price
}

// CartRepository encapsulates all MongoDB operations for cart documents.
// Each document represents one user's cart, with items embedded as an array.
type CartRepository struct {
	colProvider func() *mongo.Collection
}

// NewCartRepository creates a CartRepository backed by the given collection provider.
// The provider may return different *mongo.Collection instances over time (e.g. after reconnect).
func NewCartRepository(colProvider func() *mongo.Collection) *CartRepository {
	return &CartRepository{colProvider: colProvider}
}

func (r *CartRepository) col() (*mongo.Collection, error) {
	if r == nil || r.colProvider == nil {
		return nil, ErrDatabaseUnavailable
	}
	col := r.colProvider()
	if col == nil {
		return nil, ErrDatabaseUnavailable
	}
	return col, nil
}

// GetCartByUserID retrieves the cart document for the given user.
// If no document exists yet, an empty cart is returned (no error).
func (r *CartRepository) GetCartByUserID(userID string) (*models.Cart, error) {
	col, err := r.col()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var cart models.Cart
	err = col.FindOne(ctx, bson.D{{Key: "userId", Value: userID}}).Decode(&cart)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return models.NewCart(userID, nil), nil
	}
	if err != nil {
		return nil, err
	}
	return &cart, nil
}

// AddItem adds a product to the user's cart.
// If the product already exists in the cart, its quantity is incremented by item.Quantity.
// If the cart document does not exist yet, it is created (upsert).
// The title, unitPrice, stock and status are snapshot values passed in from the service layer.
// totalPrice is recalculated and persisted atomically on every write.
func (r *CartRepository) AddItem(userID string, item *models.CartItem) error {
	col, err := r.col()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	item.AddedAt = time.Now().UTC().Format(time.RFC3339)

	// Attempt to increment quantity when the product is already in the cart,
	// and update stock/status to latest values from catalog.
	incrResult, err := col.UpdateOne(
		ctx,
		bson.D{
			{Key: "userId", Value: userID},
			{Key: "items.productId", Value: item.ProductID},
		},
		bson.D{
			{Key: "$inc", Value: bson.D{{Key: "items.$.quantity", Value: item.Quantity}}},
			{Key: "$set", Value: bson.D{
				{Key: "items.$.stock", Value: item.Stock},
				{Key: "items.$.status", Value: item.Status},
				{Key: "items.$.unitPrice", Value: item.UnitPrice},
			}},
		},
	)
	if err != nil {
		return err
	}

	if incrResult.ModifiedCount == 0 {
		// Product not found in the cart — push a new item entry.
		_, err = col.UpdateOne(
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

// IncrementItem increments the quantity of an existing cart item by delta.
// Returns ErrCartItemNotFound if the product is not in the user's cart.
func (r *CartRepository) IncrementItem(userID string, productID string, delta int, update ProductUpdate) error {
	col, err := r.col()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := col.UpdateOne(
		ctx,
		bson.D{
			{Key: "userId", Value: userID},
			{Key: "items.productId", Value: productID},
		},
		bson.D{
			{Key: "$inc", Value: bson.D{{Key: "items.$.quantity", Value: delta}}},
			{Key: "$set", Value: bson.D{
				{Key: "items.$.stock", Value: update.Stock},
				{Key: "items.$.status", Value: update.Status},
				{Key: "items.$.unitPrice", Value: update.UnitPrice},
			}},
		},
	)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return ErrCartItemNotFound
	}

	return r.recalcTotalPrice(ctx, userID)
}

// DecrementItem decrements the quantity of an existing cart item by delta.
// The quantity is prevented from falling below 1.
// Returns ErrCartItemNotFound if the product is not in the user's cart.
func (r *CartRepository) DecrementItem(userID string, productID string, delta int, update ProductUpdate) error {
	col, err := r.col()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Fetch the full Cart document to read the current item quantity.
	var cart models.Cart
	if err := col.FindOne(ctx, bson.D{
		{Key: "userId", Value: userID},
		{Key: "items.productId", Value: productID},
	}).Decode(&cart); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return ErrCartItemNotFound
		}
		return err
	}

	// Locate the target item's current quantity.
	var currentQty int
	found := false
	for _, item := range cart.Items {
		if item.ProductID == productID {
			currentQty = item.Quantity
			found = true
			break
		}
	}
	if !found {
		return ErrCartItemNotFound
	}

	newQuantity := currentQty - delta

	if newQuantity < 1 {
		newQuantity = 1
	}

	// Set the new quantity and sync metadata.
	result, err := col.UpdateOne(
		ctx,
		bson.D{
			{Key: "userId", Value: userID},
			{Key: "items.productId", Value: productID},
		},
		bson.D{
			{Key: "$set", Value: bson.D{
				{Key: "items.$.quantity", Value: newQuantity},
				{Key: "items.$.stock", Value: update.Stock},
				{Key: "items.$.status", Value: update.Status},
				{Key: "items.$.unitPrice", Value: update.UnitPrice},
			}},
		},
	)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return ErrCartItemNotFound
	}

	return r.recalcTotalPrice(ctx, userID)
}

// UpdateProductSnapshot patches the stock and status fields for each listed product
// in the user's cart using positional arrayFilters for atomic multi-field updates.
// Also updates the price if the price snapshot is stale.
// Precondition:
//   - items and updates must have the same length.
//   - updates must be ordered in the same order as items.
//   - updates[i].ProductID must match items[i].ProductID.
func (r *CartRepository) UpdateProductSnapshot(userID string, items []models.CartItem, updates []ProductUpdate) error {
	if len(updates) == 0 {
		return nil
	}

	col, err := r.col()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for i, u := range updates {
		// TODO: Check if the price is stale and update it if it is.
		switch items[i].UnitPrice.IsPriceStale() {
		case true:
			_, err := col.UpdateOne(
				ctx,
				bson.D{
					{Key: "userId", Value: userID},
					{Key: "items.productId", Value: u.ProductID},
				},
				bson.D{
					{Key: "$set", Value: bson.D{
						{Key: "items.$.stock", Value: u.Stock},
						{Key: "items.$.status", Value: u.Status},
						{Key: "items.$.unitPrice", Value: u.UnitPrice},
					}},
				},
			)
			if err != nil {
				return err
			}
		case false:
			_, err := col.UpdateOne(
				ctx,
				bson.D{
					{Key: "userId", Value: userID},
					{Key: "items.productId", Value: u.ProductID},
				},
				bson.D{
					{Key: "$set", Value: bson.D{
						{Key: "items.$.stock", Value: u.Stock},
						{Key: "items.$.status", Value: u.Status},
					}},
				},
			)
			if err != nil {
				return err
			}
		default:
			return errors.New("unexpected error")
		}
	}
	return nil
}

// RemoveItem removes the cart entry for the given productId from the user's cart.
// Returns ErrCartItemNotFound if no such product exists in that user's cart.
// totalPrice is recalculated and persisted after the removal.
func (r *CartRepository) RemoveItem(userID string, productID string) error {
	col, err := r.col()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := col.UpdateOne(
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
func (r *CartRepository) ClearCart(userID string) error {
	col, err := r.col()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = col.UpdateOne(
		ctx,
		bson.D{{Key: "userId", Value: userID}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "items", Value: bson.A{}},
			{Key: "totalPrice", Value: models.NewPrice("0.00")},
		}}},
		options.UpdateOne().SetUpsert(true),
	)
	return err
}

// recalcTotalPrice reads the current items from the user's cart and writes the
// correct totalPrice back to the document in a single targeted $set.
func (r *CartRepository) recalcTotalPrice(ctx context.Context, userID string) error {
	col, err := r.col()
	if err != nil {
		return err
	}

	var cart models.Cart
	err = col.FindOne(ctx, bson.D{{Key: "userId", Value: userID}}).Decode(&cart)
	if err != nil {
		return err
	}

	total := models.NewPrice("0.00")
	for _, item := range cart.Items {
		for range item.Quantity {
			total = total.Add(item.UnitPrice)
		}
	}

	_, err = col.UpdateOne(
		ctx,
		bson.D{{Key: "userId", Value: userID}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "totalPrice", Value: total}}}},
	)
	return err
}
