package routes

import (
	"chat-app-backend/controllers"
	"github.com/gin-gonic/gin"
)

func SetupMessageRoutes(router *gin.Engine, messageController *controllers.MessageController) {
	// Đăng ký routes
	router.GET("/ws/messages", messageController.HandleWebSocket)

	// Thu hồi tin nhắn
	router.PUT("/api/messages/:channelID/:messageID/recall", messageController.HandleRecallMessage)

	// Xóa tin nhắn
	router.DELETE("/api/messages/:channelID/:messageID", messageController.HandleDeleteMessage)
}
