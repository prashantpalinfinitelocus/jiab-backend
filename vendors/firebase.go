package vendors

import (
	"context"
	"strings"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	log "github.com/sirupsen/logrus"

	"github.com/Infinite-Locus-Product/coca_cola_backend/config"
	"github.com/Infinite-Locus-Product/coca_cola_backend/constants"
	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
)

// IsInvalidTokenError reports whether err indicates the FCM token is permanently
// dead (uninstalled app, revoked registration, wrong sender, etc.) and should be
// invalidated rather than retried.
func IsInvalidTokenError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	for _, substr := range constants.FIREBASE_INVALID_TOKEN_ERROR_SUBSTRINGS {
		if strings.Contains(errStr, substr) {
			return true
		}
	}
	return false
}

type FirebaseClient struct {
	messagingClient *messaging.Client
	app             *firebase.App
}

// InitFirebase initializes Firebase client
func InitFirebase() *FirebaseClient {
	cfg := config.GetConfig()

	config := &firebase.Config{
		ProjectID: cfg.GcsConfig.ProjectID,
	}
	app, err := firebase.NewApp(context.Background(), config)
	if err != nil {
		log.Errorf("Failed to initialize Firebase app: %v", err)
		return nil
	}

	// Get messaging client
	messagingClient, err := app.Messaging(context.Background())
	if err != nil {
		log.Errorf("Failed to get messaging client: %v", err)
		return nil
	}

	log.Info("Firebase client initialized successfully")

	return &FirebaseClient{
		messagingClient: messagingClient,
		app:             app,
	}
}

// GetMessagingClient returns the Firebase messaging client
func (fc *FirebaseClient) GetMessagingClient() *messaging.Client {
	return fc.messagingClient
}

// SendMessage sends a single push notification
func (fc *FirebaseClient) SendMessage(ctx context.Context, token string, title string, body string, imageURL string, data map[string]string) (string, error) {
	if data == nil {
		data = map[string]string{}
	}

	data["title"] = title
	data["body"] = body
	if imageURL != "" {
		data["image_url"] = imageURL
	}

	message := &messaging.Message{
		Token: token,
		Data:  data,
		Android: &messaging.AndroidConfig{
			Priority: "high",
		},
		APNS: &messaging.APNSConfig{
			Payload: &messaging.APNSPayload{
				Aps: &messaging.Aps{
					Category: constants.NOTIFICATION_CATEGORY,
				},
			},
		},
	}

	if route, ok := data["route"]; ok && route != "" {
		base := "https://sprite-joke-in-a-bottle.coke2home.com"
		message.Webpush = &messaging.WebpushConfig{
			FCMOptions: &messaging.WebpushFCMOptions{
				Link: base + route,
			},
		}
		if message.Data == nil {
			message.Data = map[string]string{}
		}
		message.Data["web_link"] = base + route
		message.Data["deeplink"] = route
	}
	response, err := fc.messagingClient.Send(ctx, message)
	if err != nil {
		return "", errs.NewInternalServerError("failed to send push notification", err)
	}
	return response, nil
}

// BulkMessageResponse represents the response from sending bulk notifications
type BulkMessageResponse struct {
	SuccessCount int
	FailureCount int
	Responses    []*messaging.SendResponse
	// Tokens is parallel to Responses: Tokens[i] is the device token that
	// produced Responses[i], so callers can identify which tokens to invalidate.
	Tokens []string
}

// SendBulkMessage sends the same push notification to multiple devices in batches
func (fc *FirebaseClient) SendBulkMessage(ctx context.Context, tokens []string, title string, body string, imageURL string, data map[string]string) (*BulkMessageResponse, error) {
	if len(tokens) == 0 {
		return nil, errs.NewBadRequestError("no tokens provided", nil)
	}

	if data == nil {
		data = map[string]string{}
	}

	data["title"] = title
	data["body"] = body
	data["purpose"] = "campaign"
	if imageURL != "" {
		data["body_image_url"] = imageURL
	}
	base := "https://sprite-joke-in-a-bottle.coke2home.com"
	if data["base_url"] != "" {
		base = data["base_url"]
	}
	data["route"] = base

	const batchSize = 500
	totalSuccessCount := 0
	totalFailureCount := 0
	var allResponses []*messaging.SendResponse
	var allTokens []string

	// Process tokens in batches of 500
	for i := 0; i < len(tokens); i += batchSize {
		end := i + batchSize
		if end > len(tokens) {
			end = len(tokens)
		}

		batchTokens := tokens[i:end]
		log.Infof("Processing batch %d-%d of %d tokens", i, end, len(tokens))

		message := &messaging.MulticastMessage{
			Tokens: batchTokens,
			Data:   data,
			Android: &messaging.AndroidConfig{
				Priority: "high",
			},
			APNS: &messaging.APNSConfig{
				Payload: &messaging.APNSPayload{
					Aps: &messaging.Aps{
						Category: constants.NOTIFICATION_CATEGORY,
					},
				},
			},
		}

		if route, ok := data["route"]; ok && route != "" {
			message.Webpush = &messaging.WebpushConfig{
				FCMOptions: &messaging.WebpushFCMOptions{
					Link: base + route,
				},
			}
			if message.Data == nil {
				message.Data = map[string]string{}
			}
			message.Data["web_link"] = base + route
			message.Data["deeplink"] = route
		}

		batchResponse, err := fc.messagingClient.SendEachForMulticast(ctx, message)
		if err != nil {
			log.Errorf("Failed to send batch %d-%d: %v", i, end, err)
			// Continue with next batch even if this one fails
			continue
		}

		totalSuccessCount += batchResponse.SuccessCount
		totalFailureCount += batchResponse.FailureCount
		allResponses = append(allResponses, batchResponse.Responses...)
		allTokens = append(allTokens, batchTokens...)

		// Log any failures in this batch
		if batchResponse.FailureCount > 0 {
			log.Warnf("Batch %d-%d had %d failures out of %d messages",
				i, end, batchResponse.FailureCount, len(batchTokens))

			for j, resp := range batchResponse.Responses {
				if !resp.Success {
					log.Errorf("Failed to send notification to token at index %d: %v", i+j, resp.Error)
				}
			}
		}
	}

	log.Infof("Bulk notification complete: %d succeeded, %d failed out of %d total",
		totalSuccessCount, totalFailureCount, len(tokens))

	return &BulkMessageResponse{
		SuccessCount: totalSuccessCount,
		FailureCount: totalFailureCount,
		Responses:    allResponses,
		Tokens:       allTokens,
	}, nil
}
