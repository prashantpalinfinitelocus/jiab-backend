package utils

import (
	"errors"
	"time"

	"github.com/Infinite-Locus-Product/coca_cola_backend/constants"
	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func Migrate(db *gorm.DB) {
	logrus.Info("Running migrations......")

	// Enable UUID extension first
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"").Error; err != nil {
		logrus.Errorf("Failed to create UUID extension: %v", err)
		return
	}
	logrus.Info("UUID extension enabled successfully")

	// Run auto migrations
	if err := db.AutoMigrate(&entities.QuestionMaster{}, &entities.SSOToken{}, &entities.ContestBroadcastLog{}, &entities.DSRRequest{}, &entities.MemeTemplate{}); err != nil {
		logrus.Errorf("Failed to run migrations: %v", err)
		return
	}

	seedNotificationTypes(db)

	logrus.Info("Migrations completed successfully........")
}

// seedNotificationTypes idempotently ensures notification_types rows exist for
// notification type IDs introduced by application code (SendPushNotification
// hard-fails a whole transaction if the type ID isn't a row in this table).
func seedNotificationTypes(db *gorm.DB) {
	// created_by is a NOT NULL uuid column FK'd to users(id), so it must be a
	// real user's ID — there's no sentinel "system" value that satisfies that
	// FK. Any existing user works; this is a reference-data row, not an
	// attribution of authorship.
	var systemUserID string
	if err := db.Table("users").Limit(1).Pluck("id", &systemUserID).Error; err != nil || systemUserID == "" {
		logrus.Errorf("Failed to find a user to attribute seeded notification types to, skipping seed: %v", err)
		return
	}

	seedRows := []entities.NotificationType{
		{
			ID:               constants.NOTIFICATION_COMPLETE_PROFILE,
			Name:             "Complete Your Profile",
			TitleValue:       "Finish setting up your profile to unlock more Comic Coins and rewards!",
			LanguageMasterID: constants.DEFAULT_LANGUAGE_ID,
			IsBroadcast:      true,
			IsActive:         true,
			IsDeleted:        false,
			CreatedBy:        systemUserID,
			CreatedOn:        time.Now(),
		},
		{
			ID:               constants.NOTIFICATION_CONTEST_ANNOUNCEMENT,
			Name:             "Contest Announcement",
			TitleValue:       "A new contest just went live — check it out!",
			LanguageMasterID: constants.DEFAULT_LANGUAGE_ID,
			IsBroadcast:      true,
			IsActive:         true,
			IsDeleted:        false,
			CreatedBy:        systemUserID,
			CreatedOn:        time.Now(),
		},
	}

	for _, row := range seedRows {
		var existing entities.NotificationType
		err := db.Where("id = ?", row.ID).First(&existing).Error
		if err == nil {
			continue // already seeded
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			logrus.Errorf("Failed to check notification type %d: %v", row.ID, err)
			continue
		}
		if err := db.Create(&row).Error; err != nil {
			logrus.Errorf("Failed to seed notification type %d: %v", row.ID, err)
		} else {
			logrus.Infof("Seeded notification type %d (%s)", row.ID, row.Name)
		}
	}
}
