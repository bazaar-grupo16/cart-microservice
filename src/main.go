package main

import (
	"fmt"
	"os"

	"github.com/bazaar-grupo16/cart-microservice/src/config"
	"github.com/bazaar-grupo16/cart-microservice/src/controller"
	"github.com/bazaar-grupo16/cart-microservice/src/repository"
	"github.com/bazaar-grupo16/cart-microservice/src/router"
	"github.com/bazaar-grupo16/cart-microservice/src/service"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

func main() {
	// Initialize structured logger.
	logger, _ := zap.NewProduction()
	defer func() {
		if err := logger.Sync(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to sync logger: %v\n", err)
		}
	}()

	zap.ReplaceGlobals(logger)

	// Load environment variables from .env file (fallback to system env).
	if err := godotenv.Load("../.env"); err != nil {
		zap.S().Info("No .env file found; using system environment variables")
	}

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
