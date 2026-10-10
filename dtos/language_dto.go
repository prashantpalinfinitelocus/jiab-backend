package dtos

type LanguageFilter struct {
	Search        *string `form:"search"`
	Page          *int    `form:"page"`
	PageSize      *int    `form:"page_size"`
	IsActive      *bool   `form:"is_active"`
	PlatformID    *int    `form:"platform_id"`
	LanguageKeyID *int    `form:"language_key_id"`
}

type LanguageResponse struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	LanguageKey string `json:"language_key"`
	IsActive    bool   `json:"is_active"`
}

type LanguagePaginatedResponse struct {
	Languages []LanguageResponse `json:"languages,omitempty"`
	Total     int64              `json:"total"`
	Page      int                `json:"page,omitempty"`
	PageSize  int                `json:"page_size,omitempty"`
}
