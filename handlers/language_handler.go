package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"github.com/Infinite-Locus-Product/coca_cola_backend/services"
	"github.com/Infinite-Locus-Product/coca_cola_backend/utils"
	"github.com/sirupsen/logrus"
)

type LanguageHandler struct {
	languageService services.LanguageService
}

func NewLanguageHandler(languageService services.LanguageService) *LanguageHandler {
	return &LanguageHandler{
		languageService: languageService,
	}
}

// GetLanguages godoc
// @Summary Get all languages
// @Description Get a list of available languages with pagination and filtering
// @Tags Reference Data
// @Accept json
// @Produce json
// @Param search query string false "Search by name or language key"
// @Param page query int false "Page number"
// @Param page_size query int false "Page size"
// @Param is_active query bool false "Filter by active status"
// @Success 200 {object} utils.ResponseType[dtos.LanguagePaginatedResponse] "Languages retrieved successfully"
// @Failure 400 {object} utils.ErrorType "Bad Request"
// @Security BearerAuth
// @Router /reference-data/languages [get]
func (h *LanguageHandler) GetLanguages(c *gin.Context) {
	var filter dtos.LanguageFilter

	if err := c.ShouldBindQuery(&filter); err != nil {
		logrus.Errorf("invalid filter: %v", err)
		validationErrors := utils.FormatValidationErrors(err)
		c.Error(errs.NewValidationErrorWithFields("", validationErrors))
		return
	}

	response, err := h.languageService.GetLanguages(c.Request.Context(), filter)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetLanguageByID godoc
// @Summary Get language by ID
// @Description Get a specific language by its ID
// @Tags Reference Data
// @Accept json
// @Produce json
// @Param id path int true "Language ID"
// @Success 200 {object} utils.ResponseType[dtos.LanguageResponse] "Language retrieved successfully"
// @Failure 400 {object} utils.ErrorType "Bad Request"
// @Security BearerAuth
// @Router /reference-data/languages/{id} [get]
func (h *LanguageHandler) GetLanguageByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		logrus.Errorf("invalid language ID: %v", err)
		validationErrors := utils.FormatValidationErrors(err)
		c.Error(errs.NewValidationErrorWithFields("", validationErrors))
		return
	}

	language, err := h.languageService.GetLanguageByID(c.Request.Context(), id)
	if err != nil {
		c.Error(err)
		return
	}

	if language == nil {
		c.Error(errs.NewNotFoundError("Language not found", nil))
		return
	}

	c.JSON(http.StatusOK, language)
}
