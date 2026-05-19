package middleware

import (
	"fmt"
	"rosetta/models"
	"rosetta/utils"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	ContextUserID   = "user_id"
	ContextUsername = "username"
	ContextRoles    = "roles"
)

func AuthRequired(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			utils.Unauthorized(c, "未提供认证令牌")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			utils.Unauthorized(c, "认证格式错误，需要 Bearer Token")
			c.Abort()
			return
		}

		claims, err := utils.ParseToken(parts[1], secret)
		if err != nil {
			utils.Unauthorized(c, "认证令牌无效或已过期")
			c.Abort()
			return
		}

		c.Set(ContextUserID, claims.UserID)
		c.Set(ContextUsername, claims.Username)
		c.Next()
	}
}

func RequirePermission(db *gorm.DB, permission string, modelIDParam string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get(ContextUserID)
		if !exists {
			utils.Unauthorized(c, "未登录")
			c.Abort()
			return
		}

		var roles []models.RbacRole
		if err := db.Table("rbac_role").
			Joins("JOIN rbac_user_role ON rbac_user_role.role_id = rbac_role.id").
			Where("rbac_user_role.user_id = ?", userID).Find(&roles).Error; err != nil {
			utils.InternalError(c, "权限查询失败")
			c.Abort()
			return
		}

		for _, role := range roles {
			if role.RoleCode == "SUPER_ADMIN" {
				c.Set(ContextRoles, roleCodes(roles))
				c.Next()
				return
			}
		}

		modelIDStr := c.Param(modelIDParam)
		if modelIDStr == "" {
			c.Next()
			return
		}

		modelID, err := parseUint64(modelIDStr)
		if err != nil {
			utils.BadRequest(c, "无效的模型ID")
			c.Abort()
			return
		}

		if hasModelPermission(db, roles, modelID, permission) {
			c.Set(ContextRoles, roleCodes(roles))
			c.Next()
			return
		}

		utils.Forbidden(c, "无此操作权限")
		c.Abort()
	}
}

func parseUint64(s string) (uint64, error) {
	var v uint64
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0, fmt.Errorf("invalid uint64: %s", s)
		}
		v = v*10 + uint64(ch-'0')
	}
	return v, nil
}

func roleCodes(roles []models.RbacRole) []string {
	codes := make([]string, len(roles))
	for i, r := range roles {
		codes[i] = r.RoleCode
	}
	return codes
}

func hasModelPermission(db *gorm.DB, roles []models.RbacRole, modelID uint64, permission string) bool {
	for _, role := range roles {
		if role.RoleCode == "GOVERNANCE_ADMIN" && permission == "READ" {
			return true
		}
	}

	for _, role := range roles {
		var count int64
		db.Model(&models.RbacRoleModelPermission{}).
			Where("role_id = ? AND model_id = ? AND permission = ?", role.ID, modelID, permission).
			Count(&count)
		if count > 0 {
			return true
		}
	}
	return false
}
