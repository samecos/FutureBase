package email

import (
	"bytes"
	"html/template"
	"time"

	"github.com/archplatform/notification-service/internal/models"
	"gopkg.in/gomail.v2"
)

// Config holds email configuration
type Config struct {
	SMTPHost     string
	SMTPPort     int
	Username     string
	Password     string
	FromAddress  string
	FromName     string
	Enabled      bool
}

// Sender handles email sending
type Sender struct {
	config Config
	dialer *gomail.Dialer
}

// NewSender creates a new email sender
func NewSender(config Config) *Sender {
	if !config.Enabled {
		return &Sender{config: config}
	}

	dialer := gomail.NewDialer(config.SMTPHost, config.SMTPPort, config.Username, config.Password)
	
	return &Sender{
		config: config,
		dialer: dialer,
	}
}

// Send sends an email notification
func (s *Sender) Send(to string, notification *models.Notification) error {
	if !s.config.Enabled {
		return nil
	}

	m := gomail.NewMessage()
	m.SetHeader("From", m.FormatAddress(s.config.FromAddress, s.config.FromName))
	m.SetHeader("To", to)
	m.SetHeader("Subject", notification.Title)
	
	// Generate HTML body
	htmlBody := s.generateHTMLBody(notification)
	m.SetBody("text/html", htmlBody)
	
	// Add plain text alternative
	m.AddAlternative("text/plain", notification.Content)

	return s.dialer.DialAndSend(m)
}

// SendBatch sends batch emails
func (s *Sender) SendBatch(recipients []string, notification *models.Notification) []error {
	if !s.config.Enabled {
		return nil
	}

	var errors []error
	for _, to := range recipients {
		if err := s.Send(to, notification); err != nil {
			errors = append(errors, err)
		}
		time.Sleep(100 * time.Millisecond) // Rate limiting
	}
	return errors
}

// generateHTMLBody generates HTML email body
func (s *Sender) generateHTMLBody(n *models.Notification) string {
	tmpl := `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #4a90d9; color: white; padding: 20px; border-radius: 5px 5px 0 0; }
        .content { background: #f9f9f9; padding: 20px; border: 1px solid #ddd; }
        .footer { text-align: center; padding: 20px; font-size: 12px; color: #666; }
        .button { display: inline-block; padding: 10px 20px; background: #4a90d9; color: white; text-decoration: none; border-radius: 5px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h2>{{.Title}}</h2>
        </div>
        <div class="content">
            <p>{{.Content}}</p>
            {{if .ActionURL}}
            <p><a href="{{.ActionURL}}" class="button">View Details</a></p>
            {{end}}
        </div>
        <div class="footer">
            <p>This is an automated notification from ArchPlatform.</p>
        </div>
    </div>
</body>
</html>`

	t := template.Must(template.New("email").Parse(tmpl))
	var buf bytes.Buffer
	t.Execute(&buf, n)
	return buf.String()
}

// IsEnabled returns whether email is enabled
func (s *Sender) IsEnabled() bool {
	return s.config.Enabled
}
