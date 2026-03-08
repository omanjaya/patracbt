package main

import (
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	pgmigrate "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/omanjaya/patra/config"
	"github.com/omanjaya/patra/migrations"
	"github.com/omanjaya/patra/pkg/logger"
	"gorm.io/gorm"
)

// runSQLMigrations runs versioned SQL migrations using golang-migrate.
// Safe to call every startup — only applies pending migrations.
func runSQLMigrations(db *gorm.DB, cfg *config.Config) {
	sourceDriver, err := iofs.New(migrations.FS, ".")
	if err != nil {
		logger.Log.Fatalf("Failed to create migration source: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		logger.Log.Fatalf("Failed to get sql.DB for migration: %v", err)
	}

	dbDriver, err := pgmigrate.WithInstance(sqlDB, &pgmigrate.Config{})
	if err != nil {
		logger.Log.Fatalf("Failed to create migration db driver: %v", err)
	}

	m, err := migrate.NewWithInstance("iofs", sourceDriver, cfg.DB.Database, dbDriver)
	if err != nil {
		logger.Log.Fatalf("Failed to init migration: %v", err)
	}

	version, dirty, _ := m.Version()
	logger.Log.Infof("Current migration version: %d (dirty: %v)", version, dirty)

	if dirty {
		logger.Log.Warnf("Database is in dirty state at version %d, forcing version...", version)
		if err := m.Force(int(version)); err != nil {
			logger.Log.Fatalf("Failed to force migration version: %v", err)
		}
	}

	err = m.Up()
	if err == migrate.ErrNoChange {
		logger.Log.Info("Database schema is up to date")
		return
	}
	if err != nil {
		logger.Log.Fatalf("Migration failed: %v", err)
	}

	newVersion, _, _ := m.Version()
	logger.Log.Infof("Migrations applied successfully: %d → %d", version, newVersion)
	fmt.Printf("  ✓ Database migrated from v%d to v%d\n", version, newVersion)
}
