package main

import (
	"chat-app-backend/config"
	"chat-app-backend/controllers"
	"chat-app-backend/routes"
	"chat-app-backend/services"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"log"
	"os"
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

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	router.Use(cors.New(cors.Config{
		AllowOrigins: []string{
			"https://web-chat-client-ten.vercel.app",
			"http://localhost:3000",
			"http://127.0.0.1:3000",
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Accept"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// --- Router (gom routes trong index.go) ---
	routes.SetupRouter(router, messageController, channelController)

	// Chỉ serve folder /uploads khi STORAGE_PROVIDER=local (để test local)
	if os.Getenv("STORAGE_PROVIDER") == "" || os.Getenv("STORAGE_PROVIDER") == "local" {
		router.Static("/uploads", "./uploads")
	}
	router.MaxMultipartMemory = 32 << 20 // 32MB

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
