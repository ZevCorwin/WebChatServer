// Đã sửa
package controllers

import (
	"chat-app-backend/services"
	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"log"
	"net/http"
	"sync"
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
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Gửi thông báo đến một user cụ thể
func (wc *WebRTCController) NotifyUser(userID string, message interface{}) {
	wc.mu.Lock()
	conn, ok := wc.Connections[userID]
	wc.mu.Unlock()

	if ok {
		err := conn.WriteJSON(message)
		if err != nil {
			log.Printf("Error sending message to user %s: %v\n", userID, err)
		}
	}
}

func (wc *WebRTCController) BroadcastMessage(channelID primitive.ObjectID, message interface{}) {
	log.Printf("[BroadcastMessage] Vị trí 1")
	channel, err := wc.ChannelService.GetChannel(channelID)
	log.Printf("[BroadcastMessage] Lấy kênh: %s", channel)
	if err != nil {
		log.Printf("Error getting channel: %v\n", err)
		return
	}

	wc.mu.Lock()
	log.Printf("[BroadcastMessage] Vị trí 2")
	defer wc.mu.Unlock()
	log.Printf("[BroadcastMessage] Vị trí 3, Connections: %v", wc.Connections)
	for _, member := range channel.Members {
		log.Printf("[BroadcastMessage] Vị trí 4, memberID: %s, Connections keys: %v", member.MemberID.Hex(), wc.Connections)
		if conn, ok := wc.Connections[member.MemberID.Hex()]; ok {
			log.Printf("[BroadcastMessage] Vị trí 5")
			err := conn.WriteJSON(message)
			log.Printf("[BroadcastMessage]Broadcasting message to user %s: %+v\n", member.MemberID.Hex(), err)
			if err != nil {
				log.Printf("Error sending message to user %s: %v\n", member.MemberID.Hex(), err)
			}
		}
	}
}
