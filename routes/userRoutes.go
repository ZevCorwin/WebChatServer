package routes

import (
	"chat-app-backend/controllers"
	"chat-app-backend/middleware"
	"chat-app-backend/services"
	"github.com/gin-gonic/gin"
)

func SetupUserRoutes(router *gin.Engine) {
	userService := services.NewUserService()
	userController := controllers.NewUserController(userService)

	// ---- New: OTP
	otpService := services.NewOTPService()
	authController := controllers.NewAuthController(userService, otpService)

	emailChangeCtrl := controllers.NewEmailChangeController(userService, otpService)

	// Đăng ký routes
	router.POST("/register", userController.RegisterHandler) // cũ (nếu vẫn muốn để)
	// Mới - Đăng ký 2 bước
	router.POST("/register/request-otp", authController.RequestRegisterOTP)
	router.POST("/register/verify-otp", authController.VerifyRegisterOTP)

	router.POST("/login", userController.LoginHandler)
	router.GET("/users", userController.GetAllUsersHandler)
	router.GET("/users/:id", userController.GetUserByIdHandler)
	router.PUT("/users/:id", userController.UpdateProfileHandler)
	router.POST("/users/:id/avatar", userController.UpdateAvatarHandler)
	router.POST("/users/:id/cover", userController.UpdateCoverPhotoHandler)
	router.GET("/users/search", userController.SearchUserByPhoneHandler)

	router.POST("/users/:id/change-email/request-old-otp", emailChangeCtrl.RequestOldEmailOTP)
	router.POST("/users/:id/change-email/verify-old-otp", emailChangeCtrl.VerifyOldEmailOTP)
	router.POST("/users/:id/change-email/request-new-otp", emailChangeCtrl.RequestNewEmailOTP)
	router.POST("/users/:id/change-email/verify-new-otp", emailChangeCtrl.VerifyNewEmailAndChange)

	router.POST("/users/:id/change-phone", userController.ChangePhoneHandler)
	router.POST("/users/:id/change-password", userController.ChangePasswordHandler)

	protected := router.Group("/")
	protected.Use(middleware.AuthMiddleware())
	protected.GET("/users/:id/channels", userController.GetUserChannelsHandler)
}
