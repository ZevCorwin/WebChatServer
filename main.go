package main

import (
	"chat-app-backend/config"
	"chat-app-backend/controllers"
	"chat-app-backend/routes"
	"chat-app-backend/services"
	"github.com/gin-contrib/cors"
	"log"
	"strings"
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
		AllowOriginFunc: func(origin string) bool {
			// Cho phép FE production cố định
			if origin == "https://web-chat-client-ten.vercel.app" {
				return true
			}
			// Cho phép mọi subdomain vercel.app nếu bạn cần test preview
			if strings.HasSuffix(origin, ".vercel.app") {
				return true
			}
			// Local dev
			if origin == "http://localhost:3000" || origin == "http://127.0.0.1:3000" {
				return true
			}
			return false
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Accept"},
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
