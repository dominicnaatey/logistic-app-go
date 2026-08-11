package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"logistic-app-go/pkg/cache"
	"logistic-app-go/pkg/config"
	"logistic-app-go/pkg/database"
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

	// Summary
	fmt.Println("\n✅ All connections successful!")
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
