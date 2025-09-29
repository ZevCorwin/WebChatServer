package services

import (
	"chat-app-backend/config"
	"chat-app-backend/models"
	"context"
	"fmt"
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

// Lấy lịch sử chat == Trả sai thông tin user ==
func (chs *ChatHistoryService) GetChatHistory(channelID, userID primitive.ObjectID) (map[string]interface{}, error) {
	ctx := context.Background()
	chatHistoryCollection := chs.DB.Collection("chathistory")
	messagesCollection := chs.DB.Collection("messages")
	userCollection := chs.DB.Collection("users")
	channelsCollection := chs.DB.Collection("channels")
	userChannelsCollection := chs.DB.Collection("userChannels")

	result := make(map[string]interface{})

	// Lấy thông tin kênh
	var channel models.Channel
	if err := channelsCollection.FindOne(ctx, bson.M{"_id": channelID}).Decode(&channel); err != nil {
		return nil, fmt.Errorf("Không tìm thấy channel: %v", err)
	}

	// Nếu là kênh private → tìm đúng đối phương
	if channel.ChannelType == "Private" {
		var otherUC models.UserChannel
		err := userChannelsCollection.FindOne(ctx, bson.M{
			"channelID": channelID,
			"userID":    bson.M{"$ne": userID},
		}).Decode(&otherUC)
		if err == nil {
			var otherUser models.User
			if err := userCollection.FindOne(ctx, bson.M{"_id": otherUC.UserID}).Decode(&otherUser); err == nil {
				result["userName"] = otherUser.Name
				result["userAvatar"] = "http://localhost:8080" + otherUser.Avatar
			}
		}
	} else {
		result["channelName"] = channel.ChannelName
		result["channelAvatar"] = "http://localhost:8080" + channel.Avatar
	}

	// Lấy lịch sử chat
	var chatHistory models.ChatHistory
	err := chatHistoryCollection.FindOne(ctx, bson.M{"channelID": channelID}).Decode(&chatHistory)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Không có lịch sử → trả về mảng rỗng
			result["messages"] = []map[string]interface{}{}
			return result, nil
		}
		return nil, fmt.Errorf("Lỗi đọc chatHistory: %v", err)
	}

	// Lấy danh sách tin nhắn
	if len(chatHistory.Message) == 0 {
		result["messages"] = []map[string]interface{}{}
		return result, nil
	}

	cursor, err := messagesCollection.Find(ctx, bson.M{"_id": bson.M{"$in": chatHistory.Message}})
	if err != nil {
		return nil, fmt.Errorf("Lỗi truy vấn messages: %v", err)
	}
	defer cursor.Close(ctx)

	var messages []map[string]interface{}
	for cursor.Next(ctx) {
		var msg models.Message
		if err := cursor.Decode(&msg); err != nil {
			continue
		}

		// BỎ nếu user này đã ẩn tin nhắn
		skip := false
		for _, h := range msg.HiddenBy {
			if h == userID {
				skip = true
				break
			}
		}
		if skip {
			continue
		}

		var sender models.User
		_ = userCollection.FindOne(ctx, bson.M{"_id": msg.SenderID}).Decode(&sender)

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

		// Lấy danh sách userID trong channel
		var userIDsInChannel []primitive.ObjectID
		userChannelFilter := bson.M{"channelID": userChannel.ChannelID}
		userCursor, err := userChannelsCollection.Find(context.Background(), userChannelFilter)
		if err != nil {
			return nil, err
		}
		defer userCursor.Close(context.Background())

		for userCursor.Next(context.Background()) {
			var uc models.UserChannel
			if err := userCursor.Decode(&uc); err != nil {
				return nil, err
			}
			userIDsInChannel = append(userIDsInChannel, uc.UserID)
		}

		fmt.Printf("Danh sách userIDsInChannel: %+v\n", userIDsInChannel)
		// Loại bỏ user hiện tại người xem lịch sử
		for i, id := range userIDsInChannel {
			if id == userID {
				userIDsInChannel = append(userIDsInChannel[:i], userIDsInChannel[i+1:]...)
				break
			}
		}

		var user models.User
		if len(userIDsInChannel) > 0 {
			userFilter := bson.M{"_id": userIDsInChannel[0]}
			err = userCollection.FindOne(context.Background(), userFilter).Decode(&user)
			if err != nil {
				fmt.Printf("Lỗi khi tìm user với userID: %v, lỗi: %v\n", userIDsInChannel[0], err)
			} else {
				fmt.Printf("Thông tin user: %+v\n", user)
			}
		}

		// Lấy tin nhắn cuối cùng trong mảng message
		var lastMessageContent string
		if chatHistoryRecord.LastMessage != nil {
			// dữ liệu mới
			lastMessageContent = chatHistoryRecord.LastMessage.Content
		} else if len(chatHistoryRecord.Message) > 0 {
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
			"channelType":   channel.ChannelType,
			"userName":      user.Name,
			"userAvatar":    baseUrl + user.Avatar,
			"lastMessage":   lastMessageContent,
			"lastActive":    chatHistoryRecord.LastActive,
		}

		fmt.Printf("Dữ liệu chatItem: %+v\n", chatItem)
		chatHistory = append(chatHistory, chatItem)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	// Sắp xếp danh sách theo LastActive
	sort.Slice(chatHistory, func(i, j int) bool {
		return chatHistory[i]["lastActive"].(time.Time).After(chatHistory[j]["lastActive"].(time.Time))
	})

	fmt.Printf("Dữ liệu cuối cùng của chatHistory: %+v\n", chatHistory)
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
