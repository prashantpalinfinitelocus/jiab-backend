package repository

import (
	"context"
	"errors"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
)

type UserReferralRepository interface {
	GenericRepository[entities.UserReferral]
	GetUserReferrals(ctx context.Context, db *gorm.DB, userID string, filter dtos.ReferralFilter) ([]entities.UserReferral, int64, error)
	GetReferralByCode(ctx context.Context, db *gorm.DB, code string) (*entities.UserReferral, error)
	UpdateReferralStatus(ctx context.Context, db *gorm.DB, id int64, isAccept bool) error
	UpdateResendStatus(ctx context.Context, db *gorm.DB, id int64) error
	CheckActiveReferral(ctx context.Context, db *gorm.DB, userID string, referTo string) (bool, error)
	ValidateReferralCode(ctx context.Context, db *gorm.DB, referTo string, referralCode string) (*entities.UserReferral, error)
}

type userReferralRepository struct {
	*GormRepository[entities.UserReferral]
}

// ReferralWithUser is a helper struct to scan SQL results
type ReferralWithUser struct {
	entities.UserReferral
	RefereeName        *string `gorm:"column:referee_name"`
	RefereePhoneNumber string  `gorm:"column:referee_phone_number"`
	RefereeEmail       *string `gorm:"column:referee_email"`
}

func NewUserReferralRepository() UserReferralRepository {
	return &userReferralRepository{
		GormRepository: NewGormRepository[entities.UserReferral](),
	}
}

func (r *userReferralRepository) GetUserReferrals(ctx context.Context, db *gorm.DB, userID string, filter dtos.ReferralFilter) ([]entities.UserReferral, int64, error) {
	var referrals []entities.UserReferral
	var total int64

	totalQuery := db.WithContext(ctx).Model(&entities.UserReferral{}).Where("user_id = ? AND is_deleted = ?", userID, false)

	if filter.Status != nil && *filter.Status != "" {
		switch *filter.Status {
		case "Accepted":
			totalQuery = totalQuery.Where("is_accept = ?", true)
		case "Pending":
			totalQuery = totalQuery.Where("is_accept = ?", false)
		}
	}

	err := totalQuery.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	query := db.WithContext(ctx).Model(&entities.UserReferral{}).Where("user_id = ? AND is_deleted = ?", userID, false)

	if filter.Page != nil && filter.PageSize != nil {
		query = query.Offset((*filter.Page - 1) * *filter.PageSize).
			Limit(*filter.PageSize)
	}

	if filter.Status != nil && *filter.Status != "" {
		switch *filter.Status {
		case "Accepted":
			query = query.Where("is_accept = ?", true)
		case "Pending":
			query = query.Where("is_accept = ?", false)
		}
	}

	err = query.Find(&referrals).Error
	if err != nil {
		return nil, 0, err
	}

	return referrals, total, nil
}

func (r *userReferralRepository) GetReferralByCode(ctx context.Context, db *gorm.DB, code string) (*entities.UserReferral, error) {
	var referral entities.UserReferral

	err := db.WithContext(ctx).
		Where("referral_code = ? AND is_deleted = ?", code, false).
		First(&referral).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		logrus.Errorf("error finding referral by code: %v", err)
		return nil, errs.NewInternalServerError("Failed to find referral by code", err)
	}

	return &referral, nil
}

func (r *userReferralRepository) UpdateReferralStatus(ctx context.Context, db *gorm.DB, id int64, isAccept bool) error {
	updates := map[string]interface{}{
		"is_accept": isAccept,
		"accept_on": time.Now(),
		"is_active": false,
	}

	result := db.WithContext(ctx).Model(&entities.UserReferral{}).
		Where("id = ? AND is_deleted = ?", id, false).
		Updates(updates)

	if result.Error != nil {
		logrus.Errorf("error updating referral status: %v", result.Error)
		return errs.NewInternalServerError("Failed to update referral status", result.Error)
	}

	if result.RowsAffected == 0 {
		return errs.NewNotFoundError("Referral not found", nil)
	}

	return nil
}

func (r *userReferralRepository) UpdateResendStatus(ctx context.Context, db *gorm.DB, id int64) error {
	updates := map[string]interface{}{
		"is_resend":     true,
		"re_send_count": gorm.Expr("re_send_count + ?", 1),
		"re_send_on":    time.Now(),
		"is_active":     true,
	}

	result := db.WithContext(ctx).Model(&entities.UserReferral{}).
		Where("id = ? AND is_deleted = ?", id, false).
		Updates(updates)

	if result.Error != nil {
		logrus.Errorf("error updating resend status: %v", result.Error)
		return errs.NewInternalServerError("Failed to update resend status", result.Error)
	}

	if result.RowsAffected == 0 {
		return errs.NewNotFoundError("Active referral not found", nil)
	}

	return nil
}

func (r *userReferralRepository) CheckActiveReferral(ctx context.Context, db *gorm.DB, userID string, referTo string) (bool, error) {
	var count int64
	err := db.WithContext(ctx).Model(&entities.UserReferral{}).
		Where("user_id = ? AND refer_to = ? AND is_active = ? AND is_deleted = ?", userID, referTo, true, false).
		Count(&count).Error

	if err != nil {
		logrus.Errorf("error checking active referral: %v", err)
		return false, errs.NewInternalServerError("Failed to check active referral", err)
	}

	return count > 0, nil
}

func (r *userReferralRepository) ValidateReferralCode(ctx context.Context, db *gorm.DB, referTo string, referralCode string) (*entities.UserReferral, error) {
	var referral entities.UserReferral
	err := db.WithContext(ctx).Model(&entities.UserReferral{}).
		Where("referral_code = ? AND refer_to = ? AND is_active = ? AND is_deleted = ?", referralCode, referTo, true, false).
		First(&referral).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errs.NewUnauthorizedError("Invalid referral code", nil)
	} else if err != nil {
		logrus.Errorf("error validating referral code: %v", err)
		return nil, errs.NewInternalServerError("Failed to validate referral code", err)
	}

	return &referral, nil
}
