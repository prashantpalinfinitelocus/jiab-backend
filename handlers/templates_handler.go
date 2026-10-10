package handlers

import (
	"net/http"

	_ "github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
	"github.com/Infinite-Locus-Product/coca_cola_backend/services"
	"github.com/Infinite-Locus-Product/coca_cola_backend/utils"
	"github.com/gin-gonic/gin"
)

type TemplatesHandler struct {
	service services.TemplatesService
}

func NewTemplatesHandler(service services.TemplatesService) *TemplatesHandler {
	return &TemplatesHandler{service: service}
}

// @Summary Get template autocomplete suggestions
// @Description Get template autocomplete suggestions based on query
// @Tags Templates
// @Accept json
// @Produce json
// @Param q query string false "Search query"
// @Param limit query int false "Limit number of results" default(25)
// @Param offset query int false "Offset for pagination" default(0)
// @Param type query string false "Template type (gifs, stickers, or all)" default(gifs)
// @Success 200 {object} []dtos.GiphyAutocompleteItem "Template autocomplete suggestions"
// @Router /templates/autocomplete [get]
func (h *TemplatesHandler) Autocomplete(c *gin.Context) {
	q := c.Query("q")
	limit := utils.ParseIntDefault(c.Query("limit"), 25)
	offset := utils.ParseIntDefault(c.Query("offset"), 0)
	templateType := c.Query("type")
	items, err := h.service.Autocomplete(q, limit, offset, templateType)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, items)
}

// @Summary Search templates
// @Description Search templates with filters
// @Tags Templates
// @Accept json
// @Produce json
// @Param q query string false "Search query"
// @Param limit query int false "Limit number of results" default(25)
// @Param offset query int false "Offset for pagination" default(0)
// @Param rating query string false "Rating filter"
// @Param lang query string false "Language filter"
// @Param bundle query string false "Bundle filter"
// @Param type query string false "Template type (gifs, stickers, or all)" default(gifs)
// @Success 200 {object} []dtos.GiphyTemplate "Search results"
// @Router /templates/search [get]
func (h *TemplatesHandler) Search(c *gin.Context) {
	q := c.Query("q")
	limit := utils.ParseIntDefault(c.Query("limit"), 25)
	offset := utils.ParseIntDefault(c.Query("offset"), 0)
	rating := c.Query("rating")
	lang := c.Query("lang")
	bundle := c.Query("bundle")
	templateType := c.Query("type")
	out, err := h.service.Search(q, limit, offset, rating, lang, bundle, templateType)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, out)
}

// @Summary Get trending templates
// @Description Get trending templates with filters
// @Tags Templates
// @Accept json
// @Produce json
// @Param limit query int false "Limit number of results" default(25)
// @Param offset query int false "Offset for pagination" default(0)
// @Param rating query string false "Rating filter"
// @Param bundle query string false "Bundle filter"
// @Param type query string false "Template type (gifs, stickers, or all)" default(gifs)
// @Success 200 {object} []dtos.GiphyTemplate "Trending templates"
// @Router /templates/trending [get]
func (h *TemplatesHandler) Trending(c *gin.Context) {
	limit := utils.ParseIntDefault(c.Query("limit"), 25)
	offset := utils.ParseIntDefault(c.Query("offset"), 0)
	rating := c.Query("rating")
	bundle := c.Query("bundle")
	templateType := c.Query("type")
	out, err := h.service.Trending(limit, offset, rating, bundle, templateType)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, out)
}
