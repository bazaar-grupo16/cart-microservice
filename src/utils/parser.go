package utils

import (
	"net/http"
	"regexp"

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
// - user_id: UUID string
// - product_id: UUID string
//
// It writes a 400 problem+json response and returns false when either parameter
// is invalid, so callers can return immediately.
func ParseUserAndProduct(ctx *gin.Context) (userID string, productID string, ok bool) {
	strUID := ctx.Param("user_id")
	if !uuidRe.MatchString(strUID) {
		zap.S().Debugw("Invalid user ID format", "id", strUID)
		ProblemJSON(ctx, http.StatusBadRequest, "Invalid User ID", "The provided user identification format is invalid", ctx.Request.URL.String())
		return "", "", false
	}

	productID = ctx.Param("product_id")
	if !uuidRe.MatchString(productID) {
		zap.S().Debugw("Invalid product ID format", "id", productID)
		ProblemJSON(ctx, http.StatusBadRequest, "Invalid Product ID", "The provided product identification format is invalid", ctx.Request.URL.String())
		return "", "", false
	}

	return strUID, productID, true
}

func ParseUserIDHeaderAndProductID(ctx *gin.Context) (userID string, productID string, ok bool) {
	userID = ctx.GetHeader("X-User-ID")
	if userID == "" {
		zap.S().Debug("Missing X-User-ID header")
		ProblemJSON(ctx, http.StatusBadRequest, "Missing User ID", "The X-User-ID header is required", ctx.Request.URL.String())
		return "", "", false
	}

	if !IsUUID(userID) {
		zap.S().Debugw("Invalid user ID format in header", "id", userID)
		ProblemJSON(ctx, http.StatusBadRequest, "Invalid User ID", "The provided user identification format in X-User-ID header is invalid", ctx.Request.URL.String())
		return "", "", false
	}

	productID = ctx.Param("product_id")
	if !IsUUID(productID) {
		zap.S().Debugw("Invalid product ID format", "id", productID)
		ProblemJSON(ctx, http.StatusBadRequest, "Invalid Product ID", "The provided product identification format is invalid", ctx.Request.URL.String())
		return "", "", false
	}

	return userID, productID, true
}

func GetUserIDFromHeader(ctx *gin.Context) (string, bool) {
	userID := ctx.GetHeader("X-User-ID")
	if userID == "" {
		zap.S().Debug("Missing X-User-ID header")
		ProblemJSON(ctx, http.StatusBadRequest, "Missing User ID", "The X-User-ID header is required", ctx.Request.URL.String())
		return "", false
	}

	if !IsUUID(userID) {
		zap.S().Debugw("Invalid user ID format in header", "id", userID)
		ProblemJSON(ctx, http.StatusBadRequest, "Invalid User ID", "The provided user identification format in X-User-ID header is invalid", ctx.Request.URL.String())
		return "", false
	}

	return userID, true
}
