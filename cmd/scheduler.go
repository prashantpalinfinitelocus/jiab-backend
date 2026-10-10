package cmd

import (
	"context"
	"database/sql"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Infinite-Locus-Product/coca_cola_backend/constants"
	"github.com/Infinite-Locus-Product/coca_cola_backend/services"
)

// dailyNotificationSchedulerLockKey is an arbitrary fixed key for a Postgres
// advisory lock. The main-backend Cloud Run service can run multiple
// instances concurrently (min/max instances > 1), and each instance runs its
// own copy of this in-process scheduler; without a lock, every instance would
// independently fire the daily job at midnight and send duplicate
// notifications. The lock ensures only one instance actually runs the job on
// any given day, even if all instances wake up around the same time.
const dailyNotificationSchedulerLockKey = 583920147

// dailyNotificationScheduleHourIST is the hour (IST, 24h) at which the daily
// profile-completion reminder job runs.
const dailyNotificationScheduleHourIST = 10

// StartDailyNotificationScheduler runs SendDailyProfileCompletionReminders
// once per day at 10:00 IST, for as long as ctx is alive. It never blocks the
// caller — call it in a goroutine.
func StartDailyNotificationScheduler(ctx context.Context, db *gorm.DB, campaignService services.CampaignService) {
	for {
		wait := durationUntilNextISTScheduleTime()
		logrus.Infof("Daily notification scheduler: next run in %s (at next %02d:00 IST)", wait, dailyNotificationScheduleHourIST)

		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
			runDailyNotificationJob(ctx, db, campaignService)
		}
	}
}

func durationUntilNextISTScheduleTime() time.Duration {
	loc, err := time.LoadLocation(constants.IST_TIMEZONE)
	if err != nil {
		logrus.Errorf("Failed to load IST timezone, falling back to UTC: %v", err)
		loc = time.UTC
	}
	now := time.Now().In(loc)
	next := time.Date(now.Year(), now.Month(), now.Day(), dailyNotificationScheduleHourIST, 0, 0, 0, loc)
	if !next.After(now) {
		next = next.AddDate(0, 0, 1)
	}
	return next.Sub(now)
}

func runDailyNotificationJob(ctx context.Context, db *gorm.DB, campaignService services.CampaignService) {
	sqlDB, err := db.DB()
	if err != nil {
		logrus.Errorf("Daily notification scheduler: failed to get underlying sql.DB: %v", err)
		return
	}

	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		logrus.Errorf("Daily notification scheduler: failed to acquire connection: %v", err)
		return
	}
	defer conn.Close()

	var acquired bool
	if err := conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", dailyNotificationSchedulerLockKey).Scan(&acquired); err != nil {
		logrus.Errorf("Daily notification scheduler: failed to acquire advisory lock: %v", err)
		return
	}
	if !acquired {
		logrus.Info("Daily notification scheduler: another instance already holds the lock, skipping this run")
		return
	}
	defer releaseAdvisoryLock(ctx, conn)

	logrus.Info("Daily notification scheduler: starting profile-completion reminder run")
	result, err := campaignService.SendDailyProfileCompletionReminders(ctx)
	if err != nil {
		logrus.Errorf("Daily notification scheduler: run failed: %v", err)
		return
	}
	logrus.Infof("Daily notification scheduler: run complete: eligible=%d sent=%d skipped=%d failed=%d",
		result.Eligible, result.Sent, result.Skipped, result.Failed)
}

func releaseAdvisoryLock(ctx context.Context, conn *sql.Conn) {
	if _, err := conn.ExecContext(ctx, "SELECT pg_advisory_unlock($1)", dailyNotificationSchedulerLockKey); err != nil {
		logrus.Errorf("Daily notification scheduler: failed to release advisory lock: %v", err)
	}
}
