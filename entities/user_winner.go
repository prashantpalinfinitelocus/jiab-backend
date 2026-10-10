package entities

import "time"

type UserWinner struct {
	ID                   string    `gorm:"column:id;primaryKey"`
	UserID               string    `gorm:"column:user_id;not null"`
	WinnerRewardID       int       `gorm:"column:winner_reward_id;not null"`
	Coins                int       `gorm:"column:coins;not null"`
	WinnerTime           time.Time `gorm:"column:winner_time;not null"`
	IsClaimWindowOpen    bool      `gorm:"column:is_claim_window_open;not null"`
	ClaimWindowOpen      time.Time `gorm:"column:claim_window_open"`
	ClaimWindowClose     time.Time `gorm:"column:claim_window_close"`
	IsActivityRequired   bool      `gorm:"column:is_activity_required;not null"`
	IsActivityPerformed  bool      `gorm:"column:is_activity_performed;not null"`
	ActivityTime         time.Time `gorm:"column:activity_time"`
	IsRewardClaimed      bool      `gorm:"column:is_reward_claimed;not null"`
	ClaimDate            time.Time `gorm:"column:claim_date"`
	QuotaRewardDetailsID int       `gorm:"column:quota_reward_details_id"`
	IsPromoCode          bool      `gorm:"column:is_promo_code;not null"`
	UserRank             int       `gorm:"column:user_rank"`
	IsActive             bool      `gorm:"column:is_active;not null"`
	IsDeleted            bool      `gorm:"column:is_deleted;not null"`
	CreatedBy            string    `gorm:"column:created_by;not null"`
	CreatedOn            time.Time `gorm:"column:created_on;not null"`
	LastModifiedBy       string    `gorm:"column:last_modified_by"`
	LastModifiedOn       time.Time `gorm:"column:last_modified_on"`
	Mobile               string    `gorm:"column:mobile"`             // From users table
	AvatarID             *int      `gorm:"column:avatar_id"`          // From users table
	WinnerImageURL       string    `gorm:"column:winner_image_url"`   // From winner_rewards table
	WinnerDescription    string    `gorm:"column:winner_description"` // From winner_rewards table
}

func (UserWinner) TableName() string {
	return "user_winners"
}
