package entities

import (
	"time"
)

// SSOToken represents a single-use intermediary token for SSO authentication
type SSOToken struct {
	ID          string     `json:"id" gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	PhoneNumber string     `json:"phone_number" gorm:"type:varchar(15);not null;index"`
	Name        string     `json:"name" gorm:"type:varchar(255);not null"`
	Token       string     `json:"token" gorm:"type:varchar(255);uniqueIndex;not null"`
	IsUsed      bool       `json:"is_used" gorm:"type:boolean;not null;default:false"`
	UsedAt      *time.Time `json:"used_at,omitempty" gorm:"type:timestamp"`
	ExpiresAt   time.Time  `json:"expires_at" gorm:"type:timestamp;not null;index"`
	CreatedBy   string     `json:"created_by" gorm:"type:varchar(100);not null"`
	CreatedAt   time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
}

// IsExpired checks if the token has expired
func (s *SSOToken) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// IsValid checks if the token is valid (not used and not expired)
func (s *SSOToken) IsValid() bool {
	return !s.IsUsed && !s.IsExpired()
}
