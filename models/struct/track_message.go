package _struct

import (
	"time"
)

type TrackMessage struct {
	ID          uint    `json:"id"`
	Title       string  `json:"title"`
	ArtistID    uint    `json:"artistId"`
	AlbumID     uint    `json:"albumId"`
	UserID      uint    `json:"userId"`
	FilePath    string  `json:"filePath"`
	FileSize    int64   `json:"fileSize"`
	FileFormat  string  `json:"fileFormat"`
	Duration    float64 `json:"duration"`
	BitRate     int     `json:"bitRate"`
	SampleRate  int     `json:"sampleRate"`
	TrackNumber int     `json:"trackNumber"`
	Genre       string  `json:"genre"`
	PlayCount   int64   `json:"playCount"`
	CreatedAt   string  `json:"createdAt"`
	UpdatedAt   string  `json:"updatedAt"`
	IsDeleted   bool    `json:"isDeleted"`
}

func (m *TrackMessage) ToTrack() Track {
	createdAt, _ := time.Parse(time.RFC3339, m.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, m.UpdatedAt)

	return Track{
		ID:          m.ID,
		Title:       m.Title,
		ArtistID:    m.ArtistID,
		AlbumID:     &m.AlbumID,
		UserID:      m.UserID,
		FilePath:    m.FilePath,
		FileSize:    m.FileSize,
		FileFormat:  m.FileFormat,
		Duration:    m.Duration,
		BitRate:     m.BitRate,
		SampleRate:  m.SampleRate,
		TrackNumber: m.TrackNumber,
		Genre:       m.Genre,
		PlayCount:   m.PlayCount,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}
}
