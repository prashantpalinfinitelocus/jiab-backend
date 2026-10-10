package dtos

type UGCJokeUploadRequest struct {
	LanguageID int64   `json:"languageId" binding:"required"`
	Format     string  `json:"format" binding:"required"`
	Title      string  `json:"title" binding:"required"`
	MediaID    *string `json:"mediaId"`
	BotNumber  int64   `json:"botNumber" binding:"required"`
	JokeText   *string `json:"jokeText"`
}
