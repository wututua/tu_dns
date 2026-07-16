package db

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"tudns/config"
	"tudns/models"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var global *gorm.DB

func Open(cfg *config.Config) (*gorm.DB, error) {
	driver := strings.ToLower(cfg.Database.Driver)
	if driver == "" {
		return nil, fmt.Errorf("database driver not configured")
	}

	var dialector gorm.Dialector
	switch driver {
	case "sqlite":
		path := cfg.Database.Path
		if path == "" {
			path = filepath.Join(cfg.App.DataDir, "tudns.db")
		}
		if !filepath.IsAbs(path) {
			path = filepath.Join(cfg.App.DataDir, filepath.Base(path))
		}
		dialector = sqlite.Open(path + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
	case "mysql":
		if cfg.Database.DSN == "" {
			return nil, fmt.Errorf("mysql dsn required")
		}
		dialector = mysql.Open(cfg.Database.DSN)
	case "postgres", "postgresql":
		if cfg.Database.DSN == "" {
			return nil, fmt.Errorf("postgres dsn required")
		}
		dialector = postgres.Open(cfg.Database.DSN)
	default:
		return nil, fmt.Errorf("unsupported driver: %s", driver)
	}

	logLevel := logger.Warn
	if cfg.App.Mode == "dev" {
		logLevel = logger.Info
	}

	gdb, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	sqlDB, err := gdb.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(time.Hour)
	sqlDB.SetConnMaxIdleTime(30 * time.Minute)

	if err := gdb.AutoMigrate(models.AllModels()...); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	global = gdb
	return gdb, nil
}

func Get() *gorm.DB {
	return global
}

func Set(gdb *gorm.DB) {
	global = gdb
}

func Ping(gdb *gorm.DB) error {
	sqlDB, err := gdb.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}

func TestConnection(driver, dsn, sqlitePath, dataDir string) error {
	cfg := config.Default()
	cfg.App.DataDir = dataDir
	cfg.Database = config.DatabaseConfig{
		Driver: driver,
		DSN:    dsn,
		Path:   sqlitePath,
	}
	gdb, err := Open(cfg)
	if err != nil {
		return err
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()
	return sqlDB.Ping()
}
