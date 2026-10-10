package entities

import "time"

type UserWallet struct {
	ID                int64      `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID            string     `json:"user_id" gorm:"type:varchar(450);not null"`
	TotalCoinsEarned  int        `json:"total_coins_earned" gorm:"type:int;not null"`
	CurrentBalance    int        `json:"current_balance" gorm:"type:int;not null"`
	LastRedeemedCoins int        `json:"last_redeemed_coins" gorm:"type:int;not null;default:0"`
	LastRedeemedOn    *time.Time `json:"last_redeemed_on" gorm:"type:timestamp"`
	CoinsTypeID       int        `json:"coins_type_id" gorm:"type:int"`
	PlatformID        int        `json:"platform_id" gorm:"type:int"`
	WalletUpdatedOn   time.Time  `json:"wallet_updated_on" gorm:"type:timestamp"`
	IsMixCodeEnter    bool       `json:"is_mix_code_enter" gorm:"type:boolean;not null;default:false"`
	Winning           bool       `json:"winning" gorm:"type:boolean;not null;default:false"`
	IsActive          bool       `json:"is_active" gorm:"type:boolean;not null;default:true"`
	IsDeleted         bool       `json:"is_deleted" gorm:"type:boolean;not null;default:false"`
	CreatedBy         string     `json:"created_by" gorm:"type:text"`
	CreatedOn         time.Time  `json:"created_on" gorm:"type:timestamp"`
	LastModifiedBy    *string    `json:"last_modified_by" gorm:"type:text"`
	LastModifiedOn    *time.Time `json:"last_modified_on" gorm:"type:timestamp"`

	User User `json:"user" gorm:"foreignKey:UserID;references:ID"`
}

func (UserWallet) TableName() string {
	return "user_wallet"
}
