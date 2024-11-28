package routes

import (
	"chat-app-backend/controllers"

	"github.com/gin-gonic/gin"
)

func SetupWebRTCRoutes(router *gin.Engine) {
	webrtcController := controllers.NewWebRTCController()

	router.GET("/ws/realtime", webrtcController.HandleSignaling)
}
