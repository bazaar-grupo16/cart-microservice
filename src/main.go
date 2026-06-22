package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/bazaar-grupo16/cart-microservice/src/config"
	"github.com/bazaar-grupo16/cart-microservice/src/controller"
	"github.com/bazaar-grupo16/cart-microservice/src/repository"
	"github.com/bazaar-grupo16/cart-microservice/src/router"
	"github.com/bazaar-grupo16/cart-microservice/src/service"
	"github.com/bazaar-grupo16/cart-microservice/src/telemetry"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

// @title Cart Microservice API
// @version 1.0
// @description API for managing shopping carts in the Bazaar e-commerce platform.
// @host localhost:8080
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	// Load environment variables from .env file (fallback to system env) first so OTEL vars are available
	if err := godotenv.Load("../.env"); err != nil {
		log.Println("No .env file found; using system environment variables")
	}

	// Initialize structured logger.
	logger := telemetry.NewJSONLogger()
	defer func() {
		if err := logger.Sync(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to sync logger: %v\n", err)
		}
	}()

	zap.ReplaceGlobals(logger)

	// Initialize TracerProvider
	ctx := context.Background()
	shutdown, err := telemetry.InitTracer(ctx)
	if err != nil {
		logger.Fatal("Failed to initialize tracer", zap.Error(err))
	}
	defer func() {
		_ = shutdown(ctx)
	}()

	// Connect to MongoDB (reads MONGO_URI from environment).
	config.ConnectDatabase()

	// Dependency injection — cart domain only.
	cartRepo := repository.NewCartRepository(config.GetCartCollection)
	catalogClient := service.NewCatalogClient(os.Getenv("CATALOG_SERVICE_URL"))
	cartService := service.NewCartService(cartRepo, catalogClient)
	cartController := controller.NewCartController(cartService)

	r := router.SetupRouter(cartController)

	addr := fmt.Sprintf(":%s", os.Getenv("PORT"))
	zap.S().Infow("Starting cart microservice", "addr", addr)
	if err := r.Run(addr); err != nil {
		zap.S().Fatalw("Server failed to start", "error", err)
	}
}
