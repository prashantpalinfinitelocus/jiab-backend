package dtos

type JokeFormatFilter struct {
	Search   *string `form:"search"`
	Page     *int    `form:"page"`
	PageSize *int    `form:"page_size"`
	IsActive *bool   `form:"is_active"`
}

type JokeFormatResponse struct {
	ID       int    `json:"id"`
	Value    string `json:"value"`
	Title    string `json:"title"`
	IconURL  string `json:"icon_url"`
	IsActive bool   `json:"is_active"`
}

type JokeFormatPaginatedResponse struct {
	Formats  []JokeFormatResponse `json:"formats"`
	Total    int64                `json:"total"`
	Page     int                  `json:"page,omitempty"`
	PageSize int                  `json:"page_size,omitempty"`
}
