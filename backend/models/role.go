package models

import "time"

type RbacRole struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	RoleName    string    `gorm:"size:128;not null" json:"role_name"`
	RoleCode    string    `gorm:"uniqueIndex;size:64;not null" json:"role_code"`
	Description string    `gorm:"size:512" json:"description"`
	IsSystem    bool      `gorm:"not null;default:false" json:"is_system"`
	CreatedAt   time.Time `json:"created_at"`
}

func (RbacRole) TableName() string {
	return "rbac_role"
}

type RbacUserRole struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    uint64    `gorm:"not null;uniqueIndex:uk_user_role" json:"user_id"`
	RoleID    uint64    `gorm:"not null;uniqueIndex:uk_user_role" json:"role_id"`
	CreatedAt time.Time `json:"created_at"`
}

func (RbacUserRole) TableName() string {
	return "rbac_user_role"
}

type RbacRoleModelPermission struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	RoleID     uint64    `gorm:"not null;uniqueIndex:uk_role_model_perm;index:idx_role_id" json:"role_id"`
	ModelID    uint64    `gorm:"not null;uniqueIndex:uk_role_model_perm;index:idx_model_id" json:"model_id"`
	Permission string    `gorm:"size:32;not null;uniqueIndex:uk_role_model_perm" json:"permission"`
	CreatedAt  time.Time `json:"created_at"`
}

func (RbacRoleModelPermission) TableName() string {
	return "rbac_role_model_permission"
}
