package main

import (
	"github.com/gin-gonic/gin"
	"log"
	"music-conveyor/cache"
	"music-conveyor/controllers"
	"music-conveyor/database"
	"music-conveyor/storage"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	initializeServices()

	setupGracefulShutdown()

	router := gin.Default()

	setupMiddleware(router)

	setupRoutes(router)

	port := getEnv("PORT", "8080")

	log.Printf("Starting audio streaming service on :%s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func initializeServices() {
	database.ConnectDatabase()

	cache.ConnectRedis()
	storage.ConnectMinio()

	log.Println("All services initialized successfully")
}

func setupGracefulShutdown() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Println("Shutting down services...")

		sqlDB, err := database.DB.DB()
		if err != nil {
			log.Printf("Error getting database instance: %v", err)
		} else {
			sqlDB.Close()
		}

		cache.CloseRedis()

		log.Println("All services shut down gracefully")
		os.Exit(0)
	}()
}

func setupMiddleware(router *gin.Engine) {
	router.Use(cors())

	router.Use(gin.Logger())

	router.Use(gin.Recovery())
}

func setupRoutes(router *gin.Engine) {
	router.GET("/health", controllers.HealthCheck)

	streamGroup := router.Group("/api/stream")
	{
		streamGroup.GET("/:id", controllers.StreamTrack)
		streamGroup.GET("/:id/download", controllers.DownloadTrack)
		streamGroup.GET("/status", controllers.StreamStatus)
	}
}

func cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
