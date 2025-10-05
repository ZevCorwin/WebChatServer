package services

import (
	"chat-app-backend/config"
	"chat-app-backend/models"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type FileService struct{}

func NewFileService() *FileService {
	return &FileService{}
}

// SaveUpload: lưu file lên server và tạo record trong Mongo
func (fs *FileService) SaveUpload(file multipart.File, fh *multipart.FileHeader) (*models.File, error) {
	defer file.Close()

	// Đọc MIME type
	buf := make([]byte, 512)
	n, _ := file.Read(buf)
	mime := http.DetectContentType(buf[:n])
	_, err := file.Seek(0, 0) // reset lại con trỏ sau khi đọc MIME
	if err != nil {
		return nil, fmt.Errorf("không thể reset file reader: %v", err)
	}

	// Giới hạn kích thước (ví dụ: 20MB)
	const maxSize = 20 << 20 // 20MB
	if fh.Size > maxSize {
		return nil, fmt.Errorf("file quá lớn, tối đa 20MB")
	}

	// Phân loại loại file
	var fileType models.FileType
	if strings.HasPrefix(mime, "image/") {
		fileType = models.FileTypeImage
	} else if strings.HasPrefix(mime, "video/") {
		fileType = models.FileTypeVideo
	} else if strings.HasPrefix(mime, "audio/") {
		fileType = models.FileTypeAudio
	} else {
		fileType = models.FileTypeDocument
	}

	// Tạo đường dẫn lưu file
	uploadDir := "./uploads"
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		os.MkdirAll(uploadDir, os.ModePerm)
	}

	ext := filepath.Ext(fh.Filename)
	filename := primitive.NewObjectID().Hex() + ext
	fullPath := filepath.Join(uploadDir, filename)

	// Lưu file xuống disk
	dst, err := os.Create(fullPath)
	if err != nil {
		return nil, fmt.Errorf("không thể tạo file: %v", err)
	}
	defer dst.Close()

	if _, err := file.Seek(0, 0); err != nil {
		return nil, err
	}
	if _, err := dst.ReadFrom(file); err != nil {
		return nil, fmt.Errorf("lỗi khi ghi file: %v", err)
	}

	// URL công khai
	base := os.Getenv("PUBLIC_BASE_URL")
	if base == "" {
		base = "http://localhost:8080"
	}
	fileURL := base + "/uploads/" + filename

	// Tạo bản ghi file
	record := &models.File{
		ID:         primitive.NewObjectID(),
		FileName:   fh.Filename,
		FileType:   fileType,
		FileSize:   fh.Size,
		UploadTime: time.Now(),
		URL:        fileURL,
	}

	// Lưu record vào Mongo
	collection := config.DB.Collection("files")
	if _, err := collection.InsertOne(context.Background(), record); err != nil {
		return nil, fmt.Errorf("lỗi khi lưu file record: %v", err)
	}

	return record, nil
}
