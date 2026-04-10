package utils

import (
	"github.com/ValentinMarturet/is2-rojas/api/models"
	"github.com/gin-gonic/gin"
)

// ProblemJSON is a helper function to send error responses in the Problem Details format (RFC 7807) with the appropriate content type header.
func ProblemJSON(c *gin.Context, status int, title string, detail string, instance string) {
	c.Header("Content-Type", "application/problem+json")
	c.JSON(status, models.Problem{
		Type:     "about:blank",
		Title:    title,
		Status:   status,
		Detail:   detail,
		Instance: instance,
	})
}
