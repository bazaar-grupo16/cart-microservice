package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/ValentinMarturet/is2-rojas/src/config"
	"github.com/ValentinMarturet/is2-rojas/src/controller"
	"github.com/ValentinMarturet/is2-rojas/src/repository"
	"github.com/ValentinMarturet/is2-rojas/src/router"
	"github.com/ValentinMarturet/is2-rojas/src/service"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

func setupRouter() (*gin.Engine, *sql.Tx) {
	logger, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(logger)

	err := godotenv.Load("../.env", ".env")
	if err != nil {
		zap.S().Info("Error loading .env file. Using system environment variables.")
	}

	// Forzar el uso de la base de datos de pruebas para no ensuciar la de desarrollo
	os.Setenv("DATABASE_NAME", "ecommerce_test")

	config.ConnectDatabase()
	config.RunMigrations()
	tx, err := config.DB.Begin()
	if err != nil {
		zap.S().Fatalf("Failed to begin transaction: %v", err)
	}

	productRepo := repository.NewProductRepository(tx)
	productService := service.NewProductService(productRepo)
	productController := controller.NewProductController(productService)

	cartRepo := repository.NewCartRepository(tx)
	cartService := service.NewCartService(cartRepo, productRepo)
	cartController := controller.NewCartController(cartService)

	return router.SetupRouter(productController, cartController), tx
}

func TestCreateProduct(t *testing.T) {
	r, tx := setupRouter()
	defer tx.Rollback()
	data := struct {
		SellerID    int     `json:"sellerId"`
		Title       string  `json:"title"`
		Description string  `json:"description"`
		Price       float64 `json:"price"`
	}{
		SellerID:    1,
		Title:       "Test Product",
		Description: "This is a test product",
		Price:       9.99,
	}
	jsonValue, _ := json.Marshal(data)

	req := httptest.NewRequest("POST", "/products", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status code %d, got %d", http.StatusCreated, w.Code)
	}
	// get all products to check if the new product is there
	getReq := httptest.NewRequest("GET", "/products", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, getReq)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("Test Product")) {
		t.Errorf("Expected response body to contain 'Test Product', got %s", w.Body.String())
	}
}

func TestGetAllProducts(t *testing.T) {
	r, tx := setupRouter()
	defer tx.Rollback()

	req := httptest.NewRequest("GET", "/products", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}
}

func TestIntegrationCRUD(t *testing.T) {
	r, tx := setupRouter()
	defer tx.Rollback()

	data := struct {
		SellerID    int     `json:"sellerId"`
		Title       string  `json:"title"`
		Description string  `json:"description"`
		Price       float64 `json:"price"`
	}{
		SellerID:    1,
		Title:       "Test Product",
		Description: "This is a test product",
		Price:       9.99,
	}
	jsonValue, _ := json.Marshal(data)

	postReq := httptest.NewRequest("POST", "/products", bytes.NewBuffer(jsonValue))
	postReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, postReq)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status code %d, got %d", http.StatusCreated, w.Code)
	}
	var response struct {
		Data struct {
			ID int `json:"id"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &response)
	productID := response.Data.ID

	getReq := httptest.NewRequest("GET", "/products", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, getReq)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("Test Product")) {
		t.Errorf("Expected response body to contain 'Test Product', got %s", w.Body.String())
	}

	getReq = httptest.NewRequest("GET", fmt.Sprintf("/products/%d", productID), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, getReq)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("Test Product")) {
		t.Errorf("Expected response body to contain 'Test Product', got %s", w.Body.String())
	}

	updateData := struct {
		Title       string  `json:"title"`
		Description string  `json:"description"`
		Price       float64 `json:"price"`
	}{
		Title:       "Updated Product",
		Description: "This is an updated test product",
		Price:       19.99,
	}
	jsonValue, _ = json.Marshal(updateData)

	putReq := httptest.NewRequest("PUT", fmt.Sprintf("/products/%d", productID), bytes.NewBuffer(jsonValue))
	putReq.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, putReq)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	deleteReq := httptest.NewRequest("DELETE", fmt.Sprintf("/products/%d", productID), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, deleteReq)
	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status code %d, got %d", http.StatusNoContent, w.Code)
	}
}

func TestUpdateProduct(t *testing.T) {
	r, tx := setupRouter()
	defer tx.Rollback()

	data := struct {
		SellerID    int     `json:"sellerId"`
		Title       string  `json:"title"`
		Description string  `json:"description"`
		Price       float64 `json:"price"`
	}{
		SellerID:    1,
		Title:       "Test Product",
		Description: "This is a test product",
		Price:       9.99,
	}
	jsonValue, _ := json.Marshal(data)

	postReq := httptest.NewRequest("POST", "/products", bytes.NewBuffer(jsonValue))
	postReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, postReq)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status code %d, got %d", http.StatusCreated, w.Code)
	}
	var response struct {
		Data struct {
			ID int `json:"id"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &response)
	productID := response.Data.ID

	updateData := struct {
		Title       string  `json:"title"`
		Description string  `json:"description"`
		Price       float64 `json:"price"`
	}{
		Title:       "Updated Product",
		Description: "This is an updated test product",
		Price:       19.99,
	}
	jsonValue, _ = json.Marshal(updateData)
	putReq := httptest.NewRequest("PUT", fmt.Sprintf("/products/%d", productID), bytes.NewBuffer(jsonValue))
	putReq.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, putReq)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}
	// cast product to struct and check if it matches to the updated values
	var updateResponse struct {
		Data struct {
			ID          int     `json:"id"`
			SellerID    int     `json:"sellerId"`
			Title       string  `json:"title"`
			Description string  `json:"description"`
			Price       float64 `json:"price"`
			CreatedAt   string  `json:"created_at"`
			UpdatedAt   string  `json:"updated_at"`
		} `json:"data"`
	}
	if json.NewDecoder(bytes.NewBuffer(w.Body.Bytes())).Decode(&updateResponse) != nil {

	}
	updatedProduct := updateResponse.Data
	if updatedProduct.Title != updateData.Title {
		t.Errorf("Expected title %s, got %s", updateData.Title, updatedProduct.Title)
	}
	if updatedProduct.Description != updateData.Description {
		t.Errorf("Expected description %s, got %s", updateData.Description, updatedProduct.Description)
	}
	if updatedProduct.Price != updateData.Price {
		t.Errorf("Expected price %.2f, got %.2f", updateData.Price, updatedProduct.Price)
	}

}

func TestAddProductToCart(t *testing.T) {
	r, tx := setupRouter()
	defer tx.Rollback()

	productData := struct {
		SellerID    int     `json:"sellerId"`
		Title       string  `json:"title"`
		Description string  `json:"description"`
		Price       float64 `json:"price"`
	}{
		SellerID:    1,
		Title:       "Test Product",
		Description: "This is a test product",
		Price:       9.99,
	}
	jsonValue, _ := json.Marshal(productData)

	postReq := httptest.NewRequest("POST", "/products", bytes.NewBuffer(jsonValue))
	postReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, postReq)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status code %d, got %d", http.StatusCreated, w.Code)
	}
	var response struct {
		Data struct {
			ID int `json:"id"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &response)
	productID := response.Data.ID

	cartItemData := struct {
		ProductID int `json:"productId"`
	}{
		ProductID: productID,
	}
	jsonValue, _ = json.Marshal(cartItemData)

	testUserID := 1

	postReq = httptest.NewRequest("POST", fmt.Sprintf("/cart/%d/items", testUserID), bytes.NewBuffer(jsonValue))
	postReq.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, postReq)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status code %d, got %d", http.StatusCreated, w.Code)
	}
	var cartItemResp struct {
		Data struct {
			ID        int     `json:"id"`
			ProductID int     `json:"productId"`
			Title     string  `json:"title"`
			UnitPrice float64 `json:"unitPrice"`
			AddedAt   string  `json:"addedAt"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &cartItemResp)
	cartItem := cartItemResp.Data

	if cartItem.ProductID != productID {
		t.Errorf("Expected product ID %d in cart item, got %d", productID, cartItem.ProductID)
	}

	// test if cart contains the item
	getReq := httptest.NewRequest("GET", fmt.Sprintf("/cart/%d", testUserID), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, getReq)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("Test Product")) {
		t.Errorf("Expected response body to contain 'Test Product', got %s", w.Body.String())
	}
	// check cart lenght is 1
	var cartResp struct {
		Data struct {
			UserID int `json:"userId"`
			Items  []struct {
				ID        int     `json:"id"`
				ProductID int     `json:"productId"`
				Title     string  `json:"title"`
				UnitPrice float64 `json:"unitPrice"`
				AddedAt   string  `json:"addedAt"`
			} `json:"items"`
			TotalPrice float64 `json:"totalPrice"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &cartResp)
	cart := cartResp.Data

	if len(cart.Items) != 1 {
		t.Errorf("Expected cart to have 1 item, got %d items", len(cart.Items))
	}
}

func TestGetCart(t *testing.T) {
	r, tx := setupRouter()
	defer tx.Rollback()

	testUserID := 1

	req := httptest.NewRequest("GET", fmt.Sprintf("/cart/%d", testUserID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}
	// check cart is empty
	var cartResp struct {
		Data struct {
			UserID int `json:"userId"`
			Items  []struct {
				ID        int     `json:"id"`
				ProductID int     `json:"productId"`
				Title     string  `json:"title"`
				UnitPrice float64 `json:"unitPrice"`
				AddedAt   string  `json:"addedAt"`
			} `json:"items"`
			TotalPrice float64 `json:"totalPrice"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &cartResp)
	cart := cartResp.Data

	if len(cart.Items) != 0 {
		t.Errorf("Expected cart to be empty, got %d items", len(cart.Items))
	}

	HelperAddProductToCart(r, testUserID)

	req = httptest.NewRequest("GET", fmt.Sprintf("/cart/%d", testUserID), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("Test Product")) {
		t.Errorf("Expected response body to contain 'Test Product', got %s", w.Body.String())
	}
	// check cart length is 1
	json.Unmarshal(w.Body.Bytes(), &cartResp)
	cart = cartResp.Data

	if len(cart.Items) != 1 {
		t.Errorf("Expected cart to have 1 item, got %d items", len(cart.Items))
	}

}

func TestClearCart(t *testing.T) {
	r, tx := setupRouter()
	defer tx.Rollback()

	testUserID := 1
	// add a product to the cart first

	HelperAddProductToCart(r, testUserID)

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/cart/%d", testUserID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status code %d, got %d", http.StatusNoContent, w.Code)
	}

	// check cart is empty
	getReq := httptest.NewRequest("GET", fmt.Sprintf("/cart/%d", testUserID), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, getReq)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var cartResp struct {
		Data struct {
			UserID int `json:"userId"`
			Items  []struct {
				ID        int     `json:"id"`
				ProductID int     `json:"productId"`
				Title     string  `json:"title"`
				UnitPrice float64 `json:"unitPrice"`
				AddedAt   string  `json:"addedAt"`
			} `json:"items"`
			TotalPrice float64 `json:"totalPrice"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &cartResp)
	cart := cartResp.Data

	if len(cart.Items) != 0 {
		t.Errorf("Expected cart to be empty, got %d items", len(cart.Items))
	}
}

func TestRemoveFromCart(t *testing.T) {
	r, tx := setupRouter()
	defer tx.Rollback()

	testUserID := 1
	// add a product to the cart first
	HelperAddProductToCart(r, testUserID)

	// add second product to the cart
	HelperAddProductToCart(r, testUserID)

	// get cart to find cart item ID
	getReq := httptest.NewRequest("GET", fmt.Sprintf("/cart/%d", testUserID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, getReq)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var cartResp struct {
		Data struct {
			UserID int `json:"userId"`
			Items  []struct {
				ID        int     `json:"id"`
				ProductID int     `json:"productId"`
				Title     string  `json:"title"`
				UnitPrice float64 `json:"unitPrice"`
				AddedAt   string  `json:"addedAt"`
			} `json:"items"`
			TotalPrice float64 `json:"totalPrice"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &cartResp)
	cart := cartResp.Data

	if len(cart.Items) == 0 {
		t.Fatalf("Expected cart to have items, got 0")
	}

	cartItemID := cart.Items[0].ID

	deleteReq := httptest.NewRequest("DELETE", fmt.Sprintf("/cart/%d/items/%d", testUserID, cartItemID), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, deleteReq)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status code %d, got %d", http.StatusNoContent, w.Code)
	}

	// check cart item is removed
	getReq = httptest.NewRequest("GET", fmt.Sprintf("/cart/%d", testUserID), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, getReq)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	json.Unmarshal(w.Body.Bytes(), &cartResp)
	cart = cartResp.Data

	for _, item := range cart.Items {
		if item.ID == cartItemID {
			t.Errorf("Expected cart item ID %d to be removed, but it is still in the cart", cartItemID)
		}
	}
	if len(cart.Items) != 1 {
		t.Errorf("Expected cart to have 1 item after removal, got %d", len(cart.Items))
	}
}

func TestDeletingProductRemovesItFromCarts(t *testing.T) {
	r, tx := setupRouter()
	defer tx.Rollback()

	testUserID := 1
	// add a product to the cart first
	HelperAddProductToCart(r, testUserID)

	// get cart to find product ID
	getReq := httptest.NewRequest("GET", fmt.Sprintf("/cart/%d", testUserID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, getReq)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var cartResp struct {
		Data struct {
			UserID int `json:"userId"`
			Items  []struct {
				ID        int     `json:"id"`
				ProductID int     `json:"productId"`
				Title     string  `json:"title"`
				UnitPrice float64 `json:"unitPrice"`
				AddedAt   string  `json:"addedAt"`
			} `json:"items"`
			TotalPrice float64 `json:"totalPrice"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &cartResp)
	cart := cartResp.Data

	if len(cart.Items) == 0 {
		t.Fatalf("Expected cart to have items, got 0")
	}

	productID := cart.Items[0].ProductID

	deleteReq := httptest.NewRequest("DELETE", fmt.Sprintf("/products/%d", productID), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, deleteReq)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status code %d, got %d", http.StatusNoContent, w.Code)
	}

	// check cart is empty
	getReq = httptest.NewRequest("GET", fmt.Sprintf("/cart/%d", testUserID), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, getReq)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	json.Unmarshal(w.Body.Bytes(), &cartResp)
	cart = cartResp.Data

	if len(cart.Items) != 0 {
		t.Errorf("Expected cart to be empty after product deletion, got %d items", len(cart.Items))
	}
}

func TestGetEmptyCartFormat(t *testing.T) {
	r, tx := setupRouter()
	defer tx.Rollback()

	testUserID := 1

	req := httptest.NewRequest("GET", fmt.Sprintf("/cart/%d", testUserID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	// Verify the exact JSON format
	// Expected: {"data":{"userId":1,"items":[],"totalPrice":0.00}}
	bodyString := w.Body.String()
	if !bytes.Contains(w.Body.Bytes(), []byte(`"items":[]`)) {
		t.Errorf("Expected 'items' to be [], got %s", bodyString)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"totalPrice":0.00`)) {
		t.Errorf("Expected 'totalPrice' to be 0.00, got %s", bodyString)
	}
}

func HelperAddProductToCart(r *gin.Engine, userID int) {
	productData := struct {
		SellerID    int     `json:"sellerId"`
		Title       string  `json:"title"`
		Description string  `json:"description"`
		Price       float64 `json:"price"`
	}{
		SellerID:    1,
		Title:       "Test Product",
		Description: "This is a test product",
		Price:       9.99,
	}
	jsonValue, _ := json.Marshal(productData)

	postReq := httptest.NewRequest("POST", "/products", bytes.NewBuffer(jsonValue))
	postReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, postReq)

	if w.Code != http.StatusCreated {
		zap.S().Fatalf("Expected status code %d, got %d", http.StatusCreated, w.Code)
	}
	var response struct {
		Data struct {
			ID int `json:"id"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &response)
	productID := response.Data.ID

	cartItemData := struct {
		ProductID int `json:"productId"`
	}{
		ProductID: productID,
	}
	jsonValue, _ = json.Marshal(cartItemData)

	postReq = httptest.NewRequest("POST", fmt.Sprintf("/cart/%d/items", userID), bytes.NewBuffer(jsonValue))
	postReq.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, postReq)

	if w.Code != http.StatusCreated {
		zap.S().Fatalf("Expected status code %d, got %d", http.StatusCreated, w.Code)
	}
}

func TestContentTypeMiddleware(t *testing.T) {
	r, tx := setupRouter()
	defer tx.Rollback()

	// Sending a POST without Content-Type: application/json should fail with 400
	req := httptest.NewRequest("POST", "/products", bytes.NewBuffer([]byte(`{"test":"test"}`)))
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/problem+json" {
		t.Errorf("Expected Content-Type application/problem+json, got %s", contentType)
	}

	var response struct {
		Type     string `json:"type"`
		Title    string `json:"title"`
		Status   int    `json:"status"`
		Detail   string `json:"detail"`
		Instance string `json:"instance"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Title != "Invalid Content-Type" {
		t.Errorf("Expected title 'Invalid Content-Type', got %s", response.Title)
	}
	if response.Type != "about:blank" {
		t.Errorf("Expected type 'about:blank', got %s", response.Type)
	}
}

func TestUpdateProductCreatedAt(t *testing.T) {
	r, tx := setupRouter()
	defer tx.Rollback()

	// 1. Create a product
	postBody := `{"sellerId": 1, "title": "Test D", "description": "D", "price": 10.00}`
	req := httptest.NewRequest("POST", "/products", bytes.NewBuffer([]byte(postBody)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var res struct {
		Data struct {
			ID        int    `json:"id"`
			CreatedAt string `json:"createdAt"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &res)

	id := res.Data.ID
	originalCreated := res.Data.CreatedAt

	// 2. Update the product
	putBody := `{"title": "Test D2", "description": "D2", "price": 20.00}`
	reqPut := httptest.NewRequest("PUT", fmt.Sprintf("/products/%d", id), bytes.NewBuffer([]byte(putBody)))
	reqPut.Header.Set("Content-Type", "application/json")
	wPut := httptest.NewRecorder()
	r.ServeHTTP(wPut, reqPut)

	var putRes struct {
		Data struct {
			CreatedAt string `json:"createdAt"`
		} `json:"data"`
	}
	json.Unmarshal(wPut.Body.Bytes(), &putRes)
	newCreated := putRes.Data.CreatedAt

	// 3. Ensure createdAt timestamp didn't change
	if originalCreated != newCreated {
		t.Errorf("CreatedAt inadvertently changed from '%s' to '%s' during PUT operation!", originalCreated, newCreated)
	}
}

func TestCreateProduct_ReturnsErrorWhenTypesAreInvalid(t *testing.T) {
	r, tx := setupRouter()
	defer tx.Rollback()

	// Send sellerId as a boolean instead of an int
	postBody := `{"sellerId": true, "title": "Test Boolean", "description": "Desc", "price": 10.00}`
	req := httptest.NewRequest("POST", "/products", bytes.NewBuffer([]byte(postBody)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d when sending boolean for sellerId, got %d", http.StatusBadRequest, w.Code)
	}

	var response struct {
		Type     string `json:"type"`
		Title    string `json:"title"`
		Status   int    `json:"status"`
		Detail   string `json:"detail"`
		Instance string `json:"instance"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Title != "Invalid request payload" {
		t.Errorf("Expected title 'Invalid request payload', got '%s'", response.Title)
	}
}
