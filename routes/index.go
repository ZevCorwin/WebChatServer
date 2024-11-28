package routes

import (
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

// SetupRouter khởi tạo các routes chính
func SetupRouter(db *mongo.Database) *gin.Engine {
	router := gin.Default()

	// Cấu hình routes cho người dùng
	SetupUserRoutes(router, db)

	// Cấu hình routes cho tin nhắn
	SetupMessageRoutes(router, db)

	// Cấu hình routes cho WebRTC
	SetupWebRTCRoutes(router)

	// Cấu hình routes cho Channel
	SetupChannelRoutes(router, db)

	// Kiểm tra kết nối client - server
	SetupPingRoute(router)

	// Cấu hình routes cho ChatHistory
	SetupChatHistoryRoutes(router, db)

	// Cấu hình routes cho Friend
	SetupFriendRoutes(router, db)

	return router
}
