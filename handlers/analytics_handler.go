package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"github.com/Infinite-Locus-Product/coca_cola_backend/services"
)

type AnalyticsHandler interface {
	GetTokensByDate(c *gin.Context)
}

type analyticsHandler struct {
	notificationService services.NotificationService
}

func NewAnalyticsHandler(notificationService services.NotificationService) AnalyticsHandler {
	return &analyticsHandler{
		notificationService: notificationService,
	}
}

// @Summary Get FCM device tokens by date
// @Description Get all FCM device tokens that were last modified on a specific date (for analytics)
// @Tags Analytics
// @Accept json
// @Produce json
// @Param date query string true "Date in YYYY-MM-DD format (e.g., 2025-10-08)"
// @Success 200 {object} dtos.GetTokensByDateResponseDTO "Success"
// @Failure 400 {object} map[string]string "Bad Request"
// @Security ApiKeyAuth
// @Router /analytics/tokens-by-date [get]
func (h *analyticsHandler) GetTokensByDate(c *gin.Context) {
	date := c.Query("date")
	if date == "" {
		c.Error(errs.NewBadRequestError("date query parameter is required", nil))
		return
	}

	response, err := h.notificationService.GetTokensByDate(c.Request.Context(), date)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, response)
}

