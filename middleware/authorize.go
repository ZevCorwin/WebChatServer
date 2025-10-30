package middleware

import (
	"chat-app-backend/services"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"net/http"
)

func RequirePerm(acl *services.ACLService, code string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")
		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "[RequirePerm] unauthorized"})
			c.Abort()
			return
		}
		oid, err := primitive.ObjectIDFromHex(userID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id"})
			c.Abort()
			return
		}
		if !acl.Has(oid, code) {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "need": code})
			c.Abort()
			return
		}
		c.Next()
	}
}
