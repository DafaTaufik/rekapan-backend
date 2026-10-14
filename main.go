package main

import (
	"os"
	"rekap-backend/config"
	"rekap-backend/handler"
	"rekap-backend/middleware"

	"github.com/gin-gonic/gin"
)

func main() {
	// Connect to database
	config.ConnectDatabase()

	// Initialize Gin
	r := gin.Default()

	// Health check
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
			"status":  "Server is running!",
		})
	})

	// Public routes - no token required
	authRoutes := r.Group("/api/auth")
	{
		authRoutes.POST("/register", handler.Register)
		authRoutes.POST("/login", handler.Login)
		authRoutes.POST("/refresh", handler.RefreshToken)
	}

	// Protected routes - JWT token required
	api := r.Group("/api", middleware.AuthMiddleware())
	{
		// Transactions
		api.GET("/transactions", handler.GetTransactions)
		api.PATCH("/transactions/:id/toggle-payment", handler.TogglePaymentStatus)

		// Summary
		api.GET("/summary/daily", handler.GetDailySummary)
		api.GET("/summary/range", handler.GetRangeSummary)

		// Branches
		api.GET("/branches", handler.GetBranches)
	}

	// Get port from environment variable
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	r.Run(":" + port)
}