package events

import (
	"context"
	"encoding/json"
	"sync"
	"time"
	"waugzee/config"
	"waugzee/internal/logger"

	"github.com/google/uuid"
	"github.com/valkey-io/valkey-go"
)

type Channel string

func (c Channel) String() string {
	return string(c)
}

const (
	BROADCAST_CHANNEL Channel = "broadcast"
	SEND_CHANNEL      Channel = "send"
)

type MessageType string

const (
	PING          MessageType = "ping"
	PONG          MessageType = "pong"
	MESSAGE       MessageType = "message"
	BROADCAST     MessageType = "broadcast"
	SEND          MessageType = "send"
	ERROR         MessageType = "error"
	USER_JOIN     MessageType = "user_join"
	USER_LEAVE    MessageType = "user_leave"
	AUTH_REQUEST  MessageType = "auth_request"
	AUTH_RESPONSE MessageType = "auth_response"
	AUTH_SUCCESS  MessageType = "auth_success"
	AUTH_FAILURE  MessageType = "auth_failure"
	API_REQUEST   MessageType = "api_request"
	API_RESPONSE  MessageType = "api_response"
	SYNC_PROGRESS MessageType = "sync_progress"
	SYNC_COMPLETE MessageType = "sync_complete"
	SYNC_ERROR    MessageType = "sync_error"
)

type Event struct {
	ID        string         `json:"id"`
	Type      MessageType    `json:"type"`
	Channel   Channel        `json:"channel"`
	UserID    *uuid.UUID     `json:"userId,omitempty"`
	Data      map[string]any `json:"data"`
	Timestamp time.Time      `json:"timestamp"`
}

type EventHandler func(event Event) error

type EventBus struct {
	client   valkey.Client
	logger   logger.Logger
	config   config.Config
	handlers map[Channel][]EventHandler
	mutex    sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
}

func New(client valkey.Client, config config.Config) *EventBus {
	ctx, cancel := context.WithCancel(context.Background())

	return &EventBus{
		client:   client,
		logger:   logger.New("EventBus"),
		config:   config,
		handlers: make(map[Channel][]EventHandler),
		ctx:      ctx,
		cancel:   cancel,
	}
}

func (eb *EventBus) Publish(channel Channel, event Event) error {
	log := eb.logger.Function("Publish")

	if event.ID == "" {
		event.ID = uuid.New().String()
	}

	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	if event.Channel == "" {
		event.Channel = channel
	}

	eventData, err := json.Marshal(event)
	if err != nil {
		return log.Err("failed to marshal event", err, "eventID", event.ID)
	}

	ctx, cancel := context.WithTimeout(eb.ctx, 5*time.Second)
	defer cancel()

	err = eb.client.Do(ctx, eb.client.B().Publish().Channel(channel.String()).Message(string(eventData)).Build()).
		Error()
	if err != nil {
		return log.Err(
			"failed to publish event to valkey",
			err,
			"channel",
			channel,
			"eventID",
			event.ID,
		)
	}

	log.Info("Event published", "channel", channel, "eventID", event.ID, "eventType", event.Type)

	// Also notify local handlers
	eb.notifyLocalHandlers(channel, event)

	return nil
}

func (eb *EventBus) Subscribe(channel Channel, handler EventHandler) error {
	log := eb.logger.Function("Subscribe")

	eb.mutex.Lock()
	eb.handlers[channel] = append(eb.handlers[channel], handler)
	eb.mutex.Unlock()

	log.Info("Handler subscribed to channel", "channel", channel)

	// Start listening to this channel if it's the first handler
	go eb.listenToChannel(channel)

	return nil
}

func (eb *EventBus) notifyLocalHandlers(channel Channel, event Event) {
	log := eb.logger.Function("notifyLocalHandlers")

	eb.mutex.RLock()
	handlers, exists := eb.handlers[channel]
	eb.mutex.RUnlock()

	if !exists || len(handlers) == 0 {
		return
	}

	for i, handler := range handlers {
		go func(h EventHandler, handlerIndex int) {
			if err := h(event); err != nil {
				log.Er(
					"handler failed",
					err,
					"channel",
					channel,
					"eventID",
					event.ID,
					"handlerIndex",
					handlerIndex,
				)
			}
		}(handler, i)
	}
}

func (eb *EventBus) listenToChannel(channel Channel) {
	log := eb.logger.Function("listenToChannel")

	ctx, cancel := context.WithCancel(eb.ctx)
	defer cancel()

	log.Info("Starting to listen to channel", "channel", channel)

	err := eb.client.Receive(
		ctx,
		eb.client.B().Subscribe().Channel(channel.String()).Build(),
		func(msg valkey.PubSubMessage) {
			var event Event
			if err := json.Unmarshal([]byte(msg.Message), &event); err != nil {
				log.Er("failed to unmarshal event", err, "channel", channel, "message", msg.Message)
				return
			}

			log.Info(
				"Received event from valkey",
				"channel",
				channel,
				"eventID",
				event.ID,
				"eventType",
				event.Type,
			)
			eb.notifyLocalHandlers(channel, event)
		},
	)
	if err != nil {
		log.Er("failed to listen to channel", err, "channel", channel)
	}
}

func (eb *EventBus) Close() error {
	log := eb.logger.Function("Close")

	eb.cancel()

	log.Info("EventBus closed")
	return nil
}

func (eb *EventBus) PublishCacheInvalidation(
	resourceType string,
	resourceID string,
	userIDs []string,
) error {
	return eb.Publish("cache.invalidation", Event{
		Type: "cache_invalidation",
		Data: map[string]any{
			"resourceType": resourceType,
			"resourceId":   resourceID,
			"userIds":      userIDs,
		},
	})
}
