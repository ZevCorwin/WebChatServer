package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type FileType string

const (
	FileTypeImage    FileType = "Image"
	FileTypeVideo    FileType = "Video"
	FileTypeAudio    FileType = "Audio"
	FileTypeDocument FileType = "Document"
)

// File là cấu trúc đại diện cho tệp liên quan đến tin nhắn
type File struct {
	ID         primitive.ObjectID `json:"id" bson:"_id,omitempty"`      // ID tự động của MongoDB
	FileName   string             `json:"fileName" bson:"fileName"`     // Tên tệp
	FileType   FileType           `json:"fileType" bson:"fileType"`     // Loại tệp
	FileSize   int64              `json:"fileSize" bson:"fileSize"`     // Kích thước tệp (tính bằng byte)
	UploadTime time.Time          `json:"uploadTime" bson:"uploadTime"` // Thời gian tải tệp lên
	URL        string             `json:"url" bson:"url"`               // Đường dẫn tải về hoặc xem tệp
}
