package domains

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/saifwork/video-downloader-service.git/app/configs"
	"github.com/saifwork/video-downloader-service.git/app/services/domains/dtos"
	"github.com/saifwork/video-downloader-service.git/app/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type VideoDownloaderService struct {
	Bot    *tgbotapi.BotAPI
	Gin    *gin.Engine
	Conf   *configs.Config
	Client *mongo.Client
}

func NewVideoDownloaderService(bot *tgbotapi.BotAPI, gin *gin.Engine, conf *configs.Config, cli *mongo.Client) *VideoDownloaderService {
	return &VideoDownloaderService{
		Bot:    bot,
		Gin:    gin,
		Conf:   conf,
		Client: cli,
	}
}

func (s *VideoDownloaderService) StartConsuming() {
	// Start consuming

	fmt.Println("✅ Bot is running...")

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates, _ := s.Bot.GetUpdatesChan(u)

	for update := range updates {

		if update.CallbackQuery != nil {
			callbackData := update.CallbackQuery.Data
			chatID := update.CallbackQuery.Message.Chat.ID

			// Debug: Print the callback data to check if it’s detected
			fmt.Println("✅ Button Clicked: ", callbackData)

			if strings.HasPrefix(callbackData, "quality_") {
				parts := strings.Split(callbackData, "_")
				if len(parts) < 3 {
					fmt.Println("❌ Invalid callback data format: ", callbackData)
					continue
				}

				quality := parts[1]                      // e.g., "best", "720p", "audio"
				videoURL := strings.Join(parts[2:], "_") // Extract the URL

				fmt.Printf("📌 Selected Quality: %s, URL: %s\n", quality, videoURL)

				// Call the download function
				s.DownloadVideo(chatID, videoURL, quality)
			}
		}

		if update.Message == nil {
			continue
		}

		chatID := update.Message.Chat.ID
		text := update.Message.Text

		switch {

		case text == "/start":
			s.HandleStart(chatID)

		case text == "/help":
			s.HandleHelp(chatID)

		case strings.HasPrefix(text, "/feedback "):
			s.HandleFeedback(chatID, strings.TrimPrefix(text, "/feedback "))

		case text == "/about":
			s.HandleAbout(chatID)

		case strings.HasPrefix(text, "http"):
			s.HandleDownload(chatID, text)

		default:
			s.HandleUnknownCommand(chatID)
		}
	}
}

func (s *VideoDownloaderService) HandleStart(chatID int64) {
	msg := tgbotapi.NewMessage(chatID, "👋 Welcome! Send a video link to download it.")

	// Create a keyboard with options
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📥 Download Video"),
			tgbotapi.NewKeyboardButton("/help"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("/feedback"),
			tgbotapi.NewKeyboardButton("/about"),
		),
	)

	msg.ReplyMarkup = keyboard
	s.Bot.Send(msg)
}

func (s *VideoDownloaderService) HandleHelp(chatID int64) {
	helpText := `📌 *QuickVidLoader Bot Help*

📝 *How to use this bot?*
1️⃣ Send a valid video link (YouTube, Instagram, etc.).
2️⃣ The bot will download and send the video back to you.
3️⃣ Use the menu buttons for quick actions.

🔹 *Available Commands:*
✅ /start - Start the bot and see the menu
✅ /help - Show this help message
✅ /about - Learn about the bot
✅ /feedback [your message] - Send feedback
✅ Send any valid video link to download

📌 *Supported Platforms:*
✔ YouTube  
✔ Instagram  
✔ Twitter (X)  
✔ Facebook  
✔ TikTok  
`

	msg := tgbotapi.NewMessage(chatID, helpText)
	msg.ParseMode = "Markdown"
	s.Bot.Send(msg)
}

func (s *VideoDownloaderService) HandleFeedback(chatID int64, feedback string) {
	if feedback == "" {
		s.Bot.Send(tgbotapi.NewMessage(chatID, "Please provide your feedback. Example: `/feedback I love this bot!`"))
		return
	}

	// Check if user has given feedback in the last 7 days
	var lastFeedback dtos.Feedback
	err := s.Client.Database(s.Conf.MongoDatabase).Collection("feedbacks").FindOne(
		context.TODO(),
		bson.M{"chat_id": chatID},
		options.FindOne().SetSort(bson.M{"timestamp": -1}),
	).Decode(&lastFeedback)

	if err == nil {
		// Calculate time difference
		oneWeekAgo := time.Now().AddDate(0, 0, -7)
		if lastFeedback.Timestamp.After(oneWeekAgo) {
			s.Bot.Send(tgbotapi.NewMessage(chatID, "❌ You can only provide feedback once a week. Try again later!"))
			return
		}
	}

	// Create a new feedback entry
	newFeedback := dtos.Feedback{
		ChatID:    chatID,
		Message:   feedback,
		Timestamp: time.Now(),
	}

	// Insert into MongoDB
	_, err = s.Client.Database(s.Conf.MongoDatabase).Collection("feedbacks").InsertOne(context.TODO(), newFeedback)
	if err != nil {
		s.Bot.Send(tgbotapi.NewMessage(chatID, "❌ Failed to save feedback. Please try again later."))
		return
	}

	// Confirmation message
	s.Bot.Send(tgbotapi.NewMessage(chatID, "✅ Thank you for your feedback! 😊"))

	fmt.Printf("Feedback saved: %+v\n", newFeedback)
}

func (s *VideoDownloaderService) HandleAbout(chatID int64) {
	aboutText := `📢 *About QuickVidLoader Bot*  

🚀 *What does this bot do?*  
This bot allows you to download videos from various platforms by simply sending a link.  

🎥 *Supported Platforms:*  
✅ YouTube  
✅ Instagram  
✅ Twitter (X)  
✅ Facebook  
✅ TikTok  

💡 *How to use?*  
Just send a valid video link and get your video downloaded!`

	msg := tgbotapi.NewMessage(chatID, aboutText)
	msg.ParseMode = "Markdown"
	s.Bot.Send(msg)
}

func (s *VideoDownloaderService) HandleDownload(chatID int64, videoURL string) {
	s.Bot.Send(tgbotapi.NewMessage(chatID, "🎥 Choose your preferred video quality:"))

	// Create inline keyboard with predefined quality options
	qualityKeyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔹 1080p (Best)", "quality_best_"+videoURL),
			tgbotapi.NewInlineKeyboardButtonData("🔹 720p", "quality_720p_"+videoURL),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔹 480p", "quality_480p_"+videoURL),
			tgbotapi.NewInlineKeyboardButtonData("🔹 360p", "quality_360p_"+videoURL),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🎵 Audio Only", "quality_audio_"+videoURL),
		),
	)

	// Send message with the inline keyboard
	msg := tgbotapi.NewMessage(chatID, "📌 Select the quality you want to download:")
	msg.ReplyMarkup = qualityKeyboard
	s.Bot.Send(msg)
}

// Download and send video/audio based on selected quality
func (s *VideoDownloaderService) DownloadVideo(chatID int64, videoURL string, quality string) {
	s.Bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("⏳ Downloading your %s file, please wait...", quality)))

	// Define download directory
	outputDir := "downloads"
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		os.Mkdir(outputDir, os.ModePerm)
	}

	// Determine file extension
	fileExtension := "mp4" // Default for video
	if quality == "audio" {
		fileExtension = "m4a" // Default format for audio-only
	}

	// Generate unique filename
	timestamp := time.Now().Unix()
	outputFile := filepath.Join(outputDir, fmt.Sprintf("%d_%d.%s", chatID, timestamp, fileExtension))

	// Map quality to yt-dlp format selection
	qualityMap := map[string]string{
		"best":  "bestvideo+bestaudio/best",
		"1080p": "bestvideo[height<=1080]+bestaudio/best",
		"720p":  "bestvideo[height<=720]+bestaudio/best",
		"480p":  "bestvideo[height<=480]+bestaudio/best",
		"360p":  "bestvideo[height<=360]+bestaudio/best",
		"audio": "bestaudio",
	}

	// Execute yt-dlp command
	cmd := exec.Command("yt-dlp", "-f", qualityMap[quality], "-o", outputFile, videoURL)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()

	if err != nil {
		s.Bot.Send(tgbotapi.NewMessage(chatID, "❌ Failed to download. Please try another link."))
		return
	}

	// Handle merged file issue: Check if yt-dlp created a .webm instead of .mp4
	mergedFile := outputFile + ".webm"
	if utils.FileExists(mergedFile) {
		os.Rename(mergedFile, outputFile) // Rename to expected format
	}

	// Check if the final file exists
	if !utils.FileExists(outputFile) {
		s.Bot.Send(tgbotapi.NewMessage(chatID, "❌ Video file not found. Merging might have failed."))
		return
	}

	// 📌 **Step 1: Check File Size Before Sending**
	fileInfo, err := os.Stat(outputFile)
	if err == nil {
		fmt.Printf("📌 File size: %.2f MB\n", float64(fileInfo.Size())/(1024*1024))
	}

	// 📌 **Step 2: Compress Large Files (If >50MB)**
	maxSizeMB := 50.0 // Telegram bot max file size
	compressedFile := outputFile

	if float64(fileInfo.Size())/(1024*1024) > maxSizeMB {
		compressedFile = strings.Replace(outputFile, ".mp4", "_compressed.mp4", 1)
		fmt.Println("⚡ Compressing video to reduce size...")

		cmd := exec.Command("ffmpeg", "-i", outputFile, "-b:v", "800k", "-preset", "fast", compressedFile)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()

		if err != nil {
			fmt.Println("❌ FFmpeg compression failed:", err)
			compressedFile = outputFile // Send original file if compression fails
		} else {
			fmt.Println("✅ Compression successful:", compressedFile)
		}
	}

	// 📌 **Step 3: Send as Document to Skip Telegram Processing**
	var msg tgbotapi.Chattable
	if quality == "audio" {
		msg = tgbotapi.NewAudioUpload(chatID, compressedFile) // Send as audio
	} else {
		msg = tgbotapi.NewDocumentUpload(chatID, compressedFile) // Send as document to skip Telegram processing
	}

	_, sendErr := s.Bot.Send(msg)
	if sendErr != nil {
		s.Bot.Send(tgbotapi.NewMessage(chatID, "❌ Error sending file. Please try again."))
		return
	}

	// 📌 **Step 4: Delete the File After Sending**
	err = os.Remove(compressedFile)
	if err != nil {
		fmt.Println("❌ Error deleting file:", err)
	} else {
		fmt.Println("✅ File deleted:", compressedFile)
	}
}

func (s *VideoDownloaderService) HandleUnknownCommand(chatID int64) {
	msg := "❌ Unknown command. Type `/help` to see available commands."
	s.Bot.Send(tgbotapi.NewMessage(chatID, msg))
}
