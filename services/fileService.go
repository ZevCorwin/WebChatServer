package services

import (
	"chat-app-backend/config"
	"chat-app-backend/models"
	"chat-app-backend/storage"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type FileService struct {
	Provider storage.Provider
}

func NewFileService(provider storage.Provider) *FileService {
	return &FileService{Provider: provider}
}

// Singleton default provider/service for convenience in controllers
var (
	defaultFS   *FileService
	defaultOnce sync.Once
)

func GetDefaultFileService() (*FileService, error) {
	var err error
	defaultOnce.Do(func() {
		var prov storage.Provider
		prov, err = storage.NewProviderFromEnv()
		if err != nil {
			return
		}
		defaultFS = NewFileService(prov)
	})
	if err != nil {
		return nil, err
	}
	return defaultFS, nil
}

// SaveUpload: lưu file lên provider và tạo record trong Mongo
func (fs *FileService) SaveUpload(file multipart.File, fh *multipart.FileHeader) (*models.File, error) {
	defer func() {
		// some providers (cloud) Close() inside, but ensure file closed here as a safety net
		_ = file.Close()
	}()

	// Đọc MIME type
	buf := make([]byte, 512)
	n, _ := file.Read(buf)
	mime := http.DetectContentType(buf[:n])
	_, err := file.Seek(0, 0) // reset
	if err != nil {
		return nil, fmt.Errorf("không thể reset file reader: %v", err)
	}

	// Giới hạn kích thước (20MB)
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

	// Tạo tên file duy nhất
	ext := filepath.Ext(fh.Filename)
	filename := primitive.NewObjectID().Hex() + ext

	// Gọi provider để upload
	uploadedURL, err := fs.Provider.Upload(file, filename)
	if err != nil {
		return nil, fmt.Errorf("upload failed: %v", err)
	}

	// Tạo bản ghi file
	record := &models.File{
		ID:         primitive.NewObjectID(),
		FileName:   fh.Filename,
		FileType:   fileType,
		FileSize:   fh.Size,
		UploadTime: time.Now(),
		URL:        uploadedURL,
	}

	collection := config.DB.Collection("files")
	if _, err := collection.InsertOne(context.Background(), record); err != nil {
		return nil, fmt.Errorf("lỗi khi lưu file record: %v", err)
	}

	return record, nil
}
