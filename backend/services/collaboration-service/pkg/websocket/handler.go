package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/archplatform/collaboration-service/pkg/yjs"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// In production, implement proper origin checking
		return true
	},
}

// MessageType represents WebSocket message types
type MessageType string

const (
	MessageTypeAuth             MessageType = "auth"
	MessageTypeAuthSuccess      MessageType = "auth_success"
	MessageTypeAuthError        MessageType = "auth_error"
	MessageTypeJoin             MessageType = "join"
	MessageTypeLeave            MessageType = "leave"
	MessageTypeOperation        MessageType = "operation"
	MessageTypeOperationAck     MessageType = "operation_ack"
	MessageTypeCursor           MessageType = "cursor"
	MessageTypeSelection        MessageType = "selection"
	MessageTypeAwareness        MessageType = "awareness"
	MessageTypeSync             MessageType = "sync"
	MessageTypeSyncResponse     MessageType = "sync_response"
	MessageTypeError            MessageType = "error"
	MessageTypePing             MessageType = "ping"
	MessageTypePong             MessageType = "pong"
	MessageTypeUserJoined       MessageType = "user_joined"
	MessageTypeUserLeft         MessageType = "user_left"
	MessageTypeSessionClosed    MessageType = "session_closed"
)

// Message represents a WebSocket message
type Message struct {
	Type      MessageType     `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	Timestamp int64           `json:"timestamp"`
	ID        string          `json:"id"`
}

// Client represents a WebSocket client
type Client struct {
	ID         string
	SessionID  string
	UserID     string
	UserName   string
	UserAvatar string
	Conn       *websocket.Conn
	Send       chan []byte
	Server     *Server
	Permission string
	
	// Client info
	ClientType    string
	ClientVersion string
	ClientPlatform string
	
	// State
	JoinedAt     time.Time
	LastActivity time.Time
	mu           sync.RWMutex
}

// Server manages WebSocket connections
type Server struct {
	clients    map[string]*Client // clientID -> Client
	sessions   map[string]map[string]*Client // sessionID -> clientID -> Client
	register   chan *Client
	unregister chan *Client
	broadcast  chan *BroadcastMessage
	mu         sync.RWMutex
	logger     *zap.Logger
	yjsManager *yjs.DocumentManager
}

// BroadcastMessage represents a message to broadcast
type BroadcastMessage struct {
	SessionID   string
	Message     []byte
	ExcludeClient string
}

// NewServer creates a new WebSocket server
func NewServer(logger *zap.Logger, yjsManager *yjs.DocumentManager) *Server {
	return &Server{
		clients:    make(map[string]*Client),
		sessions:   make(map[string]map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *BroadcastMessage),
		logger:     logger,
		yjsManager: yjsManager,
	}
}

// Run starts the WebSocket server
func (s *Server) Run() {
	for {
		select {
		case client := <-s.register:
			s.registerClient(client)
			
		case client := <-s.unregister:
			s.unregisterClient(client)
			
		case msg := <-s.broadcast:
			s.broadcastToSession(msg)
		}
	}
}

// registerClient registers a new client
func (s *Server) registerClient(client *Client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.clients[client.ID] = client
	
	// Add to session
	if client.SessionID != "" {
		if s.sessions[client.SessionID] == nil {
			s.sessions[client.SessionID] = make(map[string]*Client)
		}
		s.sessions[client.SessionID][client.ID] = client
	}
	
	s.logger.Info("Client registered",
		zap.String("client_id", client.ID),
		zap.String("session_id", client.SessionID),
		zap.String("user_id", client.UserID),
	)
}

// unregisterClient unregisters a client
func (s *Server) unregisterClient(client *Client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if _, ok := s.clients[client.ID]; ok {
		delete(s.clients, client.ID)
		close(client.Send)
		
		// Remove from session
		if client.SessionID != "" {
			if session, ok := s.sessions[client.SessionID]; ok {
				delete(session, client.ID)
				if len(session) == 0 {
					delete(s.sessions, client.SessionID)
				}
			}
		}
		
		s.logger.Info("Client unregistered",
			zap.String("client_id", client.ID),
			zap.String("session_id", client.SessionID),
		)
	}
}

// broadcastToSession broadcasts a message to all clients in a session
func (s *Server) broadcastToSession(msg *BroadcastMessage) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	session, ok := s.sessions[msg.SessionID]
	if !ok {
		return
	}
	
	for clientID, client := range session {
		if clientID == msg.ExcludeClient {
			continue
		}
		
		select {
		case client.Send <- msg.Message:
		default:
			// Client send buffer is full, unregister
			go func(c *Client) {
				s.unregister <- c
			}(client)
		}
	}
}

// HandleWebSocket handles WebSocket connections
func (s *Server) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("Failed to upgrade connection", zap.Error(err))
		return
	}
	
	client := &Client{
		ID:           uuid.New().String(),
		Conn:         conn,
		Send:         make(chan []byte, 256),
		Server:       s,
		JoinedAt:     time.Now(),
		LastActivity: time.Now(),
	}
	
	s.register <- client
	
	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()
}

// readPump pumps messages from the WebSocket connection to the server
func (c *Client) readPump() {
	defer func() {
		c.Server.unregister <- c
		c.Conn.Close()
	}()
	
	c.Conn.SetReadLimit(10 * 1024 * 1024) // 10MB max message size
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		c.updateActivity()
		return nil
	})
	
	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.Server.logger.Error("WebSocket error", zap.Error(err))
			}
			break
		}
		
		c.updateActivity()
		c.handleMessage(message)
	}
}

// writePump pumps messages from the server to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()
	
	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			
			c.Conn.WriteMessage(websocket.TextMessage, message)
			
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage handles incoming messages
func (c *Client) handleMessage(data []byte) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		c.sendError("invalid_message", "Failed to parse message")
		return
	}
	
	switch msg.Type {
	case MessageTypeAuth:
		c.handleAuth(msg.Payload)
	case MessageTypeJoin:
		c.handleJoin(msg.Payload)
	case MessageTypeLeave:
		c.handleLeave()
	case MessageTypeOperation:
		c.handleOperation(msg.Payload)
	case MessageTypeCursor:
		c.handleCursor(msg.Payload)
	case MessageTypeSelection:
		c.handleSelection(msg.Payload)
	case MessageTypeSync:
		c.handleSync(msg.Payload)
	case MessageTypePing:
		c.handlePing()
	default:
		c.sendError("unknown_type", fmt.Sprintf("Unknown message type: %s", msg.Type))
	}
}

// handleAuth handles authentication message
func (c *Client) handleAuth(payload json.RawMessage) {
	var auth struct {
		Token string `json:"token"`
	}
	
	if err := json.Unmarshal(payload, &auth); err != nil {
		c.sendError("auth_failed", "Invalid auth payload")
		return
	}
	
	// TODO: Validate token and extract user info
	// For now, just accept the auth
	c.sendMessage(MessageTypeAuthSuccess, map[string]interface{}{
		"client_id": c.ID,
	})
}

// handleJoin handles join session message
func (c *Client) handleJoin(payload json.RawMessage) {
	var join struct {
		SessionID      string `json:"session_id"`
		UserID         string `json:"user_id"`
		UserName       string `json:"user_name"`
		UserAvatar     string `json:"user_avatar"`
		ClientType     string `json:"client_type"`
		ClientVersion  string `json:"client_version"`
		ClientPlatform string `json:"client_platform"`
	}
	
	if err := json.Unmarshal(payload, &join); err != nil {
		c.sendError("join_failed", "Invalid join payload")
		return
	}
	
	c.mu.Lock()
	c.SessionID = join.SessionID
	c.UserID = join.UserID
	c.UserName = join.UserName
	c.UserAvatar = join.UserAvatar
	c.ClientType = join.ClientType
	c.ClientVersion = join.ClientVersion
	c.ClientPlatform = join.ClientPlatform
	c.mu.Unlock()
	
	// Register with session
	c.Server.register <- c
	
	// Broadcast user joined
	c.broadcastToSession(MessageTypeUserJoined, map[string]interface{}{
		"user_id":    c.UserID,
		"user_name":  c.UserName,
		"user_avatar": c.UserAvatar,
		"joined_at":  c.JoinedAt,
	})
}

// handleLeave handles leave session message
func (c *Client) handleLeave() {
	c.broadcastToSession(MessageTypeUserLeft, map[string]interface{}{
		"user_id":   c.UserID,
		"user_name": c.UserName,
		"left_at":   time.Now(),
	})
	
	c.Server.unregister <- c
}

// handleOperation handles operation message
func (c *Client) handleOperation(payload json.RawMessage) {
	var op struct {
		OperationID string          `json:"operation_id"`
		Type        string          `json:"type"`
		TargetID    string          `json:"target_id"`
		Data        json.RawMessage `json:"data"`
		ClientClock int64           `json:"client_clock"`
	}
	
	if err := json.Unmarshal(payload, &op); err != nil {
		c.sendError("operation_failed", "Invalid operation payload")
		return
	}
	
	// Broadcast to other clients
	c.broadcastToSession(MessageTypeOperation, map[string]interface{}{
		"operation_id": op.OperationID,
		"type":         op.Type,
		"target_id":    op.TargetID,
		"data":         op.Data,
		"user_id":      c.UserID,
		"client_clock": op.ClientClock,
	})
	
	// Send ack
	c.sendMessage(MessageTypeOperationAck, map[string]interface{}{
		"operation_id": op.OperationID,
		"status":       "success",
	})
}

// handleCursor handles cursor update message
func (c *Client) handleCursor(payload json.RawMessage) {
	c.broadcastToSession(MessageTypeCursor, map[string]interface{}{
		"user_id": c.UserID,
		"cursor":  payload,
	})
}

// handleSelection handles selection update message
func (c *Client) handleSelection(payload json.RawMessage) {
	c.broadcastToSession(MessageTypeSelection, map[string]interface{}{
		"user_id":    c.UserID,
		"selection":  payload,
	})
}

// handleSync handles sync message
func (c *Client) handleSync(payload json.RawMessage) {
	var sync struct {
		StateVector []byte `json:"state_vector"`
	}
	
	if err := json.Unmarshal(payload, &sync); err != nil {
		c.sendError("sync_failed", "Invalid sync payload")
		return
	}
	
	// Get diff from Yjs manager
	diff, err := c.Server.yjsManager.ComputeDiff(c.SessionID, sync.StateVector)
	if err != nil {
		c.sendError("sync_failed", err.Error())
		return
	}
	
	c.sendMessage(MessageTypeSyncResponse, map[string]interface{}{
		"update": diff,
	})
}

// handlePing handles ping message
func (c *Client) handlePing() {
	c.sendMessage(MessageTypePong, map[string]interface{}{
		"timestamp": time.Now().Unix(),
	})
}

// broadcastToSession broadcasts a message to the session
func (c *Client) broadcastToSession(msgType MessageType, payload interface{}) {
	if c.SessionID == "" {
		return
	}
	
	data, err := json.Marshal(Message{
		Type:      msgType,
		Payload:   mustJSON(payload),
		Timestamp: time.Now().Unix(),
		ID:        uuid.New().String(),
	})
	if err != nil {
		return
	}
	
	c.Server.broadcast <- &BroadcastMessage{
		SessionID:     c.SessionID,
		Message:       data,
		ExcludeClient: c.ID,
	}
}

// sendMessage sends a message to the client
func (c *Client) sendMessage(msgType MessageType, payload interface{}) {
	data, err := json.Marshal(Message{
		Type:      msgType,
		Payload:   mustJSON(payload),
		Timestamp: time.Now().Unix(),
		ID:        uuid.New().String(),
	})
	if err != nil {
		return
	}
	
	select {
	case c.Send <- data:
	default:
		// Send buffer full
	}
}

// sendError sends an error message to the client
func (c *Client) sendError(code string, message string) {
	c.sendMessage(MessageTypeError, map[string]interface{}{
		"code":    code,
		"message": message,
	})
}

// updateActivity updates the last activity timestamp
func (c *Client) updateActivity() {
	c.mu.Lock()
	c.LastActivity = time.Now()
	c.mu.Unlock()
}

// mustJSON marshals data to JSON, panicking on error
func mustJSON(v interface{}) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage("{}")
	}
	return data
}

// GetSessionStats returns statistics for a session
func (s *Server) GetSessionStats(sessionID string) (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	session, ok := s.sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}
	
	return map[string]interface{}{
		"client_count": len(session),
	}, nil
}

// GetActiveSessionCount returns the number of active sessions
func (s *Server) GetActiveSessionCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	return len(s.sessions)
}

// GetActiveClientCount returns the number of active clients
func (s *Server) GetActiveClientCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	return len(s.clients)
}
