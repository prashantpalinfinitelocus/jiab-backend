package dtos

import "time"

type RequestOTPRequest struct {
	MobileNumber string `json:"mobile_number" binding:"required,min=10,max=10,numeric"`
}

type VerifyOTPRequest struct {
	MobileNumber string `json:"mobile_number" binding:"required,min=10,max=10"`
	Otp          string `json:"otp" binding:"required,min=6,max=6"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int64  `json:"expires_in,omitempty"`
	TokenType    string `json:"token_type,omitempty"`
	UserID       string `json:"user_id,omitempty"`
	MobileNumber string `json:"mobile_number,omitempty"`
	Name         string `json:"name,omitempty"`
	Email        string `json:"email,omitempty"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type SignUpRequest struct {
	MobileNumber string  `json:"mobile_number" binding:"required,min=10,max=10"`
	AvatarID     *int    `json:"avatar_id,omitempty"`
	FullName     string  `json:"full_name" binding:"required"`
	EmailID      *string `json:"email" binding:"omitempty,email"`
	ReferralCode *string `json:"referral_code,omitempty"`
	DeviceToken  string  `json:"device_token"`
}

type SignUpResponse struct {
	AccessToken         string `json:"access_token,omitempty"`
	RefreshToken        string `json:"refresh_token,omitempty"`
	ExpiresIn           int64  `json:"expires_in,omitempty"`
	TokenType           string `json:"token_type,omitempty"`
	UserID              string `json:"user_id,omitempty"`
	Name                string `json:"name,omitempty"`
	Email               string `json:"email,omitempty"`
	PhoneNumber         string `json:"phone_number,omitempty"`
	ReferralID          string `json:"referral_id,omitempty"`
	ReferredBy          string `json:"referred_by,omitempty"`
	ReferralPhoneNumber string `json:"referral_phone_number,omitempty"`
}

// WARegisterRequest is used for the WA registration API
// Matches the expected request body for /auth/WA/register
type WARegisterRequest struct {
	Name         string `json:"name" binding:"required"`
	Email        string `json:"email"`
	PhoneNumber  string `json:"phoneNumber" binding:"required,min=10,max=15"`
	ReferralCode string `json:"referralCode"`
	IpAddress    string `json:"ipAddress"`
	HfSession    string `json:"hfSession"`
	LanguageId   int    `json:"languageId"`
}

// WARegisterResponse is used for the WA registration API response
// Matches the expected response structure for /auth/WA/register
type WARegisterResponse struct {
	ID           string   `json:"id"`
	MobileNo     string   `json:"mobileNo"`
	Name         string   `json:"name"`
	Email        string   `json:"email"`
	Roles        []string `json:"roles"`
	IsVerified   bool     `json:"isVerified"`
	JwtToken     string   `json:"jwtToken"`
	IssuedOn     string   `json:"issuedOn"`
	ExpiresOn    string   `json:"expiresOn"`
	MainRole     string   `json:"mainRole"`
	WalletPoints int      `json:"walletPoints"`
	NewUser      bool     `json:"newUser"`
	ReferralCode string   `json:"referralCode"`
	RefreshToken string   `json:"refreshToken"`
}

// SSO Flow DTOs

// SSOInitiateRequest is the request from a partner site to initiate SSO
type SSOInitiateRequest struct {
	PhoneNumber string `json:"phone_number" binding:"required,min=10,max=10,numeric"`
	Name        string `json:"name" binding:"required,min=2,max=255"`
}

// SSOInitiateResponse contains the intermediary SSO token for the partner to redirect with
type SSOInitiateResponse struct {
	SSOToken    string    `json:"sso_token"`
	RedirectURL string    `json:"redirect_url"`
	ExpiresIn   int64     `json:"expires_in"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// SSOAuthenticateRequest is used by the frontend to exchange an SSO token for JWT tokens
type SSOAuthenticateRequest struct {
	SSOToken string `json:"sso_token" binding:"required"`
}

// SSOAuthenticateResponse contains JWT tokens and user details after SSO token exchange
type SSOAuthenticateResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
	UserID       string `json:"user_id"`
	PhoneNumber  string `json:"phone_number"`
	Name         string `json:"name"`
	IsNewUser    bool   `json:"is_new_user"`
}
