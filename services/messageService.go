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

func (ms *MessageService) SendMessage(
	channelID, senderID primitive.ObjectID,
	content string, messageType models.MessageType,
	replyTo *primitive.ObjectID,
	attachments []models.Attachment,
) (*models.Message, error) {
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
	now := time.Now()
	// recall window 2 phút (như hiện tại)
	recallDeadline := now.Add(DefaultRecallWindow)
	switch messageType {
	case models.MessageTypeFile, models.MessageTypeVoice:
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
			ID:             primitive.NewObjectID(),
			ChannelID:      channelID,
			Content:        "", // Không lưu trong Content
			Timestamp:      now,
			MessageType:    messageType,
			SenderID:       senderID,
			Status:         models.MessageStatusSending,
			Recalled:       false,
			URL:            file.URL, // Không lưu trong Url
			FileID:         &file.ID, // Liên kết với file ID
			ReplyTo:        replyTo,
			RecallDeadline: &recallDeadline,
			Attachments:    attachments,
		}

	case models.MessageTypeSticker:
		// Tạo tin nhắn cho Voice hoặc Sticker
		message = &models.Message{
			ID:             primitive.NewObjectID(),
			ChannelID:      channelID,
			Content:        "", // Không lưu trong Content
			Timestamp:      now,
			MessageType:    messageType,
			SenderID:       senderID,
			Status:         models.MessageStatusSending,
			Recalled:       false,
			URL:            content, // Lưu URL
			FileID:         nil,
			ReplyTo:        replyTo,
			RecallDeadline: &recallDeadline,
			Attachments:    attachments,
		}

	default:
		// Các loại tin nhắn khác
		message = &models.Message{
			ID:             primitive.NewObjectID(),
			ChannelID:      channelID,
			Content:        content, // Lưu nội dung vào Content
			Timestamp:      now,
			MessageType:    messageType,
			SenderID:       senderID,
			Status:         models.MessageStatusSending,
			Recalled:       false,
			URL:            "", // Không lưu trong Url
			FileID:         nil,
			ReplyTo:        replyTo,
			RecallDeadline: &recallDeadline,
			Attachments:    attachments,
		}
	}

	// Lưu tin nhắn vào collection "messages"
	log.Printf("[SendMessage] Message to insert: %+v", message)
	collection := ms.DB.Collection("messages")
	_, err = collection.InsertOne(context.Background(), message)
	if message.ReplyTo != nil {
		var parent models.Message
		if err := collection.FindOne(context.Background(), bson.M{"_id": *message.ReplyTo}).Decode(&parent); err == nil {
			message.ReplyToMessage = &parent
		}
	}
	if err != nil {
		log.Printf("[SendMessage] Insert message error: %v", err)
		return nil, err
	}
	log.Printf("[SendMessage] Insert message success")

	// Cập nhật lịch sử chat
	chatHistoryCollection := ms.DB.Collection("chathistory")
	filter := bson.M{"channelID": channelID}

	// Preview: nếu có attachments → ghi nhãn thay vì content trống
	previewContent := message.Content
	if len(message.Attachments) > 0 {
		switch message.MessageType {
		case models.MessageTypeFile:
			previewContent = "[Tệp]"
		case models.MessageTypeVoice:
			previewContent = "[Tin nhắn thoại]"
		case models.MessageTypeSticker:
			previewContent = "Sticker"
		default:
			if previewContent != "" {
				previewContent = "[Đính kèm]"
			}
		}
	}
	// lưu cả id và nội dung tin nhắn cuối
	update := bson.M{
		"$set": bson.M{
			"channelID": channelID,
			"lastMessage": models.LastMessagePreview{
				ID:      message.ID,     // để sau này dễ fetch lại
				Content: previewContent, // hiển thị preview nhanh
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

// Tìm channelID từ messageID: ƯU TIÊN đọc từ collection "messages"
// Fallback sang "chathistory" để an toàn trong thời gian chuyển tiếp.
func (ms *MessageService) findChannelIDByMessage(messageID primitive.ObjectID) (primitive.ObjectID, error) {
	// 1) Ưu tiên tra trong messages
	var m struct {
		ChannelID primitive.ObjectID `bson:"channelID"`
	}
	err := ms.DB.Collection("messages").FindOne(
		context.Background(),
		bson.M{"_id": messageID},
		options.FindOne().SetProjection(bson.M{"channelID": 1}),
	).Decode(&m)
	if err == nil && m.ChannelID != primitive.NilObjectID {
		return m.ChannelID, nil
	}

	// 2) Fallback tạm thời: tra trong chathistory (để không vỡ dữ liệu cũ)
	var out struct {
		ChannelID primitive.ObjectID `bson:"channelID"`
	}
	// support cả "message" lẫn "messages" đề phòng dữ liệu cũ
	filter := bson.M{
		"$or": []bson.M{
			{"message": messageID},
			{"messages": messageID},
			{"lastMessage.id": messageID},
		},
	}
	if err2 := ms.DB.Collection("chathistory").FindOne(context.Background(), filter).Decode(&out); err2 == nil {
		return out.ChannelID, nil
	}

	return primitive.NilObjectID, errors.New("channelID not found for message")
}

// Ẩn message cho 1 user (xóa cục bộ)
func (ms *MessageService) HideMessage(messageID, userID primitive.ObjectID) (primitive.ObjectID, error) {
	// addToSet userID vào hiddenBy
	res, err := ms.DB.Collection("messages").UpdateOne(
		context.Background(),
		bson.M{"_id": messageID},
		bson.M{"$addToSet": bson.M{"hiddenBy": userID}},
	)
	if err != nil {
		return primitive.NilObjectID, err
	}
	if res.MatchedCount == 0 {
		return primitive.NilObjectID, errors.New("Message not found")
	}

	// tìm channelID để FE biết đang ẩn trong kênh nào (phục vụ NotifyUser)
	chID, err := ms.findChannelIDByMessage(messageID)
	if err != nil {
		// không critical, vẫn cho ẩn thành công
		log.Printf("[HideMessage] warn: cannot find channelID for message %s: %v", messageID.Hex(), err)
	}
	return chID, nil
}

const DefaultRecallWindow = 2 * time.Minute

// Thu hồi message (toàn cục) — chỉ người gửi, trong khoảng thời gian cho phép
func (ms *MessageService) RecallMessage(messageID, requesterID primitive.ObjectID, window time.Duration) (primitive.ObjectID, error) {
	var msg models.Message
	if err := ms.DB.Collection("messages").FindOne(context.Background(), bson.M{"_id": messageID}).Decode(&msg); err != nil {
		return primitive.NilObjectID, errors.New("Message not found")
	}
	if msg.SenderID != requesterID {
		return primitive.NilObjectID, errors.New("Only sender can recall this message")
	}
	if time.Since(msg.Timestamp) > window {
		return primitive.NilObjectID, errors.New("Recall window has expired")
	}

	// set recalled = true (không xóa nội dung để dễ audit; FE sẽ hiển thị 'đã thu hồi')
	_, err := ms.DB.Collection("messages").UpdateOne(
		context.Background(),
		bson.M{"_id": messageID},
		bson.M{"$set": bson.M{"recalled": true}},
	)
	if err != nil {
		return primitive.NilObjectID, err
	}

	// Nếu message vừa thu hồi là lastMessage, cập nhật preview thành “Tin nhắn đã bị thu hồi”
	chID, err := ms.findChannelIDByMessage(messageID)
	if err == nil {
		_, _ = ms.DB.Collection("chathistory").UpdateOne(
			context.Background(),
			bson.M{"channelID": chID, "lastMessage.id": messageID},
			bson.M{"$set": bson.M{"lastMessage.content": "Tin nhắn đã bị thu hồi"}},
		)
	} else {
		log.Printf("[RecallMessage] warn: cannot find channelID for message %s: %v", messageID.Hex(), err)
	}

	return chID, nil
}

const EditWindow = 15 * time.Minute

func (ms *MessageService) EditMessage(messageID, editorID primitive.ObjectID, newContent string) (*models.Message, error) {
	coll := ms.DB.Collection("messages")
	var msg models.Message
	if err := coll.FindOne(context.TODO(), bson.M{"_id": messageID}).Decode(&msg); err != nil {
		return nil, err
	}

	// chỉ cho phép owner + trong khung 15'
	if msg.SenderID != editorID {
		return nil, errors.New("not your message")
	}
	if time.Since(msg.Timestamp) > EditWindow {
		return nil, errors.New("edit window expired")
	}

	now := time.Now()
	_, err := coll.UpdateOne(context.TODO(),
		bson.M{"_id": messageID},
		bson.M{"$set": bson.M{
			"content":  newContent,
			"edited":   true,
			"editedAt": now,
		}},
	)
	if err != nil {
		return nil, err
	}

	if err := coll.FindOne(context.TODO(), bson.M{"_id": messageID}).Decode(&msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

func (ms *MessageService) ToggleReaction(messageID, userID primitive.ObjectID, emoji string) (*models.Message, error) {
	coll := ms.DB.Collection("messages")
	var msg models.Message
	if err := coll.FindOne(context.TODO(), bson.M{"_id": messageID}).Decode(&msg); err != nil {
		return nil, err
	}

	// tìm reaction theo emoji
	idx := -1
	for i, r := range msg.Reactions {
		if r.Emoji == emoji {
			idx = i
			break
		}
	}

	changed := false
	if idx == -1 {
		// chưa có emoji này → thêm mới
		msg.Reactions = append(msg.Reactions, models.Reaction{
			Emoji:   emoji,
			UserIDs: []primitive.ObjectID{userID},
		})
		changed = true
	} else {
		// đã có → toggle user
		found := false
		users := msg.Reactions[idx].UserIDs
		for i, uid := range users {
			if uid == userID {
				// bỏ reaction
				msg.Reactions[idx].UserIDs = append(users[:i], users[i+1:]...)
				found = true
				changed = true
				break
			}
		}
		if !found {
			msg.Reactions[idx].UserIDs = append(users, userID)
			changed = true
		}
		// nếu rỗng thì xoá hẳn reaction đó
		if len(msg.Reactions[idx].UserIDs) == 0 {
			msg.Reactions = append(msg.Reactions[:idx], msg.Reactions[idx+1:]...)
		}
	}

	if changed {
		if _, err := coll.UpdateOne(context.TODO(),
			bson.M{"_id": messageID},
			bson.M{"$set": bson.M{"reactions": msg.Reactions}},
		); err != nil {
			return nil, err
		}
	}

	if err := coll.FindOne(context.TODO(), bson.M{"_id": messageID}).Decode(&msg); err != nil {
		return nil, err
	}
	return &msg, nil
}
