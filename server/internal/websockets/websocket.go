package websockets

import (
	"context"
	"log/slog"
	"time"
	"waugzee/config"
	"waugzee/internal/database"
	"waugzee/internal/events"
	"waugzee/internal/logger"
	"waugzee/internal/repositories"
	"waugzee/internal/services"
	"waugzee/internal/types"

	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
)

// Type alias for events.Message to avoid conflicts
type Message = events.Message

const (
	MESSAGE_TYPE_PING = "ping"
	MESSAGE_TYPE_PONG = "pong"
	API_REQUEST       = "api_request"
	API_RESPONSE      = "api_response"
	API_PROGRESS      = "api_progress"
	API_COMPLETE      = "api_complete"
	API_ERROR         = "api_error"
	USER_JOIN         = "user_join"
	BROADCAST         = "broadcast"
	SEND              = "send"
	MESSAGE           = "message"
)

const (
	PING_INTERVAL     = 30 * time.Second
	PONG_TIMEOUT      = 60 * time.Second
	WRITE_TIMEOUT     = 10 * time.Second
	MAX_MESSAGE_SIZE  = 1024 * 1024 // 1 MB
	SEND_CHANNEL_SIZE = 64
)

// ZitadelService interface to avoid circular imports
type ZitadelService interface {
	ValidateTokenWithFallback(ctx context.Context, token string) (*types.TokenInfo, string, error)
}

type Client struct {
	ID         string
	UserID     uuid.UUID
	Connection *websocket.Conn
	Manager    *Manager
	Status     int
	send       chan Message
}

type Manager struct {
	hub            *Hub
	db             database.DB
	config         config.Config
	log            logger.Logger
	eventBus       *events.EventBus
	zitadelService ZitadelService
	userRepo       repositories.UserRepository
}

func New(
	db database.DB,
	eventBus *events.EventBus,
	config config.Config,
	services services.Service,
	repos repositories.Repository,
) (*Manager, error) {
	log := logger.New("websockets")

	manager := &Manager{
		hub: &Hub{
			broadcast:  make(chan Message),
			register:   make(chan *Client),
			unregister: make(chan *Client),
			clients:    make(map[string]*Client),
		},
		db:             db,
		config:         config,
		log:            log,
		eventBus:       eventBus,
		zitadelService: services.Zitadel,
		userRepo:       repos.User,
	}

	log.Function("New").Info("Starting websocket hub")
	go manager.hub.run(manager)

	go manager.subscribeToEventBus()

	return manager, nil
}

func (m *Manager) HandleWebSocket(c *websocket.Conn) {
	log := m.log.Function("HandleWebSocket")
	clientID := uuid.New().String()

	client := &Client{
		ID:         clientID,
		UserID:     uuid.Nil,
		Connection: c,
		Manager:    m,
		Status:     STATUS_UNAUTHENTICATED,
		send:       make(chan Message, SEND_CHANNEL_SIZE),
	}

	if err := client.sendAuthRequest(); err != nil {
		if err := c.Close(); err != nil {
			log.Er("failed to close connection", err)
		}
		return
	}
	m.hub.register <- client
	defer func() {
		log.Info("Client disconnected in the defer", "clientID", clientID)
		m.hub.unregister <- client
		if err := c.Close(); err != nil {
			log.Er("failed to close connection", err)
		}
	}()

	// Start auth timeout
	client.startAuthTimeout()

	go client.readPump()
	client.writePump()
}

func (m *Manager) BroadcastMessage(message Message) {
	log := m.log.Function("BroadcastMessage")
	log.Info("Broadcasting message from ", "messageID", message.ID)

	select {
	case m.hub.broadcast <- message:
		log.Info("Message sent to broadcast channel", "messageID", message.ID)
	default:
		log.Warn("Broadcast channel is full, dropping message", "messageID", message.ID)
	}
}

func (c *Client) readPump() {
	log := c.Manager.log.Function("readPump")
	defer func() {
		c.Manager.hub.unregister <- c
		_ = c.Connection.Close()
	}()

	c.Connection.SetReadLimit(MAX_MESSAGE_SIZE)
	if err := c.Connection.SetReadDeadline(time.Now().Add(PONG_TIMEOUT)); err != nil {
		log.Er("failed to set read deadline", err, "clientID", c.ID)
	}
	c.Connection.SetPongHandler(func(string) error {
		if err := c.Connection.SetReadDeadline(time.Now().Add(PONG_TIMEOUT)); err != nil {
			log.Er("failed to set read deadline in pong handler", err, "clientID", c.ID)
		}
		return nil
	})

	for {
		var message Message
		err := c.Connection.ReadJSON(&message)
		log.Info("Read message", "clientID", c.ID, "message", message)
		if err != nil {
			log.Er("failed to read message", err)
			if websocket.IsUnexpectedCloseError(
				err,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure,
			) {
				log.Er("Unexpected close error", err, "clientID", c.ID)
			}
			break
		}

		message.ID = uuid.New().String()
		message.Timestamp = time.Now()

		c.routeMessage(message)
	}
}

func (c *Client) routeMessage(message Message) {
	log := c.Manager.log.Function("routeMessage")

	if message.Event == AUTH_RESPONSE {
		c.handleAuthResponse(message)
		return
	}

	if c.Status == STATUS_UNAUTHENTICATED {
		c.handleUnauthenticatedMessage(message)
		return
	}

	switch message.Event {
	case API_RESPONSE:
		log.Info(
			"Received API response",
			"messageID",
			message.ID,
			"clientID",
			c.ID,
			"message",
			message,
		)
	default:
		log.Warn("Unknown message event", "event", message.Event)
	}

	switch message.Service {
	case "system":
		slog.Info("System message", "messageID", message.ID, "clientID", c.ID, "message", message)
	case "user":
		slog.Info("User message", "messageID", message.ID, "clientID", c.ID, "message", message)
	}
}

func (c *Client) writePump() {
	log := c.Manager.log.Function("writePump")

	ticker := time.NewTicker(PING_INTERVAL)
	defer func() {
		ticker.Stop()
		_ = c.Connection.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			if err := c.Connection.SetWriteDeadline(time.Now().Add(WRITE_TIMEOUT)); err != nil {
				log.Er("failed to set write deadline", err, "clientID", c.ID)
			}
			if !ok {
				log.Info("Channel closed", "clientID", c.ID)
				_ = c.Connection.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Connection.WriteJSON(message); err != nil {
				log.Er("WebSocket write error", err, "clientID", c.ID, "message", message)
				return
			}

		case <-ticker.C:
			log.Debug("Sending ping", "clientID", c.ID)
			if err := c.Connection.SetWriteDeadline(time.Now().Add(WRITE_TIMEOUT)); err != nil {
				log.Er("failed to set write deadline for ping", err, "clientID", c.ID)
			}
			if err := c.Connection.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (m *Manager) subscribeToEventBus() {
	log := m.log.Function("subscribeToBroadcastEvents")
	log.Info("Starting broadcast events subscription")

	if err := m.eventBus.Subscribe(events.WEBSOCKET, func(event events.ChannelEvent) error {
		log.Info("Received broadcast event", "event", event.Event)

		switch event.Event {
		case "broadcast":
			m.sendToAuthenticatedClients(event.Message)
		case "user":
			m.sendToAuthenticatedClients(event.Message)
		}
		return nil
	}); err != nil {
		log.Er("Failed to subscribe to broadcast events", err)
	}
}

func (m *Manager) sendToAuthenticatedClients(message Message) {
	log := m.log.Function("sendToAuthenticatedClients")

	sent := 0
	for _, client := range m.hub.clients {
		if client.Status == STATUS_AUTHENTICATED {
			select {
			case client.send <- message:
				sent++
			default:
				log.Warn("Client send channel full, dropping message", "clientID", client.ID)
			}
		}
	}

	log.Info("Message sent to authenticated clients", "messageID", message.ID, "clientCount", sent)
}

