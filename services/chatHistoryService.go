package services

import (
	"chat-app-backend/config"
	"chat-app-backend/models"
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
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
func (chs *ChatHistoryService) GetChatHistory(channelID primitive.ObjectID, userID primitive.ObjectID) (map[string]interface{}, error) {
	chatHistoryCollection := chs.DB.Collection("chathistory")
	messagesCollection := chs.DB.Collection("messages")
	userCollection := chs.DB.Collection("users")
	channelsCollection := chs.DB.Collection("channels")

	// Kết quả trả về
	result := make(map[string]interface{})

	// Lấy thông tin kênh
	var channel models.Channel
	err := channelsCollection.FindOne(context.Background(), bson.M{"_id": channelID}).Decode(&channel)
	if err != nil {
		return nil, err
	}

	if channel.ChannelType == "Private" {
		// Lấy ID của đối phương bằng $ne
		var otherUser models.User
		err := userCollection.FindOne(context.Background(), bson.M{
			"_id": bson.M{"$ne": userID},
		}).Decode(&otherUser)
		if err != nil {
			return nil, err
		}

		// Chỉ trả về userName và userAvatar
		result["userName"] = otherUser.Name
		result["userAvatar"] = "http://localhost:8080" + otherUser.Avatar
	} else {
		// Trả về channelName và channelAvatar cho kênh nhóm
		result["channelName"] = channel.ChannelName
		result["channelAvatar"] = "http://localhost:8080" + channel.Avatar
	}

	// Lấy lịch sử tin nhắn
	var chatHistory models.ChatHistory
	err = chatHistoryCollection.FindOne(context.Background(), bson.M{"channelID": channelID}).Decode(&chatHistory)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			result["messages"] = []map[string]interface{}{}
			return result, nil
		}
		return nil, err
	}

	// Lấy danh sách tin nhắn
	filter := bson.M{"_id": bson.M{"$in": chatHistory.Message}}
	cursor, err := messagesCollection.Find(context.Background(), filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	var messages []map[string]interface{}
	for cursor.Next(context.Background()) {
		var msg models.Message
		if err := cursor.Decode(&msg); err != nil {
			return nil, err
		}

		// Lấy thông tin người gửi từ userCollection
		var sender models.User
		err := userCollection.FindOne(context.Background(), bson.M{"_id": msg.SenderID}).Decode(&sender)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				// Nếu không tìm thấy user, bỏ qua tin nhắn hoặc xử lý mặc định
				continue
			}
			return nil, err
		}

		// Thêm thông tin tin nhắn vào kết quả
		messages = append(messages, map[string]interface{}{
			"id":           msg.ID,
			"content":      msg.Content,
			"timestamp":    msg.Timestamp,
			"messageType":  msg.MessageType,
			"senderId":     msg.SenderID,
			"senderName":   sender.Name,
			"senderAvatar": "http://localhost:8080" + sender.Avatar,
			"status":       msg.Status,
			"recalled":     msg.Recalled,
			"url":          msg.URL,
			"fileId":       msg.FileID,
		})
	}

	result["messages"] = messages
	return result, nil
}

// Lấy lịch sử kênh chat của người dùng
func (chs *ChatHistoryService) GetChatHistoryByUserID(userID primitive.ObjectID) ([]map[string]interface{}, error) {
	userChannelsCollection := chs.DB.Collection("userChannels")
	chatHistoryCollection := chs.DB.Collection("chathistory")
	channelsCollection := chs.DB.Collection("channels")
	messagesCollection := chs.DB.Collection("messages")
	userCollection := chs.DB.Collection("users")

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

		var user models.User
		userFilter := bson.M{"_id": bson.M{"$ne": userChannel.UserID}}
		err = userCollection.FindOne(context.Background(), userFilter).Decode(&user)
		if err != nil {
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
			"channelID":     userChannel.ChannelID,
			"channelAvatar": baseUrl + channel.Avatar,
			"channelName":   channel.ChannelName,
			"userName":      user.Name,
			"userAvatar":    user.Avatar,
			"lastMessage":   lastMessageContent,
			"lastActive":    chatHistoryRecord.LastActive,
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
