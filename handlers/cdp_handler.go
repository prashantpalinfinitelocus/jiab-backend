package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"github.com/Infinite-Locus-Product/coca_cola_backend/services"
	"github.com/Infinite-Locus-Product/coca_cola_backend/utils"
)

type CDPHandler struct {
	cdpService services.CDPService
}

func NewCDPHandler(cdpService services.CDPService) *CDPHandler {
	return &CDPHandler{
		cdpService: cdpService,
	}
}

func (h *CDPHandler) SendEvent(c *gin.Context) {
	var req dtos.CDPEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logrus.Errorf("validation error: %v", err)
		validationErrors := utils.FormatValidationErrors(err)
		c.Error(errs.NewValidationErrorWithFields("", validationErrors))
		return
	}

	response, err := h.cdpService.SendEvent(c.Request.Context(), req)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, response)
}
