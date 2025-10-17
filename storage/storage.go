package storage

import "mime/multipart"

// Provider là interface chung cho storage (local hoặc cloud)
type Provider interface {
	// Upload upload từ multipart.File, trả về public URL
	Upload(file multipart.File, filename string) (string, error)
	// UploadFromPath upload từ path (nếu cần)
	UploadFromPath(path string) (string, error)
	// Delete xóa file (tuỳ chọn)
	Delete(url string) error
}
