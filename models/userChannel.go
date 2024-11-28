package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type UserChannel struct {
	ID           primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID       primitive.ObjectID `json:"userID" bson:"userID"`
	ChannelID    primitive.ObjectID `json:"channelID" bson:"channelID"`
	LastActive   time.Time          `json:"lastActive" bson:"lastActive"`
	LastUnreadAt *time.Time         `json:"lastUnreadAt" bson:"lastUnreadAt,omitempty"`
}
