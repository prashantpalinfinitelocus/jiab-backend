package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/kmsg"
)

// Producer represents a Kafka producer
type Producer struct {
	client *kgo.Client
}

// NewProducer creates a new Kafka producer
func NewProducer(client *kgo.Client) *Producer {
	return &Producer{
		client: client,
	}
}

// SendMessage sends a message to the specified topic
func (p *Producer) SendMessage(ctx context.Context, topic, key, value string) error {
	record := &kgo.Record{
		Topic: topic,
		Key:   []byte(key),
		Value: []byte(value),
	}
	fmt.Printf("record: %v\n", record)

	// Check Kafka connection before sending the message
	if err := p.Ping(ctx); err != nil {
		return fmt.Errorf("kafka connection error: %w \n", err)
	}
	fmt.Printf("Sending message to topic %q\n", topic)
	// Send the message
	results := p.client.ProduceSync(ctx, record)
	fmt.Printf("results: %v\n", results)
	fmt.Printf("Message sent to topic new  %q\n", topic)

	// Check for errors
	// for _, result := range results {
	// 	if result.Err != nil {
	// 		return fmt.Errorf("failed to produce message: %w", result.Err)
	// 	}
	// }

	return nil
}

// SendMessageAsync sends a message asynchronously
func (p *Producer) SendMessageAsync(ctx context.Context, topic, key, value string, callback func(error)) {
	record := &kgo.Record{
		Topic: topic,
		Key:   []byte(key),
		Value: []byte(value),
	}

	p.client.Produce(ctx, record, func(r *kgo.Record, err error) {
		if callback != nil {
			callback(err)
		}
	})
}

// SendJSONMessage sends a JSON message to the specified topic
func (p *Producer) SendJSONMessage(ctx context.Context, topic, key string, value interface{}) error {
	fmt.Printf("value: %v\n", value)
	jsonBytes, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Printf("jsonBytes: %v\n", jsonBytes)

	return p.SendMessage(ctx, topic, key, string(jsonBytes))
}

func (p *Producer) SendJSONMessageAsync(ctx context.Context, topic, key string, value interface{}, callback func(error)) {
	jsonBytes, err := json.Marshal(value)
	if err != nil {
		callback(fmt.Errorf("failed to marshal JSON: %w", err))
		return // Return early if marshalling fails
	}

	p.SendMessageAsync(ctx, topic, key, string(jsonBytes), callback)
}

// Ping checks if the client can connect to the Kafka cluster.
func (p *Producer) Ping(ctx context.Context) error {
	return p.client.Ping(ctx)
}

// Flush ensures all messages are sent
func (p *Producer) Flush(ctx context.Context) error {
	return p.client.Flush(ctx)
}

// Close closes the producer
func (p *Producer) Close() {
	p.client.Close()
}

// Ready checks that the client can reach the cluster AND that the
// supplied topic exists and has at least one partition.
func (p *Producer) Ready(ctx context.Context, topic string) error {
	// 1) test network / auth connectivity
	if err := p.client.Ping(ctx); err != nil {
		return fmt.Errorf("kafka ping failed: %w", err)
	}

	// 2) fetch metadata for the topic using a kmsg request (works on all franz-go versions)
	mdReq := kmsg.NewPtrMetadataRequest()
	mdReq.Topics = []kmsg.MetadataRequestTopic{{Topic: kmsg.StringPtr(topic)}}

	mdResp, err := mdReq.RequestWith(ctx, p.client)
	if err != nil {
		return fmt.Errorf("metadata fetch failed: %w", err)
	}
	if len(mdResp.Topics) == 0 {
		return fmt.Errorf("topic %q not found", topic)
	}
	t := mdResp.Topics[0]
	if t.ErrorCode != 0 {
		return fmt.Errorf("topic %q error code: %d", topic, t.ErrorCode)
	}
	if len(t.Partitions) == 0 {
		return fmt.Errorf("topic %q has no partitions", topic)
	}
	return nil
}
