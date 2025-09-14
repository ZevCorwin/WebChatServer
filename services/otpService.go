package services

import (
	"chat-app-backend/config"
	"chat-app-backend/models"
	"context"
	"crypto/rand"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"time"
)

type OTPService struct {
	DB           *mongo.Database
	EmailService *EmailService
}

func NewOTPService() *OTPService {
	return &OTPService{
		DB:           config.DB,
		EmailService: NewEmailService(),
	}
}

func (osv *OTPService) generate6Digits() (string, error) {
	// 000000 - 999999
	var n uint32
	if err := binaryReadUint32(&n); err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n%1000000), nil
}

func binaryReadUint32(out *uint32) error {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return err
	}
	*out = uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
	return nil
}

// Tạo & lưu OTP + gửi email
func (osv *OTPService) CreateAndSendRegisterOTP(email string, payload map[string]interface{}) error {
	collection := osv.DB.Collection("otps")

	var lastOTP models.OTP
	err := collection.FindOne(context.Background(), bson.M{
		"email":   email,
		"purpose": models.OTPPurposeRegister,
	}).Decode(&lastOTP)

	// Nếu tìm thấy OTP cũ → kiểm tra thời gian tạo
	if err == nil {
		if time.Since(lastOTP.CreatedAt) < 30*time.Second {
			return fmt.Errorf("bạn chỉ được yêu cầu mã OTP sau 30 giây")
		}
	} else if err != mongo.ErrNoDocuments {
		// Chỉ trả lỗi nếu là lỗi khác, không phải không tìm thấy
		return fmt.Errorf("lỗi truy vấn OTP: %v", err)
	}

	code, err := osv.generate6Digits()
	if err != nil {
		return err
	}

	otp := models.OTP{
		Email:     email,
		Code:      code, // production: có thể hash
		Purpose:   models.OTPPurposeRegister,
		ExpiresAt: time.Now().Add(10 * time.Minute),
		Attempts:  0,
		Payload:   payload,
		CreatedAt: time.Now(),
	}

	if _, err := collection.InsertOne(context.Background(), otp); err != nil {
		return err
	}

	subject := fmt.Sprintf("[%s] Mã xác thực đăng ký", osv.EmailService.app)
	body := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif">
			<h2>Xin chào,</h2>
			<p>Mã xác thực đăng ký của bạn là:</p>
			<h1 style="letter-spacing: 4px">%s</h1>
			<p>Mã có hiệu lực trong 10 phút.</p>
			<p>— %s</p>
		</div>
	`, code, osv.EmailService.app)

	return osv.EmailService.Send(email, subject, body)
}

// Xác thực OTP; trả về payload nếu hợp lệ
func (osv *OTPService) VerifyRegisterOTP(email, code string) (map[string]interface{}, error) {
	collection := osv.DB.Collection("otps")
	var record models.OTP
	err := collection.FindOne(context.Background(), bson.M{
		"email":   email,
		"purpose": models.OTPPurposeRegister,
	}).Decode(&record)
	if err != nil {
		return nil, fmt.Errorf("OTP không tồn tại hoặc đã hết hạn")
	}

	// hết hạn?
	if time.Now().After(record.ExpiresAt) {
		_, _ = collection.DeleteOne(context.Background(), bson.M{"_id": record.ID})
		return nil, fmt.Errorf("OTP đã hết hạn")
	}

	// sai mã?
	if record.Code != code {
		_ = collection.FindOneAndUpdate(context.Background(),
			bson.M{"_id": record.ID},
			bson.M{"$inc": bson.M{"attempts": 1}},
		)
		return nil, fmt.Errorf("Mã OTP không đúng")
	}

	// đúng → xoá luôn để one-time
	_, _ = collection.DeleteOne(context.Background(), bson.M{"_id": record.ID})
	return record.Payload, nil
}
