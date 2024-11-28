package routes

import (
	"chat-app-backend/controllers"
	"chat-app-backend/services"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

func SetupMessageRoutes(router *gin.Engine, db *mongo.Database) {

	// Khởi tạo services
	channelService := &services.ChannelService{} // Giả sử bạn đã khởi tạo ChannelService
	messageService := services.NewMessageService(db, channelService)

	// Khởi tạo controller
	messageController := controllers.NewMessageController(
		messageService,
		channelService,
	)

	// Đăng ký routes
	router.GET("/ws/messages", messageController.HandleWebSocket)
	router.POST("/api/messages/send", messageController.SendMessage)
}
