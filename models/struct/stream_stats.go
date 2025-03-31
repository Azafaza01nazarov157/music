package _struct

import (
	"time"

	"gorm.io/gorm"
)

type StreamStats struct {
	ID              uint           `json:"id" gorm:"primaryKey"`
	TrackID         uint           `json:"track_id" gorm:"not null"`
	TotalStreams    int64          `json:"total_streams" gorm:"default:0"`
	TotalDuration   float64        `json:"total_duration" gorm:"default:0"`
	UniqueUsers     int            `json:"unique_users" gorm:"default:0"`
	AverageDuration float64        `json:"average_duration" gorm:"default:0"`
	CompletionRate  float64        `json:"completion_rate" gorm:"default:0"`
	LastStreamedAt  *time.Time     `json:"last_streamed_at"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}
