package services

import (
	"context"
	"encoding/json"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
	pubsublib "github.com/Infinite-Locus-Product/coca_cola_backend/pubsub"
	"github.com/Infinite-Locus-Product/coca_cola_backend/repository"
	"github.com/Infinite-Locus-Product/coca_cola_backend/vendors"
)

// sendRetryAttempts is the number of in-process send attempts for a single
// message before giving up. This is intentionally NOT implemented via Pub/Sub
// Nack/redelivery: without a dead-letter policy on the subscription, Pub/Sub
// redelivers a Nacked message indefinitely, and any per-message attempt count
// stored on the received msg.Attributes is never persisted back to Pub/Sub, so
// a Nack-based retry loop never terminates. Retrying entirely within this
// handler call, then always acking, guarantees the message is never redelivered.
const sendRetryAttempts = 2

type SubscriberService struct {
	subscriber         *pubsublib.Subscriber
	firebaseClient     *vendors.FirebaseClient
	subscriptionID     string
	db                 *gorm.DB
	fcmDeviceTokenRepo repository.FCMDeviceTokenRepository
}

func NewSubscriberService(subscriber *pubsublib.Subscriber, firebaseClient *vendors.FirebaseClient, subscriptionID string, db *gorm.DB, fcmDeviceTokenRepo repository.FCMDeviceTokenRepository) *SubscriberService {
	return &SubscriberService{
		subscriber:         subscriber,
		firebaseClient:     firebaseClient,
		subscriptionID:     subscriptionID,
		db:                 db,
		fcmDeviceTokenRepo: fcmDeviceTokenRepo,
	}
}

func (s *SubscriberService) Start(ctx context.Context) error {
	handler := func(ctx context.Context, msg *pubsub.Message) error {
		logrus.Infof("Received message: %s", string(msg.Data))

		var pubsubMsg dtos.PubSubMessage
		if err := json.Unmarshal(msg.Data, &pubsubMsg); err != nil {
			logrus.Errorf("Failed to unmarshal message, dropping: %v", err)
			return nil
		}

		var result string
		var sendErr error
		for attempt := 1; attempt <= sendRetryAttempts; attempt++ {
			result, sendErr = s.firebaseClient.SendMessage(ctx, pubsubMsg.Token, pubsubMsg.Title, pubsubMsg.Body, pubsubMsg.ImageURL, map[string]string{})
			if sendErr == nil {
				logrus.Infof("Message sent to user with token: %s, result: %s", pubsubMsg.Token, result)
				return nil
			}

			if vendors.IsInvalidTokenError(sendErr) {
				logrus.Warnf("Invalid/unregistered FCM token, invalidating and dropping message: token=%s err=%v", pubsubMsg.Token, sendErr)
				s.invalidateToken(ctx, pubsubMsg.Token)
				return nil
			}

			logrus.Errorf("Error sending message (attempt %d/%d): %v", attempt, sendRetryAttempts, sendErr)
			if attempt < sendRetryAttempts {
				time.Sleep(500 * time.Millisecond)
			}
		}

		// Exhausted local retries on a non-permanent error. Do NOT return an
		// error here (that would Nack and, absent a dead-letter policy, cause
		// indefinite redelivery) — log and drop instead.
		logrus.Errorf("Giving up on message after %d attempts, dropping: %v", sendRetryAttempts, sendErr)
		return nil
	}

	// Start subscribing to messages
	if err := s.subscriber.Subscribe(ctx, s.subscriptionID, handler); err != nil {
		logrus.Errorf("Error subscribing to messages: %v", err)
		return err
	}

	logrus.Infof("Started subscribing to subscription: %s", s.subscriptionID)
	return nil
}

func (s *SubscriberService) invalidateToken(ctx context.Context, token string) {
	if s.db == nil || s.fcmDeviceTokenRepo == nil || token == "" {
		return
	}
	if err := s.fcmDeviceTokenRepo.InvalidateByDeviceToken(ctx, s.db, token); err != nil {
		logrus.Errorf("Failed to invalidate FCM token %s: %v", token, err)
	}
}

// Close closes the subscriber
func (s *SubscriberService) Close() {
	s.subscriber.Close()
}
