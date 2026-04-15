package utils

import (
	"net/http"
	"regexp"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var uuidRe = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func IsUUID(s string) bool {
	return uuidRe.MatchString(s)
}

// ParseUserAndProduct is a shared helper that extracts and validates the :user_id
// and :product_id path parameters.
//
// - user_id: positive integer
// - product_id: UUID string
//
// It writes a 400 problem+json response and returns false when either parameter
// is invalid, so callers can return immediately.
func ParseUserAndProduct(ctx *gin.Context) (userID int, productID string, ok bool) {
	strUID := ctx.Param("user_id")
	userID, err := strconv.Atoi(strUID)
	if err != nil || userID <= 0 {
		zap.S().Debugw("Invalid user ID format", "id", strUID, "error", err)
		ProblemJSON(ctx, http.StatusBadRequest, "Invalid User ID", "The provided user identification format is invalid", ctx.Request.URL.String())
		return 0, "", false
	}

	productID = ctx.Param("product_id")
	if !uuidRe.MatchString(productID) {
		zap.S().Debugw("Invalid product ID format", "id", productID)
		ProblemJSON(ctx, http.StatusBadRequest, "Invalid Product ID", "The provided product identification format is invalid", ctx.Request.URL.String())
		return 0, "", false
	}

	return userID, productID, true
}
