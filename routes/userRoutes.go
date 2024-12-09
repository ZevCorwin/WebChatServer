package routes

import (
	"chat-app-backend/controllers"
	"chat-app-backend/middleware"
	"chat-app-backend/services"
	"github.com/gin-gonic/gin"
)

// SetupUserRoutes cấu hình các routes liên quan đến người dùng
func SetupUserRoutes(router *gin.Engine) {
	userService := services.NewUserService()
	userController := controllers.NewUserController(userService)

	// Đăng ký routes
	router.POST("/register", userController.RegisterHandler)
	router.POST("/login", userController.LoginHandler)
	router.GET("/users", userController.GetAllUsersHandler)
	router.GET("/users/:id", userController.GetUserByIdHandler)
	router.PUT("/users/:id", userController.UpdateProfileHandler)
	router.GET("/users/search", userController.SearchUserByPhoneHandler)

	// Nhóm các routes cần bảo vệ
	protected := router.Group("/")
	protected.Use(middleware.AuthMiddleware())

	// Route để lấy danh sách kênh người dùng đã tham gia
	protected.GET("/users/:id/channels", userController.GetUserChannelsHandler)
}
