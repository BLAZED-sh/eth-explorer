package api

import (
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

const (
	clientSendBuf = 256
	writeTimeout  = 10 * time.Second
	pingInterval  = 30 * time.Second
)

// Helloer produces the snapshot frame sent to each newly connected client.
type Helloer interface {
	Hello() []byte
}

type client struct {
	conn *websocket.Conn
	send chan []byte
}

type Hub struct {
	upgrader websocket.Upgrader

	mu      sync.RWMutex
	clients map[*client]struct{}

	register   chan *client
	unregister chan *client
	broadcast  chan []byte

	hello Helloer
}

func NewHub() *Hub {
	return &Hub{
		// Same-origin in prod (embedded SPA) and Vite proxy in dev, so any
		// origin is acceptable here.
		upgrader:   websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }, ReadBufferSize: 1024, WriteBufferSize: 16 * 1024},
		clients:    make(map[*client]struct{}),
		register:   make(chan *client, 16),
		unregister: make(chan *client, 16),
		broadcast:  make(chan []byte, 256),
	}
}

func (h *Hub) SetHelloer(s Helloer) { h.hello = s }

func (h *Hub) Run() {
	for {
		select {
		case c := <-h.register:
			h.mu.Lock()
			h.clients[c] = struct{}{}
			h.mu.Unlock()
			if h.hello != nil {
				select {
				case c.send <- h.hello.Hello():
				default:
				}
			}
		case c := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.send)
			}
			h.mu.Unlock()
		case msg := <-h.broadcast:
			h.mu.RLock()
			for c := range h.clients {
				select {
				case c.send <- msg:
				default:
					// client is slow; drop it to keep the broadcast moving
					go h.kick(c)
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) kick(c *client) {
	select {
	case h.unregister <- c:
	default:
	}
}

func (h *Hub) Publish(msg []byte) {
	select {
	case h.broadcast <- msg:
	default:
		log.Warn().Msg("broadcast queue full; dropping frame")
	}
}

func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	c := &client{conn: conn, send: make(chan []byte, clientSendBuf)}
	h.register <- c
	go h.writePump(c)
	go h.readPump(c)
}

func (h *Hub) writePump(c *client) {
	ticker := time.NewTicker(pingInterval)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, nil)
				return
			}
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (h *Hub) readPump(c *client) {
	defer func() {
		h.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(1024)
	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			return
		}
	}
}
