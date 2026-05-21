package models

import "time"

type LogicalModel struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	DatabaseID   uint64    `gorm:"not null;index" json:"database_id"`
	SchemaID     uint64    `gorm:"not null;uniqueIndex:uk_schema_table" json:"schema_id"`
	TabName      string    `gorm:"column:table_name;size:256;not null;uniqueIndex:uk_schema_table" json:"table_name"`
	TableComment string    `gorm:"size:512" json:"table_comment"`
	TableStatus  string    `gorm:"size:32;not null;default:DRAFT" json:"table_status"`
	Source       string    `gorm:"size:32;not null;default:MANUAL" json:"source"`
	CreatedBy    *uint64   `json:"created_by"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedBy    *uint64   `json:"updated_by"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (LogicalModel) TableName() string {
	return "logical_model"
}

type ModelColumn struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	ModelID      uint64    `gorm:"not null;uniqueIndex:uk_model_column" json:"model_id"`
	Ordinal      int       `gorm:"not null" json:"ordinal"`
	ColumnName   string    `gorm:"size:256;not null;uniqueIndex:uk_model_column" json:"column_name"`
	LogicalType  string    `gorm:"size:64;not null" json:"logical_type"`
	TypeLength   *int      `json:"type_length"`
	TypeScale    *int      `json:"type_scale"`
	Nullable     bool      `gorm:"not null;default:false" json:"nullable"`
	DefaultValue string    `gorm:"size:512" json:"default_value"`
	Comment      string    `gorm:"size:512" json:"comment"`
	IsPrimaryKey bool      `gorm:"not null;default:false" json:"is_primary_key"`
	CreatedAt    time.Time `json:"created_at"`
}

func (ModelColumn) TableName() string {
	return "model_column"
}

type ModelIndex struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	ModelID   uint64    `gorm:"not null;uniqueIndex:uk_model_index" json:"model_id"`
	IndexName string    `gorm:"size:256;not null;uniqueIndex:uk_model_index" json:"index_name"`
	IndexType string    `gorm:"size:32;not null;default:NORMAL" json:"index_type"`
	Columns   string    `gorm:"type:text;not null" json:"columns"`
	CreatedAt time.Time `json:"created_at"`
}

func (ModelIndex) TableName() string {
	return "model_index"
}

type ModelForeignKey struct {
	ID            uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	ModelID       uint64    `gorm:"not null;uniqueIndex:uk_model_fk_name" json:"model_id"`
	FkName        string    `gorm:"size:256;not null;uniqueIndex:uk_model_fk_name" json:"fk_name"`
	ColumnName    string    `gorm:"size:256;not null" json:"column_name"`
	RefModelID    uint64    `gorm:"not null" json:"ref_model_id"`
	RefColumnName string    `gorm:"size:256;not null" json:"ref_column_name"`
	CreatedAt     time.Time `json:"created_at"`
}

func (ModelForeignKey) TableName() string {
	return "model_foreign_key"
}

type ModelDeployment struct {
	ID         uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ModelID    uint64     `gorm:"not null;uniqueIndex:uk_model_schema" json:"model_id"`
	SchemaID   uint64     `gorm:"not null;uniqueIndex:uk_model_schema" json:"schema_id"`
	Dialect    string     `gorm:"size:32;not null" json:"dialect"`
	DeployedDDL string    `gorm:"type:text" json:"deployed_ddl"`
	DeployedAt *time.Time `json:"deployed_at"`
	DeployedBy *uint64    `json:"deployed_by"`
	CreatedAt  time.Time  `json:"created_at"`
}

func (ModelDeployment) TableName() string {
	return "model_deployment"
}

type VirtualForeignKey struct {
	ID            uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	ModelID       uint64    `gorm:"not null;index:idx_vfk_model;uniqueIndex:uk_vfk_unique" json:"model_id"`
	ColumnName    string    `gorm:"size:256;not null;uniqueIndex:uk_vfk_unique" json:"column_name"`
	RefModelID    uint64    `gorm:"not null;uniqueIndex:uk_vfk_unique" json:"ref_model_id"`
	RefColumnName string    `gorm:"size:256;not null;uniqueIndex:uk_vfk_unique" json:"ref_column_name"`
	FkName        string    `gorm:"size:256" json:"fk_name"`
	CreatedAt     time.Time `json:"created_at"`
}

func (VirtualForeignKey) TableName() string {
	return "virtual_foreign_key"
}
