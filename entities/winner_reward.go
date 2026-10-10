package entities

import "time"

type WinnerReward struct {
	ID                 int       `gorm:"column:id;primaryKey"`
	Name               string    `gorm:"column:name;not null"`
	Description        string    `gorm:"column:description"`
	LongDesc           string    `gorm:"column:long_desc"`
	Information        string    `gorm:"column:information"`
	ImageURL           string    `gorm:"column:image_url"`
	Worth              int       `gorm:"column:worth"`
	IconURL            string    `gorm:"column:icon_url"`
	RedirectURL        string    `gorm:"column:redirect_url"`
	OLName             string    `gorm:"column:ol_name"`
	OLDescription      string    `gorm:"column:ol_description"`
	OLLongDesc         string    `gorm:"column:ol_long_desc"`
	OLImageURL         string    `gorm:"column:ol_image_url"`
	OLIconURL          string    `gorm:"column:ol_icon_url"`
	TierMasterID       int       `gorm:"column:tier_master_id;not null"`
	NoOfWinners        int       `gorm:"column:no_of_winners;not null"`
	QuotaTypeID        int       `gorm:"column:quota_type_id;not null"`
	CoinTypeID         int       `gorm:"column:coin_type_id;not null"`
	PlatformMasterID   int       `gorm:"column:platform_master_id;not null"`
	NoOfCoins          int       `gorm:"column:no_of_coins;not null"`
	MaxRewards         int       `gorm:"column:max_rewards;not null"`
	StartDate          time.Time `gorm:"column:start_date;not null"`
	IsComplete         bool      `gorm:"column:is_complete;not null"`
	TotalWins          int       `gorm:"column:total_wins;not null"`
	TotalClaims        int       `gorm:"column:total_claims;not null"`
	DisplayOrder       int       `gorm:"column:display_order;not null"`
	IsPublished        bool      `gorm:"column:is_published;not null"`
	PublishedBy        string    `gorm:"column:published_by"`
	PublishedOn        time.Time `gorm:"column:published_on"`
	IsIDRequired       bool      `gorm:"column:is_id_required;not null"`
	IsPassportRequired bool      `gorm:"column:is_passport_required;not null"`
	IsPromocode        bool      `gorm:"column:is_promocode;not null"`
	NgpsID             string    `gorm:"column:ngps_id"`
	ShareURL           string    `gorm:"column:share_url"`
	IsActive           bool      `gorm:"column:is_active;not null"`
	IsDeleted          bool      `gorm:"column:is_deleted;not null"`
	CreatedBy          string    `gorm:"column:created_by;not null"`
	CreatedOn          time.Time `gorm:"column:created_on;not null"`
	LastModifiedBy     string    `gorm:"column:last_modified_by"`
	LastModifiedOn     time.Time `gorm:"column:last_modified_on"`
}

func (WinnerReward) TableName() string {
	return "winner_rewards"
}

type WinnerRewardLanguage struct {
	ID             int       `gorm:"column:id;primaryKey"`
	WinnerRewardID int       `gorm:"column:winner_reward_id;not null"`
	LanguageID     int       `gorm:"column:language_id;not null"`
	Name           string    `gorm:"column:name;not null"`
	Description    string    `gorm:"column:description"`
	ButtonValue    string    `gorm:"column:button_value;not null"`
	IsActive       bool      `gorm:"column:is_active;not null"`
	IsDeleted      bool      `gorm:"column:is_deleted;not null"`
	CreatedBy      string    `gorm:"column:created_by;not null"`
	CreatedOn      time.Time `gorm:"column:created_on;not null"`
	LastModifiedBy string    `gorm:"column:last_modified_by"`
	LastModifiedOn time.Time `gorm:"column:last_modified_on"`
	ButtonValueEN  string    `gorm:"column:button_value_en;not null"`
	Information    string    `gorm:"column:information"`

	WinnerReward WinnerReward `gorm:"foreignKey:WinnerRewardID;references:ID"`
}

func (WinnerRewardLanguage) TableName() string {
	return "winner_reward_languages"
}
