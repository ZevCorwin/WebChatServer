// routes/adminRoutes.go
package routes

import (
	"chat-app-backend/controllers"
	"chat-app-backend/middleware"
	"chat-app-backend/services"
	"github.com/gin-gonic/gin"
)

func SetupAdminRoutes(router *gin.Engine, ac *controllers.AdminController, us *services.UserService, acl *services.ACLService) {
	router.POST("/api/admin/login", ac.AdminLogin)
	group := router.Group("/api/admin",
		middleware.AuthMiddleware(),          // set user_id
		middleware.CurrentUserMiddleware(us), // set currentUser (AdminOnly cần)
		middleware.AdminOnly(),               // chặn cổng admin bằng role cứng
	)

	{
		// ✅ Quyền động chi tiết
		group.GET("/users",
			middleware.RequirePerm(acl, "user.read"),
			ac.ListUsers,
		)

		group.POST("/users/:userID/lock",
			middleware.RequirePerm(acl, "user.lock"),
			ac.LockUser,
		)

		group.POST("/users/:userID/unlock",
			middleware.RequirePerm(acl, "user.unlock"),
			ac.UnlockUser,
		)

		// Sau này thêm:
		// group.GET("/allowlist",   middleware.RequirePerm(acl, "allowlist.read"),   ac.ListAllow)
		// group.POST("/allowlist",  middleware.RequirePerm(acl, "allowlist.create"), ac.CreateAllow)
		// group.DELETE("/allowlist/:id", middleware.RequirePerm(acl, "allowlist.delete"), ac.DeleteAllow)
		// group.GET("/stats/overview", middleware.RequirePerm(acl, "stats.read"), ac.OverviewStats)
	}
}
