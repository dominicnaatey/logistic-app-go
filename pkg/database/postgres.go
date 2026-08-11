package database

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// NewPostgresDB creates a new PostgreSQL connection with PostGIS support
func NewPostgresDB(dsn string, isDev bool) (*gorm.DB, error) {
	// Set log mode based on environment
	logLevel := logger.Silent
	if isDev {
		logLevel = logger.Info
	}

	// Open connection
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Get underlying SQL DB for connection pool settings
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying SQL DB: %w", err)
	}

	// Set connection pool settings
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)

	// Enable PostGIS extension
	if err := enablePostGIS(db); err != nil {
		return nil, err
	}

	log.Println("✓ Connected to PostgreSQL with PostGIS")
	return db, nil
}

// enablePostGIS enables the PostGIS extension if not already enabled
func enablePostGIS(db *gorm.DB) error {
	result := db.Exec("CREATE EXTENSION IF NOT EXISTS postgis;")
	if result.Error != nil {
		return fmt.Errorf("failed to enable PostGIS: %w", result.Error)
	}

	// Verify PostGIS is installed
	var version string
	err := db.Raw("SELECT PostGIS_Version();").Scan(&version).Error
	if err != nil {
		return fmt.Errorf("PostGIS verification failed: %w", err)
	}

	log.Printf("✓ PostGIS version: %s\n", version)
	return nil
}

// Close closes the database connection
func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
