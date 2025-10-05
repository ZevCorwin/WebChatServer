package controllers

import (
	"chat-app-backend/models"
	"chat-app-backend/services"
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/goccy/go-json"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"log"
	"net/http"
	"os"
	"sync"
)

// Định nghĩa upgrader cho WebSocket
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Cho phép tất cả nguồn gốc
	},
}

type MessageController struct {
	MessageService   *services.MessageService
	ChannelService   *services.ChannelService
	WebRTCController *WebRTCController
	Clients          map[*websocket.Conn]string // Lưu userID cho mỗi kết nối
	Mutex            sync.Mutex
}

func NewMessageController(ms *services.MessageService, cs *services.ChannelService, wc *WebRTCController) *MessageController {
	return &MessageController{
		MessageService:   ms,
		ChannelService:   cs,
		WebRTCController: wc,
		Clients:          make(map[*websocket.Conn]string),
	}
}

func (mc *MessageController) HandleWebSocket(ctx *gin.Context) {
	// Xác thực JWT
	authHeader := ctx.GetHeader("Authorization")
	tokenQuery := ctx.Query("token")
	log.Printf("Authorization header: %s", authHeader)
	log.Printf("Token query: %s", tokenQuery)
	tokenString := tokenQuery
	if tokenString == "" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Token is required"})
		return
	}
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		secret := os.Getenv("JWT_SECRET")
		if secret == "" {
			log.Printf("JWT_SECRET not set")
			return nil, errors.New("JWT_SECRET not set")
		}
		return []byte(secret), nil
	})
	if err != nil {
		log.Printf("JWT parse error: %v", err)
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": fmt.Sprintf("Invalid or expired token: %v", err)})
		return
	}
	if !token.Valid {
		log.Printf("Token is invalid")
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Token is invalid"})
		return
	}
	log.Printf("Token claims: %+v", claims)
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		log.Printf("Invalid claims type")
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims type"})
		return
	}
	log.Printf("Claims[sub]: %v", claims["sub"])
	log.Printf("Claims[user_id]: %v", claims["user_id"])
	if claims["user_id"] == nil {
		log.Printf("No user_id in claims")
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "No user_id in token"})
		return
	}
	userID, ok := claims["user_id"].(string)
	if !ok {
		log.Printf("Invalid userID in user_id")
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid userID in token"})
		return
	}
	log.Printf("Authenticated userID: %s", userID)

	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upgrade to WebSocket"})
		return
	}
	log.Printf("WebSocket connected for userID: %s", userID)
	defer conn.Close()

	// Lưu kết nối với userID
	mc.Mutex.Lock()
	mc.Clients[conn] = userID
	mc.WebRTCController.Connections[userID] = conn
	log.Printf("Stored connection for userID %s: %p", userID, conn)
	mc.Mutex.Unlock()

	for {
		// Đọc tin nhắn từ WebSocket
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read error for userID %s: %v", userID, err)
			mc.Mutex.Lock()
			delete(mc.Clients, conn)
			mc.Mutex.Unlock()
			break
		}
		log.Printf("Received message from userID %s: %s", userID, string(msg))

		// Giải mã tin nhắn nhận được
		var incomingMessage struct {
			ChannelID   string              `json:"channelId"`
			SenderID    string              `json:"senderId"`
			Content     string              `json:"content"`
			MessageType string              `json:"messageType"`
			ReplyTo     *string             `json:"replyTo"`
			Attachments []models.Attachment `json:"attachments"`
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
		var replyToOID *primitive.ObjectID
		if incomingMessage.ReplyTo != nil && *incomingMessage.ReplyTo != "" {
			if oid, err := primitive.ObjectIDFromHex(*incomingMessage.ReplyTo); err == nil {
				replyToOID = &oid
			}
		}

		// Sử dụng MessageService để gửi tin nhắn và lấy dữ liệu phản hồi
		message, err := mc.MessageService.SendMessage(
			channelID,
			senderID,
			incomingMessage.Content,
			models.MessageType(incomingMessage.MessageType),
			replyToOID,
			incomingMessage.Attachments,
		)
		if err != nil {
			log.Printf("Lỗi gửi tin nhắn: %v", err)
			continue
		}
		log.Printf("[HandleWebSocket] Message saved: %+v", message)

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
			"type":         "message_new",
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
			"channelId":    message.ChannelID.Hex(),
			"replyTo":      replyToOID,
			"attachments":  message.Attachments,
		}

		// Broadcast đến các thành viên kênh
		log.Printf("[HandleWebSocket] Response: %+v", response)
		mc.WebRTCController.BroadcastMessage(channelID, response)
	}
}

// Thu hồi tin nhắn — POST /api/messages/:messageID/recall
func (mc *MessageController) RecallMessageHandler(ctx *gin.Context) {
	userIDHex := ctx.GetString("user_id")
	if userIDHex == "" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	requesterID, err := primitive.ObjectIDFromHex(userIDHex)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user id"})
		return
	}

	msgHex := ctx.Param("messageID")
	msgID, err := primitive.ObjectIDFromHex(msgHex)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid message id"})
		return
	}

	chID, err := mc.MessageService.RecallMessage(msgID, requesterID, services.DefaultRecallWindow)
	if err != nil {
		ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	// Broadcast tới cả kênh: message đã bị thu hồi
	mc.WebRTCController.BroadcastMessage(chID, gin.H{
		"type":      "message_recalled",
		"channelId": chID.Hex(),
		"messageId": msgID.Hex(),
		"by":        requesterID.Hex(),
	})

	ctx.JSON(http.StatusOK, gin.H{"message": "Recalled successfully"})
}

// Ẩn tin nhắn cho riêng người gọi — DELETE /api/messages/:messageID/hide
func (mc *MessageController) HideMessageHandler(ctx *gin.Context) {
	userIDHex := ctx.GetString("user_id")
	if userIDHex == "" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID, err := primitive.ObjectIDFromHex(userIDHex)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user id"})
		return
	}

	msgHex := ctx.Param("messageID")
	msgID, err := primitive.ObjectIDFromHex(msgHex)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid message id"})
		return
	}

	chID, err := mc.MessageService.HideMessage(msgID, userID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Notify riêng user này để FE xoá item khỏi UI (không broadcast cho cả kênh)
	mc.WebRTCController.NotifyUser(userIDHex, gin.H{
		"type":      "message_hidden",
		"channelId": chID.Hex(),
		"messageId": msgID.Hex(),
	})

	ctx.JSON(http.StatusOK, gin.H{"message": "Hidden locally"})
}
