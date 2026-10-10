package dtos

import (
	"time"

	"mime/multipart"

	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
)

type AddressRequestDTO struct {
	Address1        string  `json:"address1"`
	Address2        *string `json:"address2,omitempty"`
	Pincode         int     `json:"pincode"`
	State           string  `json:"state"`
	City            string  `json:"city"`
	NearestLandmark *string `json:"nearest_landmark,omitempty"`
	ShippingMobile  *string `json:"shipping_mobile,omitempty"`
	IsDefault       bool    `json:"is_default"`
}

type AddressResponseDTO struct {
	ID              int        `json:"id"`
	Address1        string     `json:"address1"`
	Address2        *string    `json:"address2,omitempty"`
	Pincode         int        `json:"pincode"`
	State           string     `json:"state"`
	City            string     `json:"city"`
	NearestLandmark *string    `json:"nearest_landmark,omitempty"`
	ShippingMobile  *string    `json:"shipping_mobile,omitempty"`
	IsDefault       bool       `json:"is_default"`
	IsActive        bool       `json:"is_active"`
	CreatedOn       time.Time  `json:"created_on"`
	LastModifiedOn  *time.Time `json:"last_modified_on,omitempty"`
}

type EditProfileRequestDTO struct {
	Name         *string               `form:"name,omitempty"`
	Email        *string               `form:"email,omitempty"`
	Gender       *string               `form:"gender,omitempty"`
	DOB          *string               `form:"dob,omitempty"`
	AvatarID     *int                  `form:"avatar_id,omitempty"`
	IsAvatar     *bool                 `form:"is_avatar,omitempty"`
	IsProfile    *bool                 `form:"is_profile,omitempty"`
	ProfileImage *multipart.FileHeader `form:"profile_image,omitempty"`
}

type AnswerQuestionsRequestDTO struct {
	QuestionID     int `json:"question_id"`
	AnswerID       int `json:"answer_id"`
	QuestionNumber int `json:"question_number"`
}

type AnswerQuestionRequestDTO struct {
	OptionID       int  `json:"option_id" binding:"required"`
	SelectedAnswer bool `json:"selected_answer" binding:"required"`
}

type BalanceRankResponseDTO struct {
	Balance int `json:"balance"`
	Rank    int `json:"rank"`
}

type ProfileResponseDTO struct {
	User           entities.User `json:"user"`
	Rank           *int64        `json:"rank"`
	CurrentBalance int           `json:"current_balance"`
}

type OptionDTO struct {
	ID           int    `json:"id"`
	OptionText   string `json:"option_text"`
	DisplayOrder int    `json:"display_order"`
}

type QuestionResponseDTO struct {
	ID             int         `json:"id"`
	QuestionText   string      `json:"question_text"`
	LanguageID     int         `json:"language_id"`
	QuesPoint      int         `json:"ques_point"`
	Options        []OptionDTO `json:"options"`
	SelectedOption *int        `json:"selected_option,omitempty"`
}

type QuestionsResponseDTO struct {
	Total     int                   `json:"total"`
	Questions []QuestionResponseDTO `json:"questions"`
}

// ErrorResponse represents a standard error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// Bulk import users request
// Used for /main/users/bulk_import
// Each user should match the fields in entities.User

type BulkImportUsersRequest struct {
	Users []UserImportDTO `json:"users" binding:"required"`
}

type UserImportDTO struct {
	ID                     string     `json:"id"`
	Name                   *string    `json:"name"`
	Gender                 *int       `json:"gender"`
	DOB                    *time.Time `json:"dob"`
	AlternateMobile        *string    `json:"alternate_mobile"`
	ProfilePicture         *string    `json:"profile_picture"`
	IsActive               bool       `json:"is_active"`
	RefreshToken           *string    `json:"refresh_token"`
	RefreshTokenExpiryTime *time.Time `json:"refresh_token_expiry_time"`
	AvatarID               *int       `json:"avatar_id"`
	ReferralCode           *string    `json:"referral_code"`
	IsReferral             bool       `json:"is_referral"`
	ReferBy                *string    `json:"refer_by"`
	IsProfileComplete      bool       `json:"is_profile_complete"`
	IPAddress              *string    `json:"ip_address"`
	PlatformMasterID       *int       `json:"platform_master_id"`
	ProfilePercentage      int        `json:"profile_percentage"`
	LastLanguageSelection  int        `json:"last_language_selection"`
	RegisteredOn           time.Time  `json:"registered_on"`
	Email                  *string    `json:"email"`
	PhoneNumber            string     `json:"phone_number"`
	IsTamil                bool       `json:"is_tamil"`
	EmailConfirmed         bool       `json:"email_confirmed"`
	PhoneNumberConfirmed   bool       `json:"phone_number_confirmed"`
	TwoFactorEnabled       bool       `json:"two_factor_enabled"`
	LockoutEnabled         bool       `json:"lockout_enabled"`
	AccessFailedCount      int        `json:"access_failed_count"`
	IsProfilePopUp         bool       `json:"is_profile_popup"`
}

// Missing IDs request for /main/users/missing_ids
// {"ids": ["id1", "id2", ...]}
type MissingIDsRequest struct {
	IDs []string `json:"ids" binding:"required"`
}

// Generic bulk import request for /main/generic_bulk_import
type GenericBulkImportRequest struct {
	Table string        `json:"table" binding:"required"`
	Data  []interface{} `json:"data" binding:"required"`
}

type ProfilingQuestionModalResponseDTO struct {
	ShowModal      bool `json:"show_modal"`
	QuestionNumber int  `json:"question_number"`
}
