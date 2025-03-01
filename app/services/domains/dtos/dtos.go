package dtos

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Feedback struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	ChatID    int64              `bson:"chat_id"`
	Message   string             `bson:"message"`
	Timestamp time.Time          `bson:"timestamp"`
}