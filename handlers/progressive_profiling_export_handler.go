package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"github.com/Infinite-Locus-Product/coca_cola_backend/services"
)

type ProgressiveProfilingExportHandler interface {
	Export(c *gin.Context)
}

type progressiveProfilingExportHandler struct {
	exportService services.ProgressiveProfilingExportService
}

func NewProgressiveProfilingExportHandler(exportService services.ProgressiveProfilingExportService) ProgressiveProfilingExportHandler {
	return &progressiveProfilingExportHandler{exportService: exportService}
}

// Export godoc
// @Summary CDP backfill export for progressive-profiling answers
// @Description Builds an .xlsx workbook of every submitted progressive-profiling answer (bottle count, daypart preference, comedy genre preference — see constants.ProgressiveProfilingCDPEventPrefixes) as CDP-shaped events, uploads it to cloud storage, and returns a signed download URL. Excludes Buy_Now/partner events entirely.
// @Tags Analytics
// @Produce json
// @Success 200 {object} map[string]string "Signed URL to the .xlsx file in the url field"
// @Security ApiKeyAuth
// @Router /analytics/progressive-profiling/export [get]
func (h *progressiveProfilingExportHandler) Export(c *gin.Context) {
	url, err := h.exportService.Export(c.Request.Context())
	if err != nil {
		logrus.Errorf("progressive profiling CDP export failed: %v", err)
		c.Error(errs.NewInternalServerError("failed to generate progressive profiling CDP export", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"url": url})
}
