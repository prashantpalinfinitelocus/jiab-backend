package handlers

import (
	"net/http"

	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
	"github.com/Infinite-Locus-Product/coca_cola_backend/services"
	"github.com/Infinite-Locus-Product/coca_cola_backend/utils"
)

type JokeFormatHandler struct {
	formatService services.JokeFormatService
}

func NewJokeFormatHandler(formatService services.JokeFormatService) *JokeFormatHandler {
	return &JokeFormatHandler{
		formatService: formatService,
	}
}

// GetJokeFormats godoc
// @Summary Get all joke formats
// @Description Get a list of joke submission formats (text, image, audio, video) with icons
// @Tags Reference Data
// @Accept json
// @Produce json
// @Param search query string false "Search by value or title"
// @Param page query int false "Page number"
// @Param page_size query int false "Page size"
// @Param is_active query bool false "Filter by active status"
// @Success 200 {object} utils.ResponseType[dtos.JokeFormatPaginatedResponse] "Joke formats retrieved successfully"
// @Security BearerAuth
// @Router /reference-data/formats [get]
func (h *JokeFormatHandler) GetJokeFormats(c *gin.Context) {
	var filter dtos.JokeFormatFilter

	if err := c.ShouldBindQuery(&filter); err != nil {
		logrus.Errorf("invalid filter: %v", err)
		validationErrors := utils.FormatValidationErrors(err)
		c.Error(errs.NewValidationErrorWithFields("", validationErrors))
		return
	}

	response, err := h.formatService.GetJokeFormats(c.Request.Context(), filter)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, response)
}
