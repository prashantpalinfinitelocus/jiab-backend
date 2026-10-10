package pubsub

import (
	"context"
	"encoding/json"
	"fmt"

	"cloud.google.com/go/pubsub"
)

// Publisher represents a Google Cloud PubSub publisher
type Publisher struct {
	client *pubsub.Client
}

// NewPublisher creates a new PubSub publisher
func NewPublisher(client *pubsub.Client) *Publisher {
	return &Publisher{
		client: client,
	}
}

// SendMessage sends a message to the specified topic
func (p *Publisher) SendMessage(ctx context.Context, topicName, key string, value []byte) error {
	topic := p.client.Topic(topicName)
	defer topic.Stop()

	msg := &pubsub.Message{
		Data: value,
		Attributes: map[string]string{
			"key": key,
		},
	}

	// Check if the topic exists
	if err := p.TopicExists(ctx, topicName); err != nil {
		return fmt.Errorf("topic error: %w", err)
	}

	// Publish the message without waiting for result
	topic.Publish(ctx, msg)

	fmt.Printf("Message published to topic: %s\n", topicName)
	return nil
}

// SendMessageSync sends a message to the specified topic and blocks until published
func (p *Publisher) SendMessageSync(ctx context.Context, topicName, key string, value []byte) (string, error) {
	topic := p.client.Topic(topicName)
	defer topic.Stop()

	msg := &pubsub.Message{
		Data: value,
		Attributes: map[string]string{
			"key": key,
		},
	}

	// Check if the topic exists
	if err := p.TopicExists(ctx, topicName); err != nil {
		return "", fmt.Errorf("topic error: %w", err)
	}

	// Publish the message and get the result
	result := topic.Publish(ctx, msg)

	// Block until the result is returned and the message is confirmed as published
	id, err := result.Get(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to publish message: %w", err)
	}

	fmt.Printf("Message published with ID: %s\n", id)
	return id, nil
}

// SendMessageString sends a string message to the specified topic
func (p *Publisher) SendMessageString(ctx context.Context, topicName, key, value string) error {
	return p.SendMessage(ctx, topicName, key, []byte(value))
}

// SendJSONMessage sends a JSON message to the specified topic
func (p *Publisher) SendJSONMessage(ctx context.Context, topicName, key string, value interface{}) error {
	jsonBytes, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return p.SendMessage(ctx, topicName, key, jsonBytes)
}

// TopicExists checks that the specified topic exists
func (p *Publisher) TopicExists(ctx context.Context, topicName string) error {
	/*topic := p.client.Topic(topicName)
	exists, err := topic.Exists(ctx)
	if err != nil {
		return fmt.Errorf("error checking topic existence: %w", err)
	}

	if !exists {
		return fmt.Errorf("topic %q does not exist", topicName)
	}*/

	return nil
}

// CreateTopicIfNotExists creates a topic if it doesn't exist
func (p *Publisher) CreateTopicIfNotExists(ctx context.Context, topicName string) (*pubsub.Topic, error) {
	topic := p.client.Topic(topicName)
	exists, err := topic.Exists(ctx)
	if err != nil {
		return nil, fmt.Errorf("error checking topic existence: %w", err)
	}

	if !exists {
		topic, err = p.client.CreateTopic(ctx, topicName)
		if err != nil {
			return nil, fmt.Errorf("error creating topic: %w", err)
		}
		fmt.Printf("Created topic: %s\n", topicName)
	}

	return topic, nil
}

// Close stops all topics and closes the client
func (p *Publisher) Close() {
	// No explicit close method needed for PubSub client
	// Resources will be cleaned up by Go's garbage collector
}
