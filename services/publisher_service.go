package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
	pubsublib "github.com/Infinite-Locus-Product/coca_cola_backend/pubsub"
	"github.com/sirupsen/logrus"
)

type PublisherService struct {
	publisher *pubsublib.Publisher
}

func NewPublisherService(pubsubClient *pubsub.Client) *PublisherService {
	publisher := pubsublib.NewPublisher(pubsubClient)
	return &PublisherService{
		publisher: publisher,
	}
}

func (s *PublisherService) PublishMessage(ctx context.Context, topic string, key string, message dtos.PubSubMessage) {
	messageData, err := json.Marshal(message)
	if err != nil {
		logrus.Errorf("Failed to marshal message: %v", err)
		return
	}

	// Send the message - now non-blocking at the Publisher level
	if err := s.publisher.SendMessage(ctx, topic, key, messageData); err != nil {
		logrus.Errorf("Failed to publish message to topic %s: %v", topic, err)
		return
	}

	logrus.Infof("Message sent to topic: %s", topic)
}

// Ping sends a test message to verify that the publisher is working
// Blocks until the message is confirmed to be published
func (s *PublisherService) Ping(ctx context.Context, topic string) (string, error) {
	pingMessage := dtos.PubSubMessage{
		Token:    "ping-token",
		Title:    "Ping",
		Body:     fmt.Sprintf("PubSub ping test at %s", time.Now().Format(time.RFC3339)),
		ImageURL: "",
	}

	messageData, err := json.Marshal(pingMessage)
	if err != nil {
		return "", fmt.Errorf("failed to marshal ping message: %w", err)
	}

	messageID, err := s.publisher.SendMessageSync(ctx, topic, "test", messageData)
	if err != nil {
		return "", fmt.Errorf("ping failed: %w", err)
	}

	logrus.Infof("Ping message sent successfully to topic: %s, message ID: %s", topic, messageID)
	return messageID, nil
}
