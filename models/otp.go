package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type OTPPurpose string

const (
	OTPPurposeRegister       OTPPurpose = "register"
	OTPPurposeChangeEmailOld OTPPurpose = "change_email_old" // OTP gửi tới email hiện tại
	OTPPurposeChangeEmailNew OTPPurpose = "change_email_new" // OTP gửi tới email mới
	OTPPurposeChangePhone    OTPPurpose = "change_phone"
)

type OTP struct {
	ID        primitive.ObjectID     `bson:"_id,omitempty" json:"_id,omitempty"`
	Email     string                 `bson:"email" json:"email"`
	Code      string                 `bson:"code" json:"code"`
	Purpose   OTPPurpose             `bson:"purpose" json:"purpose"`
	ExpiresAt time.Time              `bson:"expires_at" json:"expires_at"`
	Attempts  int                    `bson:"attempts" json:"attempts"`
	Payload   map[string]interface{} `bson:"payload" json:"payload"`
	CreatedAt time.Time              `bson:"created_at" json:"created_at"`
}
