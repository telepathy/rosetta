package database

import (
	"rosetta/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func seed(db *gorm.DB) error {
	if err := seedRoles(db); err != nil {
		return err
	}
	if err := seedAdminUser(db); err != nil {
		return err
	}
	return nil
}

func seedRoles(db *gorm.DB) error {
	roles := []models.RbacRole{
		{RoleName: "超级管理员", RoleCode: "SUPER_ADMIN", Description: "系统级权限，管理用户、角色、实例；所有模型可读写", IsSystem: true},
		{RoleName: "数据治理管理员", RoleCode: "GOVERNANCE_ADMIN", Description: "管理字典、数据标准；所有模型可读", IsSystem: true},
		{RoleName: "数据开发", RoleCode: "DATA_DEVELOPER", Description: "被授权模型的读写权限", IsSystem: true},
		{RoleName: "数据分析", RoleCode: "DATA_ANALYST", Description: "被授权模型的只读权限", IsSystem: true},
	}

	for _, role := range roles {
		var existing models.RbacRole
		result := db.Where("role_code = ?", role.RoleCode).First(&existing)
		if result.Error == gorm.ErrRecordNotFound {
			if err := db.Create(&role).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func seedAdminUser(db *gorm.DB) error {
	var existing models.RbacUser
	result := db.Where("username = ?", "admin").First(&existing)
	if result.Error != gorm.ErrRecordNotFound {
		return nil
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	adminUser := models.RbacUser{
		Username:    "admin",
		Password:    string(hashedPassword),
		DisplayName: "系统管理员",
		Status:      "ACTIVE",
	}

	if err := db.Create(&adminUser).Error; err != nil {
		return err
	}

	var superAdminRole models.RbacRole
	if err := db.Where("role_code = ?", "SUPER_ADMIN").First(&superAdminRole).Error; err != nil {
		return err
	}

	userRole := models.RbacUserRole{
		UserID: adminUser.ID,
		RoleID: superAdminRole.ID,
	}
	return db.Create(&userRole).Error
}
