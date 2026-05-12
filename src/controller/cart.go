package controller

import (
	"errors"
	"net/http"

	"github.com/bazaar-grupo16/cart-microservice/src/models"
	"github.com/bazaar-grupo16/cart-microservice/src/repository"
	"github.com/bazaar-grupo16/cart-microservice/src/service"
	"github.com/bazaar-grupo16/cart-microservice/src/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// CartController handles cart-related HTTP requests.
type CartController struct {
	CartService *service.CartService
}

// NewCartController creates a new instance of CartController.
func NewCartController(cartService *service.CartService) *CartController {
	return &CartController{
		CartService: cartService,
	}
}

// GetCart handles GET /cart/:user_id.
func (c *CartController) GetCart(ctx *gin.Context) {
	strId := ctx.Param("user_id")
	userID := strId
	if !utils.IsUUID(userID) {
		zap.S().Debugw("Invalid user ID format for getting cart", "id", strId)
		utils.ProblemJSON(ctx, http.StatusBadRequest, "Invalid User ID", "The provided user identification format is invalid", ctx.Request.URL.String())
		return
	}

	cart, err := c.CartService.GetCartByUserID(userID)
	if err != nil {
		zap.S().Errorw("Failed to retrieve cart", "userID", userID, "error", err)
		utils.ProblemJSON(ctx, http.StatusInternalServerError, "Internal Server Error", "Could not retrieve the cart at this time", ctx.Request.URL.String())
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"data": cart})
}

// ClearCart handles DELETE /cart/:user_id.
func (c *CartController) ClearCart(ctx *gin.Context) {
	strId := ctx.Param("user_id")
	userID := strId
	if !utils.IsUUID(userID) {
		zap.S().Debugw("Invalid user ID format for clearing cart", "id", strId)
		utils.ProblemJSON(ctx, http.StatusBadRequest, "Invalid User ID", "The provided user identification format is invalid", ctx.Request.URL.String())
		return
	}

	err := c.CartService.ClearCart(userID)
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
func (c *CartController) AddToCart(ctx *gin.Context) {
	strId := ctx.Param("user_id")
	userID := strId
	if !utils.IsUUID(userID) {
		zap.S().Debugw("Invalid user ID format for adding to cart", "id", strId)
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

	err := c.CartService.AddToCart(userID, &item)
	switch {
	case errors.Is(err, service.ErrProductNotFound):
		zap.S().Debugw("Product not found in catalog", "userID", userID, "productID", req.ProductID)
		utils.ProblemJSON(ctx, http.StatusNotFound, "Product Not Found",
			"The requested product does not exist in the catalog", ctx.Request.URL.String())
	case errors.Is(err, service.ErrCatalogUnavailable):
		zap.S().Warnw("Catalog service unavailable when adding to cart", "userID", userID, "productID", req.ProductID)
		utils.ProblemJSON(ctx, http.StatusServiceUnavailable, "Catalog Unavailable",
			"The catalog service is temporarily unavailable. Please try again later.", ctx.Request.URL.String())
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

// RemoveFromCart handles DELETE /cart/:user_id/items/:product_id.
// The :product_id path parameter now refers to the catalog product ID, since cart
// items are identified by their productId (one entry per product per user).
func (c *CartController) RemoveFromCart(ctx *gin.Context) {
	strId := ctx.Param("user_id")
	userID := strId
	if !utils.IsUUID(userID) {
		zap.S().Debugw("Invalid user ID format for removing from cart", "id", strId)
		utils.ProblemJSON(ctx, http.StatusBadRequest, "Invalid User ID", "The provided user identification format is invalid", ctx.Request.URL.String())
		return
	}

	productID := ctx.Param("cart_item_id")
	if !utils.IsUUID(productID) {
		zap.S().Debugw("Invalid product ID format for removing from cart", "id", productID)
		utils.ProblemJSON(ctx, http.StatusBadRequest, "Invalid Product ID", "The provided product identification format is invalid", ctx.Request.URL.String())
		return
	}

	err := c.CartService.RemoveFromCart(userID, productID)
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

	err := c.CartService.IncrementItem(userID, productID, req.Quantity)
	switch {
	case errors.Is(err, repository.ErrCartItemNotFound):
		zap.S().Debugw("Cart item not found for increment", "userID", userID, "productID", productID)
		utils.ProblemJSON(ctx, http.StatusNotFound, "Cart Item Not Found",
			"Could not find the requested product in this user's cart", ctx.Request.URL.String())
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

	err := c.CartService.DecrementItem(userID, productID, req.Quantity)
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
func (c *CartController) GetOwnCart(ctx *gin.Context) {
	userID, ok := utils.GetUserIDFromHeader(ctx)
	if !ok {
		return
	}

	cart, err := c.CartService.GetCartByUserID(userID)
	if err != nil {
		zap.S().Errorw("Failed to retrieve cart", "userID", userID, "error", err)
		utils.ProblemJSON(ctx, http.StatusInternalServerError, "Internal Server Error", "Could not retrieve the cart at this time", ctx.Request.URL.String())
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"data": cart})
}

func (c *CartController) ClearOwnCart(ctx *gin.Context) {
	userID, ok := utils.GetUserIDFromHeader(ctx)
	if !ok {
		return
	}

	err := c.CartService.ClearCart(userID)
	if err != nil {
		zap.S().Errorw("Failed to clear cart", "userID", userID, "error", err)
		utils.ProblemJSON(ctx, http.StatusInternalServerError, "Internal Server Error", "Could not clear the cart at this time", ctx.Request.URL.String())
		return
	}

	ctx.Status(http.StatusNoContent)
}

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

	err := c.CartService.AddToCart(userID, &item)
	switch {
	case errors.Is(err, service.ErrProductNotFound):
		zap.S().Debugw("Product not found in catalog", "userID", userID, "productID", req.ProductID)
		utils.ProblemJSON(ctx, http.StatusNotFound, "Product Not Found",
			"The requested product does not exist in the catalog", ctx.Request.URL.String())
	case errors.Is(err, service.ErrCatalogUnavailable):
		zap.S().Warnw("Catalog service unavailable when adding to cart", "userID", userID, "productID", req.ProductID)
		utils.ProblemJSON(ctx, http.StatusServiceUnavailable, "Catalog Unavailable",
			"The catalog service is temporarily unavailable. Please try again later.", ctx.Request.URL.String())
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

	err := c.CartService.RemoveFromCart(userID, productID)
	if err != nil {
		zap.S().Errorw("Failed to remove item from cart", "userID", userID, "productID", productID, "error", err)
		utils.ProblemJSON(ctx, http.StatusNotFound, "Cart Item Not Found",
			"Could not find the requested product in this user's cart", ctx.Request.URL.String())
		return
	}

	ctx.Status(http.StatusNoContent)
}

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

	err := c.CartService.IncrementItem(userID, productID, req.Quantity)
	switch {
	case errors.Is(err, repository.ErrCartItemNotFound):
		zap.S().Debugw("Cart item not found for increment", "userID", userID, "productID", productID)
		utils.ProblemJSON(ctx, http.StatusNotFound, "Cart Item Not Found",
			"Could not find the requested product in this user's cart", ctx.Request.URL.String())
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

	err := c.CartService.DecrementItem(userID, productID, req.Quantity)
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
