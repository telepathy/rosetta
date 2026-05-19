package models

import "time"

type DatasourceInstance struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Name      string    `gorm:"size:128;not null" json:"name"`
	Type      string    `gorm:"size:32;not null" json:"type"`
	Host      string    `gorm:"size:256;not null" json:"host"`
	Port      int       `gorm:"not null" json:"port"`
	Credential string   `gorm:"type:text;not null" json:"credential"`
	Status    string    `gorm:"size:32;not null;default:ACTIVE" json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (DatasourceInstance) TableName() string {
	return "datasource_instance"
}

type DatasourceSchema struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	InstanceID uint64    `gorm:"not null;uniqueIndex:uk_instance_schema" json:"instance_id"`
	SchemaName string    `gorm:"size:128;not null;uniqueIndex:uk_instance_schema" json:"schema_name"`
	Layer      string    `gorm:"size:32;not null" json:"layer"`
	CreatedAt  time.Time `json:"created_at"`
}

func (DatasourceSchema) TableName() string {
	return "datasource_schema"
}
