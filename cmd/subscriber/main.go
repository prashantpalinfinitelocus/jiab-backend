package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Infinite-Locus-Product/coca_cola_backend/config"
	pubsublib "github.com/Infinite-Locus-Product/coca_cola_backend/pubsub"
	"github.com/Infinite-Locus-Product/coca_cola_backend/repository"
	"github.com/Infinite-Locus-Product/coca_cola_backend/services"
	"github.com/Infinite-Locus-Product/coca_cola_backend/vendors"
)

func main() {
	startSubscriber()
}

func startSubscriber() {
	// Initialize PubSub client
	pubsubClient := vendors.InitPubSub()
	if pubsubClient == nil {
		logrus.Fatal("Failed to initialize PubSub client")
	}

	// Initialize Firebase client
	firebaseClient := vendors.InitFirebase()
	if firebaseClient == nil {
		logrus.Error("Failed to initialize Firebase client")
	}

	// Initialize database so dead FCM tokens can be invalidated. This is a
	// secondary/optional dependency for this service: its core job is
	// relaying push notifications, which must keep working even if the DB is
	// unreachable or (as happened once) not configured for this Cloud Run
	// service at all — so a DB failure here is logged, not fatal. When db is
	// nil, SubscriberService simply skips token invalidation (see its
	// invalidateToken nil-guard) and continues relaying messages normally.
	var fcmDeviceTokenRepo repository.FCMDeviceTokenRepository
	db, err := vendors.TryInitDatabase()
	if err != nil {
		logrus.Errorf("Failed to initialize database, FCM token invalidation will be disabled: %v", err)
		db = nil
	} else {
		fcmDeviceTokenRepo = repository.NewFCMDeviceTokenRepository()
	}

	subscriber := pubsublib.NewSubscriber(pubsubClient)

	// Define the subscription ID
	subscriptionID := config.GetConfig().PubSubConfig.SubscriptionID

	// Create subscriber service
	subscriberService := services.NewSubscriberService(subscriber, firebaseClient, subscriptionID, db, fcmDeviceTokenRepo)

	// Create context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start HTTP server for health checks
	server := setupHealthCheckServer()

	// Handle OS signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start the subscriber service
	go func() {
		if err := subscriberService.Start(ctx); err != nil {
			logrus.Errorf("Error starting subscriber service: %v", err)
		}
	}()

	logrus.Info("Subscriber service started successfully")

	// Wait for shutdown signal
	<-sigChan
	logrus.Info("Shutting down subscriber...")

	// Gracefully shutdown the HTTP server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logrus.Errorf("Server shutdown error: %v", err)
	}

	subscriberService.Close()
}

func setupHealthCheckServer() *http.Server {
	// Define port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Cloud Run default port
	}

	// Create a new mux for this server
	mux := http.NewServeMux()

	// Create a simple health check handler
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// Create HTTP server with our mux
	server := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Create error channel to catch server startup errors
	errChan := make(chan error, 1)

	// Start the server in a goroutine
	go func() {
		logrus.Infof("Starting health check server on :%s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// Check for immediate startup errors (like port already in use)
	select {
	case err := <-errChan:
		logrus.Fatalf("Server failed to start: %v", err)
	case <-time.After(100 * time.Millisecond):
		// Server started successfully
	}

	return server
}
