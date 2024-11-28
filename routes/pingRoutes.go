package routes

import (
	"github.com/gin-gonic/gin"
)

func SetupPingRoute(router *gin.Engine) {
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Server is running",
		})
	})
}
