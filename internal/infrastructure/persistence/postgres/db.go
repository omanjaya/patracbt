package postgres

import (
	"fmt"
	"time"

	"github.com/omanjaya/patra/config"
	"github.com/omanjaya/patra/pkg/logger"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func NewDB(cfg *config.Config) *gorm.DB {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable TimeZone=UTC",
		cfg.DB.Host, cfg.DB.Port, cfg.DB.Username, cfg.DB.Password, cfg.DB.Database,
	)

	logLevel := gormlogger.Warn
	if cfg.App.Env == "development" {
		logLevel = gormlogger.Info
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:                 gormlogger.Default.LogMode(logLevel),
		SkipDefaultTransaction: true,
		PrepareStmt:            true,
	})
	if err != nil {
		logger.Log.Fatalf("Gagal koneksi ke PostgreSQL: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		logger.Log.Fatalf("Gagal mendapatkan sql.DB: %v", err)
	}

	sqlDB.SetMaxOpenConns(cfg.DB.MaxOpenConn)
	sqlDB.SetMaxIdleConns(cfg.DB.MaxIdleConn)
	sqlDB.SetConnMaxLifetime(time.Hour)
	sqlDB.SetConnMaxIdleTime(30 * time.Minute)

	logger.Log.Info("PostgreSQL connected")
	return db
}
