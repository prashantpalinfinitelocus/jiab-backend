package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"github.com/Infinite-Locus-Product/coca_cola_backend/services"
	"github.com/Infinite-Locus-Product/coca_cola_backend/utils"
)

type QuestionHandler struct {
	userService     services.UserService
	questionService services.QuestionService
}

func NewQuestionHandler(userService services.UserService, questionService services.QuestionService) *QuestionHandler {
	return &QuestionHandler{
		userService:     userService,
		questionService: questionService,
	}
}

// GetQuestions godoc
// @Summary Get Questions
// @Description Get all active questions with user's answers
// @Tags profile
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} utils.ResponseType[[]dtos.QuestionResponseDTO] "Questions retrieved successfully"
// @Router /profile/questions [get]
func (h *QuestionHandler) GetQuestions(ctx *gin.Context) {
	user, exists := ctx.Get("user")
	if !exists {
		ctx.Error(errs.NewUnauthorizedError("User not authenticated", fmt.Errorf("no user found in context")))
		return
	}

	userEntity := user.(*entities.User)

	languageIDStr, exists := ctx.GetQuery("language_id")
	if !exists {
		ctx.Error(errs.NewBadRequestError("Language ID is required", fmt.Errorf("language_id is required")))
		return
	}

	languageID, err := strconv.Atoi(languageIDStr)
	if err != nil {
		ctx.Error(errs.NewBadRequestError("Invalid language_id", fmt.Errorf("language_id must be an integer")))
		return
	}

	questions, err := h.userService.GetQuestions(ctx, userEntity.ID, languageID)
	if err != nil {
		ctx.Error(errs.NewInternalServerError("Failed to fetch questions", fmt.Errorf("error fetching questions for user %s: %w", userEntity.ID, err)))
		return
	}

	ctx.JSON(http.StatusOK, questions)
}

// AnswerQuestions godoc
// @Summary Answer Questions
// @Description Submit answers to multiple questions
// @Tags profile
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body []dtos.AnswerQuestionsRequestDTO true "Answer Questions Request"
// @Success 200 {object} utils.ResponseType[string] "Answers submitted successfully"
// @Router /profile/questions [post]
func (h *QuestionHandler) AnswerQuestions(ctx *gin.Context) {
	user, exists := ctx.Get("user")
	if !exists {
		ctx.Error(errs.NewUnauthorizedError("User not authenticated", fmt.Errorf("no user found in context")))
		return
	}

	userEntity := user.(*entities.User)

	var answers []dtos.AnswerQuestionsRequestDTO
	if err := ctx.ShouldBindJSON(&answers); err != nil {
		logrus.Errorf("validation error: %v", err)
		validationErrors := utils.FormatValidationErrors(err)
		ctx.Error(errs.NewValidationErrorWithFields("Invalid request body", validationErrors))
		return
	}

	// Validate that at least one answer is provided
	if len(answers) == 0 {
		ctx.Error(errs.NewBadRequestError("At least one answer must be provided",
			fmt.Errorf("no answers provided for user %s", userEntity.ID)))
		return
	}

	// Validate each answer
	for _, answer := range answers {
		if answer.QuestionID == 0 {
			ctx.Error(errs.NewBadRequestError("Question ID is required",
				fmt.Errorf("missing question ID in answer for user %s", userEntity.ID)))
			return
		}
		if answer.AnswerID == 0 {
			ctx.Error(errs.NewBadRequestError("Answer ID is required",
				fmt.Errorf("missing answer ID for question %d of user %s", answer.QuestionID, userEntity.ID)))
			return
		}
	}

	err := h.userService.AnswerQuestions(ctx, userEntity.ID, answers)
	if err != nil {
		if err.Error() == "question not found" {
			ctx.Error(errs.NewNotFoundError("Question not found",
				fmt.Errorf("one or more questions not found for user %s", userEntity.ID)))
			return
		}
		ctx.Error(errs.NewInternalServerError("Failed to submit answers",
			fmt.Errorf("error submitting answers for user %s: %w", userEntity.ID, err)))
		return
	}

	ctx.JSON(http.StatusOK, "Answers submitted successfully")
}

// GetQuestionByID godoc
// @Summary Get Question By Text
// @Description Get a question by its text and language ID
// @Tags profile
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dtos.GetQuestionByTextRequestDTO true "Get Question By Text Request"
// @Success 200 {object} utils.ResponseType[dtos.QuestionResponseDTO] "Question retrieved successfully"
// @Failure 400 {object} utils.ErrorType "Invalid request body"
// @Failure 404 {object} utils.ErrorType "Question not found"
// @Failure 500 {object} utils.ErrorType "Internal server error"
// @Router /profile/questions/text [post]
func (h *QuestionHandler) GetQuestionByID(ctx *gin.Context) {
	var req dtos.GetQuestionByTextRequestDTO
	if err := ctx.ShouldBindJSON(&req); err != nil {
		logrus.Errorf("validation error: %v", err)
		validationErrors := utils.FormatValidationErrors(err)
		ctx.Error(errs.NewValidationErrorWithFields("Invalid request body", validationErrors))
		return
	}

	questionID, err := h.userService.GetQuestionIDByText(ctx, req.QuestionText, req.LanguageID)
	if err != nil {
		if err.Error() == "question not found" {
			ctx.Error(errs.NewNotFoundError("Question not found", err))
			return
		}
		ctx.Error(errs.NewInternalServerError("Failed to find question", fmt.Errorf("error finding question by text: %w", err)))
		return
	}

	question, err := h.userService.GetQuestionByID(ctx, questionID, req.LanguageID)
	if err != nil {
		if err.Error() == "question not found" {
			ctx.Error(errs.NewNotFoundError("Question not found", err))
			return
		}
		ctx.Error(errs.NewInternalServerError("Failed to fetch question", fmt.Errorf("error fetching question %d: %w", questionID, err)))
		return
	}

	ctx.JSON(http.StatusOK, question)
}

// CreateQuestions godoc
// @Summary Create Questions
// @Description Create or update questions and options
// @Tags profile
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dtos.CreateQuestionsRequestDTO true "Create Questions Request"
// @Success 200 {object} utils.ResponseType[string] "Questions created successfully"
// @Router /profile/questions/create [post]
func (h *QuestionHandler) CreateQuestions(ctx *gin.Context) {
	user, exists := ctx.Get("user")
	if !exists {
		ctx.Error(errs.NewUnauthorizedError("User not authenticated", fmt.Errorf("no user found in context")))
		return
	}

	userEntity := user.(*entities.User)

	var req dtos.CreateQuestionsRequestDTO
	if err := ctx.ShouldBindJSON(&req); err != nil {
		logrus.Errorf("validation error: %v", err)
		validationErrors := utils.FormatValidationErrors(err)
		ctx.Error(errs.NewValidationErrorWithFields("Invalid request body", validationErrors))
		return
	}

	if err := h.questionService.CreateQuestions(ctx, userEntity.ID, req); err != nil {
		ctx.Error(errs.NewInternalServerError("Failed to create questions", err))
		return
	}

	ctx.JSON(http.StatusOK, "Questions created successfully")
}
