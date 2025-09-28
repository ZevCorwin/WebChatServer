package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type MessageType string
type MessageStatus string

const (
	MessageTypeText     MessageType = "Text"
	MessageTypeVoice    MessageType = "Voice"
	MessageTypeLink     MessageType = "Link"
	MessageTypeIcon     MessageType = "Icon"
	MessageTypeSticker  MessageType = "Sticker"
	MessageTypeLocation MessageType = "Location"
	MessageTypeContact  MessageType = "Contact"
	MessageTypeReaction MessageType = "Reaction"

	MessageTypeFile MessageType = "File"

	MessageStatusSending  MessageStatus = "Đang gửi"
	MessageStatusSent     MessageStatus = "Đã gửi"
	MessageStatusReceived MessageStatus = "Đã nhận"
	MessageStatusSeen     MessageStatus = "Đã xem"
)

type LastMessagePreview struct {
	ID      primitive.ObjectID `bson:"id" json:"id"`
	Content string             `bson:"content" json:"content"`
	Type    string             `bson:"type" json:"type"`
	Sender  primitive.ObjectID `bson:"sender" json:"sender"`
}

type Message struct {
	ID          primitive.ObjectID   `bson:"_id" json:"id"`
	Content     string               `bson:"content" json:"content"`
	Timestamp   time.Time            `bson:"timestamp" json:"timestamp"`
	MessageType MessageType          `bson:"messageType" json:"messageType"`
	SenderID    primitive.ObjectID   `bson:"senderId" json:"senderId"`
	Status      MessageStatus        `bson:"status" json:"status"`
	Recalled    bool                 `bson:"recalled" json:"recalled"`
	HiddenBy    []primitive.ObjectID `bson:"hiddenBy,omitempty" json:"-"`
	URL         string               `json:"url" bson:"url"`
	FileID      *primitive.ObjectID  `bson:"fileId" json:"fileId"`
}
