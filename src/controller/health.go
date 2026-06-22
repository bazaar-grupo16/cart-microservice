package controller

import (
	"context"
	"net/http"
	"time"

	"github.com/bazaar-grupo16/cart-microservice/src/config"
	// Required for swaggo to find models.Problem type
	_ "github.com/bazaar-grupo16/cart-microservice/src/models"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// LivezHandler reports that the process is alive.
// It performs only in-process checks (no DB, no external services),
// so Kubernetes can distinguish a crashed process from a temporarily unavailable one.
// @Summary Liveness probe
// @Description Returns HTTP 200 when the process is alive. No external checks.
// @Tags Health
// @Produce json
// @Success 200 {object} map[string]string
// @Router /livez [get]
func LivezHandler(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// ReadyzHandler reports that the service is ready to handle requests.
// It verifies connectivity to MongoDB via a lightweight Ping command.
// Returns 503 if the database is unreachable so Kubernetes stops routing traffic
// to this pod until the dependency recovers.
// @Summary Readiness probe
// @Description Attempts a lightweight database query. Returns HTTP 200 if successful, HTTP 503 otherwise.
// @Tags Health
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 503 {object} map[string]string
// @Router /readyz [get]
func ReadyzHandler(ctx *gin.Context) {
	pingCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	client := config.GetMongoClient()
	if client == nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unavailable",
			"detail": "database client not initialized",
		})
		return
	}

	if err := client.Ping(pingCtx, nil); err != nil {
		zap.S().Errorw("Readiness check failed: MongoDB unreachable", "error", err)
		ctx.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unavailable",
			"detail": "database connection failed",
		})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
}
