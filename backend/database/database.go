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
	if err := db.AutoMigrate(
		&models.RbacUser{},
		&models.RbacRole{},
		&models.RbacUserRole{},
		&models.RbacRoleModelPermission{},
		&models.DatasourceInstance{},
		&models.DatasourceSchema{},
		&models.DictDefinition{},
		&models.DictItem{},
		&models.LogicalDatabase{},
		&models.LogicalSchema{},
		&models.DatabaseInstanceMapping{},
		&models.LogicalModel{},
		&models.ModelColumn{},
		&models.ModelIndex{},
		&models.ModelForeignKey{},
		&models.ModelDeployment{},
		&models.AuditLog{},
		&models.VirtualForeignKey{},
			&models.VirtualForeignKey{},
	); err != nil {
		return err
	}
	return migrateExistingData(db)
}

func migrateExistingData(db *gorm.DB) error {
	var modelCount int64
	db.Model(&models.LogicalModel{}).Count(&modelCount)
	if modelCount == 0 {
		return nil
	}

	var dbCount int64
	db.Model(&models.LogicalDatabase{}).Count(&dbCount)
	if dbCount > 0 {
		return nil
	}

	defDB := models.LogicalDatabase{Name: "默认数据库", Description: "迁移生成的默认逻辑数据库"}
	if err := db.Create(&defDB).Error; err != nil {
		return err
	}
	defSchema := models.LogicalSchema{DatabaseID: defDB.ID, Name: "default", Description: "迁移生成的默认Schema"}
	if err := db.Create(&defSchema).Error; err != nil {
		return err
	}
	db.Model(&models.LogicalModel{}).Where("database_id = 0").Update("database_id", defDB.ID)
	db.Model(&models.LogicalModel{}).Where("schema_id = 0").Update("schema_id", defSchema.ID)
	return nil
}
