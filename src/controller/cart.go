package controller

import (
	"net/http"
	"strconv"

	"github.com/ValentinMarturet/is2-rojas/src/models"
	"github.com/ValentinMarturet/is2-rojas/src/service"
	"github.com/ValentinMarturet/is2-rojas/src/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// CartController handles cart-related HTTP requests
type CartController struct {
	CartService *service.CartService
}

// NewCartController creates a new instance of CartController
func NewCartController(cartService *service.CartService) *CartController {
	return &CartController{
		CartService: cartService,
	}
}

// GetCart handles GET /cart/:user_id
func (c *CartController) GetCart(ctx *gin.Context) {
	strId := ctx.Param("user_id")
	userID, err := strconv.Atoi(strId)
	if err != nil || userID <= 0 {
		zap.S().Debugw("Invalid user ID format for getting cart", "id", strId, "error", err)
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

// ClearCart handles DELETE /cart/:user_id
func (c *CartController) ClearCart(ctx *gin.Context) {
	strId := ctx.Param("user_id")
	userID, err := strconv.Atoi(strId)
	if err != nil || userID <= 0 {
		zap.S().Debugw("Invalid user ID format for clearing cart", "id", strId, "error", err)
		utils.ProblemJSON(ctx, http.StatusBadRequest, "Invalid User ID", "The provided user identification format is invalid", ctx.Request.URL.String())
		return
	}
 
	err = c.CartService.ClearCart(userID)
	if err != nil {
		zap.S().Errorw("Failed to clear cart", "userID", userID, "error", err)
		utils.ProblemJSON(ctx, http.StatusInternalServerError, "Internal Server Error", "Could not clear the cart at this time", ctx.Request.URL.String())
		return
	}
 
	ctx.Status(http.StatusNoContent)
}

// AddToCart handles POST /cart/:user_id/items
func (c *CartController) AddToCart(ctx *gin.Context) {
	strId := ctx.Param("user_id")
	userID, err := strconv.Atoi(strId)
	if err != nil || userID <= 0 {
		zap.S().Debugw("Invalid user ID format for adding to cart", "id", strId, "error", err)
		utils.ProblemJSON(ctx, http.StatusBadRequest, "Invalid User ID", "The provided user identification format is invalid", ctx.Request.URL.String())
		return
	}
 
	var req models.AddCartRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		zap.S().Errorw("Invalid request payload for adding to cart", "error", err, "userID", userID)
		utils.ProblemJSON(ctx, http.StatusBadRequest, "Invalid request payload", "The provided item information is malformed or invalid", ctx.Request.URL.String())
		return
	}

	item := models.CartItem{
		ProductID: req.ProductID,
	}

	err = c.CartService.AddToCart(userID, &item)
	if err != nil {
		zap.S().Errorw("Failed to add item to cart", "userID", userID, "productID", item.ProductID, "error", err)
		utils.ProblemJSON(ctx, http.StatusNotFound, "Operation Failed", "Could not add the item to the cart. Ensure the product exists.", ctx.Request.URL.String())
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{"data": item})
}

// RemoveFromCart handles DELETE /cart/:user_id/items/:cart_item_id
func (c *CartController) RemoveFromCart(ctx *gin.Context) {
	strId := ctx.Param("user_id")
	userID, err := strconv.Atoi(strId)
	if err != nil || userID <= 0 {
		zap.S().Debugw("Invalid user ID format for removing from cart", "id", strId, "error", err)
		utils.ProblemJSON(ctx, http.StatusBadRequest, "Invalid User ID", "The provided user identification format is invalid", ctx.Request.URL.String())
		return
	}
 
	strItemId := ctx.Param("cart_item_id")
	cartItemID, err := strconv.Atoi(strItemId)
	if err != nil || cartItemID <= 0 {
		zap.S().Debugw("Invalid cart item ID format", "id", strItemId, "error", err)
		utils.ProblemJSON(ctx, http.StatusBadRequest, "Invalid Item ID", "The provided item identification format is invalid", ctx.Request.URL.String())
		return
	}
 
	err = c.CartService.RemoveFromCart(userID, cartItemID)
	if err != nil {
		zap.S().Errorw("Failed to remove item from cart", "userID", userID, "cartItemID", cartItemID, "error", err)
		utils.ProblemJSON(ctx, http.StatusNotFound, "Removal Failed", "Could not find or remove the requested item from the cart", ctx.Request.URL.String())
		return
	}
 
	ctx.Status(http.StatusNoContent)
}
