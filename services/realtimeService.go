package services

import "chat-app-backend/interfaces"

type RealtimeService struct {
	WebRTCNotifier interfaces.WebRTCNotifier
}

// Khởi tạo RealtimeService
func NewRealtimeService(notifier interfaces.WebRTCNotifier) *RealtimeService {
	return &RealtimeService{WebRTCNotifier: notifier}
}

// Gửi thông báo thời gian thực
func (rs *RealtimeService) SendNotification(userID string, message interface{}) {
	rs.WebRTCNotifier.NotifyUser(userID, message)
}
