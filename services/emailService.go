package services

import (
	"crypto/tls"
	"fmt"
	"gopkg.in/gomail.v2"
	"os"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

// SendEmail là hàm điều phối chính, thay thế cho toàn bộ struct cũ.
// Bất kỳ service nào (như otpService) cũng sẽ gọi hàm này.
func SendEmail(to, subject, body string) error {
	provider := os.Getenv("EMAIL_PROVIDER")

	switch provider {
	case "sendgrid":
		fmt.Println("INFO: Using SendGrid provider to send email...")
		return sendEmailSendGrid(to, subject, body)
	case "smtp":
		fmt.Println("INFO: Using SMTP provider (gomail) to send email...")
		return sendEmailSMTP(to, subject, body)
	default:
		// Mặc định cho môi trường dev nếu không cấu hình
		fmt.Println("INFO: EMAIL_PROVIDER not set, defaulting to SMTP (gomail)...")
		return sendEmailSMTP(to, subject, body)
	}
}

// sendEmailSMTP sử dụng lại logic gomail gốc của bạn.
func sendEmailSMTP(to, subject, htmlBody string) error {
	// Đọc cấu hình SMTP từ biến môi trường
	host := os.Getenv("SMTP_HOST")
	user := os.Getenv("SMTP_USER")
	pass := os.Getenv("SMTP_PASS")
	port := 587 // Mặc định
	if v := os.Getenv("SMTP_PORT"); v != "" {
		fmt.Sscanf(v, "%d", &port)
	}

	if host == "" || user == "" {
		return fmt.Errorf("SMTP environment variables not fully configured")
	}

	m := gomail.NewMessage()
	m.SetHeader("From", user) // 'From' thường giống với 'user'
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", htmlBody)

	d := gomail.NewDialer(host, port, user, pass)
	d.TLSConfig = &tls.Config{
		ServerName: host, // Giữ lại cấu hình TLS quan trọng của bạn
	}

	fmt.Printf("[EmailService-SMTP] Sending mail to %s via %s:%d\n", to, host, port)
	if err := d.DialAndSend(m); err != nil {
		fmt.Printf("[EmailService-SMTP] Send error: %v\n", err)
		return err
	}
	fmt.Println("[EmailService-SMTP] Email sent successfully!")
	return nil
}

// sendEmailSendGrid chứa logic gửi email qua SendGrid API.
func sendEmailSendGrid(to, subject, body string) error {
	apiKey := os.Getenv("SENDGRID_API_KEY")
	senderEmail := os.Getenv("SENDER_EMAIL")
	senderName := os.Getenv("SENDER_NAME")

	if apiKey == "" || senderEmail == "" {
		return fmt.Errorf("SENDGRID_API_KEY and SENDER_EMAIL must be set")
	}

	from := mail.NewEmail(senderName, senderEmail)
	toEmail := mail.NewEmail("", to)
	message := mail.NewSingleEmail(from, subject, toEmail, body, body)
	client := sendgrid.NewSendClient(apiKey)
	response, err := client.Send(message)

	if err != nil {
		return fmt.Errorf("failed to send email via SendGrid: %w", err)
	}

	if response.StatusCode >= 200 && response.StatusCode < 300 {
		fmt.Println("[EmailService-SendGrid] Email sent successfully to", to)
		return nil
	}

	return fmt.Errorf("SendGrid API error, status code: %d, body: %s", response.StatusCode, response.Body)
}
