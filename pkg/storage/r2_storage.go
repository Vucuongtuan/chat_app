// pkg/storage/r2_storage.go
package storage

// import (
// 	"context"
// 	"mime/multipart"

// 	"github.com/aws/aws-sdk-go-v2/aws"
// 	"github.com/aws/aws-sdk-go-v2/service/s3"
// )

// type R2Storage struct {
// 	client    *s3.Client
// 	bucket    string
// 	publicUrl string
// }

// func NewR2Storage(client *s3.Client, bucket, publicUrl string) *R2Storage {
// 	return &R2Storage{client: client, bucket: bucket, publicUrl: publicUrl}
// }

// func (s *R2Storage) Upload(file *multipart.FileHeader, filename string) (string, error) {
// 	src, err := file.Open()
// 	if err != nil {
// 		return "", err
// 	}
// 	defer src.Close()

// 	_, err = s.client.PutObject(context.TODO(), &s3.PutObjectInput{
// 		Bucket: aws.String(s.bucket),
// 		Key:    aws.String(filename),
// 		Body:   src,
// 	})
// 	if err != nil {
// 		return "", err
// 	}

// 	return s.publicUrl + "/" + filename, nil
// }

// func (s *R2Storage) Delete(filename string) error {
// 	_, err := s.client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
// 		Bucket: aws.String(s.bucket),
// 		Key:    aws.String(filename),
// 	})
// 	return err
// }
