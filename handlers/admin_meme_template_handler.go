package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"github.com/Infinite-Locus-Product/coca_cola_backend/services"
)

type AdminMemeTemplateHandler struct {
	service services.MemeTemplateAdminService
}

func NewAdminMemeTemplateHandler(service services.MemeTemplateAdminService) *AdminMemeTemplateHandler {
	return &AdminMemeTemplateHandler{service: service}
}

// @Summary Bulk-register meme templates
// @Description Registers already-uploaded GCS objects as meme templates (upload and registration are separate steps)
// @Tags Admin Meme Templates
// @Accept json
// @Produce json
// @Param request body dtos.BulkRegisterMemeTemplatesRequest true "Templates to register"
// @Success 200 {object} dtos.BulkRegisterMemeTemplatesResponse "Registration summary"
// @Router /admin/meme-templates/bulk-register [post]
func (h *AdminMemeTemplateHandler) BulkRegister(c *gin.Context) {
	var req dtos.BulkRegisterMemeTemplatesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(errs.NewBadRequestError("Invalid request body", err))
		return
	}

	resp, err := h.service.BulkRegister(c.Request.Context(), req)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, resp)
}
