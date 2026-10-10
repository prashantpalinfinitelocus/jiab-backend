package entities

import "time"

type VoucherInfo struct {
	ID                  int        `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	VoucherCode         string     `json:"voucher_code" gorm:"column:voucher_code;type:text;not null"`
	SecureCodeAvailable bool       `json:"secure_code_available" gorm:"column:secure_code_available;type:boolean;not null"`
	SecureCode          *int       `json:"secure_code,omitempty" gorm:"column:secure_code;type:int"`
	WinnerRewardID      int        `json:"winner_reward_id" gorm:"column:winner_reward_id;type:int;not null"`
	AssignedToUserID    *string    `json:"assigned_to_user_id,omitempty" gorm:"column:assigned_to_user_id;type:uuid"`
	AssignedOn          *time.Time `json:"assigned_on,omitempty" gorm:"column:assigned_on;type:timestamp"`
	UserWinnersID       *int64     `json:"user_winners_id,omitempty" gorm:"column:user_winners_id;type:bigint"`
	IsActive            bool       `json:"is_active" gorm:"column:is_active;type:boolean;not null"`
	IsDeleted           bool       `json:"is_deleted" gorm:"column:is_deleted;type:boolean;not null"`
	CreatedBy           string     `json:"created_by" gorm:"column:created_by;type:text;not null"`
	CreatedOn           time.Time  `json:"created_on" gorm:"column:created_on;type:timestamp;not null"`
	LastModifiedBy      *string    `json:"last_modified_by,omitempty" gorm:"column:last_modified_by;type:text"`
	LastModifiedOn      *time.Time `json:"last_modified_on,omitempty" gorm:"column:last_modified_on;type:timestamp"`
	LastDateToRedeem    *time.Time `json:"last_date_to_redeem,omitempty" gorm:"column:last_date_to_redeem;type:timestamp;default:CURRENT_TIMESTAMP"`

	// Foreign key relationship
	WinnerReward WinnerReward `json:"winner_reward,omitempty" gorm:"foreignKey:WinnerRewardID;references:ID"`
}

func (VoucherInfo) TableName() string {
	return "vouchers_infos"
}
