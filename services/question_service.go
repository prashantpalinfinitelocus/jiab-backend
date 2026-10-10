package services

import (
	"context"
	"fmt"
	"time"

	"github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	"github.com/Infinite-Locus-Product/coca_cola_backend/repository"
	"github.com/Infinite-Locus-Product/coca_cola_backend/utils"
)

type QuestionService interface {
	CreateQuestions(ctx context.Context, userID string, req dtos.CreateQuestionsRequestDTO) error
}

type questionService struct {
	txnManager         *utils.TransactionManager
	questionAnswerRepo repository.UserQuestionAnswerRepository
	questionMasterRepo repository.QuestionMasterRepository
	optionMasterRepo   repository.OptionMasterRepository
}

func NewQuestionService(
	txnManager *utils.TransactionManager,
	questionAnswerRepo repository.UserQuestionAnswerRepository,
	questionMasterRepo repository.QuestionMasterRepository,
	optionMasterRepo repository.OptionMasterRepository,
) QuestionService {
	return &questionService{
		txnManager:         txnManager,
		questionAnswerRepo: questionAnswerRepo,
		questionMasterRepo: questionMasterRepo,
		optionMasterRepo:   optionMasterRepo,
	}
}

func (s *questionService) CreateQuestions(ctx context.Context, userID string, req dtos.CreateQuestionsRequestDTO) error {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	for _, qDTO := range req.Questions {
		var questionID int
		isActive := true
		if qDTO.IsActive != nil {
			isActive = *qDTO.IsActive
		}

		if qDTO.ID != nil && *qDTO.ID > 0 {
			q, err := s.questionMasterRepo.FindByID(ctx, tx, *qDTO.ID)
			if err != nil {
				s.txnManager.AbortTxn(tx)
				return fmt.Errorf("failed to find question %d: %w", *qDTO.ID, err)
			}
			if q == nil {
				s.txnManager.AbortTxn(tx)
				return fmt.Errorf("question with id %d not found", *qDTO.ID)
			}
			q.QuestionText = qDTO.QuestionText
			q.QuesPoint = qDTO.QuesPoint
			q.LanguageID = qDTO.LanguageID
			q.IsActive = isActive

			now := time.Now()
			q.LastModifiedOn = &now
			q.LastModifiedBy = &userID

			if err := s.questionMasterRepo.Update(ctx, tx, q); err != nil {
				s.txnManager.AbortTxn(tx)
				return fmt.Errorf("failed to update question %d: %w", *qDTO.ID, err)
			}
			questionID = q.ID
		} else {
			q := &entities.QuestionMaster{
				QuestionText: qDTO.QuestionText,
				QuesPoint:    qDTO.QuesPoint,
				LanguageID:   qDTO.LanguageID,
				IsActive:     isActive,
				IsDeleted:    false,
				CreatedOn:    time.Now(),
				CreatedBy:    userID,
			}
			if err := s.questionMasterRepo.Create(ctx, tx, q); err != nil {
				s.txnManager.AbortTxn(tx)
				return fmt.Errorf("failed to create question: %w", err)
			}
			questionID = q.ID
		}

		for _, oDTO := range qDTO.Options {
			optActive := true
			if oDTO.IsActive != nil {
				optActive = *oDTO.IsActive
			}
			if oDTO.ID != nil && *oDTO.ID > 0 {
				opt, err := s.optionMasterRepo.FindByID(ctx, tx, *oDTO.ID)
				if err != nil {
					s.txnManager.AbortTxn(tx)
					return fmt.Errorf("failed to find option %d: %w", *oDTO.ID, err)
				}
				if opt == nil {
					s.txnManager.AbortTxn(tx)
					return fmt.Errorf("option with id %d not found", *oDTO.ID)
				}
				opt.OptionText = oDTO.OptionText
				opt.DisplayOrder = oDTO.DisplayOrder
				opt.IsActive = optActive

				now := time.Now()
				opt.LastModifiedOn = &now
				opt.LastModifiedBy = &userID

				if err := s.optionMasterRepo.Update(ctx, tx, opt); err != nil {
					s.txnManager.AbortTxn(tx)
					return fmt.Errorf("failed to update option %d: %w", *oDTO.ID, err)
				}
			} else {
				opt := &entities.OptionMaster{
					QuestionMasterID: questionID,
					OptionText:       oDTO.OptionText,
					DisplayOrder:     oDTO.DisplayOrder,
					IsActive:         optActive,
					IsDeleted:        false,
					CreatedOn:        time.Now(),
					CreatedBy:        userID,
				}
				if err := s.optionMasterRepo.Create(ctx, tx, opt); err != nil {
					s.txnManager.AbortTxn(tx)
					return fmt.Errorf("failed to create option: %w", err)
				}
			}
		}
	}

	s.txnManager.CommitTxn(tx)
	return nil
}