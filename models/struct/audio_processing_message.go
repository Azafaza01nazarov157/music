package _struct

import (
	"fmt"
	"log"
)

type AudioProcessingMessage struct {
	TrackID    string `json:"track_id"`
	UserID     string `json:"user_id"`
	FilePath   string `json:"file_path"`
	FileName   string `json:"file_name"`
	FileFormat string `json:"file_format"`
}

func (m *AudioProcessingMessage) ToTrack() Track {
	var trackID uint
	if _, err := fmt.Sscanf(m.TrackID, "%d", &trackID); err != nil {
		log.Printf("Warning: Could not parse track ID as uint: %v", err)
	}

	var userID uint
	if _, err := fmt.Sscanf(m.UserID, "%d", &userID); err != nil {
		log.Printf("Warning: Could not parse user ID as uint: %v", err)
	}

	return Track{
		ID:         trackID,
		UserID:     userID,
		FilePath:   m.FilePath,
		FileFormat: m.FileFormat,
	}
}
