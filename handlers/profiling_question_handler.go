package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"github.com/Infinite-Locus-Product/coca_cola_backend/services"
	"github.com/Infinite-Locus-Product/coca_cola_backend/utils"
)

type ProfilingQuestionHandler struct {
	userService services.UserService
}

func NewProfilingQuestionHandler(userService services.UserService) *ProfilingQuestionHandler {
	return &ProfilingQuestionHandler{
		userService: userService,
	}
}

// SubmitProfilingQuestionAnswer godoc
// @Summary Submit Profiling Question Answer
// @Description Submit answers to profiling questions
// @Tags profiling-questions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body []dtos.AnswerQuestionsRequestDTO true "Answer Questions Request"
// @Success 200 {object} utils.ResponseType[string] "Answers submitted successfully"
// @Failure 400 {object} utils.ErrorType "Invalid request body or missing required fields"
// @Failure 401 {object} utils.ErrorType "User not authenticated"
// @Failure 404 {object} utils.ErrorType "Question or option not found"
// @Failure 500 {object} utils.ErrorType "Internal server error"
// @Router /profiling-questions/submit [post]
func (h *ProfilingQuestionHandler) SubmitProfilingQuestionAnswer(ctx *gin.Context) {
	user, exists := ctx.Get("user")
	if !exists {
		ctx.Error(errs.NewUnauthorizedError("User not authenticated", fmt.Errorf("no user found in context")))
		return
	}

	userEntity := user.(*entities.User)

	var answers []dtos.AnswerQuestionsRequestDTO
	if err := ctx.ShouldBindJSON(&answers); err != nil {
		validationErrors := utils.FormatValidationErrors(err)
		ctx.Error(errs.NewValidationErrorWithFields("Invalid request body", validationErrors))
		return
	}

	if len(answers) == 0 {
		ctx.Error(errs.NewBadRequestError("At least one answer must be provided",
			fmt.Errorf("no answers provided for user %s", userEntity.ID)))
		return
	}

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
		if answer.QuestionNumber == 0 {
			ctx.Error(errs.NewBadRequestError("Question number is required",
				fmt.Errorf("missing question number for question %d of user %s", answer.QuestionID, userEntity.ID)))
			return
		}
	}

	err := h.userService.SubmitProfilingQuestionAnswer(ctx, userEntity.ID, answers)
	if err != nil {
		if err.Error() == "question not found" || err.Error() == "invalid option for this question" {
			ctx.Error(errs.NewNotFoundError("Question or option not found",
				fmt.Errorf("one or more questions/options not found for user %s", userEntity.ID)))
			return
		}
		ctx.Error(errs.NewInternalServerError("Failed to submit answers",
			fmt.Errorf("error submitting answers for user %s: %w", userEntity.ID, err)))
		return
	}

	ctx.JSON(http.StatusOK, "Answers submitted successfully")
}

// UpdateProfilingQuestionLastViewed godoc
// @Summary Update Profiling Question Last Viewed
// @Description Update the last viewed timestamp for a profiling question
// @Tags profiling-questions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param question_id path int true "Question ID"
// @Param question_number path int true "Question Number"
// @Success 200 {object} utils.ResponseType[string] "Last viewed updated successfully"
// @Failure 400 {object} utils.ErrorType "Invalid question_id or question_number"
// @Failure 401 {object} utils.ErrorType "User not authenticated"
// @Failure 404 {object} utils.ErrorType "Question not found"
// @Failure 500 {object} utils.ErrorType "Internal server error"
// @Router /profiling-questions/last-viewed/{question_id}/{question_number} [put]
func (h *ProfilingQuestionHandler) UpdateProfilingQuestionLastViewed(ctx *gin.Context) {
	user, exists := ctx.Get("user")
	if !exists {
		ctx.Error(errs.NewUnauthorizedError("User not authenticated", fmt.Errorf("no user found in context")))
		return
	}

	userEntity := user.(*entities.User)

	questionIDStr := ctx.Param("question_id")
	questionNumberStr := ctx.Param("question_number")
	questionNumber, err := strconv.Atoi(questionNumberStr)
	if err != nil {
		ctx.Error(errs.NewBadRequestError("Invalid question_number", fmt.Errorf("question_number must be an integer")))
		return
	}
	questionID, err := strconv.Atoi(questionIDStr)
	if err != nil {
		ctx.Error(errs.NewBadRequestError("Invalid question_id", fmt.Errorf("question_id must be an integer")))
		return
	}

	err = h.userService.UpdateProfilingQuestionLastViewed(ctx, userEntity.ID, questionID, questionNumber)
	if err != nil {
		if err.Error() == "question not found" {
			ctx.Error(errs.NewNotFoundError("Question not found", err))
			return
		}
		ctx.Error(errs.NewInternalServerError("Failed to update last viewed",
			fmt.Errorf("error updating last viewed for question %d: %w", questionID, err)))
		return
	}

	ctx.JSON(http.StatusOK, "Last viewed updated successfully")
}

// CheckProfilingQuestionModal godoc
// @Summary Check Profiling Question Modal
// @Description Check if the profiling question modal should be shown for a specific question
// @Tags profiling-questions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param question_id path int true "Question ID"
// @Param question_number query int true "Question Number"
// @Success 200 {object} utils.ResponseType[dtos.ProfilingQuestionModalResponseDTO] "Modal status"
// @Failure 400 {object} utils.ErrorType "Invalid question_id or question_number"
// @Failure 401 {object} utils.ErrorType "User not authenticated"
// @Failure 404 {object} utils.ErrorType "Question not found"
// @Failure 500 {object} utils.ErrorType "Internal server error"
// @Router /profiling-questions/check-modal/{question_id} [get]
func (h *ProfilingQuestionHandler) CheckProfilingQuestionModal(ctx *gin.Context) {
	user, exists := ctx.Get("user")
	if !exists {
		ctx.Error(errs.NewUnauthorizedError("User not authenticated", fmt.Errorf("no user found in context")))
		return
	}

	userEntity := user.(*entities.User)

	questionIDStr := ctx.Param("question_id")
	questionID, err := strconv.Atoi(questionIDStr)
	if err != nil {
		ctx.Error(errs.NewBadRequestError("Invalid question_id", fmt.Errorf("question_id must be an integer")))
		return
	}

	questionNumberStr := ctx.Query("question_number")
	questionNumber, err := strconv.Atoi(questionNumberStr)
	if err != nil || questionNumber <= 0 {
		ctx.Error(errs.NewBadRequestError("Invalid question_number", fmt.Errorf("question_number must be a positive integer")))
		return
	}

	response, err := h.userService.CheckProfilingQuestionModal(ctx, userEntity.ID, questionID, questionNumber)
	if err != nil {
		if err.Error() == "question not found" {
			ctx.Error(errs.NewNotFoundError("Question not found", err))
			return
		}
		ctx.Error(errs.NewInternalServerError("Failed to check modal status",
			fmt.Errorf("error checking modal status for question %d: %w", questionID, err)))
		return
	}

	ctx.JSON(http.StatusOK, response)
}