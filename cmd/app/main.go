package main

import (
	"log"
	"music-conveyor/controllers"
	"music-conveyor/platform/cache"
	"music-conveyor/platform/config"
	"music-conveyor/platform/database"
	"music-conveyor/platform/kafka"
	"music-conveyor/platform/middleware"
	"music-conveyor/platform/storage"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
)

var appConfig *config.Config

func main() {
	appConfig = config.LoadConfig()

	if appConfig.Environment != "development" {
		gin.SetMode(gin.ReleaseMode)
	}

	initializeServices()
	setupGracefulShutdown()

	router := gin.Default()
	router.MaxMultipartMemory = appConfig.MaxUploadSize

	setupMiddleware(router)
	setupRoutes(router)

	log.Printf("Starting audio streaming service on :%s", appConfig.Port)
	log.Printf("Maximum upload size: %d bytes", appConfig.MaxUploadSize)
	if err := router.Run(":" + appConfig.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func setupMiddleware(router *gin.Engine) {
	router.Use(cors())
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
}

func setupRoutes(router *gin.Engine) {
	router.GET("/health", controllers.HealthCheck)

	auth := router.Group("/api/auth")
	{
		auth.POST("/login", controllers.GenerateToken)
	}

	api := router.Group("/api")
	api.Use(middleware.AuthMiddleware())
	{
		stream := api.Group("/stream")
		{
			stream.GET("/:id", controllers.StreamTrack)
			stream.GET("/:id/download", controllers.DownloadTrack)
			stream.GET("/status", controllers.StreamStatus)
		}

		tracks := api.Group("/tracks")
		{
			tracks.GET("/:id/qualities", controllers.GetStreamableQualities)
			tracks.GET("/:id/preview", controllers.StreamPreview)
		}
	}
}

func initializeServices() {
	database.ConnectDatabase()
	cache.ConnectRedis()
	storage.ConnectMinio()
	kafka.NewKafkaConfig()
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
