package models

import "time"

type LogicalDatabase struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string    `gorm:"size:128;not null;uniqueIndex" json:"name"`
	Description string    `gorm:"size:512" json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (LogicalDatabase) TableName() string { return "logical_database" }

type LogicalSchema struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	DatabaseID  uint64    `gorm:"not null;uniqueIndex:uk_db_schema_name" json:"database_id"`
	Name        string    `gorm:"size:128;not null;uniqueIndex:uk_db_schema_name" json:"name"`
	Description string    `gorm:"size:512" json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (LogicalSchema) TableName() string { return "logical_schema" }

type DatabaseInstanceMapping struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	DatabaseID uint64    `gorm:"not null;uniqueIndex:uk_db_inst" json:"database_id"`
	InstanceID uint64    `gorm:"not null;uniqueIndex:uk_db_inst" json:"instance_id"`
	CreatedAt  time.Time `json:"created_at"`
}

func (DatabaseInstanceMapping) TableName() string { return "database_instance_mapping" }
