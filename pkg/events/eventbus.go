package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

// EventType represents the type of event
type EventType string

const (
	UserCreated           EventType = "user.created"
	UserUpdated           EventType = "user.updated"
	UserDeleted           EventType = "user.deleted"
	OrderCreated          EventType = "order.created"
	OrderStatusChanged    EventType = "order.status_changed"
	OrderCancelled        EventType = "order.cancelled"
	PaymentProcessed      EventType = "payment.processed"
	PaymentFailed         EventType = "payment.failed"
	PaymentRefunded       EventType = "payment.refunded"
	ProductCreated        EventType = "product.created"
	ProductUpdated        EventType = "product.updated"
	ProductInventoryChanged EventType = "product.inventory_changed"
	NotificationSent      EventType = "notification.sent"
)

// Event represents a domain event
type Event struct {
	ID        string            `json:"id"`
	Type      EventType         `json:"type"`
	Source    string            `json:"source"`
	Subject   string            `json:"subject"`
	Data      map[string]interface{} `json:"data"`
	Metadata  map[string]string `json:"metadata"`
	Timestamp time.Time         `json:"timestamp"`
}

// EventHandler defines the interface for event handlers
type EventHandler func(ctx context.Context, event *Event) error

// EventBus defines the interface for event publishing and subscribing
type EventBus interface {
	Publish(ctx context.Context, event *Event) error
	Subscribe(eventType EventType, handler EventHandler) error
	Unsubscribe(eventType EventType) error
	Start(ctx context.Context) error
	Stop() error
}

// RedisEventBus implements EventBus using Redis Pub/Sub
type RedisEventBus struct {
	client    *redis.Client
	handlers  map[EventType][]EventHandler
	mu        sync.RWMutex
	pubsub    *redis.PubSub
	stopChan  chan struct{}
	started   bool
}

// NewRedisEventBus creates a new Redis-based event bus
func NewRedisEventBus(redisURL string) (*RedisEventBus, error) {
	client := redis.NewClient(&redis.Options{
		Addr: redisURL,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %v", err)
	}

	return &RedisEventBus{
		client:   client,
		handlers: make(map[EventType][]EventHandler),
		stopChan: make(chan struct{}),
	}, nil
}

// Publish publishes an event to the event bus
func (eb *RedisEventBus) Publish(ctx context.Context, event *Event) error {
	if event.ID == "" {
		event.ID = generateEventID()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %v", err)
	}

	channel := fmt.Sprintf("events:%s", event.Type)
	return eb.client.Publish(ctx, channel, data).Err()
}

// Subscribe subscribes to events of a specific type
func (eb *RedisEventBus) Subscribe(eventType EventType, handler EventHandler) error {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.handlers[eventType] = append(eb.handlers[eventType], handler)
	return nil
}

// Unsubscribe removes all handlers for a specific event type
func (eb *RedisEventBus) Unsubscribe(eventType EventType) error {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	delete(eb.handlers, eventType)
	return nil
}

// Start starts the event bus and begins processing events
func (eb *RedisEventBus) Start(ctx context.Context) error {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	if eb.started {
		return fmt.Errorf("event bus already started")
	}

	// Subscribe to all event types we have handlers for
	channels := make([]string, 0, len(eb.handlers))
	for eventType := range eb.handlers {
		channels = append(channels, fmt.Sprintf("events:%s", eventType))
	}

	if len(channels) == 0 {
		log.Println("No event handlers registered, not starting event bus")
		return nil
	}

	eb.pubsub = eb.client.Subscribe(ctx, channels...)
	eb.started = true

	// Start processing messages
	go eb.processMessages(ctx)

	log.Printf("Event bus started, subscribed to channels: %v", channels)
	return nil
}

// Stop stops the event bus
func (eb *RedisEventBus) Stop() error {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	if !eb.started {
		return nil
	}

	close(eb.stopChan)
	
	if eb.pubsub != nil {
		eb.pubsub.Close()
	}

	eb.started = false
	log.Println("Event bus stopped")
	return nil
}

// processMessages processes incoming messages from Redis
func (eb *RedisEventBus) processMessages(ctx context.Context) {
	ch := eb.pubsub.Channel()

	for {
		select {
		case <-eb.stopChan:
			return
		case <-ctx.Done():
			return
		case msg := <-ch:
			eb.handleMessage(ctx, msg)
		}
	}
}

// handleMessage handles a single message
func (eb *RedisEventBus) handleMessage(ctx context.Context, msg *redis.Message) {
	var event Event
	if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
		log.Printf("Failed to unmarshal event: %v", err)
		return
	}

	eb.mu.RLock()
	handlers, exists := eb.handlers[event.Type]
	eb.mu.RUnlock()

	if !exists || len(handlers) == 0 {
		return
	}

	// Execute all handlers for this event type
	for _, handler := range handlers {
		go func(h EventHandler) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Event handler panicked: %v", r)
				}
			}()

			if err := h(ctx, &event); err != nil {
				log.Printf("Event handler failed: %v", err)
			}
		}(handler)
	}
}

// generateEventID generates a unique event ID
func generateEventID() string {
	return fmt.Sprintf("evt_%d_%d", time.Now().UnixNano(), rand.Intn(10000))
}

// EventStore defines interface for storing events
type EventStore interface {
	Store(ctx context.Context, event *Event) error
	GetEvents(ctx context.Context, subject string, fromTime time.Time) ([]*Event, error)
	GetEventsByType(ctx context.Context, eventType EventType, fromTime time.Time) ([]*Event, error)
}

// RedisEventStore implements EventStore using Redis
type RedisEventStore struct {
	client *redis.Client
}

// NewRedisEventStore creates a new Redis-based event store
func NewRedisEventStore(redisURL string) (*RedisEventStore, error) {
	client := redis.NewClient(&redis.Options{
		Addr: redisURL,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %v", err)
	}

	return &RedisEventStore{client: client}, nil
}

// Store stores an event in the event store
func (es *RedisEventStore) Store(ctx context.Context, event *Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %v", err)
	}

	// Store in multiple keys for different query patterns
	pipe := es.client.Pipeline()
	
	// Store by subject
	subjectKey := fmt.Sprintf("events:subject:%s", event.Subject)
	pipe.ZAdd(ctx, subjectKey, &redis.Z{
		Score:  float64(event.Timestamp.Unix()),
		Member: string(data),
	})

	// Store by event type
	typeKey := fmt.Sprintf("events:type:%s", event.Type)
	pipe.ZAdd(ctx, typeKey, &redis.Z{
		Score:  float64(event.Timestamp.Unix()),
		Member: string(data),
	})

	// Store by source
	sourceKey := fmt.Sprintf("events:source:%s", event.Source)
	pipe.ZAdd(ctx, sourceKey, &redis.Z{
		Score:  float64(event.Timestamp.Unix()),
		Member: string(data),
	})

	_, err = pipe.Exec(ctx)
	return err
}

// GetEvents retrieves events for a subject from a specific time
func (es *RedisEventStore) GetEvents(ctx context.Context, subject string, fromTime time.Time) ([]*Event, error) {
	key := fmt.Sprintf("events:subject:%s", subject)
	return es.getEventsFromKey(ctx, key, fromTime)
}

// GetEventsByType retrieves events of a specific type from a specific time
func (es *RedisEventStore) GetEventsByType(ctx context.Context, eventType EventType, fromTime time.Time) ([]*Event, error) {
	key := fmt.Sprintf("events:type:%s", eventType)
	return es.getEventsFromKey(ctx, key, fromTime)
}

// getEventsFromKey retrieves events from a Redis sorted set key
func (es *RedisEventStore) getEventsFromKey(ctx context.Context, key string, fromTime time.Time) ([]*Event, error) {
	results, err := es.client.ZRangeByScore(ctx, key, &redis.ZRangeBy{
		Min: fmt.Sprintf("%d", fromTime.Unix()),
		Max: "+inf",
	}).Result()
	
	if err != nil {
		return nil, err
	}

	events := make([]*Event, 0, len(results))
	for _, result := range results {
		var event Event
		if err := json.Unmarshal([]byte(result), &event); err != nil {
			log.Printf("Failed to unmarshal stored event: %v", err)
			continue
		}
		events = append(events, &event)
	}

	return events, nil
}