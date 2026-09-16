package storage

import "mime/multipart"

type Storage interface {
	Upload(file *multipart.FileHeader, filename string) (url string, err error)
	Delete(filename string) error
}
