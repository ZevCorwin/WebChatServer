package controllers

import (
	"chat-app-backend/models"
	"chat-app-backend/services"
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"log"
	"net/http"
	"os"
)

// UserController chứa các phương thức xử lý yêu cầu của người dùng
type UserController struct {
	UserService *services.UserService
}

type EmailChangeController struct {
	UserService *services.UserService
	OTPService  *services.OTPService
}

// NewUserController khởi tạo một UserController mới
func NewUserController(userService *services.UserService) *UserController {
	return &UserController{
		UserService: userService,
	}
}

func NewEmailChangeController(us *services.UserService, otp *services.OTPService) *EmailChangeController {
	return &EmailChangeController{
		UserService: us,
		OTPService:  otp,
	}
}

// RegisterHandler xử lý yêu cầu đăng ký người dùng mới
func (uc *UserController) RegisterHandler(ctx *gin.Context) {
	var user models.User
	if err := ctx.ShouldBindJSON(&user); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}

	if err := models.Validate.Struct(user); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if user.Phone == "" || user.Email == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Email và số điện thoại không được bỏ trống"})
		return
	}

	newUser, err := uc.UserService.Register(user)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Đăng ký thành công", "user": newUser})
}

// LoginHandler xử lý yêu cầu đăng nhập
func (uc *UserController) LoginHandler(ctx *gin.Context) {
	var loginData struct {
		Email    string `json:"email"`
		Phone    string `json:"phone"`
		Password string `json:"password"`
	}

	// Lấy dữ liệu từ request body
	if err := ctx.ShouldBindJSON(&loginData); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}

	var user models.User
	collection := uc.UserService.DB.Collection("users")

	// Kiểm tra người dùng theo email hoặc số điện thoại
	filter := bson.M{
		"$or": []bson.M{
			{"email": loginData.Email},
			{"phone": loginData.Phone},
		},
	}
	err := collection.FindOne(context.Background(), filter).Decode(&user)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Email hoặc số điện thoại không đúng"})
		return
	}

	// Kiểm tra mật khẩu
	if !uc.UserService.CheckPasswordHash(loginData.Password, user.Password) {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Mật khẩu không đúng"})
		return
	}

	// Tạo JWT token
	token, err := uc.UserService.GenerateJWT(user.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi tạo token"})
		return
	}

	// Trả về token và userID cho người dùng
	ctx.JSON(http.StatusOK, gin.H{
		"message": "Đăng nhập thành công",
		"token":   token,
		"userID":  user.ID, // Thêm userID vào kết quả trả về
	})
}

// Lấy tất cả người dùng
func (uc *UserController) GetAllUsersHandler(ctx *gin.Context) {
	users, err := uc.UserService.GetAllUsers()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"users": users})
}

// Lấy người dùng theo ID
func (uc *UserController) GetUserByIdHandler(c *gin.Context) {
	userId := c.Param("id")

	user, err := uc.UserService.GetUserByID(userId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}

// SearchUserByPhoneHandler tìm kiếm người dùng bằng số điện thoại
func (uc *UserController) SearchUserByPhoneHandler(ctx *gin.Context) {
	phone := ctx.Query("phone") // Lấy số điện thoại từ query params

	if phone == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Vui lòng cung cấp số điện thoại"})
		return
	}

	users, err := uc.UserService.GetUserByPhone(phone)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"users": users})
}

// UpdateProfileHandler xử lý yêu cầu cập nhật thông tin người dùng
func (uc *UserController) UpdateProfileHandler(ctx *gin.Context) {
	var updatedUser models.User
	if err := ctx.ShouldBindJSON(&updatedUser); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}

	// Lấy user ID từ params hoặc token
	userId := ctx.Param("id") // Hoặc lấy từ token tùy vào logic của bạn
	objectID, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "ID người dùng không hợp lệ"})
		return
	}
	updatedUser.ID = objectID

	// Gọi service để cập nhật thông tin
	if err := uc.UserService.UpdateUserProfile(&updatedUser); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Cập nhật thông tin thành công"})
}

// THÊM handler
func (ec *EmailChangeController) RequestOldEmailOTP(ctx *gin.Context) {
	uid := ctx.Param("id")
	userID, err := primitive.ObjectIDFromHex(uid)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "ID không hợp lệ"})
		return
	}

	var body struct {
		Password string `json:"password"`
	}
	if err := ctx.ShouldBindJSON(&body); err != nil || body.Password == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu mật khẩu"})
		return
	}

	// verify pass
	ok, err := ec.UserService.VerifyUserPassword(userID, body.Password)
	if err != nil || !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Mật khẩu không đúng"})
		return
	}

	// lấy email hiện tại
	u, err := ec.UserService.GetUserByID(uid)
	if err != nil || u == nil || u.Email == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Không tìm thấy email hiện tại"})
		return
	}

	err = ec.OTPService.CreateAndSendOTP(
		models.OTPPurposeChangeEmailOld,
		u.Email,
		"[ChatApp] Xác thực email cũ",
		func(code string) string {
			return fmt.Sprintf(`
				<div style="font-family: Arial, sans-serif">
					<h2>Xác thực email hiện tại</h2>
					<p>Để đổi email, hãy nhập mã dưới đây:</p>
					<h1 style="letter-spacing:4px">%s</h1>
					<p>Mã có hiệu lực 10 phút.</p>
				</div>
			`, code)
		},
	)
	if err != nil {
		ctx.JSON(http.StatusTooManyRequests, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "Đã gửi OTP tới email hiện tại"})
}

func (ec *EmailChangeController) VerifyOldEmailOTP(ctx *gin.Context) {
	uid := ctx.Param("id")

	// kiểm tra user tồn tại
	if _, err := ec.UserService.GetUserByID(uid); err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "User không tồn tại"})
		return
	}

	// lấy email hiện tại
	u, _ := ec.UserService.GetUserByID(uid)

	var body struct {
		OTP string `json:"otp"`
	}
	if err := ctx.ShouldBindJSON(&body); err != nil || body.OTP == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu mã OTP"})
		return
	}

	if err := ec.OTPService.VerifyOTP(models.OTPPurposeChangeEmailOld, u.Email, body.OTP); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "Xác thực email cũ thành công"})
}

func (ec *EmailChangeController) RequestNewEmailOTP(ctx *gin.Context) {
	uid := ctx.Param("id")

	// verify user
	if _, err := ec.UserService.GetUserByID(uid); err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "User không tồn tại"})
		return
	}

	var body struct {
		NewEmail string `json:"newEmail"`
	}
	if err := ctx.ShouldBindJSON(&body); err != nil || body.NewEmail == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu email mới"})
		return
	}

	// gửi OTP tới email mới
	err := ec.OTPService.CreateAndSendOTP(
		models.OTPPurposeChangeEmailNew,
		body.NewEmail,
		"[ChatApp] Xác thực email mới",
		func(code string) string {
			return fmt.Sprintf(`
				<div style="font-family: Arial, sans-serif">
					<h2>Xác thực email mới</h2>
					<p>Nhập mã dưới đây để hoàn tất đổi email:</p>
					<h1 style="letter-spacing:4px">%s</h1>
					<p>Mã có hiệu lực 10 phút.</p>
				</div>
			`, code)
		},
	)
	if err != nil {
		ctx.JSON(http.StatusTooManyRequests, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "Đã gửi OTP tới email mới"})
}

func (ec *EmailChangeController) VerifyNewEmailAndChange(ctx *gin.Context) {
	uid := ctx.Param("id")
	userID, err := primitive.ObjectIDFromHex(uid)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "ID không hợp lệ"})
		return
	}

	var body struct {
		NewEmail string `json:"newEmail"`
		OTP      string `json:"otp"`
	}
	if err := ctx.ShouldBindJSON(&body); err != nil || body.NewEmail == "" || body.OTP == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu newEmail hoặc otp"})
		return
	}

	if err := ec.OTPService.VerifyOTP(models.OTPPurposeChangeEmailNew, body.NewEmail, body.OTP); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := ec.UserService.UpdateEmail(userID, body.NewEmail); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Đổi email thành công", "newEmail": body.NewEmail})
}

// GetUserChannelsHandler lấy danh sách kênh người dùng đã tham gia và tính toán thời gian hoạt động cuối cùng
func (uc *UserController) GetUserChannelsHandler(ctx *gin.Context) {
	// Lấy id người dùng từ URL hoặc token
	userId := ctx.Param("id")
	userObjectID, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "ID người dùng không hợp lệ"})
		return
	}

	// Lấy tât cả các kênh mà người dùng đã tham gia
	collection := uc.UserService.DB.Collection("userChannels")
	cursor, err := collection.Find(context.Background(), bson.M{"userID": userObjectID})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lấy kênh người dùng tham gia"})
		return
	}
	defer cursor.Close(context.Background())

	var userChannels []map[string]interface{}
	for cursor.Next(context.Background()) {
		var userChannel models.UserChannel
		if err := cursor.Decode(&userChannel); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi giải mã dữ liệu"})
			return
		}

		// Tạo thông báo dựa trên thời gian hoạt động cuối cùng
		lastActiveMsg := uc.UserService.FormatLastActive(userChannel.LastActive)

		// Gán thông báo vào phản hồi
		channelResponse := map[string]interface{}{
			"id":            userChannel.ID.Hex(),
			"userID":        userChannel.UserID.Hex(),
			"channelID":     userChannel.ChannelID.Hex(),
			"lastActive":    userChannel.LastActive,
			"lastActiveMsg": lastActiveMsg,
		}
		userChannels = append(userChannels, channelResponse)
	}

	ctx.JSON(http.StatusOK, gin.H{"userChannels": userChannels})
}

// ChangePhoneHandler xử lý yêu cầu đổi số điện thoại
func (uc *UserController) ChangePhoneHandler(ctx *gin.Context) {
	uid := ctx.Param("id")
	userID, err := primitive.ObjectIDFromHex(uid)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "ID không hợp lệ"})
		return
	}

	var body struct {
		Password string `json:"password"`
		NewPhone string `json:"newPhone"`
	}
	if err := ctx.ShouldBindJSON(&body); err != nil || body.Password == "" || body.NewPhone == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu mật khẩu hoặc số điện thoại mới"})
		return
	}

	// Xác thực mật khẩu
	ok, err := uc.UserService.VerifyUserPassword(userID, body.Password)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi kiểm tra mật khẩu"})
		return
	}
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Mật khẩu không đúng"})
		return
	}

	// Đổi số điện thoại
	if err := uc.UserService.UpdatePhone(userID, body.NewPhone); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Đổi số điện thoại thành công", "newPhone": body.NewPhone})
}

// ChangePasswordHandler xử lý đổi mật khẩu
func (uc *UserController) ChangePasswordHandler(c *gin.Context) {
	userID := c.Param("id")

	var req struct {
		OldPassword string `json:"oldPassword"`
		NewPassword string `json:"newPassword"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}

	// gọi service
	err := uc.UserService.ChangePassword(userID, req.OldPassword, req.NewPassword)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Đổi mật khẩu thành công"})
}

// UpdateAvatarHandler xử lý upload avatar
func (uc *UserController) UpdateAvatarHandler(c *gin.Context) {
	userID := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		log.Println("[UpdateAvatarHandler] Lỗi convert ObjectID:", err)
		c.JSON(400, gin.H{"error": "UserID không hợp lệ"})
		return
	}

	// Log query default
	log.Println("[UpdateAvatarHandler] Query default =", c.Query("default"))

	if c.Query("default") == "true" {
		defaultPath := os.Getenv("DEFAULT_AVATAR_URL")
		if defaultPath == "" {
			defaultPath = "/uploads/deadlineDi.jpg" // fallback cho local
		}
		log.Println("[UpdateAvatarHandler] Đặt avatar mặc định cho user:", userID)

		if err := uc.UserService.UpdateAvatar(objID, defaultPath); err != nil {
			log.Println("[UpdateAvatarHandler] Lỗi UpdateAvatar:", err)
			c.JSON(500, gin.H{"error": "Không thể cập nhật avatar mặc định"})
			return
		}

		c.JSON(200, gin.H{
			"message": "Đặt avatar mặc định thành công",
			"avatar":  defaultPath,
		})
		return
	}

	file, err := c.FormFile("avatar")
	if err != nil {
		log.Println("[UpdateAvatarHandler] Không tìm thấy file upload:", err)
		c.JSON(400, gin.H{"error": "Không tìm thấy file upload"})
		return
	}

	log.Println("[UpdateAvatarHandler] Upload file:", file.Filename)

	// Mở file để upload thông qua FileService
	f, err := file.Open()
	if err != nil {
		log.Println("[UpdateAvatarHandler] Không thể mở file:", err)
		c.JSON(400, gin.H{"error": "Không thể mở file upload"})
		return
	}

	fs, err := services.GetDefaultFileService()
	if err != nil {
		log.Println("[UpdateAvatarHandler] Lỗi provider:", err)
		c.JSON(500, gin.H{"error": "Lỗi server"})
		return
	}

	saved, err := fs.SaveUpload(f, file)
	if err != nil {
		log.Println("[UpdateAvatarHandler] Lỗi khi upload:", err)
		c.JSON(500, gin.H{"error": "Không thể upload file"})
		return
	}

	if err := uc.UserService.UpdateAvatar(objID, saved.URL); err != nil {
		log.Println("[UpdateAvatarHandler] Lỗi UpdateAvatar:", err)
		c.JSON(500, gin.H{"error": "Không thể cập nhật avatar"})
		return
	}

	log.Println("[UpdateAvatarHandler] Thành công:", saved.URL)
	c.JSON(200, gin.H{"message": "Cập nhật avatar thành công", "avatar": saved.URL})
}

// Thay thế UpdateCoverPhotoHandler tương tự:

// UpdateCoverPhotoHandler xử lý upload cover photo
func (uc *UserController) UpdateCoverPhotoHandler(c *gin.Context) {
	userID := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		c.JSON(400, gin.H{"error": "UserID không hợp lệ"})
		return
	}

	file, err := c.FormFile("cover")
	if err != nil {
		c.JSON(400, gin.H{"error": "Không tìm thấy file"})
		return
	}

	// Mở file và upload qua FileService
	f, err := file.Open()
	if err != nil {
		c.JSON(400, gin.H{"error": "Không thể mở file"})
		return
	}

	fs, err := services.GetDefaultFileService()
	if err != nil {
		c.JSON(500, gin.H{"error": "Lỗi server"})
		return
	}

	saved, err := fs.SaveUpload(f, file)
	if err != nil {
		c.JSON(500, gin.H{"error": "Không thể upload file"})
		return
	}

	if err := uc.UserService.UpdateCoverPhoto(objID, saved.URL); err != nil {
		c.JSON(500, gin.H{"error": "Không thể cập nhật ảnh bìa"})
		return
	}

	c.JSON(200, gin.H{"message": "Cập nhật ảnh bìa thành công", "cover": saved.URL})
}
