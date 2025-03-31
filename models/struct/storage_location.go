package _struct

import (
	"time"

	"gorm.io/gorm"
)

type StorageLocationType string

const (
	StorageLocal  StorageLocationType = "local"
	StorageMinio  StorageLocationType = "minio"
	StorageS3     StorageLocationType = "s3"
	StorageCustom StorageLocationType = "custom"
)

type StorageLocation struct {
	ID        uint                `json:"id" gorm:"primaryKey"`
	Name      string              `json:"name" gorm:"not null"`
	Type      StorageLocationType `json:"type" gorm:"default:local"`
	Endpoint  string              `json:"endpoint"`
	Bucket    string              `json:"bucket"`
	Region    string              `json:"region"`
	AccessKey string              `json:"access_key"`
	SecretKey string              `json:"secret_key"`
	BasePath  string              `json:"base_path"`
	IsActive  bool                `json:"is_active" gorm:"default:true"`
	IsDefault bool                `json:"is_default" gorm:"default:false"`
	CreatedAt time.Time           `json:"created_at"`
	UpdatedAt time.Time           `json:"updated_at"`
	DeletedAt gorm.DeletedAt      `json:"deleted_at" gorm:"index"`
}
