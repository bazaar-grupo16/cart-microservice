package repository

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/bazaar-grupo16/cart-microservice/src/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var testCollection *mongo.Collection
var testClient *mongo.Client

func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}
	dbName := os.Getenv("MONGO_DB_NAME")
	if dbName == "" {
		dbName = "cart_test"
	}
	collName := os.Getenv("MONGO_COLLECTION")
	if collName == "" {
		collName = "carts_test_repo"
	}

	clientOpts := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(clientOpts)
	if err != nil {
		panic(err)
	}

	testClient = client
	testCollection = client.Database(dbName).Collection(collName)

	// Ensure connection
	if err := client.Ping(ctx, nil); err != nil {
		panic(err)
	}

	code := m.Run()

	_ = testCollection.Drop(ctx)
	_ = client.Disconnect(ctx)
	os.Exit(code)
}

func clearCollection() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _ = testCollection.DeleteMany(ctx, bson.D{})
}

func getProvider() func() *mongo.Collection {
	return func() *mongo.Collection {
		return testCollection
	}
}

func TestGetCartByUserID(t *testing.T) {
	clearCollection()
	repo := NewCartRepository(getProvider())

	userID := "user123"

	// 1. Get empty cart
	cart, err := repo.GetCartByUserID(userID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cart.UserID != userID || len(cart.Items) != 0 {
		t.Errorf("expected empty cart for user, got %v", cart)
	}

	// 2. Insert dummy cart and retrieve
	_, _ = testCollection.InsertOne(context.Background(), bson.D{
		{Key: "userId", Value: userID},
		{Key: "items", Value: bson.A{
			bson.D{{Key: "productId", Value: "prod1"}, {Key: "quantity", Value: 2}},
		}},
	})

	cart, err = repo.GetCartByUserID(userID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(cart.Items) != 1 || cart.Items[0].ProductID != "prod1" {
		t.Errorf("expected cart with 1 item, got %v", cart)
	}
}

func TestAddItem(t *testing.T) {
	clearCollection()
	repo := NewCartRepository(getProvider())

	userID := "user-add"
	price := models.NewPrice("10.50")
	item := &models.CartItem{
		ProductID: "prod1",
		Quantity:  2,
		UnitPrice: price,
		Stock:     10,
		Status:    "active",
	}

	// 1. Add new item
	err := repo.AddItem(userID, item)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	cart, _ := repo.GetCartByUserID(userID)
	if len(cart.Items) != 1 || cart.Items[0].Quantity != 2 {
		t.Errorf("expected 1 item with qty 2, got %v", cart.Items)
	}
	if cart.TotalPrice.Float64() != 21.0 {
		t.Errorf("expected total 21.0, got %f", cart.TotalPrice.Float64())
	}

	// 2. Add existing item (should increment)
	item.Quantity = 3
	item.Stock = 8
	err = repo.AddItem(userID, item)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	cart, _ = repo.GetCartByUserID(userID)
	if len(cart.Items) != 1 || cart.Items[0].Quantity != 5 {
		t.Errorf("expected 1 item with qty 5, got %v", cart.Items)
	}
	if cart.Items[0].Stock != 8 {
		t.Errorf("expected stock to update to 8, got %d", cart.Items[0].Stock)
	}
	if cart.TotalPrice.Float64() != 52.50 {
		t.Errorf("expected total 52.50, got %f", cart.TotalPrice.Float64())
	}
}

func TestIncrementItem(t *testing.T) {
	clearCollection()
	repo := NewCartRepository(getProvider())
	userID := "user-inc"
	productID := "prod-inc"

	update := ProductUpdate{
		ProductID: productID,
		Stock:     20,
		Status:    "active",
		UnitPrice: models.NewPrice("5.00"),
	}

	// 1. Increment non-existent item
	err := repo.IncrementItem(userID, productID, 1, update)
	if !errors.Is(err, ErrCartItemNotFound) {
		t.Errorf("expected ErrCartItemNotFound, got %v", err)
	}

	// 2. Add item then increment
	err = repo.AddItem(userID, &models.CartItem{
		ProductID: productID,
		Quantity:  1,
		UnitPrice: models.NewPrice("5.00"),
	})
	if err != nil {
		t.Fatalf("setup AddItem failed: %v", err)
	}

	err = repo.IncrementItem(userID, productID, 2, update)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	cart, _ := repo.GetCartByUserID(userID)
	if cart.Items[0].Quantity != 3 {
		t.Errorf("expected qty 3, got %d", cart.Items[0].Quantity)
	}
	if cart.TotalPrice.Float64() != 15.00 {
		t.Errorf("expected total 15.0, got %f", cart.TotalPrice.Float64())
	}
}

func TestDecrementItem(t *testing.T) {
	clearCollection()
	repo := NewCartRepository(getProvider())
	userID := "user-dec"
	productID := "prod-dec"

	update := ProductUpdate{
		ProductID: productID,
		Stock:     20,
		Status:    "active",
		UnitPrice: models.NewPrice("5.00"),
	}

	// 1. Decrement non-existent
	err := repo.DecrementItem(userID, productID, 1, update)
	if !errors.Is(err, ErrCartItemNotFound) {
		t.Errorf("expected ErrCartItemNotFound, got %v", err)
	}

	// 2. Decrement existing
	err = repo.AddItem(userID, &models.CartItem{
		ProductID: productID,
		Quantity:  3,
		UnitPrice: models.NewPrice("5.00"),
	})
	if err != nil {
		t.Fatalf("setup AddItem failed: %v", err)
	}

	err = repo.DecrementItem(userID, productID, 1, update)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	cart, _ := repo.GetCartByUserID(userID)
	if cart.Items[0].Quantity != 2 {
		t.Errorf("expected qty 2, got %d", cart.Items[0].Quantity)
	}
	if cart.TotalPrice.Float64() != 10.00 {
		t.Errorf("expected total 10.0, got %f", cart.TotalPrice.Float64())
	}

	// 3. Decrement below 1 (should cap at 1)
	err = repo.DecrementItem(userID, productID, 5, update)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	cart, _ = repo.GetCartByUserID(userID)
	if cart.Items[0].Quantity != 1 {
		t.Errorf("expected qty 1, got %d", cart.Items[0].Quantity)
	}
}

func TestUpdateProductSnapshot(t *testing.T) {
	clearCollection()
	repo := NewCartRepository(getProvider())
	userID := "user-snap"

	err := repo.AddItem(userID, &models.CartItem{
		ProductID: "prod1",
		Quantity:  1,
		Stock:     5,
		Status:    "active",
		UnitPrice: models.NewPrice("10.00"),
	})
	if err != nil {
		t.Fatalf("setup AddItem failed: %v", err)
	}

	cart, err := repo.GetCartByUserID(userID)
	if err != nil || cart == nil || len(cart.Items) == 0 {
		t.Fatalf("failed to get cart or empty cart: err=%v", err)
	}

	// Make price stale to trigger price update
	cart.Items[0].UnitPrice.UpdatedAt = time.Now().Add(-25 * time.Hour)

	updates := []ProductUpdate{
		{
			ProductID: "prod1",
			Stock:     2,
			Status:    "inactive",
			UnitPrice: models.NewPrice("15.00"), // This price will be used since old is stale
		},
	}

	err = repo.UpdateProductSnapshot(userID, cart.Items, updates)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	cart, err = repo.GetCartByUserID(userID)
	if err != nil || cart == nil || len(cart.Items) == 0 {
		t.Fatalf("failed to get cart after update: err=%v", err)
	}
	if cart.Items[0].Stock != 2 || cart.Items[0].Status != "inactive" || cart.Items[0].UnitPrice.Float64() != 15.00 {
		t.Errorf("snapshot update failed, got %v", cart.Items[0])
	}

	// Test without price stale
	cart, err = repo.GetCartByUserID(userID) // Re-fetch
	if err != nil || cart == nil || len(cart.Items) == 0 {
		t.Fatalf("failed to get cart for second test: err=%v", err)
	}
	// Fresh price, should not update price
	updates[0].Stock = 0
	updates[0].Status = "out_of_stock"
	updates[0].UnitPrice = models.NewPrice("20.00") // This price should be IGNORED

	err = repo.UpdateProductSnapshot(userID, cart.Items, updates)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	cart, err = repo.GetCartByUserID(userID)
	if err != nil || cart == nil || len(cart.Items) == 0 {
		t.Fatalf("failed to get cart after second update: err=%v", err)
	}
	if cart.Items[0].Stock != 0 || cart.Items[0].UnitPrice.Float64() != 15.00 {
		t.Errorf("snapshot update without stale price failed, got %v", cart.Items[0])
	}

	// Empty updates
	err = repo.UpdateProductSnapshot(userID, cart.Items, []ProductUpdate{})
	if err != nil {
		t.Fatalf("expected no error on empty updates")
	}
}

func TestRemoveItem(t *testing.T) {
	clearCollection()
	repo := NewCartRepository(getProvider())
	userID := "user-rm"

	// 1. Remove non-existent
	err := repo.RemoveItem(userID, "prod1")
	if !errors.Is(err, ErrCartItemNotFound) {
		t.Errorf("expected ErrCartItemNotFound, got %v", err)
	}

	// 2. Remove existing
	err = repo.AddItem(userID, &models.CartItem{
		ProductID: "prod1", Quantity: 1, UnitPrice: models.NewPrice("10.00"),
	})
	if err != nil {
		t.Fatalf("setup AddItem failed: %v", err)
	}
	err = repo.AddItem(userID, &models.CartItem{
		ProductID: "prod2", Quantity: 2, UnitPrice: models.NewPrice("5.00"),
	})
	if err != nil {
		t.Fatalf("setup AddItem failed: %v", err)
	}

	cartBefore, _ := repo.GetCartByUserID(userID)
	t.Logf("Cart before remove: %+v", cartBefore)

	err = repo.RemoveItem(userID, "prod1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	cart, _ := repo.GetCartByUserID(userID)
	if len(cart.Items) != 1 || cart.Items[0].ProductID != "prod2" {
		t.Errorf("expected 1 item (prod2), got %v", cart.Items)
	}
	if cart.TotalPrice.Float64() != 10.00 {
		t.Errorf("expected total 10.00, got %f", cart.TotalPrice.Float64())
	}
}

func TestClearCart(t *testing.T) {
	clearCollection()
	repo := NewCartRepository(getProvider())
	userID := "user-clear"

	_ = repo.AddItem(userID, &models.CartItem{
		ProductID: "prod1", Quantity: 1, UnitPrice: models.NewPrice("10.00"),
	})

	err := repo.ClearCart(userID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	cart, _ := repo.GetCartByUserID(userID)
	if len(cart.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(cart.Items))
	}
	if cart.TotalPrice.Float64() != 0.0 {
		t.Errorf("expected total 0.0, got %f", cart.TotalPrice.Float64())
	}

	// Idempotency
	err = repo.ClearCart(userID)
	if err != nil {
		t.Fatalf("expected no error on 2nd clear, got %v", err)
	}
}

func TestDatabaseUnavailable(t *testing.T) {
	repo := NewCartRepository(func() *mongo.Collection { return nil })

	_, err := repo.GetCartByUserID("1")
	if !errors.Is(err, ErrDatabaseUnavailable) {
		t.Errorf("expected ErrDatabaseUnavailable, got %v", err)
	}
}
