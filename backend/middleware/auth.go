package middleware

import (
	"rosetta/utils"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	ContextUserID   = "user_id"
	ContextUsername = "username"
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
