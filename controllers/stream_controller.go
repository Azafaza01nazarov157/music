package controllers

import (
	"encoding/json"
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
	"gorm.io/gorm"
)

const (
	DefaultQuality = "320"
	CacheTimeout   = 24 * time.Hour
)

func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}

func StreamTrack(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	trackID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid track ID"})
		return
	}

	if !hasAccessToTrack(uint(userID.(uint)), uint(trackID)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	cacheKey := fmt.Sprintf("track:%d:info", trackID)
	var track _struct.Track
	cachedData, err := cache.GetCachedTrackData(cacheKey)
	if err == nil {
		if err := json.Unmarshal([]byte(cachedData), &track); err != nil {
			log.Printf("Error unmarshaling cached track data: %v", err)
		}
	} else {
		result := database.DB.First(&track, trackID)
		if result.Error != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Track not found"})
			return
		}

		if trackData, err := json.Marshal(track); err == nil {
			cache.CacheTrackData(cacheKey, string(trackData), CacheTimeout)
		}
	}

	quality := c.DefaultQuery("quality", DefaultQuality)
	if !isValidQuality(quality) {
		quality = DefaultQuality
	}

	sessionID := uuid.New().String()
	streamSession := _struct.StreamSession{
		UserID:       uint(userID.(uint)),
		TrackID:      uint(trackID),
		SessionID:    sessionID,
		Quality:      quality,
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

	bucketName := "audio-tracks"
	objectName := fmt.Sprintf("%d/%s.mp3", trackID, quality)

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

	c.Header("Content-Type", "audio/mpeg")
	c.Header("Accept-Ranges", "bytes")
	c.Header("Content-Length", fmt.Sprintf("%d", info.Size))
	c.Header("Cache-Control", "public, max-age=31536000")

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

func isValidQuality(quality string) bool {
	validQualities := map[string]bool{
		"64":  true,
		"128": true,
		"192": true,
		"256": true,
		"320": true,
	}
	return validQualities[quality]
}

func hasAccessToTrack(userID, trackID uint) bool {
	var track _struct.Track
	result := database.DB.First(&track, trackID)
	if result.Error != nil {
		return false
	}

	return track.UserID == userID
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
	database.DB.Model(&_struct.Track{}).
		Where("id = ?", trackID).
		UpdateColumn("play_count", gorm.Expr("play_count + ?", 1))

	statsKey := fmt.Sprintf("track:%d:stats", trackID)
	cache.RedisClient.Incr(cache.Ctx, statsKey)
}

func GetStreamableQualities(c *gin.Context) {
	trackID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid track ID"})
		return
	}

	var track _struct.Track
	result := database.DB.First(&track, trackID)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Track not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		}
		return
	}

	bucketName := "audio-tracks"
	qualities := []string{"320"}
	availableQualities := make([]string, 0)

	for _, quality := range qualities {
		objectName := fmt.Sprintf("%d/%s.mp3", trackID, quality)
		_, err := storage.GetFileInfo(bucketName, objectName)
		if err == nil {
			availableQualities = append(availableQualities, quality)
		}
	}

	qualityInfo := _struct.TrackQuality{
		TrackID:            uint(trackID),
		AvailableQualities: availableQualities,
		UpdatedAt:          time.Now(),
	}

	if err := database.DB.Where("track_id = ?", trackID).
		Assign(qualityInfo).
		FirstOrCreate(&qualityInfo).Error; err != nil {
		log.Printf("Error updating track qualities: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"track_id":  trackID,
		"qualities": availableQualities,
	})
}

func StreamPreview(c *gin.Context) {
	trackID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid track ID"})
		return
	}

	var track _struct.Track
	result := database.DB.First(&track, trackID)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Track not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		}
		return
	}

	previewPlay := _struct.PreviewPlay{
		TrackID:   uint(trackID),
		UserID:    0,
		IPAddress: c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		PlayedAt:  time.Now(),
	}

	if userID, exists := c.Get("user_id"); exists {
		previewPlay.UserID = userID.(uint)
	}

	if err := database.DB.Create(&previewPlay).Error; err != nil {
		log.Printf("Error recording preview play: %v", err)
	}

	bucketName := "audio-previews"
	objectName := fmt.Sprintf("%d/preview.mp3", trackID)

	obj, err := storage.GetAudioFile(bucketName, objectName)
	if err != nil {
		log.Printf("Error getting preview file: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error getting preview file"})
		return
	}
	defer obj.Close()

	info, err := obj.Stat()
	if err != nil {
		log.Printf("Error getting preview info: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error getting preview info"})
		return
	}

	c.Header("Content-Type", "audio/mpeg")
	c.Header("Content-Length", fmt.Sprintf("%d", info.Size))
	c.Header("Cache-Control", "public, max-age=31536000")

	if _, err := io.Copy(c.Writer, obj); err != nil {
		log.Printf("Error streaming preview: %v", err)
		return
	}
}
