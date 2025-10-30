package middleware

import (
	"chat-app-backend/models"
	"github.com/gin-gonic/gin"
	"net/http"
)

func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		v, ok := c.Get("currentUser")
		if !ok || v == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "[AdminOnly] Unauthorized"})
			return
		}

		var user models.User
		switch t := v.(type) {
		case models.User:
			user = t
		case *models.User:
			if t == nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "[AdminOnly] No user"})
				return
			}
			user = *t
		default:
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "[AdminOnly] Bad currentUser type"})
			return
		}

		if user.Role != models.RoleAdmin { // dùng enum cứng hiện tại của bạn
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
			return
		}
		c.Next()
	}
}
