package _struct

import (
	"time"

	"gorm.io/gorm"
)

type Track struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Title       string         `json:"title" gorm:"not null"`
	ArtistID    uint           `json:"artist_id" gorm:"not null"`
	AlbumID     *uint          `json:"album_id"`
	UserID      uint           `json:"user_id"`
	FilePath    string         `json:"file_path" gorm:"not null"`
	FileSize    int64          `json:"file_size"`
	FileFormat  string         `json:"file_format"`
	Duration    float64        `json:"duration"`
	BitRate     int            `json:"bit_rate"`
	SampleRate  int            `json:"sample_rate"`
	TrackNumber int            `json:"track_number"`
	Genre       string         `json:"genre"`
	PlayCount   int64          `json:"play_count" gorm:"default:0"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}
