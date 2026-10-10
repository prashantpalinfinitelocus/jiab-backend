package repository

import (
	"context"
	"errors"
	"time"

	"github.com/Infinite-Locus-Product/coca_cola_backend/constants"
	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type OtpLogsRepository interface {
	GenericRepository[entities.OTPLogs]
	VerifyOTP(ctx context.Context, db *gorm.DB, mobileNumber string, otp string) (bool, error)
	UpdateByMobileNumber(ctx context.Context, db *gorm.DB, mobileNumber string) error
	CountRecentAttempts(ctx context.Context, db *gorm.DB, mobileNumber string, duration time.Duration) (int64, error)
	IncrementAttempts(ctx context.Context, db *gorm.DB, mobileNumber string) error
	CheckVerificationRateLimit(ctx context.Context, db *gorm.DB, mobileNumber string) error
}

type otpLogsRepository struct {
	*GormRepository[entities.OTPLogs]
}

func NewOTPLogsRepository() OtpLogsRepository {
	return &otpLogsRepository{
		GormRepository: NewGormRepository[entities.OTPLogs](),
	}
}

func (r *otpLogsRepository) VerifyOTP(ctx context.Context, db *gorm.DB, mobileNumber string, otp string) (bool, error) {
	var otpLog entities.OTPLogs

	cutoffTime := time.Now().Add(-time.Duration(constants.OTP_EXPIRATION_MINUTES) * time.Minute)

	if err := db.WithContext(ctx).Model(&entities.OTPLogs{}).
		Where("mobile_no = ? AND otp = ? AND is_active = ? AND is_deleted = ? AND created_on >= ?",
			mobileNumber, otp, true, false, cutoffTime).
		First(&otpLog).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logrus.Errorf("OTP not found or expired: %v", err)
			return false, errs.NewUnauthorizedError("Invalid OTP or OTP has expired. Please request a new OTP.", err)
		}
		logrus.Errorf("error verifying OTP: %v", err)
		return false, errs.NewInternalServerError("error verifying OTP", err)
	}
	return true, nil
}

func (r *otpLogsRepository) UpdateByMobileNumber(ctx context.Context, db *gorm.DB, mobileNumber string) error {
	if err := db.WithContext(ctx).Model(&entities.OTPLogs{}).Where("mobile_no = ? AND is_active = ? AND is_deleted = ?", mobileNumber, true, false).Update("is_active", false).Error; err != nil {
		logrus.Errorf("error updating OTP log: %v", err)
		return errs.NewInternalServerError("error updating OTP log", err)
	}
	return nil
}

func (r *otpLogsRepository) CountRecentAttempts(ctx context.Context, db *gorm.DB, mobileNumber string, duration time.Duration) (int64, error) {
	var count int64
	cutoffTime := time.Now().Add(-duration)

	result := db.WithContext(ctx).Model(&entities.OTPLogs{}).
		Where("mobile_no = ? AND created_on >= ? AND is_deleted = ?", mobileNumber, cutoffTime, false).
		Count(&count)

	if result.Error != nil {
		logrus.Errorf("error counting recent OTP attempts: %v", result.Error)
		return 0, errs.NewInternalServerError("error counting recent OTP attempts", result.Error)
	}

	return count, nil
}

func (r *otpLogsRepository) IncrementAttempts(ctx context.Context, db *gorm.DB, mobileNumber string) error {
	var otpLog entities.OTPLogs
	cutoffTime := time.Now().Add(-time.Duration(constants.OTP_EXPIRATION_MINUTES) * time.Minute)

	err := db.WithContext(ctx).Model(&entities.OTPLogs{}).
		Where("mobile_no = ? AND is_active = ? AND is_deleted = ? AND created_on >= ?", mobileNumber, true, false, cutoffTime).
		Order("created_on DESC").
		First(&otpLog).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logrus.Warnf("No active OTP found for mobile number: %s", mobileNumber)
			return errs.NewNotFoundError("No active OTP found", err)
		}
		logrus.Errorf("Error finding OTP log: %v", err)
		return errs.NewInternalServerError("Error finding OTP log", err)
	}

	logrus.Infof("Found OTP log with ID: %v and current attempts: %d", otpLog.ID, otpLog.Attempts)

	otpLog.Attempts += 1
	err = db.WithContext(ctx).Save(&otpLog).Error

	if err != nil {
		logrus.Errorf("Error incrementing OTP attempts: %v", err)
		return errs.NewInternalServerError("Error incrementing OTP attempts", err)
	}

	logrus.Infof("Successfully incremented attempts for OTP log ID: %v", otpLog.ID)
	return nil
}

func (r *otpLogsRepository) CheckVerificationRateLimit(ctx context.Context, db *gorm.DB, mobileNumber string) error {
	cutoffTime := time.Now().Add(-time.Duration(constants.VERIFY_OTP_RATE_LIMIT_DURATION_MINUTES) * time.Minute)

	var totalAttempts int64
	if err := db.WithContext(ctx).Model(&entities.OTPLogs{}).
		Where("mobile_no = ? AND created_on >= ? AND is_deleted = ?", mobileNumber, cutoffTime, false).
		Select("COALESCE(SUM(attempts), 0)").
		Scan(&totalAttempts).Error; err != nil {
		logrus.Errorf("error counting verification attempts: %v", err)
		return errs.NewInternalServerError("error counting verification attempts", err)
	}

	if totalAttempts >= int64(constants.VERIFY_OTP_MAX_ATTEMPTS) {
		return errs.NewTooManyRequestsError(
			"Too many verification attempts. Please try again after 5 minutes.",
			nil,
		)
	}

	return nil
}
