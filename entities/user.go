package entities

import (
	"time"
)

type User struct {
	ID                      string     `json:"id" gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	Name                    *string    `json:"name,omitempty" gorm:"type:varchar(200)"`
	Gender                  *int       `json:"gender,omitempty" gorm:"type:int"`
	DOB                     *time.Time `json:"dob,omitempty" gorm:"type:timestamp"`
	AlternateMobile         *string    `json:"alternate_mobile,omitempty" gorm:"type:text"`
	ProfilePicture          *string    `json:"profile_picture,omitempty" gorm:"type:text"`
	IsActive                bool       `json:"is_active" gorm:"type:boolean;not null;default:true"`
	RefreshToken            *string    `json:"refresh_token,omitempty" gorm:"type:text"`
	RefreshTokenExpiryTime  *time.Time `json:"refresh_token_expiry_time,omitempty" gorm:"type:timestamp"`
	AvatarID                *int       `json:"avatar_id,omitempty" gorm:"type:int"`
	ReferralCode            *string    `json:"referral_code,omitempty" gorm:"type:varchar(20)"`
	IsReferral              bool       `json:"is_referral" gorm:"type:boolean;not null"`
	ReferBy                 *string    `json:"refer_by,omitempty" gorm:"type:text"`
	IsProfileComplete       bool       `json:"is_profile_complete" gorm:"type:boolean;not null"`
	IPAddress               *string    `json:"ip_address,omitempty" gorm:"type:text"`
	PlatformMasterID        *int       `json:"platform_master_id,omitempty" gorm:"type:int"`
	ProfilePercentage       int        `json:"profile_percentage" gorm:"type:int;not null"`
	LastLanguageSelection   int        `json:"last_language_selection" gorm:"type:int;not null"`
	RegisteredOn            time.Time  `json:"registered_on" gorm:"type:timestamp;not null"`
	Email                   *string    `json:"email,omitempty" gorm:"type:text"`
	PhoneNumber             string     `json:"phone_number,omitempty" gorm:"type:varchar(10);not null"`
	ContestOnboardingStatus bool       `json:"contest_onboarding_status" gorm:"type:boolean;not null;default:false"`
	IsTamil                 bool       `json:"is_tamil" gorm:"type:boolean;not null;default:false"`
	EmailConfirmed          bool       `json:"email_confirmed" gorm:"type:boolean;not null;default:false"`
	PhoneNumberConfirmed    bool       `json:"phone_number_confirmed" gorm:"type:boolean;not null;default:false"`
	TwoFactorEnabled        bool       `json:"two_factor_enabled" gorm:"type:boolean;not null;default:false"`
	LockoutEnabled          bool       `json:"lockout_enabled" gorm:"type:boolean;not null;default:false"`
	AccessFailedCount       int        `json:"access_failed_count" gorm:"type:int;not null;default:0"`
	HomeOnboardingStatus    bool       `json:"home_onboarding_status" gorm:"type:boolean;not null;default:false"`
	IsProfilePopUp          bool       `json:"is_profile_popup" gorm:"column:is_profile_popup;type:boolean;not null;default:false"`
}

type UserRefreshToken struct {
	ID                     int        `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID                 string     `json:"user_id" gorm:"type:uuid;not null;index"`
	RefreshToken           *string    `json:"refresh_token,omitempty" gorm:"type:text"`
	IsUsed                 bool       `json:"is_used" gorm:"type:boolean;not null;default:false"`
	RefreshTokenExpiryTime *time.Time `json:"refresh_token_expiry_time,omitempty" gorm:"type:timestamp"`
	IsActive               bool       `json:"is_active" gorm:"type:boolean;not null;default:true"`
	IsDeleted              bool       `json:"is_deleted" gorm:"type:boolean;not null;default:false"`
	CreatedOn              time.Time  `json:"created_on" gorm:"type:timestamp;not null"`
	CreatedBy              string     `json:"created_by" gorm:"type:varchar(255);not null"`
	LastModifiedOn         *time.Time `json:"last_modified_on,omitempty" gorm:"type:timestamp"`
	LastModifiedBy         string     `json:"last_modified_by" gorm:"type:varchar(255)"`

	User User `json:"user" gorm:"foreignKey:UserID;references:ID"`
}
