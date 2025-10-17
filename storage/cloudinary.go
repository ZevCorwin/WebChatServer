package storage

import (
	"context"
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

type CloudinaryProvider struct {
	cld *cloudinary.Cloudinary
}

func NewCloudinaryProviderFromURL(cloudinaryURL string) (*CloudinaryProvider, error) {
	if cloudinaryURL == "" {
		cloudinaryURL = os.Getenv("CLOUDINARY_URL")
	}
	if cloudinaryURL == "" {
		return nil, fmt.Errorf("CLOUDINARY_URL required")
	}
	cld, err := cloudinary.NewFromURL(cloudinaryURL)
	if err != nil {
		return nil, err
	}
	return &CloudinaryProvider{cld: cld}, nil
}

func (p *CloudinaryProvider) Upload(file multipart.File, filename string) (string, error) {
	defer file.Close()
	ctx := context.Background()
	publicID := filenameWithoutExt(filename)

	// Overwrite is a *bool in the SDK, so pass a pointer
	overwrite := true
	resp, err := p.cld.Upload.Upload(ctx, file, uploader.UploadParams{
		PublicID:  publicID,
		Overwrite: &overwrite,
	})
	if err != nil {
		return "", err
	}
	if resp.SecureURL != "" {
		return resp.SecureURL, nil
	}
	if resp.URL != "" {
		return resp.URL, nil
	}
	return "", fmt.Errorf("cloudinary upload returned empty url")
}

func (p *CloudinaryProvider) UploadFromPath(path string) (string, error) {
	ctx := context.Background()
	resp, err := p.cld.Upload.Upload(ctx, path, uploader.UploadParams{})
	if err != nil {
		return "", err
	}
	if resp.SecureURL != "" {
		return resp.SecureURL, nil
	}
	if resp.URL != "" {
		return resp.URL, nil
	}
	return "", fmt.Errorf("cloudinary upload returned empty url")
}

func (p *CloudinaryProvider) Delete(url string) error {
	// optional: if you store public_id, you can delete via p.cld.Upload.Destroy
	return nil
}

func filenameWithoutExt(fn string) string {
	ext := filepath.Ext(fn)
	if ext == "" {
		return fn
	}
	return fn[:len(fn)-len(ext)]
}
