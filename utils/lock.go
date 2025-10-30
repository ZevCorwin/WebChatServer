package utils

import (
	"chat-app-backend/models"
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Gọi hàm này NGAY SAU khi load user từ DB
func AutoUnlockIfExpired(db *mongo.Database, u *models.User) {
	if u == nil || u.LockedUntil == nil {
		return
	}
	if time.Now().After(*u.LockedUntil) {
		_, _ = db.Collection("users").UpdateByID(context.TODO(), u.ID, bson.M{
			"$set": bson.M{
				"adminLocked": false,
				"lockedUntil": nil,
				"lockReason":  "",
				"lockedBy":    nil,
			},
		})
		u.AdminLocked = false
		u.LockedUntil = nil
		u.LockReason = ""
		u.LockedBy = nil
	}
}
