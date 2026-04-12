package controller

import (
	"context"
	"net/http"
	"time"

	"github.com/bazaar-grupo16/cart-microservice/src/config"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// LivezHandler reports that the process is alive.
// It performs only in-process checks (no DB, no external services),
// so Kubernetes can distinguish a crashed process from a temporarily unavailable one.
func LivezHandler(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// ReadyzHandler reports that the service is ready to handle requests.
// It verifies connectivity to MongoDB via a lightweight Ping command.
// Returns 503 if the database is unreachable so Kubernetes stops routing traffic
// to this pod until the dependency recovers.
func ReadyzHandler(ctx *gin.Context) {
	pingCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := config.MongoClient.Ping(pingCtx, nil); err != nil {
		zap.S().Errorw("Readiness check failed: MongoDB unreachable", "error", err)
		ctx.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unavailable",
			"detail": "database connection failed",
		})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
}
