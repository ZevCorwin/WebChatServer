package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type ChatHistory struct {
	ID        primitive.ObjectID   `json:"id" bson:"_id,omitempty"`
	ChannelID primitive.ObjectID   `json:"channelID" bson:"channelID"`
	Message   []primitive.ObjectID `json:"message" bson:"message"`
}
