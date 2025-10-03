package services

import (
	"chat-app-backend/config"
	"chat-app-backend/models"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"os"
	"sort"
	"time"
)

type ChatHistoryService struct {
	DB *mongo.Database
}

func NewChatHistoryService() *ChatHistoryService {
	return &ChatHistoryService{DB: config.DB}
}

// GetChatHistory (sửa mới) → trả messages trực tiếp, không dùng chatHistory.Message
func (chs *ChatHistoryService) GetChatHistory(channelID, userID primitive.ObjectID) (map[string]interface{}, error) {
	ctx := context.Background()
	userCollection := chs.DB.Collection("users")
	channelsCollection := chs.DB.Collection("channels")
	messagesCollection := chs.DB.Collection("messages")

	result := make(map[string]interface{})

	// --- Channel info ---
	var channel models.Channel
	if err := channelsCollection.FindOne(ctx, bson.M{"_id": channelID}).Decode(&channel); err != nil {
		return nil, fmt.Errorf("Không tìm thấy channel: %v", err)
	}
	if channel.ChannelType == "Private" {
		// tìm đối phương
		var otherUC models.UserChannel
		if err := chs.DB.Collection("userChannels").FindOne(ctx, bson.M{
			"channelID": channelID,
			"userID":    bson.M{"$ne": userID},
		}).Decode(&otherUC); err == nil {
			var otherUser models.User
			if err := userCollection.FindOne(ctx, bson.M{"_id": otherUC.UserID}).Decode(&otherUser); err == nil {
				result["userName"] = otherUser.Name
				result["userAvatar"] = fullAvatarURL(otherUser.Avatar)
			}
		}
	} else {
		result["channelName"] = channel.ChannelName
		result["channelAvatar"] = fullAvatarURL(channel.Avatar)
	}

	// --- Messages ---
	cur, err := messagesCollection.Find(ctx, bson.M{"channelID": channelID}, options.Find().SetSort(bson.D{{"timestamp", 1}}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var messages []map[string]interface{}
	for cur.Next(ctx) {
		var msg models.Message
		if err := cur.Decode(&msg); err != nil {
			continue
		}

		// skip hidden
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
			"senderAvatar": fullAvatarURL(sender.Avatar),
			"status":       msg.Status,
			"recalled":     msg.Recalled,
			"url":          msg.URL,
			"fileId":       msg.FileID,
		})
	}

	result["messages"] = messages
	return result, nil
}

// Lấy lịch sử kênh chat của người dùng (KHÔNG phụ thuộc chatHistory.Message)
func (chs *ChatHistoryService) GetChatHistoryByUserID(userID primitive.ObjectID) ([]map[string]interface{}, error) {
	userChannelsColl := chs.DB.Collection("userChannels")
	chathistoryColl := chs.DB.Collection("chathistory")
	channelsColl := chs.DB.Collection("channels")
	messagesColl := chs.DB.Collection("messages")
	usersColl := chs.DB.Collection("users")

	// 1) Lấy các channel của user
	cur, err := userChannelsColl.Find(context.Background(), bson.M{"userID": userID})
	if err != nil {
		return nil, err
	}
	defer cur.Close(context.Background())

	base := os.Getenv("PUBLIC_BASE_URL")
	if base == "" {
		base = "http://localhost:8080"
	}

	var items []map[string]interface{}

	for cur.Next(context.Background()) {
		var uc models.UserChannel
		if err := cur.Decode(&uc); err != nil {
			return nil, err
		}

		// 2) Thông tin kênh
		var channel models.Channel
		if err := channelsColl.FindOne(context.Background(), bson.M{"_id": uc.ChannelID}).Decode(&channel); err != nil {
			// kênh không còn → bỏ qua
			continue
		}

		// 3) Chathistory record (để đọc lastActive, lastMessage preview)
		var ch models.ChatHistory
		_ = chathistoryColl.FindOne(context.Background(), bson.M{"channelID": uc.ChannelID}).Decode(&ch)

		// 4) Meta hiển thị
		var userName, userAvatar, channelName, channelAvatar string
		if channel.ChannelType == "Private" {
			// tìm đối phương
			var otherUC models.UserChannel
			if err := userChannelsColl.FindOne(
				context.Background(),
				bson.M{"channelID": uc.ChannelID, "userID": bson.M{"$ne": userID}},
			).Decode(&otherUC); err == nil {
				var other models.User
				if err := usersColl.FindOne(context.Background(), bson.M{"_id": otherUC.UserID}).Decode(&other); err == nil {
					userName = other.Name
					userAvatar = base + other.Avatar
				}
			}
		} else {
			channelName = channel.ChannelName
			channelAvatar = base + channel.Avatar
		}

		// 5) Xác định lastMessageContent & lastActive
		lastMessageContent := ""
		lastActive := ch.LastActive // ưu tiên lastActive từ chathistory

		// Ưu tiên preview có sẵn trong chathistory
		if ch.LastMessage != nil && ch.LastMessage.Content != "" {
			lastMessageContent = ch.LastMessage.Content
		} else {
			// fallback: lấy message mới nhất theo channelID (loại trừ tin user này đã ẩn)
			filter := bson.M{
				"channelID": uc.ChannelID,
				"$or": []bson.M{
					{"hiddenBy": bson.M{"$exists": false}},
					{"hiddenBy": bson.M{"$ne": userID}},
				},
			}
			var lastMsg models.Message
			if err := messagesColl.FindOne(
				context.Background(),
				filter,
				options.FindOne().SetSort(bson.D{{Key: "timestamp", Value: -1}}),
			).Decode(&lastMsg); err == nil {
				if lastMsg.Recalled {
					lastMessageContent = "Tin nhắn đã bị thu hồi"
				} else {
					switch lastMsg.MessageType {
					case models.MessageTypeFile:
						lastMessageContent = "[Tệp]"
					case models.MessageTypeVoice:
						lastMessageContent = "[Voice]"
					case models.MessageTypeSticker:
						lastMessageContent = "[Sticker]"
					default:
						lastMessageContent = lastMsg.Content
					}
				}
				// nếu chathistory chưa có lastActive thì mượn từ message mới nhất
				if lastActive.IsZero() && !lastMsg.Timestamp.IsZero() {
					lastActive = lastMsg.Timestamp
				}
			}
		}

		item := map[string]interface{}{
			"channelID":     uc.ChannelID,
			"channelAvatar": channelAvatar,
			"channelName":   channelName,
			"channelType":   channel.ChannelType,
			"userName":      userName,
			"userAvatar":    userAvatar,
			"lastMessage":   lastMessageContent,
			"lastActive":    lastActive,
		}
		items = append(items, item)
	}
	if err := cur.Err(); err != nil {
		return nil, err
	}

	// 6) Sắp xếp theo lastActive giảm dần
	sort.Slice(items, func(i, j int) bool {
		ti, _ := items[i]["lastActive"].(time.Time)
		tj, _ := items[j]["lastActive"].(time.Time)
		return ti.After(tj)
	})

	return items, nil
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

// Lấy danh sách message trực tiếp từ collection "messages"
// beforeTS == zero time => lấy mới nhất; limit (1..100), mặc định 50
func (chs *ChatHistoryService) GetChannelMessages(
	channelID primitive.ObjectID,
	viewerID primitive.ObjectID,
	beforeTS time.Time,
	limit int64,
) ([]map[string]interface{}, error) {

	if limit <= 0 || limit > 100 {
		limit = 50
	}

	match := bson.M{
		"channelID": channelID,
		"$or": []bson.M{
			{"hiddenBy": bson.M{"$exists": false}},
			{"hiddenBy": bson.M{"$ne": viewerID}},
		},
	}
	if !beforeTS.IsZero() {
		match["timestamp"] = bson.M{"$lt": beforeTS}
	}

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: match}},
		{{Key: "$sort", Value: bson.M{"timestamp": -1}}},
		{{Key: "$limit", Value: limit}},
		{
			{Key: "$lookup", Value: bson.M{
				"from":         "users",
				"localField":   "senderID",
				"foreignField": "_id",
				"as":           "sender",
			}},
		},
		{
			{Key: "$addFields", Value: bson.M{
				"senderName": bson.M{
					"$ifNull": []interface{}{
						bson.M{"$arrayElemAt": []interface{}{"$sender.name", 0}},
						"",
					},
				},
				"senderAvatar": bson.M{
					"$ifNull": []interface{}{
						bson.M{"$arrayElemAt": []interface{}{"$sender.avatar", 0}},
						"",
					},
				},
			}},
		},
		{
			{Key: "$project", Value: bson.M{
				"_id":          1,
				"content":      1,
				"timestamp":    1,
				"messageType":  1,
				"senderID":     1,
				"status":       1,
				"recalled":     1,
				"url":          1,
				"fileID":       1,
				"channelID":    1,
				"senderName":   1,
				"senderAvatar": 1,
			}},
		},
		{{Key: "$sort", Value: bson.M{"timestamp": 1}}}, // trả về theo thứ tự tăng
	}

	cur, err := chs.DB.Collection("messages").Aggregate(context.Background(), pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(context.Background())

	var out []map[string]interface{}
	for cur.Next(context.Background()) {
		var m struct {
			ID           primitive.ObjectID   `bson:"_id"`
			Content      string               `bson:"content"`
			Timestamp    time.Time            `bson:"timestamp"`
			MessageType  models.MessageType   `bson:"messageType"`
			SenderID     primitive.ObjectID   `bson:"senderID"`
			Status       models.MessageStatus `bson:"status"`
			Recalled     bool                 `bson:"recalled"`
			URL          string               `bson:"url"`
			FileID       *primitive.ObjectID  `bson:"fileID"`
			ChannelID    primitive.ObjectID   `bson:"channelID"`
			SenderName   string               `bson:"senderName"`
			SenderAvatar string               `bson:"senderAvatar"`
		}
		if err := cur.Decode(&m); err != nil {
			return nil, err
		}

		out = append(out, map[string]interface{}{
			"id":           m.ID.Hex(),
			"content":      m.Content,
			"timestamp":    m.Timestamp,
			"messageType":  m.MessageType,
			"senderId":     m.SenderID.Hex(),
			"senderName":   m.SenderName,
			"senderAvatar": fullAvatarURL(m.SenderAvatar),
			"status":       m.Status,
			"recalled":     m.Recalled,
			"url":          m.URL,
			"fileId":       m.FileID,
			"channelId":    m.ChannelID.Hex(),
		})
	}
	if err := cur.Err(); err != nil {
		return nil, err
	}

	return out, nil
}

// trả về absolute URL cho avatar (lấy từ env PUBLIC_BASE_URL, mặc định localhost)
func fullAvatarURL(path string) string {
	if path == "" {
		return ""
	}
	base := os.Getenv("PUBLIC_BASE_URL")
	if base == "" {
		base = "http://localhost:8080"
	}
	return base + path
}
