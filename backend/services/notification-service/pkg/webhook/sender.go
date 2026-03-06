package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/archplatform/notification-service/internal/models"
)

// Config holds webhook configuration
type Config struct {
	Enabled         bool
	MaxRetries      int
	TimeoutSeconds  int
	RetryDelaySeconds int
}

// Sender handles webhook sending
type Sender struct {
	config     Config
	httpClient *http.Client
}

// NewSender creates a new webhook sender
func NewSender(config Config) *Sender {
	return &Sender{
		config: config,
		httpClient: &http.Client{
			Timeout: time.Duration(config.TimeoutSeconds) * time.Second,
		},
	}
}

// Send sends a webhook notification
func (s *Sender) Send(url string, notification *models.Notification) error {
	if !s.config.Enabled {
		return nil
	}

	payload := map[string]any{
		"id":         notification.ID,
		"type":       notification.Type,
		"title":      notification.Title,
		"content":    notification.Content,
		"data":       notification.Data,
		"priority":   notification.Priority,
		"timestamp":  notification.CreatedAt,
		"action_url": notification.ActionURL,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Retry logic
	var lastErr error
	for attempt := 1; attempt <= s.config.MaxRetries; attempt++ {
		if err := s.sendRequest(url, jsonData); err != nil {
			lastErr = err
			if attempt < s.config.MaxRetries {
				time.Sleep(time.Duration(s.config.RetryDelaySeconds) * time.Second)
				continue
			}
		}
		return nil
	}

	return fmt.Errorf("webhook failed after %d attempts: %w", s.config.MaxRetries, lastErr)
}

func (s *Sender) sendRequest(url string, data []byte) error {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "ArchPlatform-Notification-Service/1.0")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// IsEnabled returns whether webhook is enabled
func (s *Sender) IsEnabled() bool {
	return s.config.Enabled
}
