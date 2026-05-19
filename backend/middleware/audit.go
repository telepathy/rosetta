package middleware

import (
	"rosetta/models"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func AuditLog(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		userID, _ := c.Get(ContextUserID)
		username, _ := c.Get(ContextUsername)
		method := c.Request.Method
		path := c.Request.URL.Path

		if method == "GET" {
			return
		}

		entry := models.AuditLog{
			UserID:     toUint64(userID),
			Username:   toString(username),
			Action:     method,
			Resource:   path,
			ResourceID: c.Param("id"),
			IP:         c.ClientIP(),
			CreatedAt:  time.Now(),
		}
		_ = db.Create(&entry)
	}
}

func toUint64(v interface{}) uint64 {
	switch val := v.(type) {
	case uint64:
		return val
	case float64:
		return uint64(val)
	}
	return 0
}

func toString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	}
	return ""
}
