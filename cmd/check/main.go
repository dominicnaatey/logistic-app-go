package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"logistic-app-go/pkg/cache"
	"logistic-app-go/pkg/config"
	"logistic-app-go/pkg/database"
	"logistic-app-go/pkg/storage"
)

func main() {
	fmt.Println("🔍 Checking external service connections...\n")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("❌ Failed to load configuration: %v\n", err)
	}
	fmt.Println("✓ Configuration loaded")

	// Check PostgreSQL + PostGIS
	fmt.Println("\n📊 Checking PostgreSQL + PostGIS...")
	if err := checkPostgreSQL(cfg.Database.URL); err != nil {
		log.Fatalf("❌ PostgreSQL check failed: %v\n", err)
	}

	// Check Upstash Redis
	fmt.Println("\n📦 Checking Upstash Redis...")
	if err := checkUpstashRedis(cfg.Redis.UpstashRestURL, cfg.Redis.UpstashRestToken); err != nil {
		log.Fatalf("❌ Upstash Redis check failed: %v\n", err)
	}

	// Check Cloudflare R2
	fmt.Println("\n☁️  Checking Cloudflare R2...")
	if err := checkR2(cfg.Storage); err != nil {
		// R2 is optional for Phase 0, just warn
		fmt.Printf("⚠️  R2 check skipped: %v\n", err)
		fmt.Println("  (R2 will be needed in Phase 1 for file uploads)")
	}

	// Summary
	fmt.Println("\n✅ All required connections successful!")
	fmt.Println("\nYou can now start the server with: go run cmd/server/main.go")
}

func checkPostgreSQL(dsn string) error {
	db, err := database.NewPostgresDB(dsn, false)
	if err != nil {
		return err
	}
	defer database.Close(db)

	// Get PostGIS version
	var version string
	if err := db.Raw("SELECT PostGIS_Version();").Scan(&version).Error; err != nil {
		return fmt.Errorf("PostGIS query failed: %w", err)
	}

	fmt.Printf("  ✓ PostgreSQL connected\n")
	fmt.Printf("  ✓ PostGIS version: %s\n", version)

	// Test a simple spatial query
	var result float64
	query := "SELECT ST_Distance(ST_MakePoint(0, 0)::geography, ST_MakePoint(1, 1)::geography);"
	if err := db.Raw(query).Scan(&result).Error; err != nil {
		return fmt.Errorf("spatial query test failed: %w", err)
	}
	fmt.Printf("  ✓ Spatial queries working (test distance: %.2f meters)\n", result)

	return nil
}

func checkUpstashRedis(restURL, token string) error {
	upstashClient, err := cache.NewUpstashClient(restURL, token)
	if err != nil {
		return err
	}
	defer upstashClient.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test SET
	testKey := "health_check_test"
	testValue := fmt.Sprintf("test_%d", time.Now().Unix())
	if err := upstashClient.Set(ctx, testKey, testValue, 10*time.Second); err != nil {
		return fmt.Errorf("SET command failed: %w", err)
	}
	fmt.Println("  ✓ Upstash SET command working")

	// Test GET
	retrieved, err := upstashClient.Get(ctx, testKey)
	if err != nil {
		return fmt.Errorf("GET command failed: %w", err)
	}
	if retrieved != testValue {
		return fmt.Errorf("GET returned wrong value: expected %s, got %s", testValue, retrieved)
	}
	fmt.Println("  ✓ Upstash GET command working")

	// Test DEL
	if err := upstashClient.Del(ctx, testKey); err != nil {
		return fmt.Errorf("DEL command failed: %w", err)
	}
	fmt.Println("  ✓ Upstash DEL command working")

	return nil
}

func checkR2(storageConfig config.StorageConfig) error {
	// Check if R2 is configured
	if storageConfig.R2AccountID == "" || storageConfig.R2AccessKeyID == "" || storageConfig.R2SecretAccessKey == "" {
		return fmt.Errorf("R2 credentials not configured (optional for Phase 0)")
	}

	if storageConfig.R2BucketName == "" {
		return fmt.Errorf("R2 bucket name not configured")
	}

	// Create R2 client
	r2Client, err := storage.NewR2Client(storage.R2Config{
		AccountID:   storageConfig.R2AccountID,
		AccessKeyID: storageConfig.R2AccessKeyID,
		SecretKey:   storageConfig.R2SecretAccessKey,
		BucketName:  storageConfig.R2BucketName,
		PublicURL:   storageConfig.R2PublicURL,
	})
	if err != nil {
		return err
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := r2Client.TestConnection(ctx); err != nil {
		return fmt.Errorf("R2 connection test failed: %w", err)
	}

	fmt.Printf("  ✓ R2 bucket '%s' accessible\n", storageConfig.R2BucketName)
	fmt.Println("  ✓ R2 connection successful")

	return nil
}
