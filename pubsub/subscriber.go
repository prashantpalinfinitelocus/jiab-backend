package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"cloud.google.com/go/pubsub"
	log "github.com/sirupsen/logrus"
)

// Subscriber represents a Google Cloud PubSub subscriber
type Subscriber struct {
	client *pubsub.Client
	subs   map[string]*pubsub.Subscription
	mu     sync.Mutex

	// retryCounts tracks per-message retry attempts client-side, since
	// mutating msg.Attributes before Nack does NOT persist back to Pub/Sub —
	// a redelivered message arrives with its original attributes, so relying
	// on msg.Attributes alone for a retry cap causes infinite redelivery for
	// any message a handler keeps failing on. Entries are removed once a
	// message is Acked (success or final drop) to keep this bounded to
	// in-flight/recently-retried messages, not every message ever seen.
	retryCounts   map[string]int
	retryCountsMu sync.Mutex
}

// MessageHandler is a function that processes a PubSub message
type MessageHandler func(ctx context.Context, msg *pubsub.Message) error

// NewSubscriber creates a new PubSub subscriber
func NewSubscriber(client *pubsub.Client) *Subscriber {
	return &Subscriber{
		client:      client,
		subs:        make(map[string]*pubsub.Subscription),
		retryCounts: make(map[string]int),
	}
}

// maxRetries is a defensive cap on client-side redelivery retries. This only
// matters for handlers that return a non-nil error; well-behaved handlers
// (e.g. SubscriberService) always ack and never rely on this path.
const maxRetries = 1

func (s *Subscriber) nextRetryCount(msgID string) int {
	s.retryCountsMu.Lock()
	defer s.retryCountsMu.Unlock()
	s.retryCounts[msgID]++
	return s.retryCounts[msgID]
}

func (s *Subscriber) clearRetryCount(msgID string) {
	s.retryCountsMu.Lock()
	defer s.retryCountsMu.Unlock()
	delete(s.retryCounts, msgID)
}

// Subscribe starts consuming messages from the specified subscription
func (s *Subscriber) Subscribe(ctx context.Context, subscriptionID string, handler MessageHandler) error {
	s.mu.Lock()
	subscription := s.client.Subscription(subscriptionID)
	s.subs[subscriptionID] = subscription
	s.mu.Unlock()

	// Check if subscription exists
	// exists, err := subscription.Exists(ctx)
	// if err != nil {
	// 	return fmt.Errorf("error checking subscription existence: %w", err)
	// }
	// if !exists {
	// 	return fmt.Errorf("subscription %s does not exist", subscriptionID)
	// }

	// Configure subscription to handle messages concurrently
	subscription.ReceiveSettings.MaxOutstandingMessages = 10
	subscription.ReceiveSettings.MaxOutstandingBytes = 1e8 // 100MB

	log.Infof("Starting to receive messages from subscription %s", subscriptionID)

	// Start receiving messages
	err := subscription.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		// Check if this is a test message
		isTestMessage := false
		if keyAttr, exists := msg.Attributes["key"]; exists && keyAttr == "test" {
			isTestMessage = true
			log.Infof("Testing message: %s", msg.ID)
		}

		if err := handler(ctx, msg); err != nil {
			if isTestMessage {
				log.Infof("Ping test message")
				msg.Ack()
				return
			}

			// Client-side retry count, NOT msg.Attributes: mutating a received
			// message's attributes before Nack never reaches Pub/Sub, so a
			// redelivered message always arrives looking like a fresh attempt.
			retryCount := s.nextRetryCount(msg.ID)
			if retryCount <= maxRetries {
				log.Infof("Retrying message (attempt %d): %s", retryCount, msg.ID)
				msg.Nack()
			} else {
				log.Errorf("Message failed after %d retries, dropping: %s, error: %v", maxRetries, msg.ID, err)
				s.clearRetryCount(msg.ID)
				msg.Ack()
			}
		} else {
			s.clearRetryCount(msg.ID)
			msg.Ack()
		}
	})

	if err != nil {
		return fmt.Errorf("error receiving messages: %w", err)
	}

	return nil
}

// CreateSubscriptionIfNotExists creates a subscription for a topic if it doesn't exist
func (s *Subscriber) CreateSubscriptionIfNotExists(
	ctx context.Context,
	subscriptionID string,
	topicName string,
) (*pubsub.Subscription, error) {
	subscription := s.client.Subscription(subscriptionID)

	exists, err := subscription.Exists(ctx)
	if err != nil {
		return nil, fmt.Errorf("error checking subscription existence: %w", err)
	}

	if !exists {
		topic := s.client.Topic(topicName)
		/*topicExists, err := topic.Exists(ctx)
		if err != nil {
			return nil, fmt.Errorf("error checking topic existence: %w", err)
		}
		if !topicExists {
			return nil, fmt.Errorf("topic %s does not exist", topicName)
		}*/

		// Create the subscription with default settings
		subscription, err = s.client.CreateSubscription(ctx, subscriptionID, pubsub.SubscriptionConfig{
			Topic:       topic,
			AckDeadline: 10 * time.Second, // Default acknowledgment deadline
		})
		if err != nil {
			return nil, fmt.Errorf("error creating subscription: %w", err)
		}
		log.Infof("Created subscription %s for topic %s", subscriptionID, topicName)
	}

	s.mu.Lock()
	s.subs[subscriptionID] = subscription
	s.mu.Unlock()

	return subscription, nil
}

// ProcessJSONMessage processes a PubSub message by unmarshaling it as JSON
func (s *Subscriber) ProcessJSONMessage(ctx context.Context, msg *pubsub.Message, target interface{}) error {
	if err := json.Unmarshal(msg.Data, target); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}
	return nil
}

// Close stops all subscriptions
func (s *Subscriber) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Clean up subscription references
	s.subs = make(map[string]*pubsub.Subscription)

	// No explicit close method needed for PubSub client
	// Resources will be cleaned up by Go's garbage collector
}
