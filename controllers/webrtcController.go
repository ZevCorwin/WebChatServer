package controllers

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"chat-app-backend/services"
	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type WebRTCController struct {
	Connections    map[string]*websocket.Conn
	MessageService *services.MessageService
	ChannelService *services.ChannelService
	mu             sync.Mutex
}

// Khởi tạo controller
func NewWebRTCController(ms *services.MessageService, cs *services.ChannelService) *WebRTCController {
	return &WebRTCController{
		Connections:    make(map[string]*websocket.Conn),
		MessageService: ms,
		ChannelService: cs,
	}
}

var upgraderWebRTC = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// ====== PRESENCE ======

func (wc *WebRTCController) setUserStatus(userID string, online bool) (lastSeen *time.Time, err error) {
	uColl := wc.MessageService.DB.Collection("users")

	if online {
		// Online: set status=Online, KHÔNG đụng lastOnlineTime
		_, err = uColl.UpdateOne(
			context.Background(),
			bson.M{"_id": toOID(userID)},
			bson.M{"$set": bson.M{"status": "Online"}},
		)
		return nil, err
	}

	// Offline: set status=Offline + lastOnlineTime = now
	now := time.Now().UTC()
	_, err = uColl.UpdateOne(
		context.Background(),
		bson.M{"_id": toOID(userID)},
		bson.M{"$set": bson.M{"status": "Offline", "lastOnlineTime": now}},
	)
	return &now, err
}

func toOID(hex string) primitive.ObjectID {
	oid, _ := primitive.ObjectIDFromHex(hex)
	return oid
}

func (wc *WebRTCController) BroadcastPresence(userID string, online bool, lastSeen *time.Time) {
	wc.mu.Lock()
	defer wc.mu.Unlock()

	payload := map[string]interface{}{
		"type":   "presence",
		"userId": userID,
		"online": online,
	}
	if !online && lastSeen != nil {
		// cho FE tính “x phút trước”
		payload["lastSeen"] = lastSeen.Format(time.RFC3339)
	}

	// MVP: phát cho TẤT CẢ user đang kết nối (đơn giản, ít đụng code).
	// Nếu muốn giới hạn theo kênh, có thể loop qua các channel của user và
	// chỉ gửi cho member của các channel đó.
	for uid, conn := range wc.Connections {
		if conn == nil {
			continue
		}
		if err := conn.WriteJSON(payload); err != nil {
			log.Printf("[Presence] send to %s err: %v", uid, err)
		}
	}
}

// Call khi user connect WS (đặt ở MessageController sau khi gán Connections[userID]=conn)
func (wc *WebRTCController) OnUserConnected(userID string) {
	if _, err := wc.setUserStatus(userID, true); err != nil {
		log.Printf("[Presence] set online err: %v", err)
	}
	wc.BroadcastPresence(userID, true, nil)
}

// Call khi user disconnect WS (đặt ở defer trong MessageController trước khi xoá map)
func (wc *WebRTCController) OnUserDisconnected(userID string) {
	last, err := wc.setUserStatus(userID, false)
	if err != nil {
		log.Printf("[Presence] set offline err: %v", err)
	}
	wc.BroadcastPresence(userID, false, last)
}

// ====== NOTIFY / BROADCAST NHẮN TIN CŨ ======

func (wc *WebRTCController) NotifyUser(userID string, message interface{}) {
	wc.mu.Lock()
	conn, ok := wc.Connections[userID]
	wc.mu.Unlock()

	if ok && conn != nil {
		if err := conn.WriteJSON(message); err != nil {
			log.Printf("Error sending message to user %s: %v\n", userID, err)
		}
	}
}

func (wc *WebRTCController) BroadcastMessage(channelID primitive.ObjectID, message interface{}) {
	channel, err := wc.ChannelService.GetChannel(channelID)
	if err != nil {
		log.Printf("Error getting channel: %v\n", err)
		return
	}

	wc.mu.Lock()
	defer wc.mu.Unlock()

	for _, member := range channel.Members {
		if conn, ok := wc.Connections[member.MemberID.Hex()]; ok && conn != nil {
			if err := conn.WriteJSON(message); err != nil {
				log.Printf("Error sending message to user %s: %v\n", member.MemberID.Hex(), err)
			}
		}
	}
}
