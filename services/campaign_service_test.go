package services

import (
	"context"
	"testing"
	"time"

	"github.com/Infinite-Locus-Product/coca_cola_backend/constants"
	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	"github.com/Infinite-Locus-Product/coca_cola_backend/repository"
	"github.com/Infinite-Locus-Product/coca_cola_backend/utils"
	"github.com/Infinite-Locus-Product/coca_cola_backend/vendors"
)

// TestCampaignServicePrimitives verifies the batching/cap/eligibility/dedupe
// primitives against a real local Postgres DB, WITHOUT ever calling the
// "page through all eligible users" entry points (SendDailyProfileCompletionReminders,
// SendProfileCompletionCampaign, SendContestAnnouncement) directly — the
// local dev DB this runs against has 290k+ real incomplete-profile users, so
// invoking those methods here would send real notifications to real users.
// Testing the underlying building blocks directly (bulk repo lookups,
// sendSingle, sendConcurrently, contest dedupe) verifies the exact same logic
// without touching that population.
func TestCampaignServicePrimitives(t *testing.T) {
	db := vendors.InitDatabase()
	utils.Migrate(db)

	txnManager := utils.NewTransactionManager(db)
	userRepo := repository.NewUserRepository()
	userNotificationRepo := repository.NewUserNotificationRepository()
	notificationTypeRepo := repository.NewNotificationTypeRepository()
	fcmDeviceTokenRepo := repository.NewFCMDeviceTokenRepository()
	contestBroadcastRepo := repository.NewContestBroadcastLogRepository()
	userWalletRepo := repository.NewUserWalletRepository()

	notificationService := NewNotificationService(txnManager, notificationTypeRepo, userNotificationRepo, fcmDeviceTokenRepo, userRepo, nil, nil, nil)
	comicCoinService := NewComicCoinService(txnManager, userWalletRepo, notificationService, nil, nil, nil, nil, nil, nil, repository.WAAssetActivityRepository{})

	cs := &campaignService{
		txnManager:           txnManager,
		userRepo:             userRepo,
		userNotificationRepo: userNotificationRepo,
		contestBroadcastRepo: contestBroadcastRepo,
		notificationService:  notificationService,
		comicCoinService:     comicCoinService,
	}

	ctx := context.Background()

	// testPlatformMarker is a sentinel PlatformMasterID value exclusive to
	// this test's own rows (it's a legacy migration-only field with no active
	// business meaning — see entities/user.go). Scoping FindPageByConditions
	// queries to this marker lets us test real pagination (multiple pages,
	// afterID continuation, ordering) using only rows this test created,
	// instead of ever matching the real ~290k-row population.
	const testPlatformMarker = 999999

	createUser := func(t *testing.T, phone string, profileComplete bool) *entities.User {
		t.Helper()
		marker := testPlatformMarker
		user := &entities.User{
			PhoneNumber:           phone,
			IsActive:              true,
			IsProfileComplete:     profileComplete,
			ProfilePercentage:     50,
			RegisteredOn:          time.Now(),
			LastLanguageSelection: 1,
			PlatformMasterID:      &marker,
		}
		if err := userRepo.Create(ctx, db, user); err != nil {
			t.Fatalf("failed to create test user %s: %v", phone, err)
		}
		t.Cleanup(func() {
			db.Exec("DELETE FROM user_notification WHERE user_id = ?", user.ID)
			db.Exec("DELETE FROM user_wallet WHERE user_id = ?", user.ID)
			db.Exec("DELETE FROM users WHERE id = ?", user.ID)
		})

		tx, err := txnManager.StartTxn()
		if err != nil {
			t.Fatalf("failed to start txn for wallet creation: %v", err)
		}
		if _, err := comicCoinService.CreateUserWallet(ctx, tx, user.ID); err != nil {
			txnManager.AbortTxn(tx)
			t.Fatalf("failed to create wallet for %s: %v", phone, err)
		}
		txnManager.CommitTxn(tx)
		return user
	}

	t.Run("FindPageByConditions pages correctly via afterID keyset continuation", func(t *testing.T) {
		userA := createUser(t, "9992220001", false)
		userB := createUser(t, "9992220002", false)
		userComplete := createUser(t, "9992220008", true) // must be excluded by the condition filter

		tx, err := txnManager.StartTxn()
		if err != nil {
			t.Fatalf("failed to start txn: %v", err)
		}
		defer txnManager.CommitTxn(tx)

		conditions := map[string]interface{}{
			"platform_master_id":  testPlatformMarker,
			"is_profile_complete": false,
		}

		// With limit=1, paging via afterID should visit exactly userA and
		// userB (in some id order), never userComplete, and never repeat.
		seen := make(map[string]bool)
		afterID := ""
		for i := 0; i < 5; i++ { // hard cap so a pagination bug can't loop forever
			page, err := userRepo.FindPageByConditions(ctx, tx, conditions, afterID, 1)
			if err != nil {
				t.Fatalf("FindPageByConditions failed: %v", err)
			}
			if len(page) == 0 {
				break
			}
			if len(page) != 1 {
				t.Fatalf("expected at most 1 row per page with limit=1, got %d", len(page))
			}
			id := page[0].ID
			if seen[id] {
				t.Fatalf("afterID pagination revisited id %s — keyset continuation is broken", id)
			}
			seen[id] = true
			afterID = id
		}

		if !seen[userA.ID] || !seen[userB.ID] {
			t.Fatalf("expected pagination to visit both incomplete-profile test users, saw: %v", seen)
		}
		if seen[userComplete.ID] {
			t.Fatal("expected the complete-profile test user to be excluded by the condition filter")
		}
		if len(seen) != 2 {
			t.Fatalf("expected exactly 2 rows visited (userA, userB), saw %d", len(seen))
		}
	})

	t.Run("FindByIDs bulk-fetches regardless of profile-complete status", func(t *testing.T) {
		userA := createUser(t, "9992220009", false)
		userB := createUser(t, "9992220010", true)

		tx, err := txnManager.StartTxn()
		if err != nil {
			t.Fatalf("failed to start txn: %v", err)
		}
		defer txnManager.CommitTxn(tx)

		byIDs, err := userRepo.FindByIDs(ctx, tx, []string{userA.ID, userB.ID})
		if err != nil {
			t.Fatalf("FindByIDs failed: %v", err)
		}
		if len(byIDs) != 2 {
			t.Fatalf("expected FindByIDs to return both users regardless of profile-complete status, got %d", len(byIDs))
		}
	})

	t.Run("FindUserIDsNotifiedWithinWindow bulk-checks a whole batch in one query", func(t *testing.T) {
		user := createUser(t, "9992220003", false)

		tx, err := txnManager.StartTxn()
		if err != nil {
			t.Fatalf("failed to start txn: %v", err)
		}
		notified, err := userNotificationRepo.FindUserIDsNotifiedWithinWindow(ctx, tx, []string{user.ID}, constants.NOTIFICATION_USER_INCOMPLETE_PROFILE, time.Hour)
		if err != nil {
			txnManager.AbortTxn(tx)
			t.Fatalf("lookup before send failed: %v", err)
		}
		if notified[user.ID] {
			t.Fatal("expected no prior notification before any send")
		}
		txnManager.CommitTxn(tx)

		if err := cs.sendSingle(ctx, user.ID, constants.NOTIFICATION_USER_INCOMPLETE_PROFILE, nil); err != nil {
			t.Fatalf("sendSingle failed: %v", err)
		}

		tx2, err := txnManager.StartTxn()
		if err != nil {
			t.Fatalf("failed to start txn: %v", err)
		}
		defer txnManager.CommitTxn(tx2)
		notifiedAfter, err := userNotificationRepo.FindUserIDsNotifiedWithinWindow(ctx, tx2, []string{user.ID}, constants.NOTIFICATION_USER_INCOMPLETE_PROFILE, time.Hour)
		if err != nil {
			t.Fatalf("lookup after send failed: %v", err)
		}
		if !notifiedAfter[user.ID] {
			t.Fatal("expected the user to show up as notified within the window after sendSingle")
		}

		// A zero-width window means "notified within the window" is
		// trivially false regardless of what was just sent.
		notifiedZeroWindow, err := userNotificationRepo.FindUserIDsNotifiedWithinWindow(ctx, tx2, []string{user.ID}, constants.NOTIFICATION_USER_INCOMPLETE_PROFILE, 0)
		if err != nil {
			t.Fatalf("zero-window lookup failed: %v", err)
		}
		if notifiedZeroWindow[user.ID] {
			t.Fatal("expected a zero-width window to never report as notified")
		}
	})

	t.Run("sendSingle with a reward credits the user's coin balance", func(t *testing.T) {
		user := createUser(t, "9992220004", false)

		walletBefore, err := comicCoinService.GetUserWalletWithoutTx(ctx, user.ID)
		if err != nil {
			t.Fatalf("failed to read wallet before send: %v", err)
		}

		reward := constants.REWARD_PROFILE_COMPLETION
		if err := cs.sendSingle(ctx, user.ID, constants.NOTIFICATION_COMPLETE_PROFILE, &reward); err != nil {
			t.Fatalf("sendSingle with reward failed: %v", err)
		}

		walletAfter, err := comicCoinService.GetUserWalletWithoutTx(ctx, user.ID)
		if err != nil {
			t.Fatalf("failed to read wallet after send: %v", err)
		}
		if walletAfter.CurrentBalance != walletBefore.CurrentBalance+reward {
			t.Fatalf("expected balance %d + %d = %d, got %d",
				walletBefore.CurrentBalance, reward, walletBefore.CurrentBalance+reward, walletAfter.CurrentBalance)
		}
	})

	t.Run("NOTIFICATION_CONTEST_ANNOUNCEMENT type is seeded and sendable", func(t *testing.T) {
		user := createUser(t, "9992220005", false)

		tx, err := txnManager.StartTxn()
		if err != nil {
			t.Fatalf("failed to start txn: %v", err)
		}
		if err := notificationService.SendPushNotification(ctx, tx, constants.NOTIFICATION_CONTEST_ANNOUNCEMENT, user.ID, nil); err != nil {
			txnManager.AbortTxn(tx)
			t.Fatalf("SendPushNotification for NOTIFICATION_CONTEST_ANNOUNCEMENT failed (is the migration seed missing?): %v", err)
		}
		txnManager.CommitTxn(tx)
	})

	t.Run("sendConcurrently processes a batch of users and reports sent/failed counts", func(t *testing.T) {
		userA := createUser(t, "9992220006", false)
		userB := createUser(t, "9992220007", false)

		sent, failed := cs.sendConcurrently(ctx, []string{userA.ID, userB.ID}, func(ctx context.Context, userID string) error {
			return cs.sendSingle(ctx, userID, constants.NOTIFICATION_USER_INCOMPLETE_PROFILE, nil)
		})
		if sent != 2 || failed != 0 {
			t.Fatalf("expected sent=2 failed=0, got sent=%d failed=%d", sent, failed)
		}

		// An unseeded notification type ID makes SendPushNotification fail
		// (it looks up the type row and errors if missing), verifying
		// failures are counted by sendConcurrently, not silently dropped.
		const nonexistentNotificationType = 999999
		sent2, failed2 := cs.sendConcurrently(ctx, []string{userA.ID}, func(ctx context.Context, userID string) error {
			return cs.sendSingle(ctx, userID, nonexistentNotificationType, nil)
		})
		if sent2 != 0 || failed2 != 1 {
			t.Fatalf("expected sent=0 failed=1 for a nonexistent notification type, got sent=%d failed=%d", sent2, failed2)
		}
	})

	t.Run("contest broadcast log dedupes by (contest_id, announcement_type)", func(t *testing.T) {
		contestID := "smoke-test-contest-primitive-1"
		t.Cleanup(func() {
			db.Exec("DELETE FROM contest_broadcast_log WHERE contest_id = ?", contestID)
		})

		existing, err := contestBroadcastRepo.FindByContestAndType(ctx, db, contestID, constants.CONTEST_ANNOUNCEMENT_TYPE_LAUNCH)
		if err != nil {
			t.Fatalf("lookup before create failed: %v", err)
		}
		if existing != nil {
			t.Fatal("expected no existing log row before creating one")
		}

		if err := contestBroadcastRepo.Create(ctx, db, &entities.ContestBroadcastLog{
			ContestID:        contestID,
			AnnouncementType: constants.CONTEST_ANNOUNCEMENT_TYPE_LAUNCH,
			RecipientCount:   42,
			CreatedOn:        time.Now(),
		}); err != nil {
			t.Fatalf("failed to create log row: %v", err)
		}

		found, err := contestBroadcastRepo.FindByContestAndType(ctx, db, contestID, constants.CONTEST_ANNOUNCEMENT_TYPE_LAUNCH)
		if err != nil {
			t.Fatalf("lookup after create failed: %v", err)
		}
		if found == nil {
			t.Fatal("expected to find the log row that was just created")
		}

		// A different announcement type for the same contest is a distinct
		// (contest_id, type) pair and must not be found.
		reminderLog, err := contestBroadcastRepo.FindByContestAndType(ctx, db, contestID, constants.CONTEST_ANNOUNCEMENT_TYPE_REMINDER)
		if err != nil {
			t.Fatalf("lookup for reminder type failed: %v", err)
		}
		if reminderLog != nil {
			t.Fatal("expected no log row for the reminder announcement type yet")
		}
	})
}
