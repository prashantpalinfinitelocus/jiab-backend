package dtos

type GenreFilter struct {
	Search     *string `form:"search"`
	Page       *int    `form:"page"`
	PageSize   *int    `form:"page_size"`
	IsActive   *bool   `form:"is_active;default:true"`
	LanguageID *int    `form:"language_master_id"`
}

type JokeGenreResponse struct {
	ID            int    `json:"id"`
	Genre         string `json:"genre"`
	GenreEmoticon string `json:"genre_emoticon"`
	ImageURL      string `json:"image_url"`
	EmoIsActive   bool   `json:"emo_is_active"`
	EmoIndex      int    `json:"emo_index"`
	IsActive      bool   `json:"is_active"`
}

type GenrePaginatedResponse struct {
	Genres   []JokeGenreResponse `json:"genres,omitempty"`
	Total    int64               `json:"total"`
	Page     int                 `json:"page,omitempty"`
	PageSize int                 `json:"page_size,omitempty"`
}
