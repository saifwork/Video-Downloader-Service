package services

import (
	"github.com/gin-gonic/gin"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/saifwork/video-downloader-service.git/app/configs"
	"github.com/saifwork/video-downloader-service.git/app/services/domains"
	"go.mongodb.org/mongo-driver/mongo"
)

type Initializer struct {
	bot    *tgbotapi.BotAPI
	gin    *gin.Engine
	conf   *configs.Config
	client *mongo.Client
}

func NewInitializer(bot *tgbotapi.BotAPI, gin *gin.Engine, conf *configs.Config, cli *mongo.Client) *Initializer {
	s := &Initializer{
		bot:    bot,
		gin:    gin,
		conf:   conf,
		client: cli,
	}
	return s
}

func (s *Initializer) RegisterDomains(domains []domains.IDomain) {
	for _, domain := range domains {
		domain.SetupRoutes()
	}
}
