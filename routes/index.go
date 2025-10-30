package routes

import (
	"chat-app-backend/controllers"
	"chat-app-backend/services"
	"github.com/gin-gonic/gin"
)

// SetupRouter khởi tạo các routes chính
func SetupRouter(
	router *gin.Engine,
	messageController *controllers.MessageController,
	channelController *controllers.ChannelController,
	ac *controllers.AdminController,
	lockMw gin.HandlerFunc,
	acl *services.ACLService,
	us *services.UserService,
) {

	SetupAdminRoutes(router, ac, us, acl)

	// Cấu hình routes cho người dùng
	SetupUserRoutes(router)

	// Cấu hình routes cho tin nhắn
	SetupMessageRoutes(router, messageController, us, lockMw)

	// Cấu hình routes cho Channel
	SetupChannelRoutes(router, channelController, us, lockMw)

	// Kiểm tra kết nối client - server
	SetupPingRoute(router)

	// Cấu hình routes cho ChatHistory
	SetupChatHistoryRoutes(router, us, lockMw)

	SetupFileRoutes(router, us, lockMw)

	// Cấu hình routes cho Friend
	SetupFriendRoutes(router, us, lockMw)
}
