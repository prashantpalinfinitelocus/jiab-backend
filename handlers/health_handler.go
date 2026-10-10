package handlers

import (
	"net/http"
	"os/exec"

	_ "github.com/Infinite-Locus-Product/coca_cola_backend/utils"
	"github.com/Infinite-Locus-Product/coca_cola_backend/version"
	"github.com/gin-gonic/gin"
)

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

type HealthHandler struct {
}

// HealthCheck godoc
// @Summary Health check endpoint
// @Description Returns a simple OK message to indicate the service is running
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} utils.ResponseType[string] "OK"
// @Router /health [get]
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	// ffmpegAvailable reports whether utils.ProcessGIF can use its preferred
	// GIF encoding path. This kept needing a guess during debugging - a
	// deployment could report the right commit while ffmpeg still silently
	// wasn't on PATH, and nothing surfaced that short of reading Cloud Run
	// logs - so it's answered directly here instead.
	_, err := exec.LookPath("ffmpeg")
	c.JSON(http.StatusOK, gin.H{
		"message":          "OK",
		"commit":           version.GitCommit,
		"ffmpeg_available": err == nil,
	})
}
