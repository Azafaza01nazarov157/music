package storage

import (
	"context"
	"fmt"
	"io"
	"log"
	"music-conveyor/platform/config"
	"time"

	"github.com/minio/minio-go/v7"

	"github.com/minio/minio-go/v7/pkg/credentials"
)

var MinioClient *minio.Client
var ctx = context.Background()

func ConnectMinio() {
	cfg := config.LoadConfig()

	client, err := minio.New(cfg.MinioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinioAccessKey, cfg.MinioSecretKey, ""),
		Secure: cfg.MinioUseSSL,
	})

	if err != nil {
		log.Fatalf("Failed to initialize MinIO client: %v", err)
	}

	MinioClient = client
	log.Println("Connected to MinIO storage")

	// Ensure all required buckets exist
	EnsureRequiredBuckets()
}

// EnsureRequiredBuckets creates all required buckets if they don't exist
func EnsureRequiredBuckets() {
	requiredBuckets := []string{
		"music-originals",
		"audio-tracks",
		"audio-previews",
		"audio-cache",
	}

	for _, bucket := range requiredBuckets {
		EnsureBucketExists(bucket)
	}
}

// EnsureBucketExists ensures a bucket exists, creating it if necessary
func EnsureBucketExists(bucketName string) {
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
	} else {
		log.Printf("Bucket already exists: %s", bucketName)
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
