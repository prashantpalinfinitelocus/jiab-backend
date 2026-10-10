package entities

import (
	"time"

	"github.com/google/uuid"
)

type UserMixCodeDetails struct {
	ID                       int        `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID                   string     `json:"user_id" gorm:"type:varchar(450);not null;foreignKey:ID;references:users"`
	MixCode                  string     `json:"mix_code" gorm:"type:text"`
	IsValidMixCode           bool       `json:"is_valid_mix_code" gorm:"not null;default:false"`
	IsAlreadyRedeemedMixCode bool       `json:"is_already_redeemed_mix_code" gorm:"not null;default:false"`
	IsActiveMixCode          bool       `json:"is_active_mix_code" gorm:"not null;default:false"`
	IsExpiredMixCode         bool       `json:"is_expired_mix_code" gorm:"not null;default:false"`
	IsRedeemedSuccessfully   bool       `json:"is_redeemed_successfully" gorm:"not null;default:false"`
	IsGrOrEq500              bool       `json:"is_gr_or_eq500" gorm:"not null;default:false"`
	Points                   *int       `json:"points" gorm:"type:int"`
	VoucherAssigned          *uuid.UUID `json:"voucher_assigned" gorm:"type:uuid"`
	IsActive                 bool       `json:"is_active" gorm:"not null;default:true"`
	IsDeleted                bool       `json:"is_deleted" gorm:"not null;default:false"`
	CreatedBy                string     `json:"created_by" gorm:"type:text;not null"`
	CreatedOn                time.Time  `json:"created_on" gorm:"not null"`
	LastModifiedBy           *string    `json:"last_modified_by" gorm:"type:text"`
	LastModifiedOn           *time.Time `json:"last_modified_on"`
}

func (UserMixCodeDetails) TableName() string {
	return "user_mixcode_details"
}
