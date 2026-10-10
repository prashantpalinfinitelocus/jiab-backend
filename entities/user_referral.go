package entities

import "time"

type UserReferral struct {
	ID             int64      `gorm:"primaryKey;autoIncrement;column:id"`
	UserID         string     `gorm:"column:user_id;type:uuid;not null"`
	ReferTo        string     `gorm:"column:refer_to;type:uuid;not null"`
	ReferOn        time.Time  `gorm:"column:refer_on;not null"`
	IsAccept       bool       `gorm:"column:is_accept;not null"`
	AcceptOn       *time.Time `gorm:"column:accept_on"`
	IsResend       bool       `gorm:"column:is_resend;not null"`
	ReSendCount    int        `gorm:"column:re_send_count;not null"`
	ReSendOn       *time.Time `gorm:"column:re_send_on"`
	ReferralCode   *string    `gorm:"column:referral_code"`
	IsTamil        bool       `gorm:"column:is_tamil;default:false"`
	IsActive       bool       `gorm:"column:is_active;not null"`
	IsDeleted      bool       `gorm:"column:is_deleted;not null"`
	CreatedBy      string     `gorm:"column:created_by;not null"`
	CreatedOn      time.Time  `gorm:"column:created_on;not null"`
	LastModifiedBy *string    `gorm:"column:last_modified_by"`
	LastModifiedOn *time.Time `gorm:"column:last_modified_on"`

	Referrer User `gorm:"foreignKey:UserID;references:ID"`
}

func (UserReferral) TableName() string {
	return "user_referral"
}
