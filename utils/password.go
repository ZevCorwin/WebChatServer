package utils

import (
	"errors"
	"golang.org/x/crypto/bcrypt"
)

// Mã hóa mật khẩu
func HashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", errors.New("Lỗi mã hóa mật khẩu")
	}
	return string(hashed), nil
}

// Kiểm tra mật khẩu khớp với hash
func VerifyPassword(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
