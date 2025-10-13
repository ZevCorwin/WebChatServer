package main

import (
	"chat-app-backend/config"
	"chat-app-backend/controllers"
	"chat-app-backend/routes"
	"chat-app-backend/services"
	"github.com/gin-contrib/cors"
	"log"
	"time"
)

func main() {
	// Kết nối DB
	config.InitDB()
	cfg := config.LoadConfig()

	// --- Services ---
	messageService := services.NewMessageService()
	channelService := services.NewChannelService()

	// --- WebRTCController ---
	webrtcController := controllers.NewWebRTCController(messageService, channelService)

	// --- Controllers ---
	messageController := controllers.NewMessageController(messageService, channelService, webrtcController)
	channelController := controllers.NewChannelController(channelService, webrtcController)

	// --- Router (gom routes trong index.go) ---
	router := routes.SetupRouter(messageController, channelController)

	router.Static("/uploads", "./uploads")
	router.MaxMultipartMemory = 32 << 20 // 32MB

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://web-chat-client-ten.vercel.app"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Run server
	port := cfg.AppPort
	if port == "" {
		port = "8080"
	}
	log.Printf("Server is running on port %s...", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Không thể khởi động server: %v", err)
	}
}
