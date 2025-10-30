package routes

import (
	"chat-app-backend/controllers"
	"chat-app-backend/middleware"
	"chat-app-backend/services"
	"github.com/gin-gonic/gin"
)

func SetupMessageRoutes(router *gin.Engine, messageController *controllers.MessageController, us *services.UserService, lockMw gin.HandlerFunc) {
	// Đăng ký routes
	router.GET("/ws/messages", messageController.HandleWebSocket, middleware.AuthMiddleware(), middleware.CurrentUserMiddleware(us), lockMw)

	protected := router.Group("/api", middleware.AuthMiddleware(), middleware.CurrentUserMiddleware(us), lockMw)
	protected.Use(middleware.AuthMiddleware())
	protected.POST("/messages/:messageID/recall", middleware.AuthMiddleware(), messageController.RecallMessageHandler)
	protected.DELETE("/messages/:messageID/hide", middleware.AuthMiddleware(), messageController.HideMessageHandler)
	protected.PUT("/messages/:messageID/", messageController.EditMessage)
	protected.POST("messages/:messageID/reaction", messageController.ToggleReaction)
}
