package handlers

import (
	"net/http"
	"strings"
	"time"

	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"github.com/Infinite-Locus-Product/coca_cola_backend/utils"
	"github.com/gin-gonic/gin"
)

const defaultSignedURLExpiry = 24 * time.Hour

type GCSHandler struct {
	gcsService utils.GCSService
}

func NewGCSHandler(gcsService utils.GCSService) *GCSHandler {
	return &GCSHandler{
		gcsService: gcsService,
	}
}

// GetSignedURL godoc
// @Summary Get signed URL for GCS object
// @Description Returns a time-limited signed URL for the given GCS object key. Default expiry is 24 hours.
// @Tags gcs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param key query string true "GCS object path (e.g. UGCImages/UGC_detail_thumb.png)"
// @Success 200 {object} map[string]string "Signed URL in url field"
// @Failure 400 {object} map[string]interface{} "Invalid or missing key"
// @Failure 500 {object} map[string]interface{} "Failed to generate signed URL"
// @Router /gcs/signed-url [get]
func (h *GCSHandler) GetSignedURL(ctx *gin.Context) {
	key := strings.TrimSpace(ctx.Query("key"))
	if key == "" {
		ctx.Error(errs.NewBadRequestError("query parameter 'key' is required", nil))
		return
	}
	if strings.Contains(key, "..") {
		ctx.Error(errs.NewBadRequestError("invalid key: path traversal not allowed", nil))
		return
	}

	url, err := h.gcsService.GetFileSignedURL(ctx.Request.Context(), key, defaultSignedURLExpiry)
	if err != nil {
		ctx.Error(errs.NewInternalServerError("failed to generate signed URL", err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"url": url})
}
