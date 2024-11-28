package controllers

import (
	"chat-app-backend/models"
	"chat-app-backend/services"
	"github.com/gorilla/websocket"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
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
func (mc *MessageController) HandleWebSocket(ctx *gin.Context) {
	// Nâng cấp kết nối HTTP lên WebSocket
	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upgrade to WebSocket"})
		return
	}
	defer conn.Close() // Đóng kết nối khi xong

	// Lắng nghe và phản hồi tin nhắn
	for {
		_, msg, err := conn.ReadMessage() // Đọc tin nhắn từ client
		if err != nil {
			break // Ngắt vòng lặp nếu xảy ra lỗi
		}

		// Phản hồi lại tin nhắn (echo)
		err = conn.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			break
		}
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
