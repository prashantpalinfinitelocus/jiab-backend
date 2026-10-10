package entities

import "time"

type ProfilingQuestionAnswer struct {
	ID               int        `gorm:"column:id;primaryKey;autoIncrement"`
	UserID           string     `gorm:"column:user_id;not null"`
	QuestionMasterID int        `gorm:"column:question_master_id;not null"`
	OptionID         int        `gorm:"column:option_id;not null"`
	SelectedAnswer   bool       `gorm:"column:selected_answer;not null"`
	IsActive         bool       `gorm:"column:is_active;not null"`
	IsDeleted        bool       `gorm:"column:is_deleted;not null"`
	CreatedBy        string     `gorm:"column:created_by;not null"`
	CreatedOn        time.Time  `gorm:"column:created_on;not null"`
	LastViewedOn     *time.Time `gorm:"column:last_viewed_on"`
	QuestionNumber   int        `gorm:"column:question_number;not null"`
}

// TableName specifies the table name for UserQuestionAnswer
func (ProfilingQuestionAnswer) TableName() string {
	return "profiling_question_answer"
}
