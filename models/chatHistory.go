package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type ChatHistory struct {
	ID          primitive.ObjectID  `json:"id" bson:"_id,omitempty"`
	ChannelID   primitive.ObjectID  `json:"channelID" bson:"channelID"`
	LastActive  time.Time           `json:"lastActive" bson:"lastActive"`
	LastMessage *LastMessagePreview `json:"lastMessage,omitempty" bson:"lastMessage,omitempty"`
}
