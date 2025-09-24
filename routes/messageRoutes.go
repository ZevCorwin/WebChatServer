package routes

import (
	"chat-app-backend/controllers"
	"chat-app-backend/services"
	"github.com/gin-gonic/gin"
)

func SetupMessageRoutes(router *gin.Engine) {
	// Khởi tạo services
	ms := services.NewMessageService()
	cs := services.NewChannelService()
	wc := controllers.NewWebRTCController(ms, cs)

	// Khởi tạo controller
	messageController := controllers.NewMessageController(ms, cs, wc)

	// Đăng ký routes
	router.GET("/ws/messages", messageController.HandleWebSocket)
}
