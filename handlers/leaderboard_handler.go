package handlers

import (
	"net/http"

	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"github.com/Infinite-Locus-Product/coca_cola_backend/services"
	_ "github.com/Infinite-Locus-Product/coca_cola_backend/utils"
	"github.com/gin-gonic/gin"
)

type LeaderboardHandler struct {
	leaderboardService services.LeaderboardService
}

func NewLeaderboardHandler(leaderboardService services.LeaderboardService) *LeaderboardHandler {
	return &LeaderboardHandler{
		leaderboardService: leaderboardService,
	}
}

// GetLeaderboard godoc
// @Summary Get Leaderboard
// @Description Get the top users for the leaderboard, with an optional date filter
// @Tags leaderboard
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param date query string false "Date in MM/DD/YYYY format"
// @Success 200 {object} dtos.LeaderboardResponseDTO "Leaderboard retrieved successfully"
// @Router /leaderboard [get]
func (h *LeaderboardHandler) GetLeaderboard(ctx *gin.Context) {
	user, exists := ctx.Get("user")
	if !exists {
		ctx.Error(errs.NewUnauthorizedError("User not authenticated", nil))
		return
	}

	userEntity := user.(*entities.User)
	date := ctx.Query("date")
	var datePtr *string
	if date != "" {
		datePtr = &date
	}

	leaderboard, err := h.leaderboardService.GetLeaderboard(ctx, userEntity, datePtr)
	if err != nil {
		ctx.Error(errs.NewInternalServerError("Failed to fetch leaderboard", err))
		return
	}

	ctx.JSON(http.StatusOK, leaderboard)
}
