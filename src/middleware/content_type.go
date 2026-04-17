package middleware

import (
	"net/http"
	"strings"

	"github.com/bazaar-grupo16/cart-microservice/src/utils"
	"github.com/gin-gonic/gin"
)

// ContentTypeMiddleware ensures that requests with bodies use application/json
func ContentTypeMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		if method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch {
			contentType := c.GetHeader("Content-Type")
			if !strings.HasPrefix(contentType, "application/json") {
				utils.ProblemJSON(
					c,
					http.StatusBadRequest,
					"Invalid Content-Type",
					"Content-Type must be application/json",
					c.Request.URL.Path,
				)
				c.Abort()
				return
			}
		}
		c.Next()
	}
}
