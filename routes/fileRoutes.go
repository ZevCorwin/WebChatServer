package routes

import (
	"chat-app-backend/controllers"
	"chat-app-backend/services"
	"log"

	"github.com/gin-gonic/gin"
)

func SetupFileRoutes(r *gin.Engine) {
	fs, err := services.GetDefaultFileService()
	if err != nil {
		// nếu provider không khởi được, log và panic/exit để dev biết
		log.Fatalf("Không thể khởi FileService: %v", err)
	}
	fc := controllers.NewFileController(fs)

	// Upload
	r.POST("/uploads", fc.Upload)
}
