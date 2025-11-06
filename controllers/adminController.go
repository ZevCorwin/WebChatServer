package controllers

import (
	"chat-app-backend/models"
	"chat-app-backend/services"
	"context"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net/http"
	"time"
)

type AdminController struct {
	UserService  *services.UserService
	StatsService *services.StatsService
	RoleService  *services.RoleService
}

func NewAdminController(us *services.UserService, ss *services.StatsService, rs *services.RoleService) *AdminController {
	return &AdminController{
		UserService:  us,
		StatsService: ss,
		RoleService:  rs,
	}
}

type lockReq struct {
	UntilISO *string `json:"untilISO"` // null/empty => khoá vĩnh viễn
	Reason   string  `json:"reason"`
}

// createRolePayload là struct để nhận dữ liệu khi tạo vai trò mới
type createRolePayload struct {
	Name          string               `json:"name" binding:"required"`
	PermissionIDs []primitive.ObjectID `json:"permissionIDs"` // Mảng các ID quyền
}

// roleAssignPayload là struct để nhận roleID khi gán/tước quyền
type roleAssignPayload struct {
	RoleID string `json:"roleID" binding:"required"`
}

func (ac *AdminController) AdminLogin(c *gin.Context) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Email == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu email/mật khẩu"})
		return
	}

	col := ac.UserService.DB.Collection("users")
	var user models.User
	if err := col.FindOne(context.Background(), bson.M{"email": req.Email}).Decode(&user); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Thông tin không đúng"})
		return
	}
	if !ac.UserService.CheckPasswordHash(req.Password, user.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Thông tin không đúng"})
		return
	}
	// Chốt quyền: chỉ cho "Quản trị viên"
	if user.Role != models.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Không có quyền admin"})
		return
	}

	// Tạo JWT có claim role
	token, err := ac.UserService.GenerateJWT(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi tạo token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "Đăng nhập admin thành công",
		"token":   token,
		"userID":  user.ID,
		"role":    user.Role,
	})
}

func (ac *AdminController) LockUser(c *gin.Context) {
	target := c.Param("userID")
	tid, err := primitive.ObjectIDFromHex(target)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid userID"})
		return
	}

	var req lockReq
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad body"})
		return
	}

	adminAny, _ := c.Get("currentUser")
	admin := adminAny.(models.User)

	var until *time.Time
	if req.UntilISO != nil && *req.UntilISO != "" {
		t, err := time.Parse(time.RFC3339, *req.UntilISO)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "untilISO must be RFC3339"})
			return
		}
		until = &t
	}

	if err := ac.UserService.LockUser(tid, admin.ID, until, req.Reason); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (ac *AdminController) UnlockUser(c *gin.Context) {
	target := c.Param("userID")
	tid, err := primitive.ObjectIDFromHex(target)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid userID"})
		return
	}
	if err := ac.UserService.UnlockUser(tid); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// optional: list users by lock state
func (ac *AdminController) ListUsers(c *gin.Context) {
	locked := c.Query("locked") == "true"
	filter := bson.M{}
	if locked {
		// locked if adminLocked==true OR lockedUntil in future
		filter = bson.M{"$or": []bson.M{
			{"adminLocked": true},
			{"lockedUntil": bson.M{"$gt": time.Now()}},
		}}
	}
	cur, err := ac.UserService.DB.Collection("users").Find(context.TODO(), filter, options.Find().SetLimit(200))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var users []models.User
	if err := cur.All(context.TODO(), &users); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"users": users})
}

func (ac *AdminController) GetOverviewStats(ctx *gin.Context) {
	// Gọi service
	stats, err := ac.StatsService.GetOverviewStats(ctx.Request.Context())
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lấy dữ liệu thống kê: " + err.Error()})
		return
	}

	// Trả về dữ liệu thành công
	ctx.JSON(http.StatusOK, stats)
}

// GetMessageActivityStats lấy thống kê về loại tin nhắn và giờ cao điểm
func (ac *AdminController) GetMessageActivityStats(ctx *gin.Context) {
	stats, err := ac.StatsService.GetMessageActivityStats(ctx.Request.Context())
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lấy dữ liệu thống kê: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, stats)
}

// GetUserGrowthStats lấy thống kê người dùng mới đăng ký theo ngày
func (ac *AdminController) GetUserGrowthStats(ctx *gin.Context) {
	stats, err := ac.StatsService.GetUserGrowthStats(ctx.Request.Context())
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lấy dữ liệu thống kê: " + err.Error()})
		return
	}

	// Trả về dữ liệu (là một mảng)
	ctx.JSON(http.StatusOK, stats)
}

// ListPermissions trả về tất cả các quyền có trong hệ thống
func (ac *AdminController) ListPermissions(ctx *gin.Context) {
	permissions, err := ac.RoleService.ListPermissions(ctx.Request.Context())
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lấy danh sách quyền: " + err.Error()})
		return
	}

	// Trả về mảng permissions
	ctx.JSON(http.StatusOK, permissions)
}

// ListRoles trả về tất cả các vai trò (roles) trong hệ thống
func (ac *AdminController) ListRoles(ctx *gin.Context) {
	roles, err := ac.RoleService.ListRoles(ctx.Request.Context())
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lấy danh sách vai trò: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, roles)
}

// CreateRole tạo một vai trò mới
func (ac *AdminController) CreateRole(ctx *gin.Context) {
	var payload createRolePayload

	// Bind JSON payload vào struct
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ: " + err.Error()})
		return
	}

	// Đảm bảo mảng permissionIDs không bị nil (gán mảng rỗng nếu nil)
	if payload.PermissionIDs == nil {
		payload.PermissionIDs = []primitive.ObjectID{}
	}

	// Gọi service để tạo
	newRole, err := ac.RoleService.CreateRole(ctx.Request.Context(), payload.Name, payload.PermissionIDs)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo vai trò: " + err.Error()})
		return
	}

	// Trả về vai trò vừa tạo
	ctx.JSON(http.StatusCreated, newRole)
}

// UpdateRole cập nhật một vai trò đã có
func (ac *AdminController) UpdateRole(ctx *gin.Context) {
	// 1. Lấy roleID từ URL param
	roleIDStr := ctx.Param("roleID")
	roleID, err := primitive.ObjectIDFromHex(roleIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Role ID không hợp lệ"})
		return
	}

	// 2. Lấy payload từ JSON body (dùng lại struct của CreateRole)
	var payload createRolePayload
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ: " + err.Error()})
		return
	}

	// Đảm bảo mảng permissionIDs không bị nil (gán mảng rỗng nếu nil)
	if payload.PermissionIDs == nil {
		payload.PermissionIDs = []primitive.ObjectID{}
	}

	// 3. Gọi service để cập nhật
	err = ac.RoleService.UpdateRole(ctx.Request.Context(), roleID, payload.Name, payload.PermissionIDs)
	if err != nil {
		// Kiểm tra lỗi "không tìm thấy"
		if err.Error() == "không tìm thấy vai trò với ID này" {
			ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		// Lỗi server khác
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật vai trò: " + err.Error()})
		return
	}

	// Trả về thành công
	ctx.JSON(http.StatusOK, gin.H{"message": "Cập nhật vai trò thành công"})
}

// DeleteRole xóa một vai trò
func (ac *AdminController) DeleteRole(ctx *gin.Context) {
	// 1. Lấy roleID từ URL param
	roleIDStr := ctx.Param("roleID")
	roleID, err := primitive.ObjectIDFromHex(roleIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Role ID không hợp lệ"})
		return
	}

	// 2. Gọi service để xóa
	err = ac.RoleService.DeleteRole(ctx.Request.Context(), roleID)
	if err != nil {
		// Kiểm tra lỗi nghiệp vụ (do service trả về)
		if err.Error() == "không tìm thấy vai trò với ID này" {
			ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "không thể xóa vai trò này vì đang có người dùng sử dụng" {
			// Lỗi 409 (Conflict) thường dùng cho trường hợp này
			ctx.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}

		// Lỗi server khác
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xóa vai trò: " + err.Error()})
		return
	}

	// Trả về thành công
	ctx.JSON(http.StatusOK, gin.H{"message": "Xóa vai trò thành công"})
}

// AssignRoleToUser gán một vai trò cho người dùng (Ban chức)
func (ac *AdminController) AssignRoleToUser(ctx *gin.Context) {
	// 1. Lấy userID từ URL param
	userIDStr := ctx.Param("userID")
	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "User ID không hợp lệ"})
		return
	}

	// 2. Lấy roleID từ JSON body
	var payload roleAssignPayload
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ, cần 'roleID'"})
		return
	}

	roleID, err := primitive.ObjectIDFromHex(payload.RoleID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Role ID không hợp lệ"})
		return
	}

	// 3. Gọi service để gán quyền
	err = ac.RoleService.AssignRoleToUser(ctx.Request.Context(), userID, roleID)
	if err != nil {
		if err.Error() == "không tìm thấy người dùng với ID này" {
			ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể gán vai trò: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Gán vai trò thành công"})
}

// RevokeRoleFromUser tước một vai trò khỏi người dùng (Giáng chức)
func (ac *AdminController) RevokeRoleFromUser(ctx *gin.Context) {
	// 1. Lấy adminID (người đang thực hiện) từ context
	adminIDStr := ctx.GetString("user_id") // Lấy từ AuthMiddleware
	adminID, err := primitive.ObjectIDFromHex(adminIDStr)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Admin ID không hợp lệ"})
		return
	}

	// 2. Lấy targetUserID (người bị giáng chức) từ URL param
	targetUserIDStr := ctx.Param("userID")
	targetUserID, err := primitive.ObjectIDFromHex(targetUserIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Target User ID không hợp lệ"})
		return
	}

	// 3. Lấy roleID từ JSON body (dùng lại struct cũ)
	var payload roleAssignPayload
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ, cần 'roleID'"})
		return
	}
	roleID, err := primitive.ObjectIDFromHex(payload.RoleID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Role ID không hợp lệ"})
		return
	}

	// 4. Gọi service (với rào chắn an toàn)
	err = ac.RoleService.RevokeRoleFromUser(ctx.Request.Context(), adminID, targetUserID, roleID)
	if err != nil {
		// Xử lý các lỗi nghiệp vụ
		if err.Error() == "không thể tự tước vai trò của chính mình" {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()}) // Lỗi 403 Cấm
			return
		}
		if err.Error() == "không tìm thấy người dùng (mục tiêu) với ID này" {
			ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		// Lỗi server khác
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tước vai trò: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Tước vai trò thành công"})
}
