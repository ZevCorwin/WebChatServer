package routes

import (
	"chat-app-backend/controllers"
	"github.com/gin-gonic/gin"
)

// SetupRouter khởi tạo các routes chính
func SetupRouter(
	router *gin.Engine,
	messageController *controllers.MessageController,
	channelController *controllers.ChannelController,
) {

	// Cấu hình routes cho người dùng
	SetupUserRoutes(router)

	// Cấu hình routes cho tin nhắn
	SetupMessageRoutes(router, messageController)

	// Cấu hình routes cho Channel
	SetupChannelRoutes(router, channelController)

	// Kiểm tra kết nối client - server
	SetupPingRoute(router)

	// Cấu hình routes cho ChatHistory
	SetupChatHistoryRoutes(router)

	SetupFileRoutes(router)

	// Cấu hình routes cho Friend
	SetupFriendRoutes(router)
}
