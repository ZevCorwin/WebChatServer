package storage

import (
	"os"
)

func NewProviderFromEnv() (Provider, error) {
	prov := os.Getenv("STORAGE_PROVIDER") // "local" hoặc "cloudinary"
	if prov == "" || prov == "local" {
		publicBase := os.Getenv("PUBLIC_BASE_URL")
		if publicBase == "" {
			publicBase = "http://localhost:8080"
		}
		uploadDir := "./uploads"
		publicRoute := "/uploads"
		return NewLocalProvider(uploadDir, publicBase, publicRoute), nil
	}
	// cloudinary
	cloudinaryURL := os.Getenv("CLOUDINARY_URL")
	return NewCloudinaryProviderFromURL(cloudinaryURL)
}
