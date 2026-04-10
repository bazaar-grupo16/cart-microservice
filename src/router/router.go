package router

import (
	"github.com/ValentinMarturet/is2-rojas/src/controller"
	"github.com/ValentinMarturet/is2-rojas/src/middleware"
	"github.com/gin-gonic/gin"
)

// SetupRouter initializes the Gin router with all routes and handlers
func SetupRouter(pc *controller.ProductController, cc *controller.CartController) *gin.Engine {
	r := gin.Default()

	// Content-Type validation middleware
	r.Use(middleware.ContentTypeMiddleware())

	// /products routes
	r.POST("/products", pc.CreateProduct)
	r.GET("/products", pc.GetAllProducts)
	r.GET("/products/:id", pc.GetProductByID)
	r.PUT("/products/:id", pc.UpdateProduct)
	r.DELETE("/products/:id", pc.DeleteProduct)

	// /cart routes
	r.GET("/cart/:user_id", cc.GetCart)
	r.DELETE("/cart/:user_id", cc.ClearCart)
	r.POST("/cart/:user_id/items", cc.AddToCart)
	r.DELETE("/cart/:user_id/items/:cart_item_id", cc.RemoveFromCart)

	return r
}
