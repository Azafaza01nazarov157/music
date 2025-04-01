package controllers

import (
	"fmt"
	"io"
	"log"
	"music-conveyor/models/struct"
	"music-conveyor/platform/cache"
	"music-conveyor/platform/database"
	"music-conveyor/platform/storage"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}

func StreamTrack(c *gin.Context) {
	trackID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid track ID"})
		return
	}

	var track _struct.Track
	result := database.DB.First(&track, trackID)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Track not found"})
		return
	}

	quality := c.DefaultQuery("quality", "high")

	userID := uint(0)
	if user, exists := c.Get("user"); exists {
		userID = user.(_struct.User).ID
	}

	sessionID := uuid.New().String()
	streamSession := _struct.StreamSession{
		UserID:       userID,
		TrackID:      uint(trackID),
		SessionID:    sessionID,
		Quality:      quality,
		CurrentPos:   0,
		IsActive:     true,
		IPAddress:    c.ClientIP(),
		UserAgent:    c.Request.UserAgent(),
		StartedAt:    time.Now(),
		LastAccessAt: time.Now(),
	}

	if err := database.DB.Create(&streamSession).Error; err != nil {
		log.Printf("Error creating stream session: %v", err)
	}

	cache.CacheStreamSession(sessionID, streamSession, time.Hour)

	rangeHeader := c.Request.Header.Get("Range")
	var offset, length int64 = 0, 0

	if rangeHeader != "" {
		_, err := fmt.Sscanf(rangeHeader, "bytes=%d-%d", &offset, &length)
		if err != nil && err != io.EOF {
			offset = 0
		}
		if length > 0 {
			length = length - offset + 1
		}
	}

	bucketName := "audio-tracks"
	objectName := track.FilePath

	obj, err := storage.GetAudioFileStream(bucketName, objectName, offset, length)
	if err != nil {
		log.Printf("Error getting audio file: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error getting audio file"})
		return
	}
	defer obj.Close()

	info, err := obj.Stat()
	if err != nil {
		log.Printf("Error getting object info: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error getting file info"})
		return
	}

	contentType := "audio/mpeg"
	if track.FileFormat == "flac" {
		contentType = "audio/flac"
	} else if track.FileFormat == "wav" {
		contentType = "audio/wav"
	} else if track.FileFormat == "ogg" {
		contentType = "audio/ogg"
	}

	c.Header("Content-Type", contentType)
	c.Header("Accept-Ranges", "bytes")
	c.Header("Content-Length", fmt.Sprintf("%d", info.Size))

	if rangeHeader != "" {
		c.Header("Content-Range", fmt.Sprintf("bytes %d-%d/%d", offset, info.Size-1, info.Size))
		c.Status(http.StatusPartialContent)
	} else {
		c.Status(http.StatusOK)
	}

	if _, err := io.Copy(c.Writer, obj); err != nil {
		log.Printf("Error streaming file: %v", err)
		return
	}

	go updateStreamStats(uint(trackID))
}

func DownloadTrack(c *gin.Context) {
	trackID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid track ID"})
		return
	}

	var track _struct.Track
	result := database.DB.First(&track, trackID)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Track not found"})
		return
	}

	bucketName := "audio-tracks"
	objectName := track.FilePath

	obj, err := storage.GetAudioFile(bucketName, objectName)
	if err != nil {
		log.Printf("Error getting audio file: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error getting audio file"})
		return
	}
	defer obj.Close()

	info, err := obj.Stat()
	if err != nil {
		log.Printf("Error getting object info: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error getting file info"})
		return
	}

	contentType := "audio/mpeg"
	if track.FileFormat == "flac" {
		contentType = "audio/flac"
	} else if track.FileFormat == "wav" {
		contentType = "audio/wav"
	} else if track.FileFormat == "ogg" {
		contentType = "audio/ogg"
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s.%s", track.Title, track.FileFormat))
	c.Header("Content-Type", contentType)
	c.Header("Content-Length", fmt.Sprintf("%d", info.Size))
	c.Status(http.StatusOK)

	if _, err := io.Copy(c.Writer, obj); err != nil {
		log.Printf("Error downloading file: %v", err)
		return
	}
}

func StreamStatus(c *gin.Context) {
	sessionID := c.Query("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session ID is required"})
		return
	}

	sessionData, err := cache.GetCachedTrackData(fmt.Sprintf("session:%s", sessionID))
	if err == nil {
		c.JSON(http.StatusOK, gin.H{
			"session_id": sessionID,
			"status":     "active",
			"data":       sessionData,
		})
		return
	}

	var session _struct.StreamSession
	result := database.DB.Where("session_id = ?", sessionID).First(&session)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}

	session.LastAccessAt = time.Now()
	database.DB.Save(&session)

	c.JSON(http.StatusOK, gin.H{
		"session_id":  sessionID,
		"status":      "active",
		"track_id":    session.TrackID,
		"current_pos": session.CurrentPos,
		"started_at":  session.StartedAt,
		"last_access": session.LastAccessAt,
	})
}

func updateStreamStats(trackID uint) {
	var stats _struct.StreamStats
	result := database.DB.Where("track_id = ?", trackID).First(&stats)

	now := time.Now()

	if result.Error != nil {
		stats = _struct.StreamStats{
			TrackID:        trackID,
			TotalStreams:   1,
			UniqueUsers:    1,
			LastStreamedAt: &now,
		}
		database.DB.Create(&stats)
	} else {
		stats.TotalStreams++
		stats.LastStreamedAt = &now
		database.DB.Save(&stats)
	}
}
