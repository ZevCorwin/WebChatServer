package services

import (
	"chat-app-backend/config"
	"chat-app-backend/models"
	"chat-app-backend/utils"
	"context"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"log"
	"os"
	"strconv"
	"time"
)

// UserService chứa các phương thức để xử lý nghiệp vụ liên quan đến người dùng
type UserService struct {
	DB *mongo.Database
}

// NewUserService khởi tạo một UserService mới
func NewUserService() *UserService {
	return &UserService{DB: config.DB}
}

func (us *UserService) CheckPasswordHash(password, hash string) bool {
	return utils.VerifyPassword(password, hash)
}

// Register thực hiện đăng ký cho người dùng mới
func (us *UserService) Register(user models.User) (*models.User, error) {
	collection := us.DB.Collection("users")

	// Kiểm tra nếu email và số điện thoại đã tồn tại
	var existingUser models.User
	err := collection.FindOne(context.Background(), bson.M{
		"$or": []bson.M{
			{"email": user.Email},
			{"phone": user.Phone},
		},
	}).Decode(&existingUser)
	if err == nil {
		return nil, errors.New("Email hoặc số điện thoại đã tồn tại")
	}

	// Mã hóa mật khẩu (sử dụng bcrypt)
	hashedPassword, err := utils.HashPassword(user.Password)
	if err != nil {
		return nil, err
	}
	user.Password = hashedPassword

	// Thêm thông tin ngày tạo tài khoản và mặc định trạng thái
	user.AccountCreatedDate = time.Now()
	user.Status = models.StatusOffline

	// Gán ảnh đại diện mặc định nếu không có
	if user.Avatar == "" {
		user.Avatar = "/uploads/deadlineDi.jpg" // Đường dẫn ảnh mặc định
	}

	// Chèn người dùng mới vào cơ sở dữ liệu
	_, err = collection.InsertOne(context.Background(), user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// GenerateJWT tạo JWT cho người dùng
func (us *UserService) GenerateJWT(userID primitive.ObjectID) (string, error) {
	expirationHours, err := strconv.Atoi(os.Getenv("JWT_EXPIRATION_HOURS"))
	if err != nil || expirationHours <= 0 {
		expirationHours = 24 // Mặc định 24 giờ
	}

	claims := jwt.MapClaims{
		"user_id": userID.Hex(),
		"exp":     time.Now().Add(time.Hour * time.Duration(expirationHours)).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	secret := os.Getenv("JWT_SECRET")
	return token.SignedString([]byte(secret))
}

func (us *UserService) GetAllUsers() ([]models.User, error) {
	collection := us.DB.Collection("users")
	var users []models.User

	// Lấy tất cả người dùng từ cơ sở dữ liệu
	cursor, err := collection.Find(context.Background(), bson.D{})
	if err != nil {
		log.Printf("Lỗi khi lấy người dùng từ DB: %v", err)
		return nil, errors.New("Không thể lấy thông tin người dùng. Vui lòng thử lại sau")
	}
	ctx := context.Background()
	defer func(cursor *mongo.Cursor, ctx context.Context) {
		err := cursor.Close(ctx)
		if err != nil {
			log.Println("Lỗi khi đóng cursor:", err)
		}
	}(cursor, ctx)

	// Duyệt qua kết quả và thêm vào slice users
	for cursor.Next(context.Background()) {
		var user models.User
		if err := cursor.Decode(&user); err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

func (us *UserService) GetUserByID(userID string) (*models.User, error) {
	collection := us.DB.Collection("users")
	var user models.User

	// Chuyển userID sang ObjectID
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		log.Printf("Lỗi khi lấy người dùng từ DB: %v", err)
		return nil, errors.New("Không thể lấy thông tin người dùng. Vui lòng thử lại sau")
	}

	// Lấy người dùng theo ID
	err = collection.FindOne(context.Background(), bson.M{"_id": objectID}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("Nguười dùng không tồn tại")
		}
		return nil, err
	}
	return &user, nil
}

// Tìm kiếm người dùng bằng số điện thoại
func (us *UserService) GetUserByPhone(phone string) (*models.User, error) {
	collection := us.DB.Collection("users")
	var user models.User

	err := collection.FindOne(context.Background(), bson.M{"phone": phone}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("Người dùng không tồn tại")
		}
		return nil, err
	}
	return &user, nil
}

func (us *UserService) UpdateUserProfile(user *models.User) error {
	// Gọi hàm cập nhật từ model
	return user.UpdateProfileInDB()
}

// VerifyUserPassword: check password hiện tại của user
func (us *UserService) VerifyUserPassword(userID primitive.ObjectID, plain string) (bool, error) {
	col := us.DB.Collection("users")
	var u models.User
	if err := col.FindOne(context.Background(), bson.M{"_id": userID}).Decode(&u); err != nil {
		return false, err
	}
	return us.CheckPasswordHash(plain, u.Password), nil
}

// UpdateEmail: đổi email (đã verify OTP ở controller/service trước đó)
func (us *UserService) UpdateEmail(userID primitive.ObjectID, newEmail string) error {
	col := us.DB.Collection("users")

	// unique check
	var exists models.User
	err := col.FindOne(context.Background(), bson.M{"email": newEmail}).Decode(&exists)
	if err == nil {
		return fmt.Errorf("Email đã tồn tại")
	}

	_, err = col.UpdateOne(context.Background(),
		bson.M{"_id": userID},
		bson.M{"$set": bson.M{"email": newEmail}},
	)
	return err
}

// GetLastActiveTime tính toán thời gian người dùng đã ngừng hoạt động trong kênh
func (us *UserService) FormatLastActive(lastActive time.Time) string {
	// Tính toán thời gian từ khi hoạt động cuối cùng
	duration := time.Since(lastActive)

	switch {
	case duration < time.Minute:
		return "Vừa mới online"
	case duration < time.Hour:
		return fmt.Sprintf("%d phút trước", int(duration.Minutes()))
	case duration < time.Hour*24:
		return fmt.Sprintf("%d giờ trước", int(duration.Hours()))
	default:
		return fmt.Sprintf("%d ngày trước", int(duration.Hours()/24))
	}
}
