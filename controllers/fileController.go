package controllers

import (
	"net/http"

	"chat-app-backend/services"
	"github.com/gin-gonic/gin"
)

type FileController struct {
	FileService *services.FileService
}

func NewFileController(fs *services.FileService) *FileController {
	return &FileController{FileService: fs}
}

// POST /uploads  (form-data: file)
func (fc *FileController) Upload(ctx *gin.Context) {
	// Giới hạn dung lượng request (25MB)
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, 25<<20)

	// Lấy file từ form-data
	fh, err := ctx.FormFile("file")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu file hoặc file không hợp lệ"})
		return
	}

	// Mở stream đọc file từ FileHeader
	f, err := fh.Open()
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Không thể mở file upload"})
		return
	}
	// Lưu ý: FileService.SaveUpload sẽ tự Close() f, nên không Close ở đây.

	// Lưu file + tạo record DB
	saved, err := fc.FileService.SaveUpload(f, fh)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Trả kết quả
	ctx.JSON(http.StatusOK, gin.H{
		"id":       saved.ID,
		"url":      saved.URL,
		"size":     saved.FileSize,
		"fileType": saved.FileType,
		// "mime":   ... // nếu muốn trả thêm, có thể detect ở FileService và thêm field
	})
}
