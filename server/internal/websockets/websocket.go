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
	MESSAGE_TYPE_PING       = "ping"
	MESSAGE_TYPE_PONG       = "pong"
	API_REQUEST             = "api_request"
	API_RESPONSE            = "api_response"
	API_PROGRESS            = "api_progress"
	API_COMPLETE            = "api_complete"
	API_ERROR               = "api_error"
	USER_JOIN               = "user_join"
	BROADCAST               = "broadcast"
	SEND                    = "send"
	MESSAGE                 = "message"
	ADMIN_DOWNLOAD_PROGRESS = "admin_download_progress"
	ADMIN_DOWNLOAD_STATUS   = "admin_download_status"
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
	hub                  *Hub
	db                   database.DB
	config               config.Config
	log                  logger.Logger
	eventBus             *events.EventBus
	zitadelService       ZitadelService
	userRepo             repositories.UserRepository
	orchestrationService *services.OrchestrationService
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
		db:                   db,
		config:               config,
		log:                  log,
		eventBus:             eventBus,
		zitadelService:       services.Zitadel,
		userRepo:             repos.User,
		orchestrationService: services.Orchestration,
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
	default:
		log.Warn("Broadcast channel is full, dropping message", "messageID", message.ID)
	}
}

func (m *Manager) sendToAdminUsers(message Message) {
	log := m.log.Function("sendToAdminUsers")
	m.hub.mutex.RLock()
	defer m.hub.mutex.RUnlock()

	sentCount := 0
	totalAdmins := 0

	for _, client := range m.hub.clients {
		if client.Status != STATUS_AUTHENTICATED {
			continue
		}

		if client.UserID == uuid.Nil {
			continue
		}

		user, err := m.userRepo.GetByID(context.Background(), nil, client.UserID)
		if err != nil || user == nil {
			log.Warn("Failed to get user for admin check", "userID", client.UserID, "error", err)
			continue
		}

		if user.IsAdmin {
			totalAdmins++
			select {
			case client.send <- message:
				sentCount++
			default:
				log.Warn("Admin client send channel full", "clientID", client.ID)
			}
		}
	}

	log.Info("Admin broadcast complete", "sentTo", sentCount, "totalAdmins", totalAdmins)
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
			c.handleAPIResponse(message)
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
	log := m.log.Function("subscribeToEventBus")
	log.Info("Starting WebSocket events subscription")

	if err := m.eventBus.Subscribe(events.WEBSOCKET, func(event events.ChannelEvent) error {

		switch event.Event {
		case "broadcast":
			m.sendToAuthenticatedClients(event.Message)
		case "user":
			m.sendToSpecificUser(event.Message)
		case "admin":
			m.sendToAdminUsers(event.Message)
		default:
			log.Warn("Unknown WebSocket event type", "event", event.Event)
		}
		return nil
	}); err != nil {
		log.Er("Failed to subscribe to WebSocket events", err)
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

}

func (m *Manager) sendToSpecificUser(message Message) {
	log := m.log.Function("sendToSpecificUser")

	if message.UserID == "" {
		log.Warn("Cannot send to specific user: UserID is empty", "messageID", message.ID)
		return
	}

	// Parse UserID from string
	userID, err := uuid.Parse(message.UserID)
	if err != nil {
		log.Er("Invalid UserID format", err, "messageID", message.ID, "userID", message.UserID)
		return
	}

	// Find the client for this specific user
	var targetClient *Client
	for _, client := range m.hub.clients {
		if client.Status == STATUS_AUTHENTICATED && client.UserID == userID {
			targetClient = client
			break
		}
	}

	if targetClient == nil {
		log.Warn("User not found or not connected", "messageID", message.ID, "userID", message.UserID)
		return
	}

	// Send message to the specific user's client
	select {
	case targetClient.send <- message:
		default:
		log.Warn("User's send channel full, dropping message", "messageID", message.ID, "userID", message.UserID, "clientID", targetClient.ID)
	}
}

// handleAPIResponse processes API responses from the client and routes them to the orchestration service
func (c *Client) handleAPIResponse(message Message) {
	log := c.Manager.log.Function("handleAPIResponse")

	if c.Status != STATUS_AUTHENTICATED {
		log.Warn("Received API response from unauthenticated client", "clientID", c.ID)
		return
	}

	// Validate that the message contains valid payload
	if message.Payload == nil {
		log.Warn("API response missing payload", "messageID", message.ID, "clientID", c.ID)
		return
	}


	// Pass the response to the orchestration service for processing
	if err := c.Manager.orchestrationService.HandleAPIResponse(context.Background(), message.Payload); err != nil {
		log.Er("Failed to process API response", err,
			"messageID", message.ID,
			"clientID", c.ID,
			"userID", c.UserID)
		return
	}

}

