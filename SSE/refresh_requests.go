package SSE

import (
	"fmt"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// SSEBroadcaster manages SSE connections and broadcasts messages to all clients.
type SSEBroadcaster struct {
	clients map[chan string]bool
	mu      sync.Mutex
}

// NewSSEBroadcaster creates a new SSEBroadcaster.
func NewSSEBroadcaster() *SSEBroadcaster {
	return &SSEBroadcaster{
		clients: make(map[chan string]bool),
	}
}

// Register adds a new client to the broadcaster.
func (b *SSEBroadcaster) Register(client chan string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.clients[client] = true
}

// Unregister removes a client from the broadcaster.
func (b *SSEBroadcaster) Unregister(client chan string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.clients, client)
	close(client)
}

// Broadcast sends a message to all registered clients.
func (b *SSEBroadcaster) Broadcast(message string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for client := range b.clients {
		select {
		case client <- message:
		case <-time.After(1 * time.Second):
			// If the client is not responding, unregister them.
			delete(b.clients, client)
			close(client)
		}
	}
}

var Broadcaster = NewSSEBroadcaster()

func RequestSSE(c *gin.Context) {
	// Set headers for SSE

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	// Create a new channel for this client
	clientChan := make(chan string)

	// Register the client channel
	Broadcaster.Register(clientChan)
	defer Broadcaster.Unregister(clientChan)
	fmt.Fprintf(c.Writer, "data: %s\n\n", "connected")
	c.Writer.Flush()
	// Listen to the client channel and send messages to the client
	for {
		select {
		case message := <-clientChan:
			// Send the message to the client
			fmt.Fprintf(c.Writer, "data: %s\n\n", message)
			c.Writer.Flush()
		case <-c.Writer.CloseNotify():
			// Client disconnected
			return
		}
	}
}
