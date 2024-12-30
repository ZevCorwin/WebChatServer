package main

import (
	"chat-app-backend/config"
	"chat-app-backend/routes"
	"github.com/gin-contrib/cors"
	"log"
	"time"
)

func main() {
	// Kết nối cơ sở dữ liệu
	// Mới
	config.InitDB()
	//db := config.DB

	// Nạp cấu hình
	cfg := config.LoadConfig()

	// Khởi tạo router với tất cả routes
	router := routes.SetupRouter()
	router.Static("/uploads", "./uploads")

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Chạy server trên cổng từ biến môi trường
	port := cfg.AppPort
	if port == "" {
		port = "8080" // Mặc định là 8080 nếu không có biến môi trường
	}
	log.Printf("Server is running on port %s...", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Không thể khởi động server: %v", err)
	}

}
