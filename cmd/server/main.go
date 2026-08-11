package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"logistic-app-go/pkg/cache"
	"logistic-app-go/pkg/config"
	"logistic-app-go/pkg/database"
	"logistic-app-go/pkg/response"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Set Gin mode
	if cfg.Server.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize database
	db, err := database.NewPostgresDB(cfg.Database.URL, cfg.Server.Env == "development")
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close(db)

	// Initialize Upstash Redis (REST API)
	upstashClient, err := cache.NewUpstashClient(cfg.Redis.UpstashRestURL, cfg.Redis.UpstashRestToken)
	if err != nil {
		log.Fatalf("Failed to connect to Upstash Redis: %v", err)
	}
	defer upstashClient.Close()

	// Initialize router
	router := setupRouter(cfg, db, upstashClient)

	// Start server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Server.Port),
		Handler: router,
	}

	// Graceful shutdown
	go func() {
		log.Printf("Server starting on port %s (env: %s)", cfg.Server.Port, cfg.Server.Env)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited gracefully")
}

func setupRouter(cfg *config.Config, db interface{}, upstash *cache.UpstashClient) *gin.Engine {
	router := gin.New()

	// Global middleware
	router.Use(gin.Recovery())
	router.Use(gin.Logger())

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		response.Success(c, gin.H{
			"status": "healthy",
			"env":    cfg.Server.Env,
		})
	})

	// API v1 group
	v1 := router.Group("/api/v1")
	{
		v1.GET("/", func(c *gin.Context) {
			response.Success(c, gin.H{
				"message": "Cross-Border Trucking Logistics API",
				"version": "1.0.0",
			})
		})
	}

	return router
}
