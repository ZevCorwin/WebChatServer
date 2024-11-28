package interfaces

// WebRTCNotifier định nghĩa giao diện để gửi thông báo
type WebRTCNotifier interface {
	NotifyUser(userID string, message interface{})
}
