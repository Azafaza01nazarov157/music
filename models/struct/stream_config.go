package _struct

import (
	"time"

	"gorm.io/gorm"
)

type StreamConfig struct {
	ID                uint           `json:"id" gorm:"primaryKey"`
	DefaultBufferSize int            `json:"default_buffer_size" gorm:"default:1024"`
	MaxBufferSize     int            `json:"max_buffer_size" gorm:"default:4096"`
	ChunkSize         int            `json:"chunk_size" gorm:"default:256"`
	DefaultQuality    string         `json:"default_quality" gorm:"default:high"`
	EnableTranscoding bool           `json:"enable_transcoding" gorm:"default:true"`
	MaxConcurrent     int            `json:"max_concurrent" gorm:"default:1000"`
	AccessLogEnabled  bool           `json:"access_log_enabled" gorm:"default:true"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}
