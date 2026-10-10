package dtos

// Request DTOs
type GenerateReferralRequest struct {
	ReferTo string `json:"refer_to" binding:"required,min=10,max=10,numeric"`
}

type SendReferralAgainRequest struct {
	ReferralID int64 `json:"referral_id" binding:"required"`
}

type VerifyReferralRequest struct {
	ReferralCode string `json:"referral_code" binding:"required"`
}

// Response DTOs
type ReferralResponse struct {
	ReferralCode string `json:"referral_code"`
}

type ReferralStatusResponse struct {
	ID           int64  `json:"id"`
	MobileNumber string `json:"mobile_number"`
	Status       string `json:"status"`
}

type ReferralListResponse struct {
	Referrals []ReferralStatusResponse `json:"referrals"`
	Total     int64                    `json:"total"`
	Page      *int                     `json:"page,omitempty"`
	PageSize  *int                     `json:"page_size,omitempty"`
}

type ReferralFilter struct {
	Page     *int    `form:"page"`
	PageSize *int    `form:"page_size"`
	Status   *string `form:"status"`
}
