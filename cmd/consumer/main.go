package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"

	"github.com/Infinite-Locus-Product/coca_cola_backend/services"
	"github.com/Infinite-Locus-Product/coca_cola_backend/vendors"
)

func main() {
	startConsumer()
}

func startConsumer() {
	// Initialize Kafka client
	kafkaClient := vendors.InitKafka()
	if kafkaClient == nil {
		logrus.Fatal("Failed to initialize Kafka client")
	}
	defer kafkaClient.Close()

	// Initialize Firebase client
	firebaseClient := vendors.InitFirebase()
	if firebaseClient == nil {
		logrus.Errorf("Failed to initialize Firebase client")
	}

	// Create consumer service
	consumerService := services.NewConsumerService(kafkaClient.GetClient(), firebaseClient)

	// Create context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle OS signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start consuming messages
	consumerService.Start(ctx)

	// Wait for shutdown signal
	<-sigChan
	logrus.Info("Shutting down consumer...")
}
