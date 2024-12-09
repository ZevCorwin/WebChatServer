package routes

import (
	"chat-app-backend/config"
	"chat-app-backend/controllers"
	"chat-app-backend/services"
	"github.com/gin-gonic/gin"
)

func SetupChatHistoryRoutes(router *gin.Engine) {
	// Tạo service và controller
	chatHistoryService := services.NewChatHistoryService(config.DB)
	chatHistoryController := controllers.NewChatHistoryController(chatHistoryService)

	chatHistory := router.Group("/api/chatHistory")
	{
		chatHistory.GET("/:channelID", chatHistoryController.GetChatHistory)
		chatHistory.GET("/user/:userID", chatHistoryController.GetChatHistoryByUserID)
		chatHistory.DELETE("/:channelID", chatHistoryController.DeleteChatHistory)
	}
}
