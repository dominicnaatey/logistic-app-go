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
	internalAuth "logistic-app-go/internal/auth"
	"logistic-app-go/internal/sms"
	"logistic-app-go/internal/user"
	"logistic-app-go/pkg/auth"
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

	// ── Database ────────────────────────────────────────────
	db, err := database.NewPostgresDB(cfg.Database.URL, cfg.Server.Env == "development")
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close(db)

	if err := database.Migrate(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// ── Cache (Upstash Redis) ────────────────────────────────
	redisClient, err := cache.NewUpstashClient(cfg.Redis.UpstashRestURL, cfg.Redis.UpstashRestToken)
	if err != nil {
		log.Fatalf("Failed to connect to Upstash Redis: %v", err)
	}
	defer redisClient.Close()

	// ── JWT ──────────────────────────────────────────────────
	jwtManager, err := auth.NewJWTManager(cfg.JWT.Secret, cfg.JWT.ExpiresIn)
	if err != nil {
		log.Fatalf("Failed to initialise JWT manager: %v", err)
	}

	// ── SMS ──────────────────────────────────────────────────
	smsSvc, err := sms.NewService(cfg.SMS.Username, cfg.SMS.APIKey, cfg.SMS.SenderID)
	if err != nil {
		log.Fatalf("Failed to initialise SMS service: %v", err)
	}

	// ── User (repository + service) ──────────────────────────
	userRepo := user.NewRepository(db)
	userSvc := user.NewService(userRepo)

	// ── Auth (OTP service + handlers) ────────────────────────
	otpSvc := internalAuth.NewOTPService(redisClient)
	authHandler := internalAuth.NewHandler(otpSvc, smsSvc, userSvc, jwtManager)

	// ── Router ───────────────────────────────────────────────
	router := setupRouter(cfg, jwtManager, authHandler)

	// ── HTTP server with graceful shutdown ───────────────────
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Server.Port),
		Handler: router,
	}

	go func() {
		log.Printf("Server starting on port %s (env: %s)", cfg.Server.Port, cfg.Server.Env)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

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

func setupRouter(
	cfg *config.Config,
	jwtManager *auth.JWTManager,
	authHandler *internalAuth.Handler,
) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(gin.Logger())

	// ── Public endpoints (no JWT) ────────────────────────────
	router.GET("/health", func(c *gin.Context) {
		response.Success(c, gin.H{"status": "healthy", "env": cfg.Server.Env})
	})

	v1 := router.Group("/api/v1")

	// Public auth routes
	authHandler.RegisterRoutes(v1, v1.Group("/", internalAuth.JWTMiddleware(jwtManager)))

	// ── Placeholder for future protected route groups ────────
	// Example usage:
	//   protected := v1.Group("/", internalAuth.JWTMiddleware(jwtManager))
	//   adminOnly  := protected.Group("/", internalAuth.RequireRoles(user.RoleAdmin))

	return router
}
