package database

import (
	"fmt"
	"time"

	"rosetta/config"
	"rosetta/models"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Init(cfg *config.DatabaseConfig) error {
	logLevel := logger.Info
	if cfg.Host == "" {
		logLevel = logger.Silent
	}

	var dialector gorm.Dialector
	switch cfg.Type {
	case "sqlite", "sqlite3":
		dialector = sqlite.Open(cfg.DSN())
	default:
		dialector = mysql.Open(cfg.DSN())
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}

	if cfg.Type != "sqlite" && cfg.Type != "sqlite3" {
		sqlDB, err := db.DB()
		if err != nil {
			return fmt.Errorf("get underlying DB: %w", err)
		}
		sqlDB.SetMaxIdleConns(10)
		sqlDB.SetMaxOpenConns(100)
		sqlDB.SetConnMaxLifetime(time.Hour)
	}

	DB = db

	if err := autoMigrate(db); err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}

	if err := seed(db); err != nil {
		return fmt.Errorf("seed: %w", err)
	}

	return nil
}

func autoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.RbacUser{},
		&models.RbacRole{},
		&models.RbacUserRole{},
		&models.RbacRoleModelPermission{},
		&models.DatasourceInstance{},
		&models.DatasourceSchema{},
		&models.DictDefinition{},
		&models.DictItem{},
		&models.LogicalModel{},
		&models.ModelColumn{},
		&models.ModelIndex{},
		&models.ModelForeignKey{},
		&models.ModelDeployment{},
		&models.AuditLog{},
	)
}
