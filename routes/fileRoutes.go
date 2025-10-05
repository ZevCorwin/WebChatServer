package routes

import (
	"chat-app-backend/controllers"
	"chat-app-backend/services"

	"github.com/gin-gonic/gin"
)

func SetupFileRoutes(r *gin.Engine) {
	fs := services.NewFileService()
	fc := controllers.NewFileController(fs)

	// Upload
	r.POST("/uploads", fc.Upload)
}
