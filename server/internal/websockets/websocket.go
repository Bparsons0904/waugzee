package websockets

import (
	"context"
	"encoding/json"
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
	AUTH_REQUEST      = "auth_request"
	AUTH_RESPONSE     = "auth_response"
	AUTH_SUCCESS      = "auth_success"
	AUTH_FAILURE      = "auth_failure"
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
	PING_INTERVAL          = 30 * time.Second
	PONG_TIMEOUT           = 60 * time.Second
	WRITE_TIMEOUT          = 10 * time.Second
	AUTH_HANDSHAKE_TIMEOUT = 10 * time.Second
	MAX_MESSAGE_SIZE       = 1024 * 1024 // 1 MB
	SEND_CHANNEL_SIZE      = 64
)

// DiscogsOrchestrationService interface to avoid circular imports
type DiscogsOrchestrationService interface {
	ProcessApiResponse(ctx context.Context, requestID string, response *types.ApiResponse) error
	HandleSyncDisconnection(ctx context.Context, userID uuid.UUID) error
	HandleSyncReconnection(ctx context.Context, userID uuid.UUID) error
}

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
	hub                     *Hub
	db                      database.DB
	config                  config.Config
	log                     logger.Logger
	eventBus                *events.EventBus
	zitadelService          ZitadelService
	userRepo                repositories.UserRepository
	discogsOrchestrationSvc DiscogsOrchestrationService
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

	authRequest := Message{
		ID:        uuid.New().String(),
		Service:   events.SYSTEM,
		Event:     AUTH_REQUEST,
		Payload:   map[string]any{"action": "authenticate"},
		Timestamp: time.Now(),
	}

	if err := c.WriteJSON(authRequest); err != nil {
		log.Er("failed to send auth request", err)
		if err := c.Close(); err != nil {
			log.Er("failed to close connection", err)
		}
		return
	}

	log.Info("Auth request sent to client", "clientID", clientID)
	m.hub.register <- client
	defer func() {
		log.Info("Client disconnected in the defer", "clientID", clientID)
		m.hub.unregister <- client
		if err := c.Close(); err != nil {
			log.Er("failed to close connection", err)
		}
	}()

	// Start auth timeout goroutine
	go func() {
		time.Sleep(AUTH_HANDSHAKE_TIMEOUT)
		if client.Status == STATUS_UNAUTHENTICATED {
			log.Warn("Client failed to authenticate within timeout, disconnecting",
				"clientID", clientID,
				"timeout", AUTH_HANDSHAKE_TIMEOUT)

			authTimeout := Message{
				ID:        uuid.New().String(),
				Service:   events.SYSTEM,
				Event:     AUTH_FAILURE,
				Payload:   map[string]any{"action": "authentication_timeout", "reason": "Authentication timeout"},
				Timestamp: time.Now(),
			}

			select {
			case client.send <- authTimeout:
				// Message sent, now close after a brief delay
				time.Sleep(100 * time.Millisecond)
			default:
				// Channel is full or closed, proceed to close immediately
			}

			if err := c.Close(); err != nil {
				log.Er("failed to close connection after auth timeout", err, "clientID", clientID)
			}
		}
	}()

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
		log.Warn(
			"Blocking message from unauthenticated client",
			"clientID",
			c.ID,
			"messageEvent",
			message.Event,
		)
		authFailure := Message{
			ID:        uuid.New().String(),
			Service:   events.SYSTEM,
			Event:     AUTH_FAILURE,
			Payload:   map[string]any{"action": "authentication_required", "reason": "Authentication required"},
			Timestamp: time.Now(),
		}
		c.send <- authFailure
		return
	}

	switch message.Event {
	case API_RESPONSE:
		c.handleDiscogsApiResponse(message)
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

func (c *Client) handleAuthResponse(message Message) {
	log := c.Manager.log.Function("handleAuthResponse")

	if c.Status != STATUS_UNAUTHENTICATED {
		log.Warn("Auth response from already authenticated client", "clientID", c.ID)
		return
	}

	token, ok := message.Payload["token"].(string)
	if !ok || token == "" {
		log.Warn("Invalid token in auth response", "clientID", c.ID)
		c.sendAuthFailure("Invalid token format")
		return
	}

	// Validate token using the consolidated method
	tokenInfo, validationMethod, err := c.Manager.zitadelService.ValidateTokenWithFallback(
		context.Background(),
		token,
	)
	if err != nil {
		log.Info("WebSocket token validation failed", "clientID", c.ID, "error", err.Error())
		c.sendAuthFailure("Authentication failed")
		return
	}

	// Get user from database using OIDC User ID
	user, err := c.Manager.userRepo.GetByOIDCUserID(context.Background(), tokenInfo.UserID)
	if err != nil {
		log.Info("WebSocket user not found in database",
			"clientID", c.ID,
			"oidcUserID", tokenInfo.UserID,
			"error", err.Error())
		c.sendAuthFailure("User not found")
		return
	}

	// Set client as authenticated with the validated user
	c.Status = STATUS_AUTHENTICATED
	c.UserID = user.ID

	log.Info("WebSocket client authenticated successfully",
		"clientID", c.ID,
		"userID", user.ID,
		"email", tokenInfo.Email,
		"method", validationMethod)

	c.Manager.promoteClientToAuthenticated(c)

	authSuccess := Message{
		ID:        uuid.New().String(),
		Service:   events.SYSTEM,
		Event:     AUTH_SUCCESS,
		UserID:    c.UserID.String(),
		Payload:   map[string]any{"action": "authenticated", "userId": c.UserID.String()},
		Timestamp: time.Now(),
	}

	c.send <- authSuccess
}

func (c *Client) sendAuthFailure(reason string) {
	log := c.Manager.log.Function("sendAuthFailure")

	authFailure := Message{
		ID:        uuid.New().String(),
		Service:   events.SYSTEM,
		Event:     AUTH_FAILURE,
		Payload:   map[string]any{"action": "authentication_failed", "reason": reason},
		Timestamp: time.Now(),
	}

	c.send <- authFailure

	log.Info("Auth failure sent, closing connection", "clientID", c.ID, "reason", reason)

	go func() {
		time.Sleep(100 * time.Millisecond)
		_ = c.Connection.Close()
	}()
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

func (c *Client) handleDiscogsApiResponse(message Message) {
	log := c.Manager.log.Function("handleDiscogsApiResponse")

	if c.Status != STATUS_AUTHENTICATED {
		log.Warn("Discogs API response from unauthenticated client", "clientID", c.ID)
		return
	}

	// Extract response data
	requestID, ok := message.Payload["requestId"].(string)
	if !ok {
		log.Warn("Invalid requestId in Discogs API response", "clientID", c.ID)
		return
	}

	status, ok := message.Payload["status"].(float64)
	if !ok {
		log.Warn("Invalid status in Discogs API response", "clientID", c.ID, "requestID", requestID)
		return
	}

	headers, ok := message.Payload["headers"].(map[string]any)
	if !ok {
		log.Warn(
			"Invalid headers in Discogs API response",
			"clientID",
			c.ID,
			"requestID",
			requestID,
		)
		return
	}

	body := message.Payload["body"]

	var errorPtr *string
	if errorMsg, exists := message.Payload["error"]; exists {
		if errorStr, ok := errorMsg.(string); ok {
			errorPtr = &errorStr
		}
	}

	// Convert headers to map[string]string
	headerMap := make(map[string]string)
	for k, v := range headers {
		if strVal, ok := v.(string); ok {
			headerMap[k] = strVal
		}
	}

	// Create API response
	apiResponse := &types.ApiResponse{
		RequestID: requestID,
		Status:    int(status),
		Headers:   headerMap,
		Error:     errorPtr,
	}

	// Marshal body to json.RawMessage
	if body != nil {
		if bodyBytes, err := json.Marshal(body); err == nil {
			apiResponse.Body = json.RawMessage(bodyBytes)
		}
	}

	// Process the response
	if err := c.Manager.discogsOrchestrationSvc.ProcessApiResponse(context.Background(), requestID, apiResponse); err != nil {
		_ = log.Error(
			"Failed to process Discogs API response",
			"error",
			err,
			"requestID",
			requestID,
		)
	} else {
		log.Info("Discogs API response processed successfully", "requestID", requestID, "status", status)
	}
}

// handleClientDisconnection manages sync state when clients disconnect
func (m *Manager) handleClientDisconnection(userID uuid.UUID) {
	log := m.log.Function("handleClientDisconnection")

	if m.discogsOrchestrationSvc == nil {
		return
	}

	if err := m.discogsOrchestrationSvc.HandleSyncDisconnection(context.Background(), userID); err != nil {
		_ = log.Error("Failed to handle sync disconnection", "error", err, "userID", userID)
	} else {
		log.Info("Sync disconnection handled", "userID", userID)
	}
}

// handleClientReconnection manages sync state when clients reconnect
func (m *Manager) handleClientReconnection(userID uuid.UUID) {
	log := m.log.Function("handleClientReconnection")

	if m.discogsOrchestrationSvc == nil {
		return
	}

	if err := m.discogsOrchestrationSvc.HandleSyncReconnection(context.Background(), userID); err != nil {
		_ = log.Error("Failed to handle sync reconnection", "error", err, "userID", userID)
	} else {
		log.Info("Sync reconnection handled", "userID", userID)
	}
}
