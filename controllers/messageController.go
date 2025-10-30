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
	"strings"
	"sync"
	"time"
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

	oid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user id"})
		return
	}
	var user models.User
	if err := mc.MessageService.DB.Collection("users").
		FindOne(context.TODO(), bson.M{"_id": oid}).Decode(&user); err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}
	if user.IsLocked() {
		var until string
		if user.LockedUntil != nil {
			until = user.LockedUntil.Format(time.RFC3339)
		}
		ctx.JSON(http.StatusLocked, gin.H{
			"error":       "Account locked",
			"lockedUntil": until,
			"reason":      user.LockReason,
		})
		return
	}

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
	mc.WebRTCController.OnUserConnected(userID)

	for {
		// Đọc tin nhắn từ WebSocket
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read error for userID %s: %v", userID, err)
			mc.Mutex.Lock()
			delete(mc.Clients, conn)
			delete(mc.WebRTCController.Connections, userID)
			mc.Mutex.Unlock()
			mc.WebRTCController.OnUserDisconnected(userID)
			break
		}
		log.Printf("Received message from userID %s: %s", userID, string(msg))

		var raw map[string]interface{}
		if err := json.Unmarshal(msg, &raw); err == nil {
			// Nếu client gửi event typing: { type: "typing", channelId: "...", senderId: "...", isTyping: true/false, senderName?: "..."}
			if t, ok := raw["type"].(string); ok && t == "typing" {
				chanIDStr, _ := raw["channelId"].(string)
				senderIDStr, _ := raw["senderId"].(string)
				isTyping, _ := raw["isTyping"].(bool)
				senderName, _ := raw["senderName"].(string) // optional

				if chanIDStr == "" {
					// thiếu channel -> bỏ qua
					continue
				}

				cid, err := primitive.ObjectIDFromHex(chanIDStr)
				if err != nil {
					log.Printf("[HandleWebSocket] Invalid channelId in typing event: %v", err)
					continue
				}

				resp := map[string]interface{}{
					"type":      "typing",
					"channelId": chanIDStr,
					"senderId":  senderIDStr,
					"isTyping":  isTyping,
				}
				// nếu có senderName gửi kèm thì kèm luôn (giúp hiển thị tên nhanh ở client)
				if senderName != "" {
					resp["senderName"] = senderName
				}

				// Broadcast tới tất cả member trong channel
				mc.WebRTCController.BroadcastMessage(cid, resp)
				// Không lưu DB, không tiếp tục xử lý gửi như tin nhắn
				continue
			}
		}

		if t, ok := raw["type"].(string); ok {
			switch t {
			case "message_delivered":
				mid, _ := raw["messageId"].(string)
				cid, _ := raw["channelId"].(string)
				uid, _ := raw["userId"].(string)
				if mid == "" || cid == "" || uid == "" {
					continue
				}

				msgOID, err1 := primitive.ObjectIDFromHex(mid)
				chanOID, err2 := primitive.ObjectIDFromHex(cid)
				userOID, err3 := primitive.ObjectIDFromHex(uid)
				if err1 != nil || err2 != nil || err3 != nil {
					continue
				}

				// ĐỌC MSG HIỆN TẠI ĐỂ GATE
				var cur models.Message
				if err := mc.MessageService.DB.Collection("messages").
					FindOne(context.TODO(), bson.M{"_id": msgOID}).Decode(&cur); err != nil {
					continue
				}
				// Nếu đã >= Received thì bỏ qua (tránh notify lặp)
				if int(cur.StatusStage) >= int(models.StatusStageReceived) {
					continue
				}

				// Lưu delivery (có thể vẫn bị gọi trùng nhưng không sao)
				if err := mc.MessageService.AddDeliveryReceipt(msgOID, chanOID, userOID); err != nil {
					log.Printf("[WS] AddDeliveryReceipt error: %v", err)
					continue
				}

				// Nâng cấp tiến cấp
				newStatus, err := mc.MessageService.AdvanceStatus(msgOID, models.StatusStageReceived)
				if err != nil {
					// Không nâng thêm được (có thể có race) -> thôi, khỏi notify
					continue
				}

				// Lấy sender để notify
				var msg models.Message
				_ = mc.MessageService.DB.Collection("messages").
					FindOne(context.TODO(), bson.M{"_id": msgOID}).Decode(&msg)

				payload := map[string]interface{}{
					"type":        "message_status_update",
					"messageId":   mid,
					"status":      string(newStatus),
					"statusStage": int(models.StatusStageReceived), // ✅ số
					"userId":      uid,
					"channelId":   cid,
				}
				mc.WebRTCController.NotifyUser(msg.SenderID.Hex(), payload)
				continue

			case "message_read":
				mid, _ := raw["messageId"].(string)
				cid, _ := raw["channelId"].(string)
				uid, _ := raw["userId"].(string)
				if mid == "" || cid == "" || uid == "" {
					continue
				}
				msgOID, err1 := primitive.ObjectIDFromHex(mid)
				chanOID, err2 := primitive.ObjectIDFromHex(cid)
				userOID, err3 := primitive.ObjectIDFromHex(uid)
				if err1 != nil || err2 != nil || err3 != nil {
					continue
				}

				// ĐỌC MSG HIỆN TẠI ĐỂ GATE
				var cur models.Message
				if err := mc.MessageService.DB.Collection("messages").
					FindOne(context.TODO(), bson.M{"_id": msgOID}).Decode(&cur); err != nil {
					continue
				}
				// Nếu đã >= Received thì bỏ qua (tránh notify lặp)
				if int(cur.StatusStage) >= int(models.StatusStageSeen) {
					continue
				}

				// Lưu delivery (có thể vẫn bị gọi trùng nhưng không sao)
				if err := mc.MessageService.AddReadReceipt(msgOID, chanOID, userOID); err != nil {
					log.Printf("[WS] AddDeliveryReceipt error: %v", err)
					continue
				}

				// Nâng cấp tiến cấp
				newStatus, err := mc.MessageService.AdvanceStatus(msgOID, models.StatusStageSeen)
				if err != nil {
					// Không nâng thêm được (có thể có race) -> thôi, khỏi notify
					continue
				}

				// Lấy sender để notify
				var msg models.Message
				_ = mc.MessageService.DB.Collection("messages").
					FindOne(context.TODO(), bson.M{"_id": msgOID}).Decode(&msg)

				// Notify sender (nếu online)
				payload := map[string]interface{}{
					"type":        "message_status_update",
					"messageId":   mid,
					"status":      string(newStatus),
					"statusStage": int(models.StatusStageSeen), // "Đã xem"
					"userId":      uid,
					"channelId":   cid,
				}
				mc.WebRTCController.NotifyUser(msg.SenderID.Hex(), payload)
				continue
			} // end switch t
		} // end if t

		// Giải mã tin nhắn nhận được
		var incomingMessage struct {
			ChannelID       string              `json:"channelId"`
			SenderID        string              `json:"senderId"`
			Content         string              `json:"content"`
			MessageType     string              `json:"messageType"`
			ReplyTo         *string             `json:"replyTo"`
			Attachments     []models.Attachment `json:"attachments"`
			ClientMessageID string              `json:"clientMessageId"`
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
		_, _ = mc.MessageService.AdvanceStatus(message.ID, models.StatusStageSent)
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
		var replyPreview map[string]interface{}
		if message.ReplyTo != nil && message.ReplyToMessage != nil {
			replyPreview = map[string]interface{}{
				"id":       message.ReplyToMessage.ID.Hex(),
				"content":  message.ReplyToMessage.Content,
				"senderId": message.ReplyToMessage.SenderID.Hex(),
				"senderName": func() string {
					var u struct {
						Name string `bson:"name"`
					}
					_ = mc.MessageService.DB.Collection("users").FindOne(
						context.TODO(),
						bson.M{"_id": message.ReplyToMessage.SenderID},
					).Decode(&u)
					return u.Name
				}(),
				"messageType": message.ReplyToMessage.MessageType,
			}
		}

		response := map[string]interface{}{
			"type":            "message_new",
			"id":              message.ID.Hex(),
			"clientMessageId": incomingMessage.ClientMessageID,
			"content":         message.Content,
			"timestamp":       message.Timestamp,
			"messageType":     message.MessageType,
			"senderId":        incomingMessage.SenderID,
			"senderName":      sender.Name,
			"senderAvatar":    "http://localhost:8080" + sender.Avatar,
			"status":          models.MessageStatusSent,
			"statusStage":     int(models.StatusStageSent),
			"recalled":        message.Recalled,
			"url":             message.URL,
			"fileId":          message.FileID,
			"channelId":       message.ChannelID.Hex(),
			"replyTo":         replyPreview,
			"attachments":     message.Attachments,
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

func (mc *MessageController) EditMessage(ctx *gin.Context) {
	msgIDHex := ctx.Param("messageID")
	msgID, err := primitive.ObjectIDFromHex(msgIDHex)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid message id"})
		return
	}

	// auth user id từ JWT (bạn đang dùng ctx header token ở WS; với REST bạn đang có middleware auth rồi)
	userIDHex := ctx.GetString("user_id") // nếu middleware set; nếu chưa có, parse giống WS
	if userIDHex == "" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	editorID, _ := primitive.ObjectIDFromHex(userIDHex)

	var body struct {
		Content string `json:"content"`
	}
	if err := ctx.BindJSON(&body); err != nil || strings.TrimSpace(body.Content) == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "content required"})
		return
	}

	msg, err := mc.MessageService.EditMessage(msgID, editorID, body.Content)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 🔧 LẤY THÔNG TIN NGƯỜI GỬI để trả về đầy đủ cho FE
	var sender struct {
		Name   string `bson:"name"`
		Avatar string `bson:"avatar"`
	}
	_ = mc.MessageService.DB.Collection("users").FindOne(
		context.TODO(),
		bson.M{"_id": msg.SenderID},
	).Decode(&sender)

	// broadcast
	resp := map[string]interface{}{
		"type":         "message_updated",
		"id":           msg.ID.Hex(),
		"channelId":    msg.ChannelID.Hex(),
		"content":      msg.Content,
		"edited":       msg.Edited,
		"editedAt":     msg.EditedAt,
		"messageType":  msg.MessageType, // ✅ thêm loại tin nhắn
		"senderId":     msg.SenderID.Hex(),
		"senderName":   sender.Name,
		"senderAvatar": "http://localhost:8080" + sender.Avatar,
		"timestamp":    msg.Timestamp, // ✅ thêm timestamp
		"recalled":     msg.Recalled,
		"status":       msg.Status,
	}
	mc.WebRTCController.BroadcastMessage(msg.ChannelID, resp)

	ctx.JSON(http.StatusOK, msg)
}

func (mc *MessageController) ToggleReaction(ctx *gin.Context) {
	msgIDHex := ctx.Param("messageID")
	msgID, err := primitive.ObjectIDFromHex(msgIDHex)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid message id"})
		return
	}

	userIDHex := ctx.GetString("user_id")
	if userIDHex == "" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID, _ := primitive.ObjectIDFromHex(userIDHex)

	var body struct {
		Emoji string `json:"emoji"`
	}
	if err := ctx.BindJSON(&body); err != nil || body.Emoji == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "emoji required"})
		return
	}

	msg, err := mc.MessageService.ToggleReaction(msgID, userID, body.Emoji)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// build reaction summary (emoji + count + hasMine)
	type R struct {
		Emoji   string               `json:"emoji"`
		UserIDs []primitive.ObjectID `json:"userIDs"`
		Count   int                  `json:"count"`
	}
	var rs []R
	for _, r := range msg.Reactions {
		rs = append(rs, R{
			Emoji:   r.Emoji,
			UserIDs: r.UserIDs,
			Count:   len(r.UserIDs),
		})
	}

	resp := map[string]interface{}{
		"type":      "message_reaction",
		"messageId": msg.ID.Hex(),
		"channelId": msg.ChannelID.Hex(),
		"reactions": rs,
	}
	mc.WebRTCController.BroadcastMessage(msg.ChannelID, resp)

	ctx.JSON(http.StatusOK, msg)
}
