package controllers

import (
	"chat-app-backend/models"
	"chat-app-backend/services"
	"context"
	"github.com/gin-gonic/gin"
	"github.com/goccy/go-json"
	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"log"
	"net/http"
	"sync"
)

// Định nghĩa upgrader cho WebSocket
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Cho phép tất cả nguồn gốc
	},
}

type MessageController struct {
	MessageService *services.MessageService
	ChannelService *services.ChannelService
}

func NewMessageController(messageService *services.MessageService, channelService *services.ChannelService) *MessageController {
	return &MessageController{MessageService: messageService, ChannelService: channelService}
}

// HandleWebSocket xử lý kết nối WebSocket
var (
	clients   = make(map[*websocket.Conn]bool) // Lưu trữ danh sách kết nối
	broadcast = make(chan []byte)              // Kênh để broadcast tin nhắn
	mutex     sync.Mutex                       // Đảm bảo thread-safe
)

// Goroutine để gửi tin nhắn đến tất cả client
func init() {
	go func() {
		for {
			message := <-broadcast
			mutex.Lock()
			for client := range clients {
				err := client.WriteMessage(websocket.TextMessage, message)
				if err != nil {
					client.Close()
					delete(clients, client)
				}
			}
			mutex.Unlock()
		}
	}()
}

func (mc *MessageController) HandleWebSocket(ctx *gin.Context) {
	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upgrade to WebSocket"})
		return
	}
	defer conn.Close()

	// Thêm kết nối WebSocket vào danh sách clients
	mutex.Lock()
	clients[conn] = true
	mutex.Unlock()

	for {
		// Đọc tin nhắn từ WebSocket
		_, msg, err := conn.ReadMessage()
		if err != nil {
			mutex.Lock()
			delete(clients, conn)
			mutex.Unlock()
			break
		}

		// Giải mã tin nhắn nhận được
		var incomingMessage struct {
			ChannelID   string `json:"channelId"`
			SenderID    string `json:"senderId"`
			Content     string `json:"content"`
			MessageType string `json:"messageType"`
		}
		if err := json.Unmarshal(msg, &incomingMessage); err != nil {
			log.Printf("Lỗi giải mã tin nhắn: %v", err)
			continue
		}

		// Chuyển đổi ChannelID và SenderID sang ObjectID
		channelID, err := primitive.ObjectIDFromHex(incomingMessage.ChannelID)
		if err != nil {
			log.Printf("Lỗi chuyển đổi ChannelID: %v", err)
			continue
		}

		senderID, err := primitive.ObjectIDFromHex(incomingMessage.SenderID)
		if err != nil {
			log.Printf("Lỗi chuyển đổi SenderID: %v", err)
			continue
		}

		// Sử dụng MessageService để gửi tin nhắn và lấy dữ liệu phản hồi
		message, err := mc.MessageService.SendMessage(
			channelID,
			senderID,
			incomingMessage.Content,
			models.MessageType(incomingMessage.MessageType),
		)
		if err != nil {
			log.Printf("Lỗi gửi tin nhắn: %v", err)
			continue
		}

		// Truy vấn thông tin người gửi để tạo phản hồi nhất quán
		var sender struct {
			Name   string `bson:"name"`
			Avatar string `bson:"avatar"`
		}
		err = mc.MessageService.DB.Collection("users").FindOne(
			context.TODO(),
			bson.M{"_id": senderID},
		).Decode(&sender)
		if err != nil {
			log.Printf("Lỗi truy vấn thông tin người gửi: %v", err)
			continue
		}

		// Chuẩn hóa phản hồi
		response := map[string]interface{}{
			"id":           message.ID.Hex(),
			"content":      message.Content,
			"timestamp":    message.Timestamp,
			"messageType":  message.MessageType,
			"senderId":     incomingMessage.SenderID,
			"senderName":   sender.Name,
			"senderAvatar": "http://localhost:8080" + sender.Avatar,
			"status":       message.Status,
			"recalled":     message.Recalled,
			"url":          message.URL,
			"fileId":       message.FileID,
		}

		// Broadcast tin nhắn đến tất cả client
		broadcast <- func() []byte {
			resp, _ := json.Marshal(response)
			return resp
		}()
	}
}

// SendMessage API
func (mc *MessageController) SendMessage(ctx *gin.Context) {
	var request struct {
		ChannelID   string `json:"channelID"`
		SenderID    string `json:"senderID"`
		Content     string `json:"content"`
		MessageType string `json:"messageType"`
	}

	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	channelID, err := primitive.ObjectIDFromHex(request.ChannelID)
	if err != nil {
		log.Print(request.ChannelID)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel ID"})
		return
	}

	senderID, err := primitive.ObjectIDFromHex(request.SenderID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid sender ID"})
		return
	}

	messageType := models.MessageType(request.MessageType)
	if messageType != models.MessageTypeText && messageType != models.MessageTypeVoice {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported message type"})
		return
	}

	message, err := mc.MessageService.SendMessage(channelID, senderID, request.Content, messageType)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": message})
}
