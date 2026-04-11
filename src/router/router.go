package router

import (
	"github.com/ValentinMarturet/is2-rojas/src/controller"
	"github.com/ValentinMarturet/is2-rojas/src/middleware"
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
	r.GET("/cart/:user_id", cc.GetCart)
	r.DELETE("/cart/:user_id", cc.ClearCart)
	r.POST("/cart/:user_id/items", cc.AddToCart)
	r.DELETE("/cart/:user_id/items/:cart_item_id", cc.RemoveFromCart)

	return r
}
