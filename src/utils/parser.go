package utils

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// parseUserAndProduct is a shared helper that extracts and validates the :user_id
// and :product_id path parameters. It writes a 400 problem+json response and returns
// false when either parameter is invalid, so callers can return immediately.
func ParseUserAndProduct(ctx *gin.Context) (userID int, productID int, ok bool) {
	strUID := ctx.Param("user_id")
	userID, err := strconv.Atoi(strUID)
	if err != nil || userID <= 0 {
		zap.S().Debugw("Invalid user ID format", "id", strUID, "error", err)
		ProblemJSON(ctx, http.StatusBadRequest, "Invalid User ID", "The provided user identification format is invalid", ctx.Request.URL.String())
		return 0, 0, false
	}

	strPID := ctx.Param("product_id")
	productID, err = strconv.Atoi(strPID)
	if err != nil || productID <= 0 {
		zap.S().Debugw("Invalid product ID format", "id", strPID, "error", err)
		ProblemJSON(ctx, http.StatusBadRequest, "Invalid Product ID", "The provided product identification format is invalid", ctx.Request.URL.String())
		return 0, 0, false
	}

	return userID, productID, true
}
