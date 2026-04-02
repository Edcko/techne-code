package db

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// Open opens (or creates) the SQLite database at the given path.
// It configures the database with WAL mode, busy timeout, and foreign key constraints.
// Parent directories are created if they don't exist.
// All models are auto-migrated on Open.
func Open(dbPath string) (*gorm.DB, error) {
	// Ensure parent directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database with SQLite driver
	dialector := sqlite.Open(dbPath)

	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure SQLite pragmas for better performance and reliability
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying database: %w", err)
	}

	// Set connection pool settings for SQLite
	sqlDB.SetMaxOpenConns(1) // SQLite works best with single connection

	// Configure SQLite pragmas
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA busy_timeout=5000",
		"PRAGMA foreign_keys=ON",
	}

	for _, pragma := range pragmas {
		if _, err := sqlDB.Exec(pragma); err != nil {
			return nil, fmt.Errorf("failed to set pragma '%s': %w", pragma, err)
		}
	}

	// Auto-migrate all models
	if err := db.AutoMigrate(
		&SessionModel{},
		&MessageModel{},
		&ReadFileModel{},
		&PermissionGrantModel{},
	); err != nil {
		return nil, fmt.Errorf("failed to auto-migrate: %w", err)
	}

	return db, nil
}
