package _struct

import (
	"time"

	"gorm.io/gorm"
)

type ConversionStatus string

const (
	ConversionPending    ConversionStatus = "pending"
	ConversionProcessing ConversionStatus = "processing"
	ConversionCompleted  ConversionStatus = "completed"
	ConversionFailed     ConversionStatus = "failed"
)

type ConversionJob struct {
	ID             uint             `json:"id" gorm:"primaryKey"`
	TrackID        uint             `json:"track_id" gorm:"not null"`
	SourceFormatID uint             `json:"source_format_id" gorm:"not null"`
	TargetFormatID uint             `json:"target_format_id" gorm:"not null"`
	Status         ConversionStatus `json:"status" gorm:"default:pending"`
	Priority       int              `json:"priority" gorm:"default:1"`
	Progress       float64          `json:"progress" gorm:"default:0"`
	ErrorMessage   string           `json:"error_message"`
	StartedAt      *time.Time       `json:"started_at"`
	CompletedAt    *time.Time       `json:"completed_at"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
	DeletedAt      gorm.DeletedAt   `json:"deleted_at" gorm:"index"`
}
