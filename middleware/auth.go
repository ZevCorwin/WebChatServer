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
		authHeader := c.GetHeader("Authorization")

		// Kiểm tra xem có Header Authorization hay không
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token không được cung cấp"})
			c.Abort()
			return
		}

		// Xử lý token từ Header
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		secret := os.Getenv("JWT_SECRET")

		// Kiểm tra nếu biến môi trường JWT_SECRET không được cấu hình
		if secret == "" {
			log.Println("JWT_SECRET không được cấu hình")
			c.JSON(http.StatusInternalServerError, gin.H{"err": "Lỗi máy chủ"})
			c.Abort()
			return
		}

		// Kiểm tra phân tích token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Kiểm tra phương pháp ký token
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				log.Println("Phương pháp ký không hợp lệ", token.Method)
				return nil, errors.New("phương pháp ký không hợp lệ")
			}
			return []byte(secret), nil
		})

		if err != nil {
			// Xử lý các loại lỗi token khác nhau
			if strings.Contains(err.Error(), "token is expired") {
				log.Println("token đã hết hạn", err)
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Token đã hết hạn"})
			} else if strings.Contains(err.Error(), "signature is invalid") {
				log.Println("Chữ ký token không hợp lệ", err)
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Chữ ký token không hợp lệ"})
			} else {
				log.Println("Token không hợp lệ", err)
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Token không hợp lệ"})
			}
			c.Abort()
			return
		}

		// Nếu token hợp lệ, kiểm tra claims
		if clamis, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			userID, ok := clamis["user_id"].(string)
			if !ok || userID == "" {
				log.Println("Token không chứa thông tin user_id hợp lệ")
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Token không chưa thông tin người dùng hợp lệ"})
				c.Abort()
				return
			}

			// Lưu user_id vào context
			log.Println("Xác thực thành công, user_id:", userID)
			c.Set("user_id", userID)
		} else {
			log.Println("Token không hợp lệ hoặc claims không hợp lệ")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token không hợp lệ"})
			c.Abort()
			return
		}

		// Tiếp tục xử lý request
		c.Next()
	}
}
