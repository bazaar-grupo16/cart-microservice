package router

import (
	"github.com/bazaar-grupo16/cart-microservice/src/controller"
	"github.com/bazaar-grupo16/cart-microservice/src/middleware"
	"github.com/gin-gonic/gin"
)

// SetupRouter initializes the Gin router with all cart routes and health probes.
func SetupRouter(cc *controller.CartController) *gin.Engine {
	r := gin.Default()

	// Content-Type validation middleware applies to POST/PUT/PATCH requests.
	r.Use(middleware.ContentTypeMiddleware())

	// Health probes — must respond even when DB is temporarily unavailable.
	// These are intentionally outside any versioned prefix for Kubernetes probe compatibility.
	r.GET("/livez", controller.LivezHandler)
	r.GET("/readyz", controller.ReadyzHandler)

	// Cart routes (scope of this microservice)
	r.GET("/cart", cc.GetOwnCart) // For authenticated users to get their own cart
	r.DELETE("/cart", cc.ClearOwnCart)
	r.POST("/cart/items", cc.AddToOwnCart)
	r.DELETE("/cart/items/:cart_item_id", cc.RemoveFromOwnCart)
	r.PUT("/cart/increment/:product_id", cc.IncrementOwnItem)
	r.PUT("/cart/decrement/:product_id", cc.DecrementOwnItem)

	// Not authenticated routes (for now these are for retrocompatibility and testing, but they may be removed in the future) (This routes should be protected by an API Gateway in production, but we keep them here for testing and retrocompatibility)

	r.GET("/cart/:user_id", cc.GetCart)
	r.DELETE("/cart/:user_id", cc.ClearCart)
	r.POST("/cart/:user_id/items", cc.AddToCart)
	r.DELETE("/cart/:user_id/items/:cart_item_id", cc.RemoveFromCart)
	r.PUT("/cart/:user_id/:product_id/increment", cc.IncrementItem)
	r.PUT("/cart/:user_id/:product_id/decrement", cc.DecrementItem)

	return r
}
