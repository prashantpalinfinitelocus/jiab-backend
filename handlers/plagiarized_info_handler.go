package handlers

import (
	"net/http"

	"github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"github.com/Infinite-Locus-Product/coca_cola_backend/services"
	"github.com/Infinite-Locus-Product/coca_cola_backend/utils"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type PlagiarizedInfoHandler interface {
	CreatePlagiarizedInfo(c *gin.Context)
	ChangePlagiarizedInfoStatus(c *gin.Context)
}

type plagiarizedInfoHandler struct {
	plagiarizedInfoService services.PlagiarizedInfoService
}

func NewPlagiarizedInfoHandler(plagiarizedInfoService services.PlagiarizedInfoService) PlagiarizedInfoHandler {
	return &plagiarizedInfoHandler{
		plagiarizedInfoService: plagiarizedInfoService,
	}
}

// @Summary Create or Update Plagiarized Info
// @Description Create or update plagiarized information for a UGC asset
// @Tags Joke
// @Accept json
// @Produce json
// @Param request body dtos.PlagiarizedInfoRequest true "Plagiarized Info Request"
// @Success 200 {object} utils.ResponseType[dtos.PlagiarizedInfoResponse] "Plagiarized Info Response"
// @Security BearerAuth
// @Router /joke/ugc/plagiarize_content [post]
func (h *plagiarizedInfoHandler) CreatePlagiarizedInfo(c *gin.Context) {
	var request dtos.PlagiarizedInfoRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		logrus.Errorf("validation error: %v", err)
		validationErrors := utils.FormatValidationErrors(err)
		c.Error(errs.NewValidationErrorWithFields("Invalid request", validationErrors))
		return
	}

	userValue, exists := c.Get("user")
	if !exists {
		c.Error(errs.NewUnauthorizedError("Unauthorized", nil))
		return
	}
	user := userValue.(*entities.User)

	response, err := h.plagiarizedInfoService.CreatePlagiarizedInfo(c.Request.Context(), request, user.ID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, response)
}

// @Summary Change Plagiarized Info Status
// @Description Change the status of a plagiarized content
// @Tags Joke
// @Accept json
// @Produce json
// @Param request body dtos.ChangePlagiarizedInfoStatusRequest true "Change Plagiarized Info Status Request"
// @Success 200 {object} utils.ResponseType[string] "Plagiarized Info Response"
// @Security BearerAuth
// @Router /joke/plag/change-status [post]
func (h *plagiarizedInfoHandler) ChangePlagiarizedInfoStatus(c *gin.Context) {
	var request dtos.ChangePlagiarizedInfoStatusRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		logrus.Errorf("validation error: %v", err)
		validationErrors := utils.FormatValidationErrors(err)
		c.Error(errs.NewValidationErrorWithFields("Invalid request", validationErrors))
		return
	}

	err := h.plagiarizedInfoService.RemovePlagiarizedUGC(c.Request.Context(), request)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Plagiarized content successfully removed",
	})
}
