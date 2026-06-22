package controller

import (
	"context"
	"errors"
	"net/http"

	"github.com/bazaar-grupo16/cart-microservice/src/models"
	"github.com/bazaar-grupo16/cart-microservice/src/repository"
	"github.com/bazaar-grupo16/cart-microservice/src/service"
	"github.com/bazaar-grupo16/cart-microservice/src/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// CartServiceInterface defines the methods required by CartController.
type CartServiceInterface interface {
	GetCartByUserID(ctx context.Context, userID string) (*models.Cart, error)
	ClearCart(ctx context.Context, userID string) error
	AddToCart(ctx context.Context, userID string, item *models.CartItem) error
	RemoveFromCart(ctx context.Context, userID string, productID string) error
	IncrementItem(ctx context.Context, userID string, productID string, qty int) error
	DecrementItem(ctx context.Context, userID string, productID string, qty int) error
}

// CartController handles cart-related HTTP requests.
type CartController struct {
	CartService CartServiceInterface
}

// NewCartController creates a new instance of CartController.
func NewCartController(cartService CartServiceInterface) *CartController {
	return &CartController{
		CartService: cartService,
	}
}

// GetCart handles GET /cart/:user_id.
// @Summary Get cart by user ID (Unauthenticated)
// @Description Retrieves the shopping cart for a given user ID.
// @Tags cart
// @Produce json
// @Param user_id path string true "User ID"
// @Success 200 {object} models.Cart
// @Failure 400 {object} models.Problem
// @Failure 500 {object} models.Problem
// @Router /cart/{user_id} [get]
func (c *CartController) GetCart(ctx *gin.Context) {
	strID := ctx.Param("user_id")
	userID := strID
	if !utils.IsUUID(userID) {
		zap.S().Debugw("Invalid user ID format for getting cart", "id", strID)
		utils.ProblemJSON(ctx, http.StatusBadRequest, "Invalid User ID", "The provided user identification format is invalid", ctx.Request.URL.String())
		return
	}

	cart, err := c.CartService.GetCartByUserID(ctx, userID)
	if err != nil {
		zap.S().Errorw("Failed to retrieve cart", "userID", userID, "error", err)
		utils.ProblemJSON(ctx, http.StatusInternalServerError, "Internal Server Error", "Could not retrieve the cart at this time", ctx.Request.URL.String())
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"data": cart})
}

// ClearCart handles DELETE /cart/:user_id.
// @Summary Clear cart by user ID (Unauthenticated)
// @Description Clears all items from the shopping cart for a given user ID.
// @Tags cart
// @Param user_id path string true "User ID"
// @Success 204
// @Failure 400 {object} models.Problem
// @Failure 500 {object} models.Problem
// @Router /cart/{user_id} [delete]
func (c *CartController) ClearCart(ctx *gin.Context) {
	strID := ctx.Param("user_id")
	userID := strID
	if !utils.IsUUID(userID) {
		zap.S().Debugw("Invalid user ID format for clearing cart", "id", strID)
		utils.ProblemJSON(ctx, http.StatusBadRequest, "Invalid User ID", "The provided user identification format is invalid", ctx.Request.URL.String())
		return
	}

	err := c.CartService.ClearCart(ctx, userID)
	if err != nil {
		zap.S().Errorw("Failed to clear cart", "userID", userID, "error", err)
		utils.ProblemJSON(ctx, http.StatusInternalServerError, "Internal Server Error", "Could not clear the cart at this time", ctx.Request.URL.String())
		return
	}

	ctx.Status(http.StatusNoContent)
}

// AddToCart handles POST /cart/:user_id/items.
// Accepts a productId; the title and unitPrice are fetched from the Catalog service
// and stored as an immutable snapshot. If the product is already in the cart, its
// quantity is incremented instead of adding a duplicate entry.
// @Summary Add item to cart (Unauthenticated)
// @Description Adds a product to the cart for a given user ID.
// @Tags cart
// @Accept json
// @Produce json
// @Param user_id path string true "User ID"
// @Param request body models.AddCartRequest true "Item to add"
// @Success 201 {object} models.CartItem
// @Failure 400 {object} models.Problem
// @Failure 403 {object} models.Problem
// @Failure 404 {object} models.Problem
// @Failure 409 {object} models.Problem
// @Failure 422 {object} models.Problem
// @Failure 500 {object} models.Problem
// @Failure 503 {object} models.Problem
// @Router /cart/{user_id}/items [post]
func (c *CartController) AddToCart(ctx *gin.Context) {
	strID := ctx.Param("user_id")
	userID := strID
	if !utils.IsUUID(userID) {
		zap.S().Debugw("Invalid user ID format for adding to cart", "id", strID)
		utils.ProblemJSON(ctx, http.StatusBadRequest, "Invalid User ID", "The provided user identification format is invalid", ctx.Request.URL.String())
		return
	}

	var req models.AddCartRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		zap.S().Errorw("Invalid request payload for adding to cart", "error", err, "userID", userID)
		utils.ProblemJSON(ctx, http.StatusBadRequest, "Invalid Request Payload", "The provided item information is malformed or invalid", ctx.Request.URL.String())
		return
	}

	if !utils.IsUUID(req.ProductID) {
		zap.S().Debugw("Invalid product ID format for adding to cart", "id", req.ProductID, "userID", userID)
		utils.ProblemJSON(ctx, http.StatusBadRequest, "Invalid Product ID", "The provided product identification format is invalid", ctx.Request.URL.String())
		return
	}

	qty := 1
	if req.Quantity != nil {
		qty = *req.Quantity
	}

	item := models.CartItem{
		ProductID: req.ProductID,
		Quantity:  qty,
	}

	err := c.CartService.AddToCart(ctx, userID, &item)
	switch {
	case errors.Is(err, service.ErrProductNotFound):
		zap.S().Debugw("Product not found in catalog", "userID", userID, "productID", req.ProductID)
		utils.ProblemJSON(ctx, http.StatusNotFound, "Product Not Found",
			"The requested product does not exist in the catalog", ctx.Request.URL.String())
	case errors.Is(err, service.ErrCatalogUnavailable):
		zap.S().Warnw("Catalog service unavailable when adding to cart", "userID", userID, "productID", req.ProductID)
		utils.ProblemJSON(ctx, http.StatusServiceUnavailable, "Catalog Unavailable",
			"The catalog service is temporarily unavailable. Please try again later.", ctx.Request.URL.String())
	case errors.Is(err, service.ErrCannotAddOwnProduct):
		zap.S().Debugw("Cannot add own product to cart", "userID", userID, "productID", req.ProductID)
		utils.ProblemJSON(ctx, http.StatusConflict, "Cannot Add Own Product",
			"You cannot add your own product to the cart", ctx.Request.URL.String())
	case errors.Is(err, service.ErrProductBlocked):
		zap.S().Debugw("Cannot add blocked product to cart", "userID", userID, "productID", req.ProductID)
		utils.ProblemJSON(ctx, http.StatusForbidden, "Product Blocked",
			"This product is blocked and cannot be added to the cart", ctx.Request.URL.String())
	case errors.Is(err, service.ErrInsufficientStock):
		zap.S().Debugw("Insufficient stock", "userID", userID, "productID", req.ProductID)
		utils.ProblemJSON(ctx, http.StatusUnprocessableEntity, "Insufficient Stock",
			"The requested quantity exceeds available stock", ctx.Request.URL.String())
	case err != nil:
		zap.S().Errorw("Unexpected error adding item to cart", "userID", userID, "productID", req.ProductID, "error", err)
		utils.ProblemJSON(ctx, http.StatusInternalServerError, "Internal Server Error",
			"Could not add the item to the cart", ctx.Request.URL.String())
	default:
		ctx.JSON(http.StatusCreated, gin.H{"data": item})
	}
}

// RemoveFromCart handles DELETE /cart/:user_id/items/:cart_item_id.
// The :cart_item_id path parameter now refers to the catalog product ID, since cart
// items are identified by their productId (one entry per product per user).
// @Summary Remove item from cart (Unauthenticated)
// @Description Removes a specific product from the cart for a given user ID.
// @Tags cart
// @Param user_id path string true "User ID"
// @Param cart_item_id path string true "Product ID"
// @Success 204
// @Failure 400 {object} models.Problem
// @Failure 404 {object} models.Problem
// @Failure 500 {object} models.Problem
// @Router /cart/{user_id}/items/{cart_item_id} [delete]
func (c *CartController) RemoveFromCart(ctx *gin.Context) {
	strID := ctx.Param("user_id")
	userID := strID
	if !utils.IsUUID(userID) {
		zap.S().Debugw("Invalid user ID format for removing from cart", "id", strID)
		utils.ProblemJSON(ctx, http.StatusBadRequest, "Invalid User ID", "The provided user identification format is invalid", ctx.Request.URL.String())
		return
	}

	productID := ctx.Param("cart_item_id")
	if !utils.IsUUID(productID) {
		zap.S().Debugw("Invalid product ID format for removing from cart", "id", productID)
		utils.ProblemJSON(ctx, http.StatusBadRequest, "Invalid Product ID", "The provided product identification format is invalid", ctx.Request.URL.String())
		return
	}

	err := c.CartService.RemoveFromCart(ctx, userID, productID)
	if err != nil {
		zap.S().Errorw("Failed to remove item from cart", "userID", userID, "productID", productID, "error", err)
		utils.ProblemJSON(ctx, http.StatusNotFound, "Cart Item Not Found",
			"Could not find the requested product in this user's cart", ctx.Request.URL.String())
		return
	}

	ctx.Status(http.StatusNoContent)
}

// IncrementItem handles PUT /cart/:user_id/:product_id/increment.
// Increments the quantity of the specified product in the user's cart by the given amount.
// @Summary Increment item quantity (Unauthenticated)
// @Description Increments the quantity of a specific product in the cart for a given user ID.
// @Tags cart
// @Accept json
// @Param user_id path string true "User ID"
// @Param product_id path string true "Product ID"
// @Param request body models.QuantityRequest true "Quantity to increment"
// @Success 204
// @Failure 400 {object} models.Problem
// @Failure 403 {object} models.Problem
// @Failure 404 {object} models.Problem
// @Failure 409 {object} models.Problem
// @Failure 422 {object} models.Problem
// @Failure 500 {object} models.Problem
// @Router /cart/{user_id}/{product_id}/increment [put]
func (c *CartController) IncrementItem(ctx *gin.Context) {
	userID, productID, ok := utils.ParseUserAndProduct(ctx)
	if !ok {
		return
	}

	var req models.QuantityRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		zap.S().Errorw("Invalid request payload for increment", "error", err, "userID", userID, "productID", productID)
		utils.ProblemJSON(ctx, http.StatusBadRequest, "Invalid Request Payload", "The provided quantity is malformed or missing", ctx.Request.URL.String())
		return
	}

	err := c.CartService.IncrementItem(ctx, userID, productID, req.Quantity)
	switch {
	case errors.Is(err, repository.ErrCartItemNotFound):
		zap.S().Debugw("Cart item not found for increment", "userID", userID, "productID", productID)
		utils.ProblemJSON(ctx, http.StatusNotFound, "Cart Item Not Found",
			"Could not find the requested product in this user's cart", ctx.Request.URL.String())
	case errors.Is(err, service.ErrCannotAddOwnProduct):
		zap.S().Debugw("Cannot increment own product", "userID", userID, "productID", productID)
		utils.ProblemJSON(ctx, http.StatusConflict, "Cannot Add Own Product",
			"You cannot add your own product to the cart", ctx.Request.URL.String())
	case errors.Is(err, service.ErrProductBlocked):
		zap.S().Debugw("Cannot increment blocked product", "userID", userID, "productID", productID)
		utils.ProblemJSON(ctx, http.StatusForbidden, "Product Blocked",
			"This product is blocked and cannot be added to the cart", ctx.Request.URL.String())
	case errors.Is(err, service.ErrInsufficientStock):
		zap.S().Debugw("Insufficient stock for increment", "userID", userID, "productID", productID)
		utils.ProblemJSON(ctx, http.StatusUnprocessableEntity, "Insufficient Stock",
			"The requested quantity exceeds available stock", ctx.Request.URL.String())
	case err != nil:
		zap.S().Errorw("Failed to increment cart item", "userID", userID, "productID", productID, "error", err)
		utils.ProblemJSON(ctx, http.StatusInternalServerError, "Internal Server Error",
			"Could not increment the cart item at this time", ctx.Request.URL.String())
	default:
		ctx.Status(http.StatusNoContent)
	}
}

// DecrementItem handles PUT /cart/:user_id/:product_id/decrement.
// Decrements the quantity of the specified product in the user's cart by the given amount.
// If the resulting quantity is <= 0, the item is removed from the cart automatically.
// @Summary Decrement item quantity (Unauthenticated)
// @Description Decrements the quantity of a specific product in the cart for a given user ID.
// @Tags cart
// @Accept json
// @Param user_id path string true "User ID"
// @Param product_id path string true "Product ID"
// @Param request body models.QuantityRequest true "Quantity to decrement"
// @Success 204
// @Failure 400 {object} models.Problem
// @Failure 404 {object} models.Problem
// @Failure 500 {object} models.Problem
// @Router /cart/{user_id}/{product_id}/decrement [put]
func (c *CartController) DecrementItem(ctx *gin.Context) {
	userID, productID, ok := utils.ParseUserAndProduct(ctx)
	if !ok {
		return
	}

	var req models.QuantityRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		zap.S().Errorw("Invalid request payload for decrement", "error", err, "userID", userID, "productID", productID)
		utils.ProblemJSON(ctx, http.StatusBadRequest, "Invalid Request Payload", "The provided quantity is malformed or missing", ctx.Request.URL.String())
		return
	}

	err := c.CartService.DecrementItem(ctx, userID, productID, req.Quantity)
	switch {
	case errors.Is(err, repository.ErrCartItemNotFound):
		zap.S().Debugw("Cart item not found for decrement", "userID", userID, "productID", productID)
		utils.ProblemJSON(ctx, http.StatusNotFound, "Cart Item Not Found",
			"Could not find the requested product in this user's cart", ctx.Request.URL.String())
	case err != nil:
		zap.S().Errorw("Failed to decrement cart item", "userID", userID, "productID", productID, "error", err)
		utils.ProblemJSON(ctx, http.StatusInternalServerError, "Internal Server Error",
			"Could not decrement the cart item at this time", ctx.Request.URL.String())
	default:
		ctx.Status(http.StatusNoContent)
	}
}

// AUNTENTICATED (OWN CART) VERSIONS OF THE ABOVE METHODS
// The following methods are the "own cart" versions of the above, which extract the user ID from the X-User-ID header instead of the path parameter.
// GetOwnCart handles GET /cart.
// @Summary Get own cart (Authenticated)
// @Description Retrieves the shopping cart for the authenticated user.
// @Tags cart
// @Security BearerAuth
// @Produce json
// @Success 200 {object} models.Cart
// @Failure 400 {object} models.Problem
// @Failure 401 {object} models.Problem
// @Failure 500 {object} models.Problem
// @Router /cart [get]
func (c *CartController) GetOwnCart(ctx *gin.Context) {
	userID, ok := utils.GetUserIDFromHeader(ctx)
	if !ok {
		return
	}

	cart, err := c.CartService.GetCartByUserID(ctx, userID)
	if err != nil {
		zap.S().Errorw("Failed to retrieve cart", "userID", userID, "error", err)
		utils.ProblemJSON(ctx, http.StatusInternalServerError, "Internal Server Error", "Could not retrieve the cart at this time", ctx.Request.URL.String())
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"data": cart})
}

// ClearOwnCart handles DELETE /cart.
// @Summary Clear own cart (Authenticated)
// @Description Clears all items from the shopping cart for the authenticated user.
// @Tags cart
// @Security BearerAuth
// @Success 204
// @Failure 400 {object} models.Problem
// @Failure 401 {object} models.Problem
// @Failure 500 {object} models.Problem
// @Router /cart [delete]
func (c *CartController) ClearOwnCart(ctx *gin.Context) {
	userID, ok := utils.GetUserIDFromHeader(ctx)
	if !ok {
		return
	}

	err := c.CartService.ClearCart(ctx, userID)
	if err != nil {
		zap.S().Errorw("Failed to clear cart", "userID", userID, "error", err)
		utils.ProblemJSON(ctx, http.StatusInternalServerError, "Internal Server Error", "Could not clear the cart at this time", ctx.Request.URL.String())
		return
	}

	ctx.Status(http.StatusNoContent)
}

// AddToOwnCart handles POST /cart/items.
// @Summary Add item to own cart (Authenticated)
// @Description Adds a product to the cart for the authenticated user.
// @Tags cart
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body models.AddCartRequest true "Item to add"
// @Success 201 {object} models.CartItem
// @Failure 400 {object} models.Problem
// @Failure 401 {object} models.Problem
// @Failure 403 {object} models.Problem
// @Failure 404 {object} models.Problem
// @Failure 409 {object} models.Problem
// @Failure 422 {object} models.Problem
// @Failure 500 {object} models.Problem
// @Failure 503 {object} models.Problem
// @Router /cart/items [post]
func (c *CartController) AddToOwnCart(ctx *gin.Context) {
	userID, ok := utils.GetUserIDFromHeader(ctx)
	if !ok {
		return
	}

	var req models.AddCartRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		zap.S().Errorw("Invalid request payload for adding to cart", "error", err, "userID", userID)
		utils.ProblemJSON(ctx, http.StatusBadRequest, "Invalid Request Payload", "The provided item information is malformed or invalid", ctx.Request.URL.String())
		return
	}

	if !utils.IsUUID(req.ProductID) {
		zap.S().Debugw("Invalid product ID format for adding to cart", "id", req.ProductID, "userID", userID)
		utils.ProblemJSON(ctx, http.StatusBadRequest, "Invalid Product ID", "The provided product identification format is invalid", ctx.Request.URL.String())
		return
	}

	qty := 1
	if req.Quantity != nil {
		qty = *req.Quantity
	}

	item := models.CartItem{
		ProductID: req.ProductID,
		Quantity:  qty,
	}

	err := c.CartService.AddToCart(ctx, userID, &item)
	switch {
	case errors.Is(err, service.ErrProductNotFound):
		zap.S().Debugw("Product not found in catalog", "userID", userID, "productID", req.ProductID)
		utils.ProblemJSON(ctx, http.StatusNotFound, "Product Not Found",
			"The requested product does not exist in the catalog", ctx.Request.URL.String())
	case errors.Is(err, service.ErrCatalogUnavailable):
		zap.S().Warnw("Catalog service unavailable when adding to cart", "userID", userID, "productID", req.ProductID)
		utils.ProblemJSON(ctx, http.StatusServiceUnavailable, "Catalog Unavailable",
			"The catalog service is temporarily unavailable. Please try again later.", ctx.Request.URL.String())
	case errors.Is(err, service.ErrCannotAddOwnProduct):
		zap.S().Debugw("Cannot add own product to cart", "userID", userID, "productID", req.ProductID)
		utils.ProblemJSON(ctx, http.StatusConflict, "Cannot Add Own Product",
			"You cannot add your own product to the cart", ctx.Request.URL.String())
	case errors.Is(err, service.ErrProductBlocked):
		zap.S().Debugw("Cannot add blocked product to cart", "userID", userID, "productID", req.ProductID)
		utils.ProblemJSON(ctx, http.StatusForbidden, "Product Blocked",
			"This product is blocked and cannot be added to the cart", ctx.Request.URL.String())
	case errors.Is(err, service.ErrInsufficientStock):
		zap.S().Debugw("Insufficient stock", "userID", userID, "productID", req.ProductID)
		utils.ProblemJSON(ctx, http.StatusUnprocessableEntity, "Insufficient Stock",
			"The requested quantity exceeds available stock", ctx.Request.URL.String())
	case err != nil:
		zap.S().Errorw("Unexpected error adding item to cart", "userID", userID, "productID", req.ProductID, "error", err)
		utils.ProblemJSON(ctx, http.StatusInternalServerError, "Internal Server Error",
			"Could not add the item to the cart", ctx.Request.URL.String())
	default:
		ctx.JSON(http.StatusCreated, gin.H{"data": item})
	}
}

// RemoveFromOwnCart handles DELETE /cart/items/:cart_item_id.
// @Summary Remove item from own cart (Authenticated)
// @Description Removes a specific product from the cart for the authenticated user.
// @Tags cart
// @Security BearerAuth
// @Param cart_item_id path string true "Product ID"
// @Success 204
// @Failure 400 {object} models.Problem
// @Failure 401 {object} models.Problem
// @Failure 404 {object} models.Problem
// @Failure 500 {object} models.Problem
// @Router /cart/items/{cart_item_id} [delete]
func (c *CartController) RemoveFromOwnCart(ctx *gin.Context) {
	userID, ok := utils.GetUserIDFromHeader(ctx)
	if !ok {
		return
	}

	productID := ctx.Param("cart_item_id")
	if !utils.IsUUID(productID) {
		zap.S().Debugw("Invalid product ID format for removing from cart", "id", productID)
		utils.ProblemJSON(ctx, http.StatusBadRequest, "Invalid Product ID", "The provided product identification format is invalid", ctx.Request.URL.String())
		return
	}

	err := c.CartService.RemoveFromCart(ctx, userID, productID)
	if err != nil {
		zap.S().Errorw("Failed to remove item from cart", "userID", userID, "productID", productID, "error", err)
		utils.ProblemJSON(ctx, http.StatusNotFound, "Cart Item Not Found",
			"Could not find the requested product in this user's cart", ctx.Request.URL.String())
		return
	}

	ctx.Status(http.StatusNoContent)
}

// IncrementOwnItem handles PUT /cart/increment/:product_id.
// @Summary Increment item quantity in own cart (Authenticated)
// @Description Increments the quantity of a specific product in the cart for the authenticated user.
// @Tags cart
// @Security BearerAuth
// @Accept json
// @Param product_id path string true "Product ID"
// @Param request body models.QuantityRequest true "Quantity to increment"
// @Success 204
// @Failure 400 {object} models.Problem
// @Failure 401 {object} models.Problem
// @Failure 403 {object} models.Problem
// @Failure 404 {object} models.Problem
// @Failure 409 {object} models.Problem
// @Failure 422 {object} models.Problem
// @Failure 500 {object} models.Problem
// @Router /cart/increment/{product_id} [put]
func (c *CartController) IncrementOwnItem(ctx *gin.Context) {
	userID, productID, ok := utils.ParseUserIDHeaderAndProductID(ctx)
	if !ok {
		return
	}

	var req models.QuantityRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		zap.S().Errorw("Invalid request payload for increment", "error", err, "userID", userID, "productID", productID)
		utils.ProblemJSON(ctx, http.StatusBadRequest, "Invalid Request Payload", "The provided quantity is malformed or missing", ctx.Request.URL.String())
		return
	}

	err := c.CartService.IncrementItem(ctx, userID, productID, req.Quantity)
	switch {
	case errors.Is(err, repository.ErrCartItemNotFound):
		zap.S().Debugw("Cart item not found for increment", "userID", userID, "productID", productID)
		utils.ProblemJSON(ctx, http.StatusNotFound, "Cart Item Not Found",
			"Could not find the requested product in this user's cart", ctx.Request.URL.String())
	case errors.Is(err, service.ErrCannotAddOwnProduct):
		zap.S().Debugw("Cannot increment own product", "userID", userID, "productID", productID)
		utils.ProblemJSON(ctx, http.StatusConflict, "Cannot Add Own Product",
			"You cannot add your own product to the cart", ctx.Request.URL.String())
	case errors.Is(err, service.ErrProductBlocked):
		zap.S().Debugw("Cannot increment blocked product", "userID", userID, "productID", productID)
		utils.ProblemJSON(ctx, http.StatusForbidden, "Product Blocked",
			"This product is blocked and cannot be added to the cart", ctx.Request.URL.String())
	case errors.Is(err, service.ErrInsufficientStock):
		zap.S().Debugw("Insufficient stock for increment", "userID", userID, "productID", productID)
		utils.ProblemJSON(ctx, http.StatusUnprocessableEntity, "Insufficient Stock",
			"The requested quantity exceeds available stock", ctx.Request.URL.String())
	case err != nil:
		zap.S().Errorw("Failed to increment cart item", "userID", userID, "productID", productID, "error", err)
		utils.ProblemJSON(ctx, http.StatusInternalServerError, "Internal Server Error",
			"Could not increment the cart item at this time", ctx.Request.URL.String())
	default:
		ctx.Status(http.StatusNoContent)
	}
}

// DecrementOwnItem handles PUT /cart/decrement/:product_id.
// @Summary Decrement item quantity in own cart (Authenticated)
// @Description Decrements the quantity of a specific product in the cart for the authenticated user.
// @Tags cart
// @Security BearerAuth
// @Accept json
// @Param product_id path string true "Product ID"
// @Param request body models.QuantityRequest true "Quantity to decrement"
// @Success 204
// @Failure 400 {object} models.Problem
// @Failure 401 {object} models.Problem
// @Failure 404 {object} models.Problem
// @Failure 500 {object} models.Problem
// @Router /cart/decrement/{product_id} [put]
func (c *CartController) DecrementOwnItem(ctx *gin.Context) {
	userID, productID, ok := utils.ParseUserIDHeaderAndProductID(ctx)
	if !ok {
		return
	}

	var req models.QuantityRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		zap.S().Errorw("Invalid request payload for decrement", "error", err, "userID", userID, "productID", productID)
		utils.ProblemJSON(ctx, http.StatusBadRequest, "Invalid Request Payload", "The provided quantity is malformed or missing", ctx.Request.URL.String())
		return
	}

	err := c.CartService.DecrementItem(ctx, userID, productID, req.Quantity)
	switch {
	case errors.Is(err, repository.ErrCartItemNotFound):
		zap.S().Debugw("Cart item not found for decrement", "userID", userID, "productID", productID)
		utils.ProblemJSON(ctx, http.StatusNotFound, "Cart Item Not Found",
			"Could not find the requested product in this user's cart", ctx.Request.URL.String())
	case err != nil:
		zap.S().Errorw("Failed to decrement cart item", "userID", userID, "productID", productID, "error", err)
		utils.ProblemJSON(ctx, http.StatusInternalServerError, "Internal Server Error",
			"Could not decrement the cart item at this time", ctx.Request.URL.String())
	default:
		ctx.Status(http.StatusNoContent)
	}
}
