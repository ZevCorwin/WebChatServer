package routes

import (
	"chat-app-backend/controllers"
	"chat-app-backend/middleware"
	"chat-app-backend/services"
	"github.com/gin-gonic/gin"
)

func SetupChannelRoutes(router *gin.Engine) {
	// Tạo một channel controller và service mới
	channelService := services.NewChannelService()
	channelController := controllers.NewChannelController(channelService)

	// Group routes cho channels
	channelRoutes := router.Group("/api/channels", middleware.AuthMiddleware(), middleware.CurrentUserMiddleware())
	{
		channelRoutes.POST("", channelController.CreateChannelHandler)
		channelRoutes.PUT("/:channelID/members/:memberID", channelController.AddMemberHandler)
		channelRoutes.DELETE("/:channelID/members/:memberID", channelController.RemoveMemberHandler)
		channelRoutes.GET("/:channelID/members", channelController.ListMembersHandler)
		channelRoutes.PUT("/:channelID/approval", channelController.ToggleApprovalHandler)
		channelRoutes.POST("/:channelID/leave/:memberID", channelController.LeaveChannelHandler)                 // Thành viên rời khỏi kênh
		channelRoutes.DELETE("/:channelID/dissolve/:leaderID", channelController.DissolveChannelHandler)         // Giải tán kênh
		channelRoutes.POST("/:channelID/block/:blockerID/:memberID", channelController.BlockMemberHandler)       // Chặn thành viên
		channelRoutes.POST("/:channelID/unblock/:unblockerID/:memberID", channelController.UnblockMemberHandler) // Bỏ chặn thành viên
		channelRoutes.GET("/search", channelController.SearchChannelsHandler)
		channelRoutes.GET("/user/:userID/channels", channelController.GetUserChannelsHandler)
		channelRoutes.GET("/find-private-channel", channelController.FindPrivateChannelHandler)
	}
}
