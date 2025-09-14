package services

import (
	"crypto/tls"
	"fmt"
	"gopkg.in/gomail.v2"
	"os"
)

type EmailService struct {
	host string
	port int
	user string
	pass string
	from string
	app  string
}

func NewEmailService() *EmailService {
	// port lấy từ .env string -> int
	port := 587
	if v := os.Getenv("SMTP_PORT"); v != "" {
		fmt.Sscanf(v, "%d", &port)
	}
	return &EmailService{
		host: os.Getenv("SMTP_HOST"),
		port: port,
		user: os.Getenv("SMTP_USER"),
		pass: os.Getenv("SMTP_PASS"),
		from: os.Getenv("SMTP_USER"),
		app:  os.Getenv("APP_NAME"),
	}
}

func (es *EmailService) Send(to, subject, htmlBody string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", es.from)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", htmlBody)

	d := gomail.NewDialer(es.host, es.port, es.user, es.pass)
	d.TLSConfig = &tls.Config{
		ServerName: es.host, // 👈 BẮT BUỘC để TLS handshake đúng
	}

	fmt.Printf("[EmailService] Sending mail to %s via %s:%d as %s\n", to, es.host, es.port, es.user)
	if err := d.DialAndSend(m); err != nil {
		fmt.Printf("[EmailService] Send error: %v\n", err)
		return err
	}
	fmt.Println("[EmailService] Email sent successfully!")
	return nil
}
