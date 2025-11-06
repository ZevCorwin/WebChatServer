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
		// --- Quản lý User (Vòng 2: Phân quyền động) ---
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
		// API Ban chức
		group.POST("/users/:userID/assign-role",
			middleware.RequirePerm(acl, "user.assign_role"),
			ac.AssignRoleToUser,
		)
		group.POST("/users/:userID/revoke-role", middleware.RequirePerm(acl, "user.revoke_role"), ac.RevokeRoleFromUser)

		// --- Quản lý Vai trò & Quyền ---
		group.GET("/permissions",
			middleware.RequirePerm(acl, "permission.read"),
			ac.ListPermissions,
		)
		group.GET("/roles",
			middleware.RequirePerm(acl, "role.read"),
			ac.ListRoles,
		)
		group.POST("/roles",
			middleware.RequirePerm(acl, "role.create"),
			ac.CreateRole,
		)
		group.PUT("/roles/:roleID",
			middleware.RequirePerm(acl, "role.update"),
			ac.UpdateRole,
		)
		group.DELETE("/roles/:roleID",
			middleware.RequirePerm(acl, "role.delete"),
			ac.DeleteRole,
		)

		// --- Thống kê ---
		// Cả 3 API thống kê đều dùng chung 1 quyền "stats.read"
		group.GET("/stats/overview",
			middleware.RequirePerm(acl, "stats.read"),
			ac.GetOverviewStats,
		)
		group.GET("/stats/messages",
			middleware.RequirePerm(acl, "stats.read"),
			ac.GetMessageActivityStats,
		)
		group.GET("/stats/user-growth",
			middleware.RequirePerm(acl, "stats.read"),
			ac.GetUserGrowthStats,
		)

		// Sau này thêm:
		// group.GET("/allowlist",   middleware.RequirePerm(acl, "allowlist.read"),   ac.ListAllow)
		// group.POST("/allowlist",  middleware.RequirePerm(acl, "allowlist.create"), ac.CreateAllow)
		// group.DELETE("/allowlist/:id", middleware.RequirePerm(acl, "allowlist.delete"), ac.DeleteAllow)
		// group.GET("/stats/overview", middleware.RequirePerm(acl, "stats.read"), ac.OverviewStats)
	}
}
