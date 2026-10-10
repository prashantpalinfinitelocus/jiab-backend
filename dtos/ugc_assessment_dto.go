package dtos

import "time"

type UGCAssessmentRequest struct {
	StartDate  time.Time `json:"start_date" binding:"required"`
	EndDate    time.Time `json:"end_date" binding:"required"`
	LanguageID *int      `json:"language_id" binding:"omitempty"`
	GenreID    *int      `json:"genre_id" binding:"omitempty"`
	Source     *int      `json:"source" binding:"omitempty"`
}

type UGCAssessmentFilter struct {
	StartDate  time.Time
	EndDate    time.Time
	LanguageID *int
	GenreID    *int
	Source     *int
	Page       int
	Limit      int
}

func NewUGCAssessmentFilter(request UGCAssessmentRequest, page, limit int) UGCAssessmentFilter {
	return UGCAssessmentFilter{
		StartDate:  request.StartDate,
		EndDate:    request.EndDate,
		LanguageID: request.LanguageID,
		GenreID:    request.GenreID,
		Source:     request.Source,
		Page:       page,
		Limit:      limit,
	}
}

type UGCAssessmentResponse struct {
	ContentID       string    `json:"content_id"`
	ContentURL      string    `json:"content_url"`
	ContentGenre    *string   `json:"content_genre,omitempty"`
	ContentLanguage string    `json:"content_language"`
	Title           string    `json:"title"`
	UserID          string    `json:"user_id"`
	CreatedOn       time.Time `json:"created_on"`
	Status          string    `json:"status"`
}

type UGCAssessmentListResponse struct {
	Data       []UGCAssessmentResponse `json:"data"`
	TotalCount int64                   `json:"total_count"`
	Page       int                     `json:"page"`
	Limit      int                     `json:"limit"`
}
