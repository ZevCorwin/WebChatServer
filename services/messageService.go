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
	"log"
	"time"
)

type MessageService struct {
	DB                 *mongo.Database
	ChannelService     *ChannelService
	UserChannelService *UserChannelService
	ChatHistoryService *ChatHistoryService
}

func NewMessageService() *MessageService {
	return &MessageService{
		DB:                 config.DB,
		ChannelService:     NewChannelService(),
		UserChannelService: NewUserChannelService(),
		ChatHistoryService: NewChatHistoryService(),
	}
}

func (ms *MessageService) SendMessage(channelID, senderID primitive.ObjectID, content string, messageType models.MessageType) (*models.Message, error) {
	// Sử dụng ChannelService để lấy thông tin kênh
	log.Printf("[SendMessage] channelID=%s senderID=%s content=%s messageType=%s", channelID.Hex(), senderID.Hex(), content, messageType)
	channel, err := ms.ChannelService.GetChannel(channelID)
	if err != nil {
		log.Printf("[SendMessage] GetChannel error: %v", err)
		return nil, err
	}
	log.Printf("[SendMessage] Found channel: %+v", channel)

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
	log.Printf("[SendMessage] Message to insert: %+v", message)
	collection := ms.DB.Collection("messages")
	_, err = collection.InsertOne(context.Background(), message)
	if err != nil {
		log.Printf("[SendMessage] Insert message error: %v", err)
		return nil, err
	}
	log.Printf("[SendMessage] Insert message success")

	// Cập nhật lịch sử chat
	chatHistoryCollection := ms.DB.Collection("chathistory")
	filter := bson.M{"channelID": channelID}

	// lưu cả id và nội dung tin nhắn cuối
	update := bson.M{
		"$push": bson.M{"messages": message.ID},
		"$set": bson.M{
			"channelID": channelID,
			"lastMessage": models.LastMessagePreview{
				ID:      message.ID,      // để sau này dễ fetch lại
				Content: message.Content, // hiển thị preview nhanh
				Type:    string(message.MessageType),
				Sender:  senderID,
			},
			"lastActive": message.Timestamp,
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err = chatHistoryCollection.UpdateOne(context.Background(), filter, update, opts)
	if err != nil {
		log.Printf("[SendMessage] Update chat history error: %v", err)
		return nil, err
	}
	log.Printf("[SendMessage] Update chat history success")

	err = ms.ChatHistoryService.UpdateLastActive(channelID, message.Timestamp)
	if err != nil {
		return nil, err
	}

	err = ms.UserChannelService.UpdateLastActive(senderID, channelID)
	if err != nil {
		return nil, err
	}

	return message, nil
}

// Đã sửa
func (ms *MessageService) UpdateMessageStatus(messageID, channelID primitive.ObjectID, status models.MessageStatus) error {
	_, err := ms.DB.Collection("messages").UpdateOne(
		context.Background(),
		bson.M{"_id": messageID, "channelID": channelID},
		bson.M{"$set": bson.M{"status": status}},
	)
	return err

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
