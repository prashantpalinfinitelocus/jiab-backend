package dtos

import (
	"mime/multipart"
)

type JokeDTO struct {
	Genre    *int    `form:"genre" binding:"omitempty"`
	Language *int    `form:"language" binding:"omitempty"`
	Exclude  *string `form:"exclude" binding:"omitempty"`
}

type WAJokeDTO struct {
	UserId     string `form:"UserId" binding:"required"`
	GenreId    *int   `form:"genreId" binding:"required"`
	LanguageID *int   `form:"languageID" binding:"required"`
}

type JokeScrollDTO struct {
	Limit          *int    `form:"limit" binding:"omitempty"`
	SelectedJoke   *string `form:"selected_joke" binding:"omitempty"`
	Language       *int    `form:"language" binding:"required"`
	PreferredJokes *string `form:"preferred_jokes" binding:"omitempty"`
	Type           *string `form:"type" binding:"omitempty"`
	IsRefreshRing  *bool   `form:"is_refresh_ring" binding:"omitempty"`
}

type UserReaction struct {
	Sad     int `json:"sad"`
	Neutral int `json:"neutral"`
	Laugh   int `json:"laugh"`
	Like    int `json:"like"`
	Share   int `json:"share"`
}

type JokeResponseDTO struct {
	ID           string       `json:"id"`
	URL          string       `json:"url"`
	Title        string       `json:"title"`
	Type         string       `json:"type"`
	LikeCount    int          `json:"like_count"`
	JokeLanguage string       `json:"joke_language"`
	JokeGenre    string       `json:"joke_genre"`
	GenreImage   string       `json:"genre_image"`
	ViewCount    int          `json:"view_count"`
	ThumbnailUrl string       `json:"thumbnail_url"`
	UserReaction UserReaction `json:"user_reaction"`
	Creator      string       `json:"joke_creator"`
	ArtistId     string       `json:"artist_id"`
	IsExhausted  bool         `json:"is_exhausted"`
}

type JokeIngestionDTO struct {
	LanguageID  int                   `form:"language_id" binding:"required" swaggertype:"integer"`
	Format      string                `form:"format" binding:"required,oneof=Image Text Audio Video" swaggertype:"string"`
	File        *multipart.FileHeader `form:"file" binding:"required" swaggerignore:"true"`
	Title       string                `form:"title" binding:"required" swaggertype:"string"`
	JokeGenreID *int                  `form:"joke_genre_id,omitempty" swaggertype:"integer"`
}

type UGCContentByIDsDTO struct {
	IDs []string `json:"ids" binding:"required"`
}

type UGCContentByIDsResponseDTO struct {
	ID      string `json:"id"`
	Content string `json:"content"`
}

type UGCContentResponseDTO struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	ThumbnailUrl string `json:"thumbnail_url"`
	Format       string `json:"format"`
	Date         string `json:"date"`
	ContentURL   string `json:"content_url,omitempty"`
	Status       string `json:"status,omitempty"`
}

type UGCContentArtistDTO struct {
	ID         string `json:"id"`
	ArtistName string `json:"artist_name"`
}

type WAJokeResponseDTO struct {
	JokeId        string           `json:"jokeId"`
	Joke          string           `json:"joke"`
	JokeType      string           `json:"jokeType"`
	ArtistId      string           `json:"artistId"`
	JokeDetails   WAJokeDetailsDTO `json:"jokeDetails"`
	Genre         int              `json:"genre"`
	LanguageId    int              `json:"languageId"`
	JokesConsumed int              `json:"jokesConsumed"`
}

type WAJokeDetailsDTO struct {
	Title               string       `json:"title"`
	JokeContentType     string       `json:"jokeContentType"`
	ThumbnailUrl        string       `json:"thumbnailUrl"`
	JokeCreator         string       `json:"jokeCreator"`
	JokeLanguage        string       `json:"jokeLanguage"`
	JokeGenre           string       `json:"jokeGenre"`
	JokeReleaseDateTime string       `json:"jokeReleaseDateTime"`
	Reaction            UserReaction `json:"reaction"`
	ViewsCount          int          `json:"viewsCount"`
}

type AddUserReactPointRequest struct {
	AssetID      string `json:"assetId" binding:"required"`
	ActivityType int    `json:"activityType" binding:"required"`
}

type AddUserReactPointResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type AddUserReelReactionRequest struct {
	AssetID  string `json:"assetId" binding:"required"`
	Reaction string `json:"reaction" binding:"required"`
}

type ChangeUGCStatusRequest struct {
	VideoID string `json:"videoId" binding:"required"`
	Status  int    `json:"status" binding:"required,min=2,max=4"` // 2: Accepted, 4: Rejected, 3: Plagiarized
}

type ActivateJokeConsumptionRequest struct {
	JokeID string `json:"joke_id" binding:"required"`
}

type JokeWithLanguage struct {
	Joke        map[string]interface{}
	Language    string
	LanguageID  int
	LanguageKey string
}
