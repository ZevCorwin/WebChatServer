package controllers

import (
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"sync"
)

type WebRTCController struct {
	Connections map[string]*websocket.Conn
	mu          sync.Mutex
}

// Khởi tạo controller
func NewWebRTCController() *WebRTCController {
	return &WebRTCController{
		Connections: make(map[string]*websocket.Conn),
	}
}

var upgraderWebRTC = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Xử lý kết nối WebSocket
func (wc *WebRTCController) HandleSignaling(ctx *gin.Context) {
	userID := ctx.Query("userID")
	if userID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "userID is required"})
		return
	}

	conn, err := upgraderWebRTC.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		log.Println("WebSocket Upgrade error:", err)
		return
	}
	defer conn.Close()

	// Thêm kết nối vào danh sách
	wc.mu.Lock()
	wc.Connections[userID] = conn
	wc.mu.Unlock()

	log.Printf("User %s connected\n", userID)

	for {
		var msg map[string]interface{}
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("Error reading from user %s: %v\n", userID, err)
			break
		}

		// Xử lý tin nhắn nhận được (nếu cần)
		log.Printf("Received message from user %s: %v\n", userID, msg)
	}
	wc.mu.Lock()
	delete(wc.Connections, userID)
	wc.mu.Unlock()
	log.Printf("User %s disconnected\n", userID)
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
