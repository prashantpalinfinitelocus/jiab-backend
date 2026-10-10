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

type GenreHandler struct {
	genreService services.GenreService
}

func NewGenreHandler(genreService services.GenreService) *GenreHandler {
	return &GenreHandler{
		genreService: genreService,
	}
}

// GetGenres godoc
// @Summary Get all genres
// @Description Get a list of available joke genres with pagination and filtering
// @Tags Reference Data
// @Accept json
// @Produce json
// @Param search query string false "Search by genre name"
// @Param page query int false "Page number"
// @Param page_size query int false "Page size"
// @Param is_active query bool false "Filter by active status"
// @Param language_id query int false "Filter by language ID"
// @Success 200 {object} utils.ResponseType[dtos.GenrePaginatedResponse] "Genres retrieved successfully"
// @Security BearerAuth
// @Router /reference-data/genres [get]
func (h *GenreHandler) GetGenres(c *gin.Context) {
	var filter dtos.GenreFilter

	if err := c.ShouldBindQuery(&filter); err != nil {
		logrus.Errorf("invalid filter: %v", err)
		validationErrors := utils.FormatValidationErrors(err)
		c.Error(errs.NewValidationErrorWithFields("", validationErrors))
		return
	}

	response, err := h.genreService.GetGenres(c.Request.Context(), filter)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, utils.ResponseType[*dtos.GenrePaginatedResponse]{
		Success: true,
		Status:  http.StatusOK,
		Message: "Genres retrieved successfully",
		Data:    response,
	})
}

// GetGenreByID godoc
// @Summary Get genre by ID
// @Description Get a specific genre by its ID
// @Tags Reference Data
// @Accept json
// @Produce json
// @Param id path int true "Genre ID"
// @Success 200 {object} utils.ResponseType[dtos.JokeGenreResponse] "Genre retrieved successfully"
// @Failure 400 {object} utils.ErrorType "Bad Request"
// @Security BearerAuth
// @Router /reference-data/genres/{id} [get]
func (h *GenreHandler) GetGenreByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		logrus.Errorf("invalid genre ID: %v", err)
		validationErrors := utils.FormatValidationErrors(err)
		c.Error(errs.NewValidationErrorWithFields("", validationErrors))
		return
	}

	genre, err := h.genreService.GetGenreByID(c.Request.Context(), id)
	if err != nil {
		c.Error(err)
		return
	}

	if genre == nil {
		c.Error(errs.NewNotFoundError("Genre not found", nil))
		return
	}

	c.JSON(http.StatusOK, utils.ResponseType[*dtos.JokeGenreResponse]{
		Success: true,
		Status:  http.StatusOK,
		Message: "Genre retrieved successfully",
		Data:    genre,
	})
}
