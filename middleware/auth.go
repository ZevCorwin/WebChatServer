package middleware

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"log"
	"net/http"
	"os"
	"strings"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token không được cung cấp"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		secret := os.Getenv("JWT_SECRET")

		if secret == "" {
			log.Fatal("JWT_SECRET không được cấu hình")
		}

		// Kiểm tra và phân tích token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Kiểm tra nếu phương thức ký là HMAC
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("phương pháp ký không hợp lệ")
			}
			return []byte(secret), nil
		})

		// Xử lý lỗi token
		if err != nil {
			var errorMsg string
			// Kiểm tra lỗi token bằng cách kiểm tra chuỗi lỗi
			if strings.Contains(err.Error(), "token is expired") {
				errorMsg = "Token đã hết hạn"
			} else if strings.Contains(err.Error(), "signature is invalid") {
				errorMsg = "Chữ ký token không hợp lệ"
			} else {
				errorMsg = "Token không hợp lệ"
			}
			c.JSON(http.StatusUnauthorized, gin.H{"error": errorMsg})
			c.Abort()
			return
		}

		// Lưu thông tin người dùng vào context để sử dụng trong các handler tiếp theo
		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			// Giả sử token chứa "user_id", bạn có thể thêm thông tin này vào context
			userID, ok := claims["user_id"].(string)
			if !ok || userID == "" {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Token không chứa thông tin người dùng hợp lệ"})
				c.Abort()
				return
			}
			c.Set("user_id", userID)
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token không hợp lệ"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// CurrentUserMiddleware - Middleware để lấy thông tin người dùng hiện tại từ context
func CurrentUserMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Lấy thông tin user_id từ context (đã được set bởi AuthMiddleware)
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Không tìm thấy thông tin người dùng"})
			c.Abort()
			return
		}

		// Set thông tin người dùng vào context để sử dụng trong các controller tiếp theo
		c.Set("currentUserID", userID)
		c.Next()
	}
}
