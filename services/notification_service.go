package services

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Infinite-Locus-Product/coca_cola_backend/constants"
	"github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	"github.com/Infinite-Locus-Product/coca_cola_backend/repository"
	"github.com/Infinite-Locus-Product/coca_cola_backend/utils"
	"github.com/Infinite-Locus-Product/coca_cola_backend/vendors"
)

type NotificationService interface {
	RegisterDevice(ctx context.Context, userId string, deviceToken string) error
	GetUserNotifications(ctx context.Context, userId string, filter dtos.NotificationFilter) ([]dtos.NotificationResponseDTO, int, error)
	GetNotificationCount(ctx context.Context, userId string) (int, error)
	SendPushNotification(ctx context.Context, tx *gorm.DB, notificationType int, userId string, amount *int) error
	BroadcastNotification(ctx context.Context, title, body, imageURL, baseURL string) error
	SendNotificationByPhone(ctx context.Context, phoneNumber, title, body, imageURL string) error
	MarkAsRead(ctx context.Context, userId string) error
	RegisterWithTxn(ctx context.Context, tx *gorm.DB, userId string, deviceToken string) error
	GetTokensByDate(ctx context.Context, dateStr string) (dtos.GetTokensByDateResponseDTO, error)
}

type notificationService struct {
	txnManager           *utils.TransactionManager
	notificationTypeRepo repository.NotificationTypeRepository
	notificationRepo     repository.UserNotificationRepository
	fcmDeviceTokenRepo   repository.FCMDeviceTokenRepository
	userRepo             repository.UserRepository
	publisher            *PublisherService
	gcsService           utils.GCSService
	firebaseClient       *vendors.FirebaseClient
}

// NewNotificationService creates a new notification service instance
func NewNotificationService(txnManager *utils.TransactionManager, notificationTypeRepo repository.NotificationTypeRepository, notificationRepo repository.UserNotificationRepository, fcmDeviceTokenRepo repository.FCMDeviceTokenRepository, userRepo repository.UserRepository, publisher *PublisherService, gcsService utils.GCSService, firebaseClient *vendors.FirebaseClient) NotificationService {
	return &notificationService{
		txnManager:           txnManager,
		notificationTypeRepo: notificationTypeRepo,
		notificationRepo:     notificationRepo,
		fcmDeviceTokenRepo:   fcmDeviceTokenRepo,
		userRepo:             userRepo,
		publisher:            publisher,
		gcsService:           gcsService,
		firebaseClient:       firebaseClient,
	}
}

func (s *notificationService) RegisterDevice(ctx context.Context, userId string, deviceToken string) error {

	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return err
	}

	now := time.Now()
	token, err := s.fcmDeviceTokenRepo.FindByField(ctx, tx, "user_id", userId)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return err
	}

	if token == nil {
		token = &entities.FCMDeviceToken{
			UserID:         userId,
			DeviceToken:    deviceToken,
			LastModifiedOn: &now,
			IsActive:       true,
			IsDeleted:      false,
		}
		err = s.fcmDeviceTokenRepo.Create(ctx, tx, token)
		if err != nil {
			s.txnManager.AbortTxn(tx)
			return err
		}
	} else {
		token.DeviceToken = deviceToken
		token.LastModifiedOn = &now
		err = s.fcmDeviceTokenRepo.Update(ctx, tx, token.ID, token)
		if err != nil {
			s.txnManager.AbortTxn(tx)
			return err
		}
	}

	s.txnManager.CommitTxn(tx)
	return nil
}

func (s *notificationService) RegisterWithTxn(ctx context.Context, tx *gorm.DB, userId string, deviceToken string) error {
	now := time.Now()
	token, err := s.fcmDeviceTokenRepo.FindByField(ctx, tx, "user_id", userId)
	if err != nil {
		return err
	}

	if token == nil {
		token = &entities.FCMDeviceToken{
			UserID:         userId,
			DeviceToken:    deviceToken,
			LastModifiedOn: &now,
			IsActive:       true,
			IsDeleted:      false,
		}
		err = s.fcmDeviceTokenRepo.Create(ctx, tx, token)
		if err != nil {
			return err
		}
	} else {
		token.DeviceToken = deviceToken
		token.LastModifiedOn = &now
		err = s.fcmDeviceTokenRepo.Update(ctx, tx, token.ID, token)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *notificationService) GetUserNotifications(ctx context.Context, userId string, filter dtos.NotificationFilter) ([]dtos.NotificationResponseDTO, int, error) {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return nil, 0, err
	}

	notifications, totalCount, err := s.notificationRepo.GetUserNotifications(ctx, tx, userId, filter)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, 0, err
	}

	notificationResponseDTOs := make([]dtos.NotificationResponseDTO, 0)
	for _, notification := range notifications {
		signedURL := ""
		if notification.NotificationType.IconURL != nil {
			signedURL, _ = s.gcsService.GetFileSignedURL(ctx, *notification.NotificationType.IconURL, 24*time.Hour)
		}
		notificationResponseDTOs = append(notificationResponseDTOs, dtos.NotificationResponseDTO{
			NotificationTitle: notification.NotificationTitle,
			NotificationText:  notification.NotificationText,
			IsNew:             notification.IsNew,
			IsRead:            notification.IsRead,
			IconURL:           &signedURL,
			LaunchDate:        notification.LaunchDate,
		})
	}

	s.txnManager.CommitTxn(tx)
	return notificationResponseDTOs, totalCount, nil
}

// GetNotificationCount returns the user's unread notification count.
//
// This used to also lazily fire the "complete your profile" nudge on every
// call, gated only by "has this been sent today" — with no check that the
// profile was actually still incomplete, so a fully-complete user would keep
// getting nagged forever. That nudge is now sent by a proper daily scheduled
// job (see SendDailyProfileCompletionReminders) which re-checks completeness
// at send time, so this stays a pure read.
func (s *notificationService) GetNotificationCount(ctx context.Context, userId string) (int, error) {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return 0, err
	}

	notificationCount, err := s.notificationRepo.CountWithConditions(ctx, tx, map[string]interface{}{"user_id": userId, "is_read": false, "is_new": true})
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return 0, err
	}

	s.txnManager.CommitTxn(tx)
	return int(notificationCount), nil
}

func (s *notificationService) MarkAsRead(ctx context.Context, userId string) error {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return err
	}

	err = s.notificationRepo.MarkAsRead(ctx, tx, userId)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return err
	}

	s.txnManager.CommitTxn(tx)
	return nil
}

func (s *notificationService) SendPushNotification(ctx context.Context, tx *gorm.DB, notificationType int, userId string, amount *int) error {
	// Only fetch a still-active token: an already-invalidated (dead) token
	// would otherwise keep getting fetched and re-queued for every future
	// notification to this user, repeating a send we already know will fail.
	token, err := s.fcmDeviceTokenRepo.FindOneWithConditions(ctx, tx, map[string]interface{}{
		"user_id":    userId,
		"is_active":  true,
		"is_deleted": false,
	})
	if err != nil {
		return err
	}

	notification, err := s.notificationTypeRepo.FindByField(ctx, tx, "id", notificationType)
	if err != nil {
		return err
	}

	if notification == nil {
		return fmt.Errorf("notification type not found with id: %d", notificationType)
	}

	if notificationType == constants.NOTIFICATION_WIN_COIN {
		if amount != nil {
			newTitle := utils.TitleWithAmount(notification.TitleValue, *amount)
			notification.TitleValue = newTitle
		}
	}

	now := time.Now()
	err = s.notificationRepo.Create(ctx, tx, &entities.UserNotification{
		UserID:             userId,
		NotificationTypeID: notification.ID,
		NotificationTitle:  notification.Name,
		NotificationText:   notification.TitleValue,
		LaunchDate:         &now,
		IsActive:           true,
		IsDeleted:          false,
		IsNew:              true,
		IsRead:             false,
		CreatedOn:          now,
		LastModifiedOn:     &now,
	})
	if err != nil {
		return err
	}

	var imageURL string
	if notification.IconURL != nil {
		imageURL = *notification.IconURL
	} else {
		imageURL = ""
	}

	pubsubCtx := context.Background()
	if token != nil {
		s.publisher.PublishMessage(pubsubCtx, constants.DEFAULT_TOPIC, userId, dtos.PubSubMessage{
			Token:    token.DeviceToken,
			Title:    notification.Name,
			Body:     notification.TitleValue,
			ImageURL: imageURL,
		})
	}

	return nil
}

func (s *notificationService) BroadcastNotification(ctx context.Context, title, body, imageURL, baseURL string) error {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	// Find all active FCM device tokens with non-empty device_token
	fcmTokens, err := s.fcmDeviceTokenRepo.FindWithConditions(ctx, tx, map[string]interface{}{
		"is_active":  true,
		"is_deleted": false,
	})
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return fmt.Errorf("failed to fetch FCM tokens: %w", err)
	}

	// Commit transaction before external operation (notification sending)
	s.txnManager.CommitTxn(tx)

	// Extract device tokens and filter out empty ones
	deviceTokens := make([]string, 0)
	for _, token := range fcmTokens {
		if token.DeviceToken != "" {
			deviceTokens = append(deviceTokens, token.DeviceToken)
		}
	}

	if len(deviceTokens) == 0 {
		return fmt.Errorf("no valid device tokens found")
	}

	// Send bulk notification (external operation, not part of transaction)
	response, err := s.firebaseClient.SendBulkMessage(
		ctx,
		deviceTokens,
		title,
		body,
		imageURL,
		map[string]string{
			"base_url": baseURL,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to send bulk notification: %w", err)
	}

	s.invalidateFailedTokens(ctx, response)

	logrus.Infof("Bulk notification sent: %d successful, %d failed", response.SuccessCount, response.FailureCount)

	return nil
}

// invalidateFailedTokens inspects a bulk-send response and marks any tokens
// that failed with a permanent/dead-token error as inactive.
func (s *notificationService) invalidateFailedTokens(ctx context.Context, response *vendors.BulkMessageResponse) {
	if response == nil {
		return
	}

	var tokensToInvalidate []string
	for i, resp := range response.Responses {
		if resp == nil || resp.Success || resp.Error == nil {
			continue
		}
		if !vendors.IsInvalidTokenError(resp.Error) {
			continue
		}
		if i < len(response.Tokens) && response.Tokens[i] != "" {
			tokensToInvalidate = append(tokensToInvalidate, response.Tokens[i])
		}
	}

	if len(tokensToInvalidate) == 0 {
		return
	}

	tx, err := s.txnManager.StartTxn()
	if err != nil {
		logrus.Errorf("Failed to start transaction to invalidate FCM tokens: %v", err)
		return
	}

	rowsAffected, err := s.fcmDeviceTokenRepo.BatchInvalidateByDeviceTokens(ctx, tx, tokensToInvalidate)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		logrus.Errorf("Failed to batch invalidate %d FCM tokens: %v", len(tokensToInvalidate), err)
		return
	}
	s.txnManager.CommitTxn(tx)
	logrus.Infof("Invalidated %d FCM tokens after permanent send failures", rowsAffected)
}

func (s *notificationService) SendNotificationByPhone(ctx context.Context, phoneNumber, title, body, imageURL string) error {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	// Find user by phone number
	users, err := s.userRepo.FindWithConditions(ctx, tx, map[string]interface{}{
		"phone_number": phoneNumber,
		"is_active":    true,
	})
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return fmt.Errorf("failed to find user by phone number: %w", err)
	}

	if len(users) == 0 {
		s.txnManager.AbortTxn(tx)
		return fmt.Errorf("user not found with phone number: %s", phoneNumber)
	}

	user := users[0]

	// Find FCM device tokens for this user
	fcmTokens, err := s.fcmDeviceTokenRepo.FindWithConditions(ctx, tx, map[string]interface{}{
		"user_id":    user.ID,
		"is_active":  true,
		"is_deleted": false,
	})
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return fmt.Errorf("failed to fetch FCM tokens for user: %w", err)
	}

	s.txnManager.CommitTxn(tx)

	// Extract device tokens and filter out empty ones
	deviceTokens := make([]string, 0)
	for _, token := range fcmTokens {
		if token.DeviceToken != "" {
			deviceTokens = append(deviceTokens, token.DeviceToken)
		}
	}

	if len(deviceTokens) == 0 {
		return fmt.Errorf("no valid device tokens found for user with phone number: %s", phoneNumber)
	}

	response, err := s.firebaseClient.SendBulkMessage(
		ctx,
		deviceTokens,
		title,
		body,
		imageURL,
		map[string]string{},
	)
	if err != nil {
		return fmt.Errorf("failed to send notification to user: %w", err)
	}

	s.invalidateFailedTokens(ctx, response)

	logrus.Infof("Notification sent to user %s (phone: %s): %d successful, %d failed",
		user.ID, utils.SanitizeForLog(phoneNumber), response.SuccessCount, response.FailureCount)

	return nil
}

func (s *notificationService) GetTokensByDate(ctx context.Context, dateStr string) (dtos.GetTokensByDateResponseDTO, error) {
	// Parse the date string (expected format: YYYY-MM-DD)
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return dtos.GetTokensByDateResponseDTO{}, fmt.Errorf("invalid date format, expected YYYY-MM-DD: %w", err)
	}

	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return dtos.GetTokensByDateResponseDTO{}, err
	}

	// Fetch tokens by date
	tokens, err := s.fcmDeviceTokenRepo.FindByDate(ctx, tx, date)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return dtos.GetTokensByDateResponseDTO{}, err
	}

	// Convert to response DTOs
	tokenDTOs := make([]dtos.FCMDeviceTokenResponseDTO, len(tokens))
	for i, token := range tokens {
		tokenDTOs[i] = dtos.FCMDeviceTokenResponseDTO{
			ID:             token.ID,
			UserID:         token.UserID,
			DeviceToken:    token.DeviceToken,
			IsActive:       token.IsActive,
			IsDeleted:      token.IsDeleted,
			CreatedOn:      token.CreatedOn,
			LastModifiedOn: token.LastModifiedOn,
		}
	}

	response := dtos.GetTokensByDateResponseDTO{
		Tokens: tokenDTOs,
		Count:  len(tokenDTOs),
		Date:   dateStr,
	}

	s.txnManager.CommitTxn(tx)
	return response, nil
}
