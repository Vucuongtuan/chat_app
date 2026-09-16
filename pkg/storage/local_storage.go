package storage

import (
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
)

type LocalStorage struct {
	baseDir string
	baseUrl string // "/uploads"
}

func NewLocalStorage(baseDir, baseUrl string) *LocalStorage {
	return &LocalStorage{baseDir: baseDir, baseUrl: baseUrl}
}

func (s *LocalStorage) Upload(file *multipart.FileHeader, filename string) (string, error) {
	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	dstPath := filepath.Join(s.baseDir, filename)
	dst, err := os.Create(dstPath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return "", err
	}

	return s.baseUrl + "/" + filename, nil
}

func (s *LocalStorage) Delete(filename string) error {
	return os.Remove(filepath.Join(s.baseDir, filename))
}
