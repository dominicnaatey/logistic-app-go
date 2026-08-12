package database

import (
	"fmt"
	"log"

	"gorm.io/gorm"

	"logistic-app-go/internal/user"
)

// Migrate runs GORM auto-migration for all registered models.
// It creates or alters tables to match the current model definitions.
// Safe to run on every startup — GORM only adds missing columns/indexes,
// it never drops existing ones.
func Migrate(db *gorm.DB) error {
	log.Println("Running database migrations...")

	err := db.AutoMigrate(
		// Phase 1: Authentication & User Management
		&user.User{},
	)
	if err != nil {
		return fmt.Errorf("auto-migration failed: %w", err)
	}

	log.Println("✓ Database migrations complete")
	return nil
}
