package routes

import (
	"chat-app-backend/controllers"
	"chat-app-backend/middleware"
	"chat-app-backend/services"
	"github.com/gin-gonic/gin"
)

func SetupChatHistoryRoutes(router *gin.Engine, us *services.UserService, lockMw gin.HandlerFunc) {
	// Tạo service và controller
	chatHistoryService := services.NewChatHistoryService()
	chatHistoryController := controllers.NewChatHistoryController(chatHistoryService)

	chatHistory := router.Group("/api/chatHistory", middleware.AuthMiddleware(), middleware.CurrentUserMiddleware(us), lockMw)
	{
		chatHistory.GET("/:channelID/:userID", chatHistoryController.GetChatHistory)
		chatHistory.GET("/user/:userID", chatHistoryController.GetChatHistoryByUserID)
		chatHistory.DELETE("/:channelID", chatHistoryController.DeleteChatHistory)
	}
}
