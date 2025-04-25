package _struct

import (
	"time"

	"gorm.io/gorm"
)

type TrackQuality struct {
	ID                 uint           `json:"id" gorm:"primaryKey"`
	TrackID            uint           `json:"track_id" gorm:"uniqueIndex"`
	AvailableQualities []string       `json:"available_qualities" gorm:"type:json"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}
