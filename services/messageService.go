package services

import (
	"chat-app-backend/config"
	"chat-app-backend/models"
	"context"
	"errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

type MessageService struct {
	DB             *mongo.Database
	ChannelService *ChannelService
}

func NewMessageService(cs *ChannelService) *MessageService {
	return &MessageService{DB: config.DB, ChannelService: cs}
}

func (ms *MessageService) SendMessage(channelID primitive.ObjectID, senderID primitive.ObjectID, content string, messageType models.MessageType) (*models.Message, error) {
	// Sử dụng ChannelService để lấy thông tin kênh
	channel, err := ms.ChannelService.GetChannel(channelID)
	if err != nil {
		return nil, err
	}

	// Kiểm tra xem người gửi có phải là thành viên của kênh hay không
	if !ms.ChannelService.IsMember(channel, senderID) {
		return nil, errors.New("Sender is not a member of the channel")
	}

	var message *models.Message
	switch messageType {
	case models.MessageTypeFile:
		// Tạo file và lưu vào collection files
		file := &models.File{
			ID:         primitive.NewObjectID(),
			FileName:   content,                 // giả định `content` là tên tệp
			FileType:   models.FileTypeDocument, // Loại file mặc định
			FileSize:   0,                       // Giá trị giả định, cần bổ sung logic để lấy kích thước file
			UploadTime: time.Now(),
			URL:        content, // giả định `content` là URL file
		}
		_, err := ms.DB.Collection("files").InsertOne(context.Background(), file)
		if err != nil {
			return nil, err
		}

		// Tạo tin nhắn
		message = &models.Message{
			ID:          primitive.NewObjectID(),
			Content:     "", // Không lưu trong Content
			Timestamp:   time.Now(),
			MessageType: messageType,
			SenderID:    senderID,
			Status:      models.MessageStatusSending,
			Recalled:    false,
			URL:         "",       // Không lưu trong Url
			FileID:      &file.ID, // Liên kết với file ID
		}

	case models.MessageTypeVoice, models.MessageTypeSticker:
		// Tạo tin nhắn cho Voice hoặc Sticker
		message = &models.Message{
			ID:          primitive.NewObjectID(),
			Content:     "", // Không lưu trong Content
			Timestamp:   time.Now(),
			MessageType: messageType,
			SenderID:    senderID,
			Status:      models.MessageStatusSending,
			Recalled:    false,
			URL:         content, // Lưu URL
			FileID:      nil,
		}

	default:
		// Các loại tin nhắn khác
		message = &models.Message{
			ID:          primitive.NewObjectID(),
			Content:     content, // Lưu nội dung vào Content
			Timestamp:   time.Now(),
			MessageType: messageType,
			SenderID:    senderID,
			Status:      models.MessageStatusSending,
			Recalled:    false,
			URL:         "", // Không lưu trong Url
			FileID:      nil,
		}
	}

	// Lưu tin nhắn vào collection "messages"
	collection := ms.DB.Collection("messages")
	_, err = collection.InsertOne(context.Background(), message)
	if err != nil {
		return nil, err
	}

	// Cập nhật lịch sử chat
	chatHistoryCollection := ms.DB.Collection("chathistory")
	filter := bson.M{"channelID": channelID}
	update := bson.M{"$push": bson.M{"message": message.ID}}
	opts := options.FindOneAndUpdate().SetUpsert(true)
	err = chatHistoryCollection.FindOneAndUpdate(context.Background(), filter, update, opts).Err()
	if err != nil {
		return nil, err
	}

	return message, nil
}

// Kiểm tra người dùng có vai trò nhất định trong kênh không
func (ms *MessageService) hasRole(channel *models.Channel, userID primitive.ObjectID, roles []models.MemberRole) bool {
	for _, member := range channel.Members {
		if member.MemberID == userID {
			for _, role := range roles {
				if member.Role == role {
					return true
				}
			}
		}
	}
	return false
}
