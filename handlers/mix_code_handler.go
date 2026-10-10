package handlers

import (
	"fmt"
	"net/http"

	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"github.com/Infinite-Locus-Product/coca_cola_backend/services"
	_ "github.com/Infinite-Locus-Product/coca_cola_backend/utils"
	"github.com/gin-gonic/gin"
)

type MixCodeHandler struct {
	mixCodeService *services.MixCodeService
}

func NewMixCodeHandler(mixCodeService *services.MixCodeService) *MixCodeHandler {
	return &MixCodeHandler{
		mixCodeService: mixCodeService,
	}
}

// @Summary Redeem a mix code
// @Description Redeem a mix code to get comic coins
// @Tags mix code
// @Accept json
// @Produce json
// @Param mix_code query string true "Mix code to redeem"
// @Security BearerAuth
// @Success 200 {object} utils.ResponseType[string] "Mix code redeemed successfully"
// @Failure 400 {object} utils.ErrorType "Bad Request"
// @Failure 500 {object} utils.ErrorType "Internal Server Error"
// @Router /mix-code/redeem [post]
func (h *MixCodeHandler) RedeemMixCode(ctx *gin.Context) {
	mixCode := ctx.Query("mix_code")
	if mixCode == "" {
		ctx.Error(errs.NewBadRequestError("Mix code is required", nil))
		return
	}

	user, exists := ctx.Get("user")
	if !exists {
		ctx.Error(errs.NewUnauthorizedError("User not authenticated", fmt.Errorf("no user found in context")))
		return
	}
	userEntity := user.(*entities.User)

	if err := h.mixCodeService.RedeemMixCode(ctx.Request.Context(), userEntity.ID, mixCode); err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Mix code redeemed successfully",
	})
}
