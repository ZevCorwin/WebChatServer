package config

import (
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"os"
)

// Config chứa các biến môi trường cần thiết
type Config struct {
	AppPort   string
	DBHost    string
	DBPort    string
	DBName    string
	JWTSecret string
	//RedisHost     string
	MongoURI      string
	WebSocketPort string
	WebSocketPath string
}

// LoadEnv nạp biến môi trường từ tệp .env dựa trên APP_ENV
func LoadEnv() {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development" // Mặc định là "development"
	}
	envFile := fmt.Sprintf(".env.%s", env)

	// Nạp biến môi trường từ tệp
	if err := godotenv.Load(envFile); err != nil {
		log.Fatalf("Lỗi: không thể nạp tệp %s: %v", envFile, err)
	}
}

// LoadConfig nạp và trả về cấu hình từ các biến môi trường
func LoadConfig() Config {
	LoadEnv() // Nạp biến môi trường từ file .env trước khi đọc

	config := Config{
		AppPort:   os.Getenv("APP_PORT"),
		DBHost:    os.Getenv("DB_HOST"),
		DBPort:    os.Getenv("DB_PORT"),
		DBName:    os.Getenv("DB_NAME"),
		JWTSecret: os.Getenv("JWT_SECRET"),
		//RedisHost:     os.Getenv("REDIS_HOST"),
		MongoURI:      os.Getenv("MONGODB_URI"),
		WebSocketPort: os.Getenv("WEBSOCKET_PORT"),
		WebSocketPath: os.Getenv("WEBSOCKET_PATH"),
	}

	// Kiểm tra và báo lỗi nếu thiếu bất kỳ biến môi trường bắt buộc nào
	if config.AppPort == "" {
		log.Fatal("Lỗi cấu hình: Biến môi trường APP_PORT không được để trống")
	}
	if config.DBHost == "" {
		log.Fatal("Lỗi cấu hình: Biến môi trường DB_HOST không được để trống")
	}
	if config.DBPort == "" {
		log.Fatal("Lỗi cấu hình: Biến môi trường DB_PORT không được để trống")
	}
	if config.DBName == "" {
		log.Fatal("Lỗi cấu hình: Biến môi trường DB_NAME không được để trống")
	}
	if config.JWTSecret == "" {
		log.Fatal("Lỗi cấu hình: Biến môi trường JWT_SECRET không được để trống")
	}
	//if config.RedisHost == "" {
	//	log.Fatal("Lỗi cấu hình: Biến môi trường REDIS_HOST không được để trống")
	//}
	if config.WebSocketPort == "" {
		log.Fatal("Lỗi cấu hình: Biến môi trường WEBSOCKET_PORT không được để trống")
	}

	// ❗ Nếu KHÔNG có MongoURI thì mới yêu cầu DB_HOST/DB_PORT
	if config.MongoURI == "" {
		if config.DBHost == "" {
			log.Fatal("Lỗi cấu hình: DB_HOST không được để trống (hoặc dùng MONGODB_URI)")
		}
		if config.DBPort == "" {
			log.Fatal("Lỗi cấu hình: DB_PORT không được để trống (hoặc dùng MONGODB_URI)")
		}
	}

	return config
}
