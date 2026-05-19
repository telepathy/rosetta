package models

import "time"

type RbacUser struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Username    string    `gorm:"uniqueIndex;size:128;not null" json:"username"`
	Password    string    `gorm:"size:256;not null" json:"-"`
	DisplayName string    `gorm:"size:128;not null" json:"display_name"`
	Email       string    `gorm:"size:256" json:"email"`
	Status      string    `gorm:"size:32;not null;default:ACTIVE" json:"status"`
	LastLoginAt *time.Time `json:"last_login_at"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	Roles []RbacRole `gorm:"many2many:rbac_user_role;joinForeignKey:UserID;joinReferences:RoleID" json:"roles,omitempty"`
}

func (RbacUser) TableName() string {
	return "rbac_user"
}
