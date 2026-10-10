package dtos

type JokeGenreLanguageResponseDTO struct {
	ID                    int                         `json:"id"`
	Genre                 string                      `json:"genre"`
	ImageURL              *string                     `json:"imageUrl"`
	JokeGenreLanguageList []JokeGenreLanguageListItem `json:"jokeGenreLanguageList"`
}

type JokeGenreLanguageListItem struct {
	LanguageID     int    `json:"languageId" gorm:"column:language_id"`
	Language       string `json:"language" gorm:"column:language_name"`
	LanguageScript string `json:"languageScript" gorm:"column:script"`
}
