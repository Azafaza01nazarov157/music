package storage

import (
	"context"
	"fmt"
	"github.com/minio/minio-go/v7"
	"io"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/minio/minio-go/v7/pkg/credentials"
)

var MinioClient *minio.Client
var ctx = context.Background()

func ConnectMinio() {
	endpoint := getEnv("MINIO_ENDPOINT", "localhost:9000")
	accessKey := getEnv("MINIO_ACCESS_KEY", "adminUser")
	secretKey := getEnv("MINIO_SECRET_KEY", "adminUser")
	useSSL := getEnv("MINIO_USE_SSL", "false")

	ssl, err := strconv.ParseBool(useSSL)
	if err != nil {
		log.Printf("Invalid MINIO_USE_SSL value: %v, using default (false)", err)
		ssl = false
	}

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: ssl,
	})

	if err != nil {
		log.Fatalf("Failed to initialize MinIO client: %v", err)
	}

	MinioClient = client
	log.Println("Connected to MinIO storage")

	ensureBucketExists("audio-tracks")
	ensureBucketExists("audio-cache")
}

func ensureBucketExists(bucketName string) {
	exists, err := MinioClient.BucketExists(ctx, bucketName)
	if err != nil {
		log.Printf("Error checking bucket %s: %v", bucketName, err)
		return
	}

	if !exists {
		err = MinioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
		if err != nil {
			log.Printf("Error creating bucket %s: %v", bucketName, err)
			return
		}
		log.Printf("Created bucket: %s", bucketName)
	}
}

func UploadAudioFile(bucketName, objectName string, reader io.Reader, size int64, contentType string) error {
	_, err := MinioClient.PutObject(ctx, bucketName, objectName, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	return err
}

func GetAudioFile(bucketName, objectName string) (*minio.Object, error) {
	return MinioClient.GetObject(ctx, bucketName, objectName, minio.GetObjectOptions{})
}

func GetAudioFileStream(bucketName, objectName string, offset, length int64) (*minio.Object, error) {
	opts := minio.GetObjectOptions{}

	if offset > 0 || length > 0 {
		err := opts.SetRange(offset, offset+length-1)
		if err != nil {
			return nil, fmt.Errorf("error setting range: %w", err)
		}
	}

	return MinioClient.GetObject(ctx, bucketName, objectName, opts)
}

func GetAudioFileURL(bucketName, objectName string, expiry time.Duration) (string, error) {
	url, err := MinioClient.PresignedGetObject(ctx, bucketName, objectName, expiry, nil)
	if err != nil {
		return "", err
	}
	return url.String(), nil
}

func DeleteAudioFile(bucketName, objectName string) error {
	return MinioClient.RemoveObject(ctx, bucketName, objectName, minio.RemoveObjectOptions{})
}

func CopyAudioFile(srcBucket, srcObject, destBucket, destObject string) error {
	srcOpts := minio.CopySrcOptions{
		Bucket: srcBucket,
		Object: srcObject,
	}

	dstOpts := minio.CopyDestOptions{
		Bucket: destBucket,
		Object: destObject,
	}

	_, err := MinioClient.CopyObject(ctx, dstOpts, srcOpts)
	return err
}

func GetFileInfo(bucketName, objectName string) (minio.ObjectInfo, error) {
	return MinioClient.StatObject(ctx, bucketName, objectName, minio.StatObjectOptions{})
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
