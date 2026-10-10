package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"github.com/Infinite-Locus-Product/coca_cola_backend/services"
)

type DSRHandler interface {
	HandleRequest(c *gin.Context)
}

type dsrHandler struct {
	dsrService services.DSRService
}

func NewDSRHandler(dsrService services.DSRService) DSRHandler {
	return &dsrHandler{dsrService: dsrService}
}

// HandleRequest godoc
// @Summary Fulfill a Data Subject Rights request
// @Description Called by OneTrust's privacy-portal automation to delete/anonymize, update, or export a user's data, matched by phone number.
// @Tags DSR
// @Accept json
// @Produce json
// @Param request body dtos.DSRRequestDTO true "DSR request"
// @Success 200 {object} map[string]interface{} "request_id and status in the data field"
// @Security ApiKeyAuth
// @Router /dsr/requests [post]
func (h *dsrHandler) HandleRequest(c *gin.Context) {
	var req dtos.DSRRequestDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(errs.NewBadRequestError("invalid request body", err))
		return
	}

	resp, err := h.dsrService.HandleRequest(c.Request.Context(), req)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, resp)
}
