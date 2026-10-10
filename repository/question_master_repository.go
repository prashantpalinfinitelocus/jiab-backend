package repository

import (
	"context"

	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	"gorm.io/gorm"
)

type QuestionMasterRepository interface {
	FindByID(ctx context.Context, tx *gorm.DB, id int) (*entities.QuestionMaster, error)
	FindAll(ctx context.Context, tx *gorm.DB) ([]entities.QuestionMaster, error)
	FindActive(ctx context.Context, tx *gorm.DB) ([]entities.QuestionMaster, error)
	FindByQuestionTextAndLanguageID(ctx context.Context, tx *gorm.DB, questionText string, languageID int) (*entities.QuestionMaster, error)
	Create(ctx context.Context, tx *gorm.DB, question *entities.QuestionMaster) error
	Update(ctx context.Context, tx *gorm.DB, question *entities.QuestionMaster) error
	Delete(ctx context.Context, tx *gorm.DB, id int) error
	FindAllByLanguage(ctx context.Context, tx *gorm.DB, languageID *int) ([]entities.QuestionMaster, error)
	FindActiveByLanguageWithUniquePoints(ctx context.Context, tx *gorm.DB, languageID int) ([]entities.QuestionMaster, error)
	FindActiveWithLanguageText(ctx context.Context, tx *gorm.DB, languageID int) ([]entities.QuestionMaster, error)
	CountActive(ctx context.Context, tx *gorm.DB) (int64, error)
}

type questionMasterRepository struct {
	db *gorm.DB
}

func NewQuestionMasterRepository(db *gorm.DB) QuestionMasterRepository {
	return &questionMasterRepository{
		db: db,
	}
}

func (r *questionMasterRepository) FindByID(ctx context.Context, tx *gorm.DB, id int) (*entities.QuestionMaster, error) {
	var question entities.QuestionMaster
	if err := tx.Where("id = ? AND is_deleted = false", id).First(&question).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &question, nil
}

func (r *questionMasterRepository) FindAll(ctx context.Context, tx *gorm.DB) ([]entities.QuestionMaster, error) {
	var questions []entities.QuestionMaster
	if err := tx.Where("is_deleted = false").Find(&questions).Error; err != nil {
		return nil, err
	}
	return questions, nil
}

func (r *questionMasterRepository) FindActive(ctx context.Context, tx *gorm.DB) ([]entities.QuestionMaster, error) {
	var questions []entities.QuestionMaster
	if err := tx.Where("is_active = true AND is_deleted = false AND profile_only = true").Find(&questions).Error; err != nil {
		return nil, err
	}
	return questions, nil
}

func (r *questionMasterRepository) FindByQuestionTextAndLanguageID(ctx context.Context, tx *gorm.DB, questionText string, languageID int) (*entities.QuestionMaster, error) {
	var question entities.QuestionMaster
	if err := tx.Where("question_text = ? AND language_id = ? AND is_deleted = false", questionText, languageID).First(&question).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &question, nil
}

func (r *questionMasterRepository) Create(ctx context.Context, tx *gorm.DB, question *entities.QuestionMaster) error {
	return tx.Create(question).Error
}

func (r *questionMasterRepository) Update(ctx context.Context, tx *gorm.DB, question *entities.QuestionMaster) error {
	return tx.Save(question).Error
}

func (r *questionMasterRepository) Delete(ctx context.Context, tx *gorm.DB, id int) error {
	return tx.Model(&entities.QuestionMaster{}).Where("id = ?", id).Update("is_deleted", true).Error
}

func (r *questionMasterRepository) FindAllByLanguage(ctx context.Context, tx *gorm.DB, languageID *int) ([]entities.QuestionMaster, error) {
	var questions []entities.QuestionMaster
	if err := tx.Where("language_id = ? AND is_deleted = false", languageID).Find(&questions).Error; err != nil {
		return nil, err
	}
	return questions, nil
}

func (r *questionMasterRepository) FindActiveByLanguageWithUniquePoints(ctx context.Context, tx *gorm.DB, languageID int) ([]entities.QuestionMaster, error) {
	var questions []entities.QuestionMaster
	db := r.db
	if tx != nil {
		db = tx
	}

	subQuery := db.Model(&entities.QuestionMaster{}).
		Select("id, question_text, language_id, ques_point, is_active, is_deleted, created_by, created_on, last_modified_by, last_modified_on, ROW_NUMBER() OVER (PARTITION BY ques_point ORDER BY created_on DESC) as rn").
		Where("language_id = ? AND is_active = ? AND is_deleted = ?", languageID, true, false)

	query := db.Table("(?) as sub", subQuery).Where("rn = 1")

	err := query.Find(&questions).Error

	return questions, err
}

func (r *questionMasterRepository) FindActiveWithLanguageText(ctx context.Context, tx *gorm.DB, languageID int) ([]entities.QuestionMaster, error) {
	var questions []entities.QuestionMaster
	if err := tx.Where("language_id = ? AND is_active = true AND is_deleted = false", languageID).Find(&questions).Error; err != nil {
		return nil, err
	}
	return questions, nil
}

func (r *questionMasterRepository) CountActive(ctx context.Context, tx *gorm.DB) (int64, error) {
	var count int64
	if err := tx.Model(&entities.QuestionMaster{}).Where("is_active = true AND is_deleted = false").Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}