package events

import (
	"context"
	"encoding/json"
	"sync"
	"time"
	"waugzee/config"
	"waugzee/internal/logger"

	"github.com/valkey-io/valkey-go"
)

type Channel string

const (
	WEBSOCKET  Channel = "websocket"
	CONTROLLER Channel = "controller"
)

func (c Channel) String() string {
	return string(c)
}

type ChannelEvent struct {
	Event   string
	Message Message
}

type EventHandler func(event ChannelEvent) error

type EventBus struct {
	client   valkey.Client
	logger   logger.Logger
	config   config.Config
	handlers map[Channel][]EventHandler
	mutex    sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
}

type Service string

const (
	SYSTEM Service = "system"
	USER   Service = "user"
	API    Service = "api"
)

type EventType string

const (
	ADMIN_DOWNLOAD_PROGRESS   EventType = "admin_download_progress"
	ADMIN_DOWNLOAD_STATUS     EventType = "admin_download_status"
	ADMIN_PROCESSING_PROGRESS EventType = "admin_processing_progress"
)

type Message struct {
	ID        string         `json:"id"`
	Service   Service        `json:"service,omitempty"`
	Event     string         `json:"event"`
	UserID    string         `json:"userId,omitempty"`
	Payload   map[string]any `json:"payload,omitempty"`
	Timestamp time.Time      `json:"timestamp"`
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

func (eb *EventBus) Publish(
	channel Channel,
	eventTypeOrChannelEvent any,
	message ...Message,
) error {
	log := eb.logger.Function("Publish")

	var channelEvent ChannelEvent

	// Handle different parameter combinations
	// TODO: There is no reason to handle 2 different parameter combinations, refactor
	switch v := eventTypeOrChannelEvent.(type) {
	case ChannelEvent:
		// Legacy usage: Publish(channel, channelEvent)
		channelEvent = v
	case string:
		// New usage: Publish(channel, "eventType", message)
		if len(message) != 1 {
			return log.Err("message parameter required when using string event type", nil)
		}
		channelEvent = ChannelEvent{
			Event:   v,
			Message: message[0],
		}
	default:
		return log.Err("invalid event type parameter", nil)
	}

	eventData, err := json.Marshal(channelEvent)
	if err != nil {
		return log.Err("failed to marshal event", err, "channel", channel, "event", channelEvent)
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
			"event",
			channelEvent,
		)
	}

	eb.notifyLocalHandlers(channel, channelEvent)

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

func (eb *EventBus) notifyLocalHandlers(channel Channel, event ChannelEvent) {
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
			var event ChannelEvent
			if err := json.Unmarshal([]byte(msg.Message), &event); err != nil {
				log.Er("failed to unmarshal event", err, "channel", channel, "message", msg.Message)
				return
			}

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
