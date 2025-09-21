package websockets

import (
	"context"
	"time"
	"waugzee/internal/events"

	"github.com/google/uuid"
)

// Authentication-related constants
const (
	AUTH_REQUEST              = "auth_request"
	AUTH_RESPONSE             = "auth_response"
	AUTH_SUCCESS              = "auth_success"
	AUTH_FAILURE              = "auth_failure"
	AUTH_HANDSHAKE_TIMEOUT    = 10 * time.Second
)

// startAuthTimeout initiates the authentication timeout for a client
func (c *Client) startAuthTimeout() {
	log := c.Manager.log.Function("startAuthTimeout")

	go func() {
		time.Sleep(AUTH_HANDSHAKE_TIMEOUT)
		if c.Status == STATUS_UNAUTHENTICATED {
			log.Warn("Client failed to authenticate within timeout, disconnecting",
				"clientID", c.ID,
				"timeout", AUTH_HANDSHAKE_TIMEOUT)

			authTimeout := Message{
				ID:        uuid.New().String(),
				Service:   events.SYSTEM,
				Event:     AUTH_FAILURE,
				Payload:   map[string]any{"action": "authentication_timeout", "reason": "Authentication timeout"},
				Timestamp: time.Now(),
			}

			select {
			case c.send <- authTimeout:
				// Message sent, now close after a brief delay
				time.Sleep(100 * time.Millisecond)
			default:
				// Channel is full or closed, proceed to close immediately
			}

			if err := c.Connection.Close(); err != nil {
				log.Er("failed to close connection after auth timeout", err, "clientID", c.ID)
			}
		}
	}()
}

// handleAuthResponse processes authentication response from client
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

// sendAuthFailure sends authentication failure response and closes connection
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

// sendAuthRequest sends initial authentication request to client
func (c *Client) sendAuthRequest() error {
	log := c.Manager.log.Function("sendAuthRequest")

	authRequest := Message{
		ID:        uuid.New().String(),
		Service:   events.SYSTEM,
		Event:     AUTH_REQUEST,
		Payload:   map[string]any{"action": "authenticate"},
		Timestamp: time.Now(),
	}

	if err := c.Connection.WriteJSON(authRequest); err != nil {
		log.Er("failed to send auth request", err)
		return err
	}

	log.Info("Auth request sent to client", "clientID", c.ID)
	return nil
}

// handleUnauthenticatedMessage handles messages from unauthenticated clients
func (c *Client) handleUnauthenticatedMessage(message Message) {
	log := c.Manager.log.Function("handleUnauthenticatedMessage")

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
}
