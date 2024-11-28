package routes

import (
	"chat-app-backend/controllers"
	"chat-app-backend/services"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

// SetupFriendRoutes sets up friend-related routes.
func SetupFriendRoutes(router *gin.Engine, db *mongo.Database) {
	friendService := services.NewFriendService(db)
	friendController := controllers.NewFriendController(friendService)

	friendRoute := router.Group("/friends")
	{
		friendRoute.POST("/:userID/send/:friendID", friendController.SendFriendRequest)
		friendRoute.DELETE("/:userID/cancel/:friendID", friendController.CancelFriendRequest)
		friendRoute.PUT("/:userID/accept/:friendID", friendController.AcceptFriendRequest)
		friendRoute.PUT("/:userID/decline/:friendID", friendController.DeclineFriendRequest)
		friendRoute.GET("/:userID/list", friendController.GetFriends)
		friendRoute.GET("/:userID/requests", friendController.GetFriendRequests)
		friendRoute.DELETE("/:userID/remove/:friendID", friendController.RemoveFriend)
		friendRoute.GET("/:userID/search", friendController.SearchFriendsByName)
		friendRoute.GET("/:userID/status/:friendID", friendController.CheckFriendStatus)
	}
}
