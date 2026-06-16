package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/TheInfamousToTo/gold-tracker/backend/internal/api"
	"github.com/TheInfamousToTo/gold-tracker/backend/internal/repository"
)

func main() {
	// Load .env if it exists
	_ = godotenv.Load()

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	// Initialize repository
	repo, err := repository.NewPostgresRepository()
	if err != nil {
		log.Fatalf("Failed to initialize repository: %v", err)
	}
	defer repo.Close()

	// Initialize handler
	h := api.NewHandler(repo)

	r := gin.Default()

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Routes
	apiGroup := r.Group("/api")
	{
		apiGroup.GET("/health", h.Health)
		
		apiGroup.GET("/items", h.GetItems)
		apiGroup.POST("/items", h.CreateItem)
		apiGroup.PUT("/items/:id", h.UpdateItem)
		apiGroup.DELETE("/items/:id", h.DeleteItem)
		
		apiGroup.GET("/portfolio", h.GetPortfolio)
		
		apiGroup.GET("/prices", h.GetPrices)
		apiGroup.POST("/prices", h.CreatePrice)
		
		apiGroup.GET("/signals", h.GetSignals)
	}

	fmt.Printf("Gold Tracker Go API listening on port %s\n", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
