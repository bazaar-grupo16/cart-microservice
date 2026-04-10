package controller

import (
	"net/http"
	"strconv"

	"github.com/ValentinMarturet/is2-rojas/src/models"
	"github.com/ValentinMarturet/is2-rojas/src/service"
	"github.com/ValentinMarturet/is2-rojas/src/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ProductController handles HTTP requests related to products
type ProductController struct {
	Service *service.ProductService
}

// NewProductController creates a new instance of ProductController
func NewProductController(service *service.ProductService) *ProductController {
	return &ProductController{
		Service: service,
	}
}

// CreateProduct handles POST /products to create a new product
func (c *ProductController) CreateProduct(ctx *gin.Context) {
	var req models.CreateProductRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		zap.S().Errorw("Invalid request payload for creating product", "error", err, "url", ctx.Request.URL.String())
		utils.ProblemJSON(ctx, http.StatusBadRequest, "Invalid request payload", "The provided product information is malformed or invalid", ctx.Request.URL.String())
		return
	}

	p := models.Product{
		SellerID:    req.SellerID,
		Title:       req.Title,
		Description: req.Description,
		Price:       req.Price,
	}

	if err := c.Service.CreateProduct(&p); err != nil {
		zap.S().Errorw("Failed to create product", "error", err, "product", p)
		utils.ProblemJSON(ctx, http.StatusBadRequest, "Failed to create product", err.Error(), ctx.Request.URL.String())
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{"data": p})
}

// GetAllProducts handles GET /products to retrieve all products
func (c *ProductController) GetAllProducts(ctx *gin.Context) {
	products, err := c.Service.GetAllProducts()
	if err != nil {
		zap.S().Errorw("Failed to retrieve products", "error", err)
		utils.ProblemJSON(ctx, http.StatusInternalServerError, "Internal Server Error", "Could not retrieve the list of products at this time", ctx.Request.URL.String())
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"data": products})
}

// GetProductByID handles GET /products/:id to retrieve a product by its ID
func (c *ProductController) GetProductByID(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		zap.S().Debugw("Invalid product ID format", "id", idStr, "error", err)
		utils.ProblemJSON(ctx, http.StatusBadRequest, "Invalid ID", "The provided identification format is invalid", ctx.Request.URL.String())
		return
	}

	product, err := c.Service.GetProductByID(id)
	if err != nil {
		zap.S().Errorw("Product not found", "id", id, "error", err)
		utils.ProblemJSON(ctx, http.StatusNotFound, "Product not found", "Could not find a product with the given identification", ctx.Request.URL.String())
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"data": product})
}

// UpdateProduct handles PUT /products/:id to update an existing product
func (c *ProductController) UpdateProduct(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		zap.S().Debugw("Invalid product ID format for update", "id", idStr, "error", err)
		utils.ProblemJSON(ctx, http.StatusBadRequest, "Invalid ID", "The provided identification format is invalid", ctx.Request.URL.String())
		return
	}

	var req models.UpdateProductRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		zap.S().Errorw("Invalid request payload for updating product", "error", err, "id", id)
		utils.ProblemJSON(ctx, http.StatusBadRequest, "Invalid request payload", "The updated product information is malformed or invalid", ctx.Request.URL.String())
		return
	}

	p := models.Product{
		ID:          id,
		Title:       req.Title,
		Description: req.Description,
		Price:       req.Price,
	}

	if err := c.Service.UpdateProduct(&p); err != nil {
		zap.S().Errorw("Failed to update product", "error", err, "id", id)
		if err.Error() == "product not found" {
			utils.ProblemJSON(ctx, http.StatusNotFound, "Product not found", "Could not find a product with the given identification", ctx.Request.URL.String())
		} else {
			utils.ProblemJSON(ctx, http.StatusBadRequest, "Update Failed", err.Error(), ctx.Request.URL.String())
		}
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"data": p})
}

// DeleteProduct handles DELETE /products/:id to delete a product by its ID
func (c *ProductController) DeleteProduct(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		zap.S().Debugw("Invalid product ID format for deletion", "id", idStr, "error", err)
		utils.ProblemJSON(ctx, http.StatusBadRequest, "Invalid ID", "The provided identification format is invalid", ctx.Request.URL.String())
		return
	}

	if err := c.Service.DeleteProduct(id); err != nil {
		zap.S().Errorw("Failed to delete product", "id", id, "error", err)
		utils.ProblemJSON(ctx, http.StatusNotFound, "Deletion Failed", "The requested product could not be found or deleted", ctx.Request.URL.String())
		return
	}

	ctx.Status(http.StatusNoContent)
}
