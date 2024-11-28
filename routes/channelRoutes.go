package routes

import (
	"chat-app-backend/controllers"
	"chat-app-backend/middleware"
	"chat-app-backend/services"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

func SetupChannelRoutes(router *gin.Engine, db *mongo.Database) {
	// Tạo một channel controller mới
	channelService := services.NewChannelService(db)
	channelController := controllers.NewChannelController(channelService)

	// Tạo một channel service mới

	// Group routes cho channels
	channelRoutes := router.Group("/api/channels", middleware.CurrentUserMiddleware())
	{
		channelRoutes.POST("", channelController.CreateChannelHandler)
		channelRoutes.PUT("/:channelId/members/:memberId", channelController.AddMemberHandler)
		channelRoutes.DELETE("/:channelId/members/:memberId", channelController.RemoveMemberHandler)
		channelRoutes.GET("/:channelId/members", channelController.ListMembersHandler)
		channelRoutes.PUT("/:channelId/approval", channelController.ToggleApprovalHandler)
		channelRoutes.POST("/:channelId/leave/:memberId", channelController.LeaveChannelHandler)                 // Thành viên rời khỏi kênh
		channelRoutes.DELETE("/:channelId/dissolve/:leaderId", channelController.DissolveChannelHandler)         // Giải tán kênh
		channelRoutes.POST("/:channelId/block/:blockerId/:memberId", channelController.BlockMemberHandler)       // Chặn thành viên
		channelRoutes.POST("/:channelId/unblock/:unblockerId/:memberId", channelController.UnblockMemberHandler) // Bỏ chặn thành viên
		channelRoutes.GET("/search", channelController.SearchChannelsHandler)
		channelRoutes.GET("/user/:userID/channels", channelController.GetUserChannelsHandler)
		channelRoutes.GET("/find-private-channel", channelController.FindPrivateChannelHandler)
	}
}
