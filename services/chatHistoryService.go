package services

import (
	"chat-app-backend/config"
	"chat-app-backend/models"
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"log"
	"sort"
	"time"
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
func (chs *ChatHistoryService) GetChatHistoryByUserID(userID primitive.ObjectID) ([]map[string]interface{}, error) {
	userChannelsCollection := chs.DB.Collection("userChannels")
	chatHistoryCollection := chs.DB.Collection("chathistory")
	channelsCollection := chs.DB.Collection("channels")
	messagesCollection := chs.DB.Collection("messages")

	// Lấy danh sách các kênh của user
	filter := bson.M{"userID": userID}
	cursor, err := userChannelsCollection.Find(context.Background(), filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	baseUrl := "http://localhost:8080"
	var chatHistory []map[string]interface{}
	for cursor.Next(context.Background()) {
		var userChannel models.UserChannel
		if err := cursor.Decode(&userChannel); err != nil {
			return nil, err
		}

		// Lấy thông tin kênh
		var channel models.Channel
		channelFilter := bson.M{"_id": userChannel.ChannelID}
		err = channelsCollection.FindOne(context.Background(), channelFilter).Decode(&channel)
		if err != nil {
			return nil, err
		}

		// Lấy thông tin lịch sử chat
		var chatHistoryRecord models.ChatHistory
		chatHistoryFilter := bson.M{"channelID": userChannel.ChannelID}
		err = chatHistoryCollection.FindOne(context.Background(), chatHistoryFilter).Decode(&chatHistoryRecord)
		if err != nil && err != mongo.ErrNoDocuments {
			return nil, err
		}

		// Lấy tin nhắn cuối cùng trong mảng message
		var lastMessageContent string
		if len(chatHistoryRecord.Message) > 0 {
			lastMessageID := chatHistoryRecord.Message[len(chatHistoryRecord.Message)-1]

			// Truy vấn để lấy nội dung tin nhắn
			var lastMessage models.Message
			messageFilter := bson.M{"_id": lastMessageID}
			err = messagesCollection.FindOne(context.Background(), messageFilter).Decode(&lastMessage)
			if err == nil {
				lastMessageContent = lastMessage.Content
			}
		}

		// Tạo đối tượng trả về
		chatItem := map[string]interface{}{
			"channelID":   userChannel.ChannelID,
			"avatar":      baseUrl + channel.Avatar,
			"channelName": channel.ChannelName,
			"lastMessage": lastMessageContent,
			"lastActive":  chatHistoryRecord.LastActive,
		}

		chatHistory = append(chatHistory, chatItem)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	// Sắp xếp danh sách theo LastActive
	sort.Slice(chatHistory, func(i, j int) bool {
		return chatHistory[i]["lastActive"].(time.Time).After(chatHistory[j]["lastActive"].(time.Time))
	})

	return chatHistory, nil
}

// Xóa lịch sử chat
func (chs *ChatHistoryService) DeleteChatHistory(channelID primitive.ObjectID) error {
	chatHistoryCollection := chs.DB.Collection("chathistory")
	_, err := chatHistoryCollection.DeleteOne(context.Background(), bson.M{"channelID": channelID})
	return err
}

func (chs *ChatHistoryService) UpdateLastActive(channelID primitive.ObjectID, timestamp time.Time) error {
	collection := chs.DB.Collection("chathistory")
	filter := bson.M{"channelID": channelID}
	update := bson.M{"$set": bson.M{"lastActive": timestamp}}

	_, err := collection.UpdateOne(context.Background(), filter, update)
	return err
}
