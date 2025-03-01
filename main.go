package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/saifwork/video-downloader-service.git/app/configs"
	"github.com/saifwork/video-downloader-service.git/app/middleware"
	"github.com/saifwork/video-downloader-service.git/app/services/domains"
	"github.com/saifwork/video-downloader-service.git/database"
)

func main() {
	runServer()
}

// service restart

func runServer() {
	// Load the configurations
	log.Println("Loading config ...")
	config := configs.NewConfig("")

	bot, err := tgbotapi.NewBotAPI(config.TelegramBotToken)
	if err != nil {
		log.Print("Error creating bot:", err)
		return
	}

	log.Println("Parsing environment ...")
	port := os.Getenv("PORT") // Use Railway’s assigned port
	if port == "" {
		port = "8080" // Default fallback for local testing
	}

	// Database connection
	log.Println("Initialize db ...")
	client, err := database.InitMongo(config)
	if err != nil {
		log.Fatal(err)
	}

	// Close the connection
	defer client.Disconnect(context.Background())

	// Setting routes endpoints
	log.Println("Creating the service ...")
	r := gin.New()

	// Global middleware
	log.Printf("Logging channel: %s to %s", config.LoggingChannel, config.LoggingEndpoint)
	if config.LoggingChannel == "file" {
		logfile, err := os.OpenFile(middleware.GetLogfilePath(config), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0664)
		if err != nil {
			panic(err)
		}
		defer func(logfile *os.File) {
			log.Println("Logfile closed")
			_ = logfile.Close()
		}(logfile)

		r.Use(middleware.DefaultStructuredLogger(config, logfile))
	} else {
		log.Printf("Using default gin logger")
		r.Use(gin.Logger())
	}

	// Recovery middleware
	r.Use(gin.Recovery())

	// Enable CORS middleware
	r.Use(CORSMiddleware())

	// Setup services
	r.GET("/healthcheck", Healthcheck)


	domains.NewVideoDownloaderService(bot, r, config, client).StartConsuming()

	isHttps, err := strconv.Atoi(os.Getenv("SERVICE_HTTPS"))
	if err == nil && isHttps == 1 {
		crt := os.Getenv("SERVICE_CERT")
		key := os.Getenv("SERVICE_KEY")
		log.Printf("Starting the HTTPS server on port %s", port)
		err := r.RunTLS(":"+port, crt, key)
		if err != nil {
			log.Fatalf("Error on starting the HTTPS service: %v", err)
		}
	} else {
		log.Printf("Starting the HTTP server on port %s", port)
		err := r.Run(":" + port) // Only use port, no host
		if err != nil {
			log.Fatalf("Error on starting the HTTP service: %v", err)
		}
	}
}

func Healthcheck(c *gin.Context) {
	version := os.Getenv("VERSION")
	if version == "" {
		version = "OK"
	}
	response := map[string]string{
		"status":  "up",
		"version": version,
	}
	c.JSON(http.StatusOK, response)
}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
