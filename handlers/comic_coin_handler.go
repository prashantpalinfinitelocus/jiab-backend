package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Infinite-Locus-Product/coca_cola_backend/constants"
	"github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
	_ "github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	"github.com/Infinite-Locus-Product/coca_cola_backend/services"
	_ "github.com/Infinite-Locus-Product/coca_cola_backend/utils"
)

// ComicCoinHandler interface for comic coin operations
type ComicCoinHandler interface {
	IncrementCoinCount(c *gin.Context)
	GetComicCoin(c *gin.Context)
}

type comicCoinHandler struct {
	comicCoinService services.ComicCoinService
}

// NewComicCoinHandler creates a new instance of ComicCoinHandler
func NewComicCoinHandler(comicCoinService services.ComicCoinService) ComicCoinHandler {
	return &comicCoinHandler{
		comicCoinService: comicCoinService,
	}
}

// GetComicCoins godoc
// @Summary Increment Comic Coins
// @Description Increment coin count for the current user
// @Tags comic-coins
// @Param type query string false "Type of increment (vote or other)"
// @Param genre query string false "Genre of the comic"
// @Param language query string false "Language of the comic"
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} utils.ResponseType[string] "Successfully incremented coin count"
// @Router /comic-coin/increment [post]
func (h *comicCoinHandler) IncrementCoinCount(c *gin.Context) {
	var req dtos.IncrementCoinCountRequestDTO
	if err := c.ShouldBindQuery(&req); err != nil {
		c.Error(err)
		return
	}

	user, _ := c.Get("user")
	userId := user.(*entities.User).ID

	wallet, err := h.comicCoinService.IncrementCoinCount(c, userId, constants.COIN_COUNT_INCREMENT_VALUE, req)
	if err != nil {
		c.Error(err)
		return
	}

	if req.Type != nil && *req.Type == "vote" {
		if wallet != nil {
			c.JSON(http.StatusOK, dtos.IncrementCoinCountResponseDTO{
				Message:           "Successfully incremented coin count",
				ShowCoinAnimation: true,
			})
		} else {
			c.JSON(http.StatusOK, dtos.IncrementCoinCountResponseDTO{
				Message:           "Successfully voted",
				ShowCoinAnimation: false,
			})
		}
	} else {
		c.JSON(http.StatusOK, gin.H{
			"message": "Successfully incremented coin count",
		})
	}
}

// GetComicCoin godoc
// @Summary Get Comic Coin Balance
// @Description Get the current user's comic coin balance
// @Tags comic-coins
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} utils.ResponseType[dtos.GetComicCoinResponseDTO] "Comic coin balance retrieved successfully"
// @Router /comic-coin [get]
func (h *comicCoinHandler) GetComicCoin(c *gin.Context) {
	user, _ := c.Get("user")
	userId := user.(*entities.User).ID

	wallet, err := h.comicCoinService.GetUserWalletWithoutTx(c, userId)
	if err != nil {
		c.Error(err)
		return
	}

	if wallet == nil {
		c.JSON(http.StatusOK, dtos.GetComicCoinResponseDTO{
			ComicCoin: 0,
			CreatedAt: time.Now(),
			UpdatedAt: nil,
		})
		return
	}

	response := dtos.GetComicCoinResponseDTO{
		ComicCoin: wallet.CurrentBalance,
		CreatedAt: wallet.CreatedOn,
		UpdatedAt: &wallet.WalletUpdatedOn,
	}

	c.JSON(http.StatusOK, response)
}
