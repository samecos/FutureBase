package websocket

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHub(t *testing.T) {
	hub := NewHub()
	assert.NotNil(t, hub)
	assert.NotNil(t, hub.clients)
	assert.NotNil(t, hub.broadcast)
	assert.NotNil(t, hub.register)
	assert.NotNil(t, hub.unregister)
}

func TestHub_Run(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Give hub time to start
	time.Sleep(10 * time.Millisecond)

	// Test registration
	client := &Client{
		hub:  hub,
		send: make(chan []byte, 256),
		userID: uuid.New().String(),
	}

	hub.register <- client
	time.Sleep(10 * time.Millisecond)

	assert.Contains(t, hub.clients, client)

	// Test unregistration
	hub.unregister <- client
	time.Sleep(10 * time.Millisecond)

	assert.NotContains(t, hub.clients, client)
}

func TestHub_Broadcast(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	// Create test clients
	client1 := &Client{
		hub:     hub,
		send:    make(chan []byte, 256),
		userID:  uuid.New().String(),
		projectID: uuid.New().String(),
	}
	client2 := &Client{
		hub:     hub,
		send:    make(chan []byte, 256),
		userID:  uuid.New().String(),
		projectID: client1.projectID,
	}

	hub.register <- client1
	hub.register <- client2
	time.Sleep(10 * time.Millisecond)

	// Broadcast message
	message := []byte(`{"type":"test","data":"hello"}`)
	hub.broadcast <- Message{
		ProjectID: client1.projectID,
		Data:      message,
	}

	// Give time for message to be delivered
	time.Sleep(50 * time.Millisecond)

	// Both clients should receive the message
	select {
	case msg := <-client1.send:
		assert.Equal(t, message, msg)
	case <-time.After(100 * time.Millisecond):
		t.Error("Client 1 did not receive message")
	}

	select {
	case msg := <-client2.send:
		assert.Equal(t, message, msg)
	case <-time.After(100 * time.Millisecond):
		t.Error("Client 2 did not receive message")
	}
}

func TestServeWs(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ServeWs(hub, w, r)
	}))
	defer server.Close()

	// Convert http:// to ws://
	url := "ws" + server.URL[4:]

	// Test WebSocket connection
	ws, _, err := websocket.DefaultDialer.Dial(url, nil)
	require.NoError(t, err)
	defer ws.Close()

	// Give time for connection to be established
	time.Sleep(50 * time.Millisecond)

	// Verify client was registered
	assert.Equal(t, 1, len(hub.clients))
}

func TestClient_writePump(t *testing.T) {
	client := &Client{
		send: make(chan []byte, 256),
	}

	// Start write pump
	go client.writePump()

	// Send message
	message := []byte(`{"type":"update","data":"test"}`)
	client.send <- message

	// Give time for pump to process
	time.Sleep(50 * time.Millisecond)

	// Pump should continue running until channel is closed
	close(client.send)
	time.Sleep(50 * time.Millisecond)
}

func TestClient_readPump(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ServeWs(hub, w, r)
	}))
	defer server.Close()

	url := "ws" + server.URL[4:]
	ws, _, err := websocket.DefaultDialer.Dial(url, nil)
	require.NoError(t, err)
	defer ws.Close()

	time.Sleep(50 * time.Millisecond)

	// Send message to server
	msg := map[string]interface{}{
		"type": "sync",
		"data": "test data",
	}
	err = ws.WriteJSON(msg)
	require.NoError(t, err)

	// Give time for message to be processed
	time.Sleep(50 * time.Millisecond)

	// Client should still be connected
	assert.Equal(t, 1, len(hub.clients))
}

func TestHandleDocumentUpdate(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Create test client
	client := &Client{
		hub:       hub,
		send:      make(chan []byte, 256),
		userID:    uuid.New().String(),
		projectID: uuid.New().String(),
	}
	hub.register <- client
	time.Sleep(10 * time.Millisecond)

	// Create update request
	update := map[string]interface{}{
		"type":      "update",
		"projectId": client.projectID,
		"document": map[string]interface{}{
			"id":      uuid.New().String(),
			"content": "updated content",
		},
	}

	body, _ := json.Marshal(update)
	req := httptest.NewRequest("POST", "/api/v1/collaboration/update", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		HandleDocumentUpdate(hub, w, r)
	})

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Give time for broadcast
	time.Sleep(50 * time.Millisecond)

	// Client should receive update
	select {
	case <-client.send:
		// Success
	case <-time.After(100 * time.Millisecond):
		t.Error("Client did not receive update")
	}
}

func TestHandleGetActiveUsers(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Create test clients
	projectID := uuid.New().String()
	client1 := &Client{
		hub:       hub,
		send:      make(chan []byte, 256),
		userID:    uuid.New().String(),
		projectID: projectID,
	}
	client2 := &Client{
		hub:       hub,
		send:      make(chan []byte, 256),
		userID:    uuid.New().String(),
		projectID: projectID,
	}

	hub.register <- client1
	hub.register <- client2
	time.Sleep(10 * time.Millisecond)

	req := httptest.NewRequest("GET", "/api/v1/collaboration/active-users?projectId="+projectID, nil)
	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		HandleGetActiveUsers(hub, w, r)
	})

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	users, ok := response["users"].([]interface{})
	assert.True(t, ok)
	assert.Equal(t, 2, len(users))
}
