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
	UserService *services.UserService
}

func NewAdminController(us *services.UserService) *AdminController {
	return &AdminController{UserService: us}
}

type lockReq struct {
	UntilISO *string `json:"untilISO"` // null/empty => khoá vĩnh viễn
	Reason   string  `json:"reason"`
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
