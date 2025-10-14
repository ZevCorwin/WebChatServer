package services

import (
	"chat-app-backend/config"
	"chat-app-backend/models"
	"context"
	"crypto/rand"
	"encoding/binary" // Thay thế cho binaryReadUint32
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"os"
	"time"
)

// OTPService struct bây giờ không còn chứa EmailService nữa.
type OTPService struct {
	DB *mongo.Database
}

// NewOTPService bây giờ chỉ cần DB.
func NewOTPService() *OTPService {
	return &OTPService{
		DB: config.DB,
	}
}

// Tôi giữ lại các hàm helper của bạn và sửa lại một chút cho an toàn hơn.
func (osv *OTPService) generate6Digits() (string, error) {
	var b [4]byte
	_, err := rand.Read(b[:])
	if err != nil {
		return "", err
	}
	n := binary.LittleEndian.Uint32(b[:])
	return fmt.Sprintf("%06d", n%1000000), nil
}

// Tạo & lưu OTP + gửi email (giữ nguyên logic gốc của bạn)
func (osv *OTPService) CreateAndSendRegisterOTP(email string, payload map[string]interface{}) error {
	collection := osv.DB.Collection("otps")

	var lastOTP models.OTP
	err := collection.FindOne(context.Background(), bson.M{
		"email":   email,
		"purpose": models.OTPPurposeRegister,
	}).Decode(&lastOTP)

	if err == nil {
		if time.Since(lastOTP.CreatedAt) < 30*time.Second {
			return fmt.Errorf("Bạn chỉ được yêu cầu mã OTP sau 30 giây")
		}
	}

	code, err := osv.generate6Digits()
	if err != nil {
		return err
	}

	// THAY ĐỔI LOGIC: Gửi email trước, thành công mới lưu vào DB
	appName := os.Getenv("APP_NAME")
	if appName == "" {
		appName = "WebChat" // Giá trị mặc định
	}
	subject := fmt.Sprintf("[%s] Mã xác thực đăng ký", appName)
	body := fmt.Sprintf(`
       <div style="font-family: Arial, sans-serif">
          <h2>Xin chào,</h2>
          <p>Mã xác thực đăng ký của bạn là:</p>
          <h1 style="letter-spacing: 4px">%s</h1>
          <p>Mã có hiệu lực trong 10 phút.</p>
          <p>— %s</p>
       </div>
    `, code, appName)

	// SỬA ĐỔI QUAN TRỌNG: Gọi trực tiếp hàm SendEmail mới
	if err := SendEmail(email, subject, body); err != nil {
		return err // Báo lỗi nếu gửi email thất bại
	}

	// Nếu gửi mail thành công, tiến hành lưu OTP
	otp := models.OTP{
		Email:     email,
		Code:      code,
		Purpose:   models.OTPPurposeRegister,
		ExpiresAt: time.Now().Add(10 * time.Minute),
		Attempts:  0,
		Payload:   payload,
		CreatedAt: time.Now(),
	}

	if _, err := collection.InsertOne(context.Background(), otp); err != nil {
		return err
	}

	return nil
}

// Hàm VerifyRegisterOTP của bạn được giữ nguyên vì nó không bị ảnh hưởng.
func (osv *OTPService) VerifyRegisterOTP(email, code string) (map[string]interface{}, error) {
	// ... code gốc của bạn ...
	collection := osv.DB.Collection("otps")
	var record models.OTP
	err := collection.FindOne(context.Background(), bson.M{
		"email":   email,
		"purpose": models.OTPPurposeRegister,
	}).Decode(&record)
	if err != nil {
		return nil, fmt.Errorf("OTP không tồn tại hoặc đã hết hạn")
	}

	if time.Now().After(record.ExpiresAt) {
		_, _ = collection.DeleteOne(context.Background(), bson.M{"_id": record.ID})
		return nil, fmt.Errorf("OTP đã hết hạn")
	}

	if record.Code != code {
		_ = collection.FindOneAndUpdate(context.Background(),
			bson.M{"_id": record.ID},
			bson.M{"$inc": bson.M{"attempts": 1}},
		)
		return nil, fmt.Errorf("Mã OTP không đúng")
	}

	_, _ = collection.DeleteOne(context.Background(), bson.M{"_id": record.ID})
	return record.Payload, nil
}

// Hàm CreateAndSendOTP chung của bạn được sửa lại tương tự
func (osv *OTPService) CreateAndSendOTP(purpose models.OTPPurpose, email string, subject string, htmlBodyFunc func(code string) string) error {
	collection := osv.DB.Collection("otps")

	var last models.OTP
	_ = collection.FindOne(context.Background(), bson.M{
		"email":   email,
		"purpose": purpose,
	}).Decode(&last)
	if !last.CreatedAt.IsZero() && time.Since(last.CreatedAt) < 30*time.Second {
		return fmt.Errorf("Bạn chỉ được yêu cầu mã OTP sau 30 giây")
	}

	code, err := osv.generate6Digits()
	if err != nil {
		return err
	}

	body := htmlBodyFunc(code)
	// SỬA ĐỔI QUAN TRỌNG: Gọi trực tiếp hàm SendEmail mới
	if err := SendEmail(email, subject, body); err != nil {
		return err
	}

	otp := models.OTP{
		Email:     email,
		Code:      code,
		Purpose:   purpose,
		ExpiresAt: time.Now().Add(10 * time.Minute),
		Attempts:  0,
		Payload:   map[string]interface{}{},
		CreatedAt: time.Now(),
	}

	if _, err := collection.InsertOne(context.Background(), otp); err != nil {
		return err
	}

	return nil
}

// Hàm VerifyOTP của bạn được giữ nguyên.
func (osv *OTPService) VerifyOTP(purpose models.OTPPurpose, email, code string) error {
	// ... code gốc của bạn ...
	collection := osv.DB.Collection("otps")
	var record models.OTP
	if err := collection.FindOne(context.Background(), bson.M{
		"email":   email,
		"purpose": purpose,
	}).Decode(&record); err != nil {
		return fmt.Errorf("OTP không tồn tại hoặc đã hết hạn")
	}

	if time.Now().After(record.ExpiresAt) {
		_, _ = collection.DeleteOne(context.Background(), bson.M{"_id": record.ID})
		return fmt.Errorf("OTP đã hết hạn")
	}

	if record.Code != code {
		_ = collection.FindOneAndUpdate(context.Background(),
			bson.M{"_id": record.ID},
			bson.M{"$inc": bson.M{"attempts": 1}},
		)
		return fmt.Errorf("Mã OTP không đúng")
	}

	_, _ = collection.DeleteOne(context.Background(), bson.M{"_id": record.ID})
	return nil
}
