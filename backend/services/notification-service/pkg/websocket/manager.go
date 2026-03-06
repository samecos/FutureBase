package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/archplatform/notification-service/internal/models"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Manager manages WebSocket connections
type Manager struct {
	upgrader  websocket.Upgrader
	clients   map[uuid.UUID]*Client
	broadcast chan *models.Notification
	mu        sync.RWMutex
}

// Client represents a WebSocket client
type Client struct {
	UserID uuid.UUID
	Conn   *websocket.Conn
	Send   chan []byte
}

// NewManager creates a new WebSocket manager
func NewManager() *Manager {
	return &Manager{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins in development
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		clients:   make(map[uuid.UUID]*Client),
		broadcast: make(chan *models.Notification, 100),
	}
}

// Start starts the WebSocket manager
func (m *Manager) Start() {
	go m.handleBroadcast()
}

// HandleConnection handles a new WebSocket connection
func (m *Manager) HandleConnection(w http.ResponseWriter, r *http.Request, userID uuid.UUID) {
	conn, err := m.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	client := &Client{
		UserID: userID,
		Conn:   conn,
		Send:   make(chan []byte, 256),
	}

	m.mu.Lock()
	// Close existing connection for this user
	if existing, ok := m.clients[userID]; ok {
		close(existing.Send)
		existing.Conn.Close()
	}
	m.clients[userID] = client
	m.mu.Unlock()

	// Start goroutines for reading and writing
	go m.readPump(client)
	go m.writePump(client)

	// Send welcome message
	welcome := models.WebSocketMessage{
		Type:    "connected",
		Payload: map[string]string{"status": "connected", "user_id": userID.String()},
	}
	if data, err := json.Marshal(welcome); err == nil {
		client.Send <- data
	}
}

// SendNotification sends a notification to a specific user
func (m *Manager) SendNotification(userID uuid.UUID, notification *models.Notification) {
	m.broadcast <- notification
}

// Broadcast sends a notification to all connected users
func (m *Manager) Broadcast(notification *models.Notification) {
	m.broadcast <- notification
}

func (m *Manager) handleBroadcast() {
	for notification := range m.broadcast {
		message := models.WebSocketMessage{
			Type:    "notification",
			Payload: notification,
		}

		data, err := json.Marshal(message)
		if err != nil {
			continue
		}

		m.mu.RLock()
		client, ok := m.clients[notification.UserID]
		m.mu.RUnlock()

		if ok {
			select {
			case client.Send <- data:
			default:
				// Client buffer is full, close connection
				m.removeClient(client)
			}
		}
	}
}

func (m *Manager) readPump(client *Client) {
	defer func() {
		m.removeClient(client)
	}()

	client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	client.Conn.SetPongHandler(func(string) error {
		client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := client.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}
	}
}

func (m *Manager) writePump(client *Client) {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		client.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.Send:
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			client.Conn.WriteMessage(websocket.TextMessage, message)

		case <-ticker.C:
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (m *Manager) removeClient(client *Client) {
	m.mu.Lock()
	if existing, ok := m.clients[client.UserID]; ok && existing == client {
		delete(m.clients, client.UserID)
	}
	m.mu.Unlock()

	close(client.Send)
	client.Conn.Close()
}

// GetConnectedCount returns the number of connected clients
func (m *Manager) GetConnectedCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.clients)
}
