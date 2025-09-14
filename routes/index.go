package routes

import (
	"github.com/gin-gonic/gin"
)

// SetupRouter khởi tạo các routes chính
func SetupRouter() *gin.Engine {
	router := gin.Default()

	// Cấu hình routes cho người dùng
	SetupUserRoutes(router)

	// Cấu hình routes cho tin nhắn
	SetupMessageRoutes(router)

	// Cấu hình routes cho Channel
	SetupChannelRoutes(router)

	// Kiểm tra kết nối client - server
	SetupPingRoute(router)

	// Cấu hình routes cho ChatHistory
	SetupChatHistoryRoutes(router)

	// Cấu hình routes cho Friend
	SetupFriendRoutes(router)

	return router
}
