package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

// Consumer represents a Kafka consumer
type Consumer struct {
	client *kgo.Client
}

type MessageHandler func(ctx context.Context, record *kgo.Record) error

// NewConsumer creates a new Kafka consumer
func NewConsumer(client *kgo.Client) *Consumer {
	return &Consumer{
		client: client,
	}
}

// ConsumeMessages starts consuming messages from subscribed topics
func (c *Consumer) ConsumeMessages(ctx context.Context, handler MessageHandler) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			fetches := c.client.PollFetches(ctx)
			if errs := fetches.Errors(); len(errs) > 0 {
				// Handle fetch errors
				for _, err := range errs {
					log.Errorf("Fetch error: %v", err)
				}
				continue
			}

			// Process records
			fetches.EachPartition(func(p kgo.FetchTopicPartition) {
				for _, record := range p.Records {
					if err := handler(ctx, record); err != nil {
						log.Errorf("Error handling message: %v", err)
					}
				}
			})

			// Commit offsets if auto-commit is disabled
			if err := c.client.CommitUncommittedOffsets(ctx); err != nil {
				log.Errorf("Error committing offsets: %v", err)
			}
		}
	}
}

// ConsumeMessage consumes a single message (useful for testing or batch processing)
func (c *Consumer) ConsumeMessage(ctx context.Context, timeout time.Duration) (*kgo.Record, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	fetches := c.client.PollFetches(timeoutCtx)
	if errs := fetches.Errors(); len(errs) > 0 {
		return nil, fmt.Errorf("fetch errors: %v", errs)
	}

	var record *kgo.Record
	fetches.EachPartition(func(p kgo.FetchTopicPartition) {
		if len(p.Records) > 0 && record == nil {
			record = p.Records[0]
		}
	})

	if record == nil {
		return nil, fmt.Errorf("no message received within timeout")
	}

	return record, nil
}

// ConsumeJSONMessage consumes a message and unmarshals it as JSON
func (c *Consumer) ConsumeJSONMessage(ctx context.Context, timeout time.Duration, target interface{}) error {
	record, err := c.ConsumeMessage(ctx, timeout)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(record.Value, target); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return nil
}

// CommitOffsets manually commits current offsets
func (c *Consumer) CommitOffsets(ctx context.Context) error {
	return c.client.CommitUncommittedOffsets(ctx)
}

// GetCurrentOffsets returns the current committed offsets for all partitions
func (c *Consumer) GetCurrentOffsets(ctx context.Context) (map[string]map[int32]int64, error) {
	offsets := c.client.CommittedOffsets()
	result := make(map[string]map[int32]int64)

	for topic, partitions := range offsets {
		result[topic] = make(map[int32]int64)
		for partition, epochOffset := range partitions {
			result[topic][partition] = epochOffset.Offset
		}
	}

	return result, nil
}

// Close closes the consumer
func (c *Consumer) Close() {
	c.client.Close()
}
