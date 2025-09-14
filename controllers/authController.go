package controllers

import (
	"chat-app-backend/models"
	"chat-app-backend/services"
	"github.com/gin-gonic/gin"
	"net/http"
)

type AuthController struct {
	UserService *services.UserService
	OTPService  *services.OTPService
}

func NewAuthController(us *services.UserService, otps *services.OTPService) *AuthController {
	return &AuthController{
		UserService: us,
		OTPService:  otps,
	}
}

// B1: Gửi OTP + giữ tạm payload
func (ac *AuthController) RequestRegisterOTP(ctx *gin.Context) {
	var req struct {
		Name     string `json:"name"`
		Birth    string `json:"dob"`
		Email    string `json:"email"`
		Phone    string `json:"phone"`
		Password string `json:"password"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}
	if req.Email == "" || req.Phone == "" || req.Password == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu email/phone/password"})
		return
	}

	// check trùng
	if _, err := ac.UserService.GetUserByPhone(req.Phone); err == nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "SĐT đã tồn tại"})
		return
	}
	if users, _ := ac.UserService.GetAllUsers(); len(users) > 0 {
		// kiểm tra tồn tại email nhanh gọn
		for _, u := range users {
			if u.Email == req.Email {
				ctx.JSON(http.StatusBadRequest, gin.H{"error": "Email đã tồn tại"})
				return
			}
		}
	}

	payload := map[string]interface{}{
		"name":     req.Name,
		"dob":      req.Birth,
		"email":    req.Email,
		"phone":    req.Phone,
		"password": req.Password,
	}

	if err := ac.OTPService.CreateAndSendRegisterOTP(req.Email, payload); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Không gửi được OTP: " + err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "Đã gửi OTP đến email"})
}

// B2: Xác thực OTP → tạo user chính thức
func (ac *AuthController) VerifyRegisterOTP(ctx *gin.Context) {
	var req struct {
		Email string `json:"email"`
		Code  string `json:"code"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil || req.Email == "" || req.Code == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu email/mã OTP"})
		return
	}

	payload, err := ac.OTPService.VerifyRegisterOTP(req.Email, req.Code)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// map payload -> models.User
	user := models.User{
		Name:     toString(payload["name"]),
		Email:    toString(payload["email"]),
		Phone:    toString(payload["phone"]),
		Password: toString(payload["password"]),
		// bạn đang dùng BirthDate string -> để tạm dob
		BirthDate: toString(payload["dob"]),
		// Avatar default sẽ set trong Register()
	}
	newUser, regErr := ac.UserService.Register(user)
	if regErr != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": regErr.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "Xác thực OTP thành công, tài khoản đã tạo", "user": newUser})
}

func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
