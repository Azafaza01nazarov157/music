package _struct

import (
	"time"

	"gorm.io/gorm"
)

type AudioCache struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	TrackID   uint           `json:"track_id" gorm:"uniqueIndex:idx_track_format;not null"`
	FormatID  uint           `json:"format_id" gorm:"uniqueIndex:idx_track_format;not null"`
	FilePath  string         `json:"file_path" gorm:"not null"`
	FileSize  int64          `json:"file_size"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	AccessAt  time.Time      `json:"access_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}
