package repository

import (
	"context"

	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	"gorm.io/gorm"
)

type VoucherInfoRepository interface {
	GenericRepository[entities.VoucherInfo]
	GetVoucherInfo(ctx context.Context, tx *gorm.DB, userId string) ([]entities.VoucherInfo, error)
	FindTranslation(ctx context.Context, tx *gorm.DB, languageId int, rewardId int) (*entities.WinnerRewardLanguage, error)
}

type voucherInfoRepository struct {
	*GormRepository[entities.VoucherInfo]
}

func NewVoucherInfoRepository() VoucherInfoRepository {
	return &voucherInfoRepository{
		GormRepository: NewGormRepository[entities.VoucherInfo](),
	}
}

func (r *voucherInfoRepository) GetVoucherInfo(ctx context.Context, tx *gorm.DB, userId string) ([]entities.VoucherInfo, error) {
	var voucherInfos []entities.VoucherInfo

	err := tx.Preload("WinnerReward").Where("assigned_to_user_id = ? and is_active = ? and is_deleted = ?", userId, true, false).Find(&voucherInfos).Error
	if err != nil {
		return nil, err
	}

	return voucherInfos, nil
}

func (r *voucherInfoRepository) FindTranslation(ctx context.Context, tx *gorm.DB, languageId int, rewardId int) (*entities.WinnerRewardLanguage, error) {
	var winnerRewardsLanguage entities.WinnerRewardLanguage

	err := tx.Model(&entities.WinnerRewardLanguage{}).Where("language_id = ? and winner_reward_id = ? and is_active = ? and is_deleted = ?", languageId, rewardId, true, false).First(&winnerRewardsLanguage).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &winnerRewardsLanguage, nil
}
