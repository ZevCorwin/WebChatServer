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

type Attachment struct {
	URL      string `bson:"url" json:"url"`
	Mime     string `bson:"mime" json:"mime"`
	Size     int64  `bson:"size,omitempty" json:"size,omitempty"`
	Width    int32  `bson:"width,omitempty" json:"width,omitempty"`
	Height   int32  `bson:"height,omitempty" json:"height,omitempty"`
	Duration int32  `bson:"duration,omitempty" json:"duration,omitempty"` // audio/video (giây)
}

type ReadReceipt struct {
	UserID primitive.ObjectID `bson:"userId" json:"userId"`
	SeenAt time.Time          `bson:"seenAt" json:"seenAt"`
}

type DeliveryReceipt struct {
	UserID      primitive.ObjectID `bson:"userId" json:"userId"`
	DeliveredAt time.Time          `bson:"deliveredAt" json:"deliveredAt"`
}

type Message struct {
	ID             primitive.ObjectID              `bson:"_id" json:"id"`
	ChannelID      primitive.ObjectID              `bson:"channelID" json:"channelId"`
	Content        string                          `bson:"content" json:"content"`
	Timestamp      time.Time                       `bson:"timestamp" json:"timestamp"`
	MessageType    MessageType                     `bson:"messageType" json:"messageType"`
	SenderID       primitive.ObjectID              `bson:"senderId" json:"senderId"`
	Status         MessageStatus                   `bson:"status" json:"status"`
	Recalled       bool                            `bson:"recalled" json:"recalled"`
	HiddenBy       []primitive.ObjectID            `bson:"hiddenBy,omitempty" json:"-"`
	URL            string                          `json:"url" bson:"url"`
	FileID         *primitive.ObjectID             `bson:"fileId" json:"fileId"`
	ReplyTo        *primitive.ObjectID             `bson:"replyTo,omitempty" json:"replyTo,omitempty"`
	EditedAt       *time.Time                      `bson:"editedAt,omitempty" json:"editedAt,omitempty"`
	RecallDeadline *time.Time                      `bson:"recallDeadline,omitempty" json:"recallDeadline,omitempty"`
	Reactions      map[string][]primitive.ObjectID `bson:"reactions,omitempty" json:"reactions,omitempty"`
	ReadBy         []ReadReceipt                   `bson:"readBy,omitempty" json:"readBy,omitempty"`
	DeliveredBy    []DeliveryReceipt               `bson:"deliveredBy,omitempty" json:"deliveredBy,omitempty"`
	Attachments    []Attachment                    `bson:"attachments,omitempty" json:"attachments,omitempty"`
}
