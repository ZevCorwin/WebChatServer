package storage

import (
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
)

type LocalProvider struct {
	UploadDir   string
	PublicBase  string // ví dụ: http://localhost:8080
	PublicRoute string // ví dụ: /uploads
}

func NewLocalProvider(uploadDir, publicBase, publicRoute string) *LocalProvider {
	// ensure dir exists
	_ = os.MkdirAll(uploadDir, os.ModePerm)
	return &LocalProvider{
		UploadDir:   uploadDir,
		PublicBase:  publicBase,
		PublicRoute: publicRoute,
	}
}

func (p *LocalProvider) Upload(file multipart.File, filename string) (string, error) {
	dstPath := filepath.Join(p.UploadDir, filename)

	dst, err := os.Create(dstPath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	// reset file reader to start
	if _, err := file.Seek(0, 0); err != nil {
		// ignore seek error, continue
	}
	if _, err := dst.ReadFrom(file); err != nil {
		return "", err
	}

	base := p.PublicBase
	if base == "" {
		base = "http://localhost:8080"
	}
	// ensure public route has leading slash
	route := p.PublicRoute
	if route == "" {
		route = "/uploads"
	}
	return base + route + "/" + filename, nil
}

func (p *LocalProvider) UploadFromPath(path string) (string, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "", fmt.Errorf("file does not exist: %s", path)
	}
	filename := filepath.Base(path)
	base := p.PublicBase
	if base == "" {
		base = "http://localhost:8080"
	}
	route := p.PublicRoute
	if route == "" {
		route = "/uploads"
	}
	return base + route + "/" + filename, nil
}

func (p *LocalProvider) Delete(url string) error {
	// optional: parse filename from url and delete from disk
	return nil
}
