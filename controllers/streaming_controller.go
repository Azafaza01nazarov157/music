package controllers

import (
	"encoding/json"
	"fmt"
	"log"
	_struct "music-conveyor/models/struct"
	"music-conveyor/platform/cache"
	"music-conveyor/platform/database"
	"music-conveyor/platform/storage"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func GetStreamableQualities(c *gin.Context) {
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

	cacheKey := fmt.Sprintf("track:%d:qualities", trackID)
	cachedData, err := cache.GetCachedTrackData(cacheKey)
	if err == nil {
		var qualities []string
		if err := json.Unmarshal([]byte(cachedData), &qualities); err == nil {
			c.JSON(http.StatusOK, gin.H{
				"track_id":  trackID,
				"qualities": qualities,
			})
			return
		}
	}

	bucketName := "audio-tracks"
	availableQualities := []string{}

	potentialQualities := []string{"64", "128", "192", "256", "320"}
	for _, quality := range potentialQualities {
		objectName := fmt.Sprintf("%d/%s.mp3", trackID, quality)
		_, err := storage.GetFileInfo(bucketName, objectName)
		if err == nil {
			availableQualities = append(availableQualities, quality)
		}
	}

	qualitiesJSON, _ := json.Marshal(availableQualities)
	cache.CacheTrackData(cacheKey, string(qualitiesJSON), 12*time.Hour)

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
		c.JSON(http.StatusNotFound, gin.H{"error": "Track not found"})
		return
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
		log.Printf("Error getting object info: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error getting file info"})
		return
	}

	c.Header("Content-Type", "audio/mpeg")
	c.Header("Content-Length", fmt.Sprintf("%d", info.Size))
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s_preview.mp3", track.Title))
	c.Status(http.StatusOK)

	http.ServeContent(c.Writer, c.Request, track.Title, time.Now(), obj)
}

func GetTrackStreamingStatus(c *gin.Context) {
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

	processingStatusKey := fmt.Sprintf("track:%d:processing_status", trackID)
	processingStatus, err := cache.GetCachedTrackData(processingStatusKey)
	if err != nil {
		processingStatus = "{\"status\":\"unknown\"}"
	}

	var statusData map[string]interface{}
	if err := json.Unmarshal([]byte(processingStatus), &statusData); err != nil {
		log.Printf("Error parsing processing status: %v", err)
		statusData = map[string]interface{}{
			"status": "unknown",
		}
	}

	var stats _struct.StreamStats
	database.DB.Where("track_id = ?", trackID).First(&stats)

	c.JSON(http.StatusOK, gin.H{
		"track_id":          trackID,
		"title":             track.Title,
		"artist_id":         track.ArtistID,
		"processing_status": statusData["status"],
		"stream_count":      stats.TotalStreams,
		"unique_listeners":  stats.UniqueUsers,
		"last_streamed_at":  stats.LastStreamedAt,
	})
}

func TrackPlaybackProgress(c *gin.Context) {
	sessionID := c.Query("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session ID is required"})
		return
	}

	var session _struct.StreamSession
	result := database.DB.Where("session_id = ?", sessionID).First(&session)
	if result.Error != nil {
		if c.Request.Method == "POST" {
			trackID, err := strconv.ParseUint(c.PostForm("track_id"), 10, 64)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid track ID"})
				return
			}

			userID := uint(0)
			if user, exists := c.Get("user"); exists {
				userID = user.(_struct.User).ID
			}

			quality := c.DefaultPostForm("quality", "high")

			newSessionID := uuid.New().String()
			session = _struct.StreamSession{
				UserID:       userID,
				TrackID:      uint(trackID),
				SessionID:    newSessionID,
				Quality:      quality,
				CurrentPos:   0,
				IsActive:     true,
				IPAddress:    c.ClientIP(),
				UserAgent:    c.Request.UserAgent(),
				StartedAt:    time.Now(),
				LastAccessAt: time.Now(),
			}

			database.DB.Create(&session)
			cache.CacheStreamSession(newSessionID, session, time.Hour)

			c.JSON(http.StatusOK, gin.H{
				"session_id":  newSessionID,
				"track_id":    trackID,
				"current_pos": 0,
				"started_at":  session.StartedAt,
			})
			return
		}

		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}

	if c.Request.Method == "POST" {
		positionStr := c.PostForm("position")
		position, err := strconv.ParseFloat(positionStr, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid position value"})
			return
		}

		session.CurrentPos = position
		session.LastAccessAt = time.Now()
		session.IsActive = true

		database.DB.Save(&session)
		cache.UpdateStreamPosition(sessionID, position)

		c.JSON(http.StatusOK, gin.H{
			"session_id":  sessionID,
			"track_id":    session.TrackID,
			"current_pos": position,
			"updated_at":  session.LastAccessAt,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"session_id":  sessionID,
		"track_id":    session.TrackID,
		"current_pos": session.CurrentPos,
		"started_at":  session.StartedAt,
		"last_access": session.LastAccessAt,
		"is_active":   session.IsActive,
	})
}

func GetPopularTracks(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}

	var stats []_struct.StreamStats
	database.DB.Order("total_streams DESC").Limit(limit).Find(&stats)

	if len(stats) == 0 {
		c.JSON(http.StatusOK, gin.H{"tracks": []interface{}{}})
		return
	}

	var trackIDs []uint
	for _, stat := range stats {
		trackIDs = append(trackIDs, stat.TrackID)
	}

	var tracks []_struct.Track
	database.DB.Where("id IN ?", trackIDs).Find(&tracks)

	trackMap := make(map[uint]_struct.Track)
	for _, track := range tracks {
		trackMap[track.ID] = track
	}

	var results []gin.H
	for _, stat := range stats {
		if track, ok := trackMap[stat.TrackID]; ok {
			results = append(results, gin.H{
				"id":               track.ID,
				"title":            track.Title,
				"artist_id":        track.ArtistID,
				"total_streams":    stat.TotalStreams,
				"unique_users":     stat.UniqueUsers,
				"last_streamed_at": stat.LastStreamedAt,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{"tracks": results})
}
