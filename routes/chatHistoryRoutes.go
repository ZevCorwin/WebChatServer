package routes

import (
	"chat-app-backend/controllers"
	"chat-app-backend/services"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

func SetupChatHistoryRoutes(router *gin.Engine, db *mongo.Database) {
	// Tạo service và controller
	chatHistoryService := services.NewChatHistoryService(db)
	chatHistoryController := controllers.NewChatHistoryController(chatHistoryService)

	chatHistory := router.Group("/api/chatHistory")
	{
		chatHistory.GET("/:channelID", chatHistoryController.GetChatHistory)
		chatHistory.GET("/user/:userID", chatHistoryController.GetChatHistoryByUserID)
		chatHistory.DELETE("/:channelID", chatHistoryController.DeleteChatHistory)
	}
}
