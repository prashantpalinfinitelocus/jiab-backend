package services

import (
	"context"
	"encoding/json"

	"github.com/sirupsen/logrus"
	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
	"github.com/Infinite-Locus-Product/coca_cola_backend/kafka"
	"github.com/Infinite-Locus-Product/coca_cola_backend/vendors"
)

type ConsumerService struct {
	kafkaConsumer  *kafka.Consumer
	firebaseClient *vendors.FirebaseClient
}

func NewConsumerService(kafkaClient *kgo.Client, firebaseClient *vendors.FirebaseClient) *ConsumerService {
	consumer := kafka.NewConsumer(kafkaClient)
	return &ConsumerService{
		kafkaConsumer:  consumer,
		firebaseClient: firebaseClient,
	}
}

func (s *ConsumerService) Start(ctx context.Context) {
	handler := func(ctx context.Context, record *kgo.Record) error {
		logrus.Infof("Received message: %s", string(record.Value))
		var msg dtos.KafkaMessage
		if err := json.Unmarshal(record.Value, &msg); err != nil {
			logrus.Errorf("Failed to unmarshal message: %v", err)
			return err
		}
		result, err := s.firebaseClient.SendMessage(ctx, msg.Token, msg.Title, msg.Body, msg.ImageURL, map[string]string{})
		logrus.Infof("message sent to user : %v\n", record.Key)
		logrus.Infof("result: %v\n", result)
		if err != nil {
			logrus.Errorf("Error sending message: %v", err)
		}
		return nil
	}

	go func() {
		if err := s.kafkaConsumer.ConsumeMessages(ctx, handler); err != nil {
			logrus.Errorf("Error consuming messages: %v", err)
		}
	}()
}
