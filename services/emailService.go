package services

import (
	"fmt"
	"github.com/sendgrid/sendgrid-go"
	"net/smtp"
	"os"

	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

// SendEmail là hàm điều phối chính.
// Nó sẽ đọc biến môi trường EMAIL_PROVIDER để quyết định dùng dịch vụ nào.
func SendEmail(to, subject, body string) error {
	provider := os.Getenv("EMAIL_PROVIDER")

	switch provider {
	case "sendgrid":
		fmt.Println("INFO: Using SendGrid provider to send email...")
		return sendEmailSendGrid(to, subject, body)
	case "smtp":
		fmt.Println("INFO: Using SMTP provider to send email...")
		return sendEmailSMTP(to, subject, body)
	default:
		// Mặc định cho môi trường dev nếu không cấu hình
		fmt.Println("INFO: EMAIL_PROVIDER not set, defaulting to SMTP...")
		return sendEmailSMTP(to, subject, body)
	}
}

// sendEmailSMTP chứa logic gửi email qua SMTP truyền thống.
func sendEmailSMTP(to, subject, body string) error {
	smtpServer := os.Getenv("SMTP_SERVER")
	smtpPort := os.Getenv("SMTP_PORT")
	smtpUser := os.Getenv("SMTP_USER")
	smtpPassword := os.Getenv("SMTP_PASSWORD")

	if smtpServer == "" || smtpPort == "" || smtpUser == "" || smtpPassword == "" {
		return fmt.Errorf("SMTP environment variables not fully configured")
	}

	auth := smtp.PlainAuth("", smtpUser, smtpPassword, smtpServer)
	addr := fmt.Sprintf("%s:%s", smtpServer, smtpPort)

	// Định dạng message theo chuẩn MIME để email hiển thị đúng HTML
	msg := []byte("To: " + to + "\r\n" +
		"From: " + smtpUser + "\r\n" + // Thêm dòng From
		"Subject: " + subject + "\r\n" +
		"MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n" +
		body)

	err := smtp.SendMail(addr, auth, smtpUser, []string{to}, msg)
	if err != nil {
		return fmt.Errorf("failed to send email via SMTP: %w", err)
	}

	fmt.Println("SUCCESS: Email sent successfully via SMTP to", to)
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
	htmlContent := body
	plainTextContent := "Please view this email in an HTML-compatible client."

	message := mail.NewSingleEmail(from, subject, toEmail, plainTextContent, htmlContent)
	client := sendgrid.NewSendClient(apiKey)
	response, err := client.Send(message)

	if err != nil {
		return fmt.Errorf("failed to send email via SendGrid: %w", err)
	}

	if response.StatusCode >= 200 && response.StatusCode < 300 {
		fmt.Println("SUCCESS: Email sent successfully via SendGrid to", to)
		return nil
	}

	return fmt.Errorf("SendGrid API error, status code: %d, body: %s", response.StatusCode, response.Body)
}
