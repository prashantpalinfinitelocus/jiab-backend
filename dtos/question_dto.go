package dtos

type QuestionAnswerRequestDTO struct {
	Question string   `json:"question"`
	Answer   []string `json:"answer"`
}

type CreateOptionDTO struct {
	ID           *int   `json:"id"` // Nullable for creation
	OptionText   string `json:"option_text"`
	DisplayOrder int    `json:"display_order"`
	IsActive     *bool  `json:"is_active"`
}

type CreateQuestionDTO struct {
	ID           *int              `json:"id"`
	QuestionText string            `json:"question_text"`
	QuesPoint    int               `json:"ques_point"`
	LanguageID   int               `json:"language_id"`
	IsActive     *bool             `json:"is_active"`
	Options      []CreateOptionDTO `json:"options"`
}

type CreateQuestionsRequestDTO struct {
	Questions []CreateQuestionDTO `json:"questions"`
}

type GetQuestionByTextRequestDTO struct {
	QuestionText string `json:"question_text" binding:"required"`
	LanguageID   int    `json:"language_id" binding:"required"`
}
