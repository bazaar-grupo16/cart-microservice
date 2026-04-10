package main

import (
	"fmt"
	"os"

	"github.com/ValentinMarturet/is2-rojas/api/config"
	"github.com/ValentinMarturet/is2-rojas/api/controller"
	"github.com/ValentinMarturet/is2-rojas/api/repository"
	"github.com/ValentinMarturet/is2-rojas/api/router"
	"github.com/ValentinMarturet/is2-rojas/api/service"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

func main() {
	// Initialize global Zap logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	zap.ReplaceGlobals(logger)

	// Load environment variables
	err := godotenv.Load("../.env")
	if err != nil {
		zap.S().Info("Error loading .env file. Using system environment variables.")
	}

	config.ConnectDatabase()
	config.RunMigrations()

	productRepo := repository.NewProductRepository(config.DB)
	productService := service.NewProductService(productRepo)
	productController := controller.NewProductController(productService)

	cartRepo := repository.NewCartRepository(config.DB)
	cartService := service.NewCartService(cartRepo, productRepo)
	cartController := controller.NewCartController(cartService)

	r := router.SetupRouter(productController, cartController)

	_ = r.Run(fmt.Sprintf(":%s", os.Getenv("PORT")))
}
