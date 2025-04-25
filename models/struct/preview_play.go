package _struct

import (
	"time"

	"gorm.io/gorm"
)

type PreviewPlay struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	TrackID   uint           `json:"track_id" gorm:"index"`
	UserID    uint           `json:"user_id" gorm:"index"`
	IPAddress string         `json:"ip_address"`
	UserAgent string         `json:"user_agent"`
	PlayedAt  time.Time      `json:"played_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}
