package dtos

import "time"

type WATokenRequest struct {
	MobileNumber string `json:"mobileNo" binding:"required"`
}

type WATokenResponse struct {
	ID           string    `json:"id"`
	MobileNumber string    `json:"mobileNo"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	Roles        []string  `json:"roles"`
	IsVerified   bool      `json:"isVerified"`
	JwtToken     string    `json:"jwtToken"`
	IssuedOn     time.Time `json:"issuedOn"`
	ExpiresOn    time.Time `json:"expiresOn"`
	MainRole     string    `json:"mainRole"`
	WalletPoints int       `json:"walletPoints"`
	NewUser      bool      `json:"newUser"`
	RefreshToken string    `json:"refreshToken"`
}
