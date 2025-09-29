package routes

import (
	"chat-app-backend/controllers"
	"chat-app-backend/middleware"
	"github.com/gin-gonic/gin"
)

func SetupMessageRoutes(router *gin.Engine, messageController *controllers.MessageController) {
	// Đăng ký routes
	router.GET("/ws/messages", messageController.HandleWebSocket)

	protected := router.Group("/api")
	protected.Use(middleware.AuthMiddleware())
	protected.POST("/messages/:messageID/recall", middleware.AuthMiddleware(), messageController.RecallMessageHandler)
	protected.DELETE("/messages/:messageID/hide", middleware.AuthMiddleware(), messageController.HideMessageHandler)
}
