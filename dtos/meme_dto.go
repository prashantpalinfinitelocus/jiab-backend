package dtos

type MemeSubmitRequest struct {
	LanguageID int    `form:"language_id" binding:"required"`
	TemplateID string `form:"template_id" binding:"required"`
	Text1      string `form:"text1"`
	Text2      string `form:"text2"`
}

type MemeSubmitResponseData struct {
	MemeID string `json:"memeId"`
	Status string `json:"status"`
}

type MemeFeedRequest struct {
	Language *int    `form:"language"`
	Q        *string `form:"q"`
	Limit    *int    `form:"limit"`
	Cursor   *string `form:"cursor"`
	Trending *bool   `form:"trending"`
}

type ReactionCounts struct {
	Happy   int `json:"happy"`
	Sad     int `json:"sad"`
	Neutral int `json:"neutral"`
}

type ReactionClick struct {
	Happy   int `json:"happy"`
	Sad     int `json:"sad"`
	Neutral int `json:"neutral"`
}

type MemeFeedItem struct {
	ID             string         `json:"id"`
	UserID         string         `json:"userId"`
	AssetURL       string         `json:"assetUrl"`
	Language       string         `json:"language"`
	ViewCount      int            `json:"viewCount"`
	ReactionCounts ReactionCounts `json:"reactionCounts"`
	ReactionClick  ReactionClick  `json:"reactionClick"`
	HasVoted       bool           `json:"hasVoted"`
	Username       *string        `json:"username"`
	AvatarImage    *string        `json:"avatarImage"`
}

type MemeFeedResponse struct {
	Items  []MemeFeedItem `json:"items"`
	Cursor *string        `json:"cursor"`
}

type MemeDetailResponse struct {
	ID             string         `json:"id"`
	UserID         string         `json:"userId"`
	Status         int            `json:"status"`
	AssetURL       string         `json:"assetUrl"`
	Language       string         `json:"language"`
	ViewCount      int            `json:"viewCount"`
	ReactionCounts ReactionCounts `json:"reactionCounts"`
	ReactionClick  ReactionClick  `json:"reactionClick"`
	HasVoted       bool           `json:"hasVoted"`
}

type MemeVoteRequest struct {
	AssetID string `json:"asset_id" binding:"required"`
}

type MemeReactRequest struct {
	Reaction string `json:"reaction" binding:"required"`
}

type MemeIncrementViewResponse struct {
	MemeID    string `json:"memeId"`
	ViewCount int    `json:"viewCount"`
}

type ReactionCount struct {
	MemeID  string
	Happy   int
	Sad     int
	Neutral int
}

type MemeReactionResult struct {
	MemeID   string
	Reaction string
	Count    int
}

type UserMemesRequest struct {
	Status *int    `form:"status"`
	Limit  *int    `form:"limit"`
	Cursor *string `form:"cursor"`
}

type UserMemesItem struct {
	ID             string         `json:"id"`
	UserID         string         `json:"userId"`
	AssetURL       string         `json:"assetUrl"`
	Language       string         `json:"language"`
	Status         int            `json:"status"`
	ViewCount      int            `json:"viewCount"`
	ReactionCounts ReactionCounts `json:"reactionCounts"`
	ReactionClick  ReactionClick  `json:"reactionClick"`
	HasVoted       bool           `json:"hasVoted"`
	CreatedAt      string         `json:"createdAt"`
	Username       *string        `json:"username"`
	AvatarImage    *string        `json:"avatarImage"`
}

type UserMemesResponse struct {
	Items  []UserMemesItem `json:"items"`
	Cursor *string         `json:"cursor"`
}
