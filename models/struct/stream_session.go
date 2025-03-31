package _struct

import (
	"time"

	"gorm.io/gorm"
)

type StreamSession struct {
	ID           uint           `json:"id" gorm:"primaryKey"`
	UserID       uint           `json:"user_id" gorm:"not null"`
	TrackID      uint           `json:"track_id" gorm:"not null"`
	SessionID    string         `json:"session_id" gorm:"uniqueIndex;not null"`
	Quality      string         `json:"quality" gorm:"default:high"`
	CurrentPos   float64        `json:"current_pos"`
	BufferSize   int            `json:"buffer_size"`
	IsActive     bool           `json:"is_active" gorm:"default:true"`
	IPAddress    string         `json:"ip_address"`
	UserAgent    string         `json:"user_agent"`
	StartedAt    time.Time      `json:"started_at"`
	LastAccessAt time.Time      `json:"last_access_at"`
	EndedAt      *time.Time     `json:"ended_at"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}
