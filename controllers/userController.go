package controllers

import (
	"chat-app-backend/models"
	"chat-app-backend/services"
	"context"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"net/http"
)

// UserController chứa các phương thức xử lý yêu cầu của người dùng
type UserController struct {
	UserService *services.UserService
}

// NewUserController khởi tạo một UserController mới
func NewUserController(userService *services.UserService) *UserController {
	return &UserController{
		UserService: userService,
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

	//// Kiểm tra xem người dùng đã có danh sách bạn bè chưa
	//friendCollection := uc.UserService.DB.Collection("listFriends")
	//friendFilter := bson.M{"userID": user.ID}
	//count, err := friendCollection.CountDocuments(context.Background(), friendFilter)
	//if err != nil {
	//	ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi kiểm tra danh sách bạn bè"})
	//	return
	//}
	//
	//// Nếu không có danh sách bạn bè thì tạo danh sách bạn bè trống
	//if count == 0 {
	//	emptyList := models.ListFriends{
	//		UserID:     user.ID,
	//		FriendID:   user.ID, // Bạn có thể để friendID là chính người dùng để tạo một bản ghi trống
	//		FriendType: models.FriendTypeSelf,
	//	}
	//	_, err := friendCollection.InsertOne(context.Background(), emptyList)
	//	if err != nil {
	//		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi tạo danh sách bạn bè"})
	//		return
	//	}
	//}

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
