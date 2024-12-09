package services

import (
	"chat-app-backend/config"
	"chat-app-backend/models"
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"log"
)

type ChatHistoryService struct {
	DB *mongo.Database
}

func NewChatHistoryService() *ChatHistoryService {
	return &ChatHistoryService{DB: config.DB}
}

// Lấy lịch sử chat
func (chs *ChatHistoryService) GetChatHistory(channelID primitive.ObjectID) ([]models.Message, error) {
	chatHistoryCollection := chs.DB.Collection("chathistory")
	var chatHistory models.ChatHistory
	err := chatHistoryCollection.FindOne(context.Background(), bson.M{"channelID": channelID}).Decode(&chatHistory)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return []models.Message{}, nil
		}
		return nil, err
	}

	// Lấy danh sách tin nhắn
	messagesCollection := chs.DB.Collection("messages")
	filter := bson.M{"_id": bson.M{"$in": chatHistory.Message}}
	cursor, err := messagesCollection.Find(context.Background(), filter)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	defer func(cursor *mongo.Cursor, ctx context.Context) {
		err := cursor.Close(ctx)
		if err != nil {
			log.Println("Lỗi khi đóng cursor:", err)
		}
	}(cursor, ctx)

	var messages []models.Message
	for cursor.Next(context.Background()) {
		var msg models.Message
		if err := cursor.Decode(&msg); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// Lấy lịch sử kênh chat của người dùng
func (chs *ChatHistoryService) GetChatHistoryByUserID(userID primitive.ObjectID) ([]models.Channel, error) {
	chatHistoryCollection := chs.DB.Collection("chathistory")
	filter := bson.M{"userID": userID}
	cursor, err := chatHistoryCollection.Find(context.Background(), filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	var channels []models.Channel
	for cursor.Next(context.Background()) {
		var channel models.Channel
		if err := cursor.Decode(&channel); err != nil {
			return nil, err
		}
		channels = append(channels, channel)
	}
	if len(channels) == 0 {
		return []models.Channel{}, nil
	}

	return channels, nil
}

// Xóa lịch sử chat
func (chs *ChatHistoryService) DeleteChatHistory(channelID primitive.ObjectID) error {
	chatHistoryCollection := chs.DB.Collection("chathistory")
	_, err := chatHistoryCollection.DeleteOne(context.Background(), bson.M{"channelID": channelID})
	return err
}
