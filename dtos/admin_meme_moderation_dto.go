package dtos

type AdminPendingMemeItem struct {
	ID          string `json:"id"`
	AssetURL    string `json:"asseturl"`
	LanguageID  string `json:"languageId"`
	Category    string `json:"category"`
	Text1       string `json:"text1"`
	Text2       string `json:"text2"`
	SubmittedBy string `json:"submittedBy"`
	SubmittedAt string `json:"submittedAt"`
}

type AdminPendingMemesResponse struct {
	Items  []AdminPendingMemeItem `json:"items"`
	Cursor *string                `json:"cursor"`
}

type ApproveMemeRequest struct {
	LanguageID *int    `json:"languageId"`
	Notes      *string `json:"notes"`
}

type ApproveMemeResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type RejectMemeRequest struct {
	Reason string `json:"reason" binding:"required"`
}

type RejectMemeResponse struct {
	Status string `json:"status"`
}
