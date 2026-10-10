package services

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Infinite-Locus-Product/coca_cola_backend/constants"
	"github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"github.com/Infinite-Locus-Product/coca_cola_backend/repository"
	"github.com/Infinite-Locus-Product/coca_cola_backend/utils"
)

// CampaignService owns scheduled/ops-triggered notification campaigns that
// need both notification delivery and coin-wallet updates. It sits above
// NotificationService and ComicCoinService (which itself depends on
// NotificationService) to avoid a circular dependency between the two.
//
// All three campaigns here can run against the entire active user base
// (hundreds of thousands of rows in production), so they page through users
// with keyset pagination (id > lastSeenID) instead of loading everyone into
// memory at once, batch their "already sent"/eligibility lookups into one
// query per batch instead of one query per user, and send within a batch
// using a bounded worker pool instead of strictly sequentially.
type CampaignService interface {
	// SendDailyProfileCompletionReminders sends the "complete your profile"
	// nudge (NOTIFICATION_USER_INCOMPLETE_PROFILE) to every active user whose
	// profile is still incomplete, at most once per PROFILE_COMPLETION_REMINDER_COOLDOWN
	// window. Intended to be run once per day by the scheduler.
	SendDailyProfileCompletionReminders(ctx context.Context) (dtos.NotificationSendResult, error)

	// SendProfileCompletionCampaign is the ops-triggered bulk campaign
	// (NOTIFICATION_COMPLETE_PROFILE) with a coin reward attached, capped at
	// once per PROFILE_COMPLETION_CAMPAIGN_COOLDOWN window per user, with
	// eligibility (profile still incomplete) re-checked at send time.
	SendProfileCompletionCampaign(ctx context.Context, req dtos.ProfileCompletionCampaignDTO) (dtos.NotificationSendResult, error)

	// SendContestAnnouncement broadcasts a contest announcement
	// (NOTIFICATION_CONTEST_ANNOUNCEMENT) to all active users, deduplicated
	// by (contest_id, announcement_type) so the same contest never gets the
	// same announcement broadcast twice.
	SendContestAnnouncement(ctx context.Context, req dtos.ContestAnnouncementDTO) (dtos.NotificationSendResult, error)
}

type campaignService struct {
	txnManager           *utils.TransactionManager
	userRepo             repository.UserRepository
	userNotificationRepo repository.UserNotificationRepository
	contestBroadcastRepo repository.ContestBroadcastLogRepository
	notificationService  NotificationService
	comicCoinService     ComicCoinService
}

func NewCampaignService(
	txnManager *utils.TransactionManager,
	userRepo repository.UserRepository,
	userNotificationRepo repository.UserNotificationRepository,
	contestBroadcastRepo repository.ContestBroadcastLogRepository,
	notificationService NotificationService,
	comicCoinService ComicCoinService,
) CampaignService {
	return &campaignService{
		txnManager:           txnManager,
		userRepo:             userRepo,
		userNotificationRepo: userNotificationRepo,
		contestBroadcastRepo: contestBroadcastRepo,
		notificationService:  notificationService,
		comicCoinService:     comicCoinService,
	}
}

func (s *campaignService) SendDailyProfileCompletionReminders(ctx context.Context) (dtos.NotificationSendResult, error) {
	result, err := s.runIncompleteProfileCampaign(ctx, constants.NOTIFICATION_USER_INCOMPLETE_PROFILE, constants.PROFILE_COMPLETION_REMINDER_COOLDOWN, nil)
	logrus.Infof("Daily profile-completion reminder run complete: eligible=%d sent=%d skipped=%d failed=%d",
		result.Eligible, result.Sent, result.Skipped, result.Failed)
	return result, err
}

func (s *campaignService) SendProfileCompletionCampaign(ctx context.Context, req dtos.ProfileCompletionCampaignDTO) (dtos.NotificationSendResult, error) {
	campaignReward := constants.REWARD_PROFILE_COMPLETION
	result, err := s.runIncompleteProfileCampaign(ctx, constants.NOTIFICATION_COMPLETE_PROFILE, constants.PROFILE_COMPLETION_CAMPAIGN_COOLDOWN, &campaignReward)
	logrus.Infof("Profile-completion campaign run complete: eligible=%d sent=%d skipped=%d failed=%d",
		result.Eligible, result.Sent, result.Skipped, result.Failed)
	return result, err
}

// runIncompleteProfileCampaign pages through active, incomplete-profile users
// in batches of CAMPAIGN_BATCH_SIZE (keyset pagination on id, cheap at any
// scale unlike OFFSET/LIMIT) and processes each batch via processIncompleteProfileBatch.
func (s *campaignService) runIncompleteProfileCampaign(ctx context.Context, notificationType int, window time.Duration, reward *int) (dtos.NotificationSendResult, error) {
	result := dtos.NotificationSendResult{}
	afterID := ""

	for {
		if err := ctx.Err(); err != nil {
			return result, err
		}

		tx, err := s.txnManager.StartTxn()
		if err != nil {
			return result, err
		}
		batch, err := s.userRepo.FindPageByConditions(ctx, tx, map[string]interface{}{
			"is_active":           true,
			"is_profile_complete": false,
		}, afterID, constants.CAMPAIGN_BATCH_SIZE)
		if err != nil {
			s.txnManager.AbortTxn(tx)
			return result, err
		}
		s.txnManager.CommitTxn(tx)

		if len(batch) == 0 {
			break
		}
		afterID = batch[len(batch)-1].ID
		result.Eligible += len(batch)

		sent, skipped, failed := s.processIncompleteProfileBatch(ctx, batch, notificationType, window, reward)
		result.Sent += sent
		result.Skipped += skipped
		result.Failed += failed

		logrus.Infof("Campaign batch complete (notification_type=%d): batch_size=%d sent=%d skipped=%d failed=%d running_sent=%d",
			notificationType, len(batch), sent, skipped, failed, result.Sent)

		if len(batch) < constants.CAMPAIGN_BATCH_SIZE {
			break
		}
	}

	return result, nil
}

// processIncompleteProfileBatch bulk-checks which users in the batch already
// received notificationType within window and which are still eligible
// (profile still incomplete, re-checked now rather than trusting the
// original page query), then sends to the remainder with bounded concurrency.
func (s *campaignService) processIncompleteProfileBatch(ctx context.Context, batch []entities.User, notificationType int, window time.Duration, reward *int) (sent, skipped, failed int) {
	ids := make([]string, len(batch))
	for i, u := range batch {
		ids[i] = u.ID
	}

	tx, err := s.txnManager.StartTxn()
	if err != nil {
		logrus.Errorf("Failed to start txn for batch lookups: %v", err)
		return 0, 0, len(batch)
	}

	alreadySent, err := s.userNotificationRepo.FindUserIDsNotifiedWithinWindow(ctx, tx, ids, notificationType, window)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		logrus.Errorf("Failed to bulk-check already-sent users: %v", err)
		return 0, 0, len(batch)
	}

	currentUsers, err := s.userRepo.FindByIDs(ctx, tx, ids)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		logrus.Errorf("Failed to bulk re-verify batch eligibility: %v", err)
		return 0, 0, len(batch)
	}
	s.txnManager.CommitTxn(tx)

	stillIncomplete := make(map[string]bool, len(currentUsers))
	for _, u := range currentUsers {
		if u.IsActive && !u.IsProfileComplete {
			stillIncomplete[u.ID] = true
		}
	}

	var toSend []string
	for _, id := range ids {
		if alreadySent[id] {
			skipped++
			continue
		}
		if !stillIncomplete[id] {
			skipped++
			continue
		}
		toSend = append(toSend, id)
	}

	sentCount, failedCount := s.sendConcurrently(ctx, toSend, func(ctx context.Context, userID string) error {
		return s.sendSingle(ctx, userID, notificationType, reward)
	})
	return sentCount, skipped, failedCount
}

// sendSingle credits reward (if non-nil) and sends notificationType to
// userID, in one transaction.
func (s *campaignService) sendSingle(ctx context.Context, userID string, notificationType int, reward *int) error {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return err
	}

	if reward != nil {
		if _, err := s.comicCoinService.UpdateBalance(ctx, tx, userID, *reward); err != nil {
			s.txnManager.AbortTxn(tx)
			return err
		}
	}

	if err := s.notificationService.SendPushNotification(ctx, tx, notificationType, userID, reward); err != nil {
		s.txnManager.AbortTxn(tx)
		return err
	}

	s.txnManager.CommitTxn(tx)
	return nil
}

// sendConcurrently runs work(ctx, userID) for every id in userIDs using up to
// CAMPAIGN_SEND_CONCURRENCY workers, and returns how many succeeded/failed.
// Each call runs in its own DB transaction (started inside work), so this is
// safe to parallelize — the connection pool (100 max open conns) has ample
// headroom for this concurrency level.
func (s *campaignService) sendConcurrently(ctx context.Context, userIDs []string, work func(ctx context.Context, userID string) error) (sent, failed int) {
	if len(userIDs) == 0 {
		return 0, 0
	}

	jobs := make(chan string)
	var wg sync.WaitGroup
	var mu sync.Mutex

	worker := func() {
		defer wg.Done()
		for userID := range jobs {
			if err := work(ctx, userID); err != nil {
				mu.Lock()
				failed++
				mu.Unlock()
				logrus.Errorf("Campaign send failed for user %s: %v", userID, err)
				continue
			}
			mu.Lock()
			sent++
			mu.Unlock()
		}
	}

	workerCount := constants.CAMPAIGN_SEND_CONCURRENCY
	if workerCount > len(userIDs) {
		workerCount = len(userIDs)
	}
	wg.Add(workerCount)
	for i := 0; i < workerCount; i++ {
		go worker()
	}

	for _, id := range userIDs {
		jobs <- id
	}
	close(jobs)
	wg.Wait()

	return sent, failed
}

func (s *campaignService) SendContestAnnouncement(ctx context.Context, req dtos.ContestAnnouncementDTO) (dtos.NotificationSendResult, error) {
	result := dtos.NotificationSendResult{}

	if req.AnnouncementType != constants.CONTEST_ANNOUNCEMENT_TYPE_LAUNCH && req.AnnouncementType != constants.CONTEST_ANNOUNCEMENT_TYPE_REMINDER {
		return result, errs.NewBadRequestError("announcement_type must be 'launch' or 'reminder'", nil)
	}

	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return result, err
	}
	existing, err := s.contestBroadcastRepo.FindByContestAndType(ctx, tx, req.ContestID, req.AnnouncementType)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return result, err
	}
	if existing != nil {
		s.txnManager.AbortTxn(tx)
		return result, errs.NewConflictError("this contest has already had a "+req.AnnouncementType+" announcement broadcast", nil)
	}
	s.txnManager.CommitTxn(tx)

	afterID := ""
	for {
		if err := ctx.Err(); err != nil {
			return result, err
		}

		pageTx, err := s.txnManager.StartTxn()
		if err != nil {
			return result, err
		}
		batch, err := s.userRepo.FindPageByConditions(ctx, pageTx, map[string]interface{}{
			"is_active": true,
		}, afterID, constants.CAMPAIGN_BATCH_SIZE)
		if err != nil {
			s.txnManager.AbortTxn(pageTx)
			return result, err
		}
		s.txnManager.CommitTxn(pageTx)

		if len(batch) == 0 {
			break
		}
		afterID = batch[len(batch)-1].ID
		result.Eligible += len(batch)

		ids := make([]string, len(batch))
		for i, u := range batch {
			ids[i] = u.ID
		}
		sent, failed := s.sendConcurrently(ctx, ids, func(ctx context.Context, userID string) error {
			sendTx, err := s.txnManager.StartTxn()
			if err != nil {
				return err
			}
			if err := s.notificationService.SendPushNotification(ctx, sendTx, constants.NOTIFICATION_CONTEST_ANNOUNCEMENT, userID, nil); err != nil {
				s.txnManager.AbortTxn(sendTx)
				return err
			}
			s.txnManager.CommitTxn(sendTx)
			return nil
		})
		result.Sent += sent
		result.Failed += failed

		logrus.Infof("Contest announcement batch complete (%s, %s): batch_size=%d sent=%d failed=%d running_sent=%d",
			utils.SanitizeForLog(req.ContestID), utils.SanitizeForLog(req.AnnouncementType), len(batch), sent, failed, result.Sent)

		if len(batch) < constants.CAMPAIGN_BATCH_SIZE {
			break
		}
	}

	logTx, err := s.txnManager.StartTxn()
	if err != nil {
		return result, err
	}
	if err := s.contestBroadcastRepo.Create(ctx, logTx, &entities.ContestBroadcastLog{
		ContestID:        req.ContestID,
		AnnouncementType: req.AnnouncementType,
		RecipientCount:   result.Sent,
		CreatedOn:        time.Now(),
	}); err != nil {
		s.txnManager.AbortTxn(logTx)
		return result, err
	}
	s.txnManager.CommitTxn(logTx)

	logrus.Infof("Contest announcement (%s, %s) complete: eligible=%d sent=%d failed=%d",
		utils.SanitizeForLog(req.ContestID), utils.SanitizeForLog(req.AnnouncementType), result.Eligible, result.Sent, result.Failed)
	return result, nil
}
