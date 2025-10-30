package middleware

import (
	"chat-app-backend/models"
	"chat-app-backend/services"
	"chat-app-backend/utils"
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func MakeLockCheckMiddleware(us *services.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		uidHex := c.GetString("user_id")
		if uidHex == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		uid, err := primitive.ObjectIDFromHex(uidHex)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid user id"})
			return
		}

		var u models.User
		if err := us.DB.Collection("users").FindOne(context.TODO(), bson.M{"_id": uid}).Decode(&u); err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
			return
		}

		// Tự mở nếu quá hạn
		utils.AutoUnlockIfExpired(us.DB, &u)

		// Còn đang khóa?
		locked := u.AdminLocked || (u.LockedUntil != nil && time.Now().Before(*u.LockedUntil))
		if locked {
			var until string
			if u.LockedUntil != nil {
				until = u.LockedUntil.UTC().Format(time.RFC3339)
			}
			c.AbortWithStatusJSON(http.StatusLocked, gin.H{
				"error":       "Tài khoản đã bị khoá",
				"lockedUntil": until,
				"reason":      u.LockReason,
			})
			return
		}

		c.Next()
	}
}
