package interfaces

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// WebRTCNotifier định nghĩa giao diện để gửi thông báo
type WebRTCNotifier interface {
	NotifyUser(userID string, message interface{})
	BroadcastMessage(channelID primitive.ObjectID, message interface{}) // Đã sửa
}
