package vendors

import (
	"crypto/tls"
	"fmt"
	"time"

	"github.com/Infinite-Locus-Product/coca_cola_backend/config"
	log "github.com/sirupsen/logrus"
	"github.com/twmb/franz-go/pkg/kgo"
)

type KafkaClient struct {
	client *kgo.Client
}

// InitKafka initializes Kafka client
func InitKafka() *KafkaClient {
	cfg := config.GetConfig()

	fmt.Printf("Kafka brokers : %v\n", cfg.KafkaConfig.Brokers)
	opts := []kgo.Opt{
		kgo.SeedBrokers(cfg.KafkaConfig.Brokers...),
		kgo.ClientID(cfg.KafkaConfig.ClientID),
		kgo.DefaultProduceTopic(cfg.KafkaConfig.DefaultTopic),
		kgo.ProducerBatchCompression(kgo.SnappyCompression()),
		kgo.ProduceRequestTimeout(15 * time.Second),
		kgo.RequestTimeoutOverhead(15 * time.Second),

		// Consumer Config
		kgo.ConsumerGroup(cfg.KafkaConfig.ConsumerGroup),
		kgo.ConsumeTopics(cfg.KafkaConfig.DefaultTopic),
	}

	// Add SSL/SASL configuration if needed
	if cfg.KafkaConfig.EnableSSL {
		opts = append(opts, kgo.DialTLSConfig(&tls.Config{MinVersion: tls.VersionTLS12}))
	}

	client, err := kgo.NewClient(opts...)
	if err != nil {
		log.Fatalf("Failed to initialize Kafka client: %v", err)
		return nil
	}
	log.Info("Kafka client initialized successfully")

	return &KafkaClient{
		client: client,
	}
}

// GetClient returns the underlying Kafka client
func (kc *KafkaClient) GetClient() *kgo.Client {
	return kc.client
}

func (k *KafkaClient) Close() {
	if k.client != nil {
		k.client.Close()
	}
}
