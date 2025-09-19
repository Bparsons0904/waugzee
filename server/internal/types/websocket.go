package types

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// TokenInfo represents validated token information
type TokenInfo struct {
	UserID          string
	Email           string
	Name            string
	GivenName       string
	FamilyName      string
	PreferredName   string
	EmailVerified   bool
	Roles           []string
	ProjectID       string
	Nonce           string
	Valid           bool
}

// ApiResponse represents API response data for Discogs operations
type ApiResponse struct {
	RequestID string            `json:"requestId"`
	Status    int               `json:"status"`
	Headers   map[string]string `json:"headers"`
	Body      json.RawMessage   `json:"body"`
	Error     *string           `json:"error,omitempty"`
}

// WebSocketMessage is a generic message interface for WebSocket communication
type WebSocketMessage interface {
	GetID() string
	GetType() string
	GetChannel() string
	GetAction() string
	GetUserID() string
	GetData() map[string]interface{}
	GetTimestamp() time.Time
}

// WebSocketSender interface for sending messages to users
type WebSocketSender interface {
	SendMessageToUser(userID uuid.UUID, message WebSocketMessage)
}

// WebSocketMessageImpl implements the WebSocketMessage interface
type WebSocketMessageImpl struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Channel   string                 `json:"channel"`
	Action    string                 `json:"action"`
	UserID    string                 `json:"userId,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// Implement WebSocketMessage interface
func (m *WebSocketMessageImpl) GetID() string                     { return m.ID }
func (m *WebSocketMessageImpl) GetType() string                   { return m.Type }
func (m *WebSocketMessageImpl) GetChannel() string               { return m.Channel }
func (m *WebSocketMessageImpl) GetAction() string                { return m.Action }
func (m *WebSocketMessageImpl) GetUserID() string                { return m.UserID }
func (m *WebSocketMessageImpl) GetData() map[string]interface{}  { return m.Data }
func (m *WebSocketMessageImpl) GetTimestamp() time.Time          { return m.Timestamp }