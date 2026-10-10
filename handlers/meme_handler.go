package handlers

import (
	"net/http"

	"github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"github.com/Infinite-Locus-Product/coca_cola_backend/services"
	"github.com/gin-gonic/gin"
)

type MemeHandler struct {
	service services.MemeService
}

func NewMemeHandler(service services.MemeService) *MemeHandler { return &MemeHandler{service: service} }

// @Summary Submit Meme
// @Description Validate inputs, upload to GCS, persist as PENDING
// @Tags Memes
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param language_id formData int true "Language ID"
// @Param template_id formData string true "Template ID"
// @Param text1 formData string false "Top Text"
// @Param text2 formData string false "Bottom Text"
// @Param asset formData file true "Meme asset"
// @Success 200 {object} dtos.MemeSubmitResponseData
// @Router /memes [post]
func (h *MemeHandler) Submit(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.Error(errs.NewUnauthorizedError("User not authenticated", nil))
		return
	}
	userID := user.(*entities.User).ID

	var req dtos.MemeSubmitRequest
	if err := c.ShouldBind(&req); err != nil {
		c.Error(errs.NewBadRequestError("Invalid form data", err))
		return
	}

	file, err := c.FormFile("asset")
	if err != nil {
		c.Error(errs.NewBadRequestError("asset file is required", err))
		return
	}

	meme, err := h.service.Submit(c, userID, req.LanguageID, req.TemplateID, req.Text1, req.Text2, file)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, dtos.MemeSubmitResponseData{
		MemeID: meme.ID,
		Status: "Pending",
	})
}

// @Summary Fetch Feed
// @Description Paginated approved memes (recency + reaction score); optional filters
// @Tags Memes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param language query int false "Language ID"
// @Param q query string false "Search"
// @Param limit query int false "Limit" default(15)
// @Param cursor query string false "Cursor"
// @Param trending query bool false "Sort by most viewed (trending)"
// @Success 200 {object} dtos.MemeFeedResponse
// @Router /memes/feed [get]
func (h *MemeHandler) Feed(c *gin.Context) {
	var req dtos.MemeFeedRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.Error(errs.NewBadRequestError("Invalid query parameters", err))
		return
	}

	userID := ""
	if user, exists := c.Get("user"); exists {
		userID = user.(*entities.User).ID
	}

	limit := 15
	if req.Limit != nil {
		limit = *req.Limit
	}

	results, next, err := h.service.Feed(c, userID, req.Language, req.Q, limit, req.Cursor, req.Trending)
	if err != nil {
		c.Error(err)
		return
	}

	feedItems := make([]dtos.MemeFeedItem, len(results))
	for i, result := range results {
		feedItems[i] = dtos.MemeFeedItem{
			ID:        result.Meme.ID,
			AssetURL:  result.AssetURL,
			Language:  result.LanguageCode,
			ViewCount: result.Meme.ViewCount,
			ReactionCounts: dtos.ReactionCounts{
				Happy:   result.ReactionCounts.Happy,
				Sad:     result.ReactionCounts.Sad,
				Neutral: result.ReactionCounts.Neutral,
			},
			ReactionClick: dtos.ReactionClick{
				Happy:   result.ReactionClick.Happy,
				Sad:     result.ReactionClick.Sad,
				Neutral: result.ReactionClick.Neutral,
			},
			HasVoted:    result.HasVoted,
			Username:    result.Username,
			AvatarImage: result.AvatarImage,
		}
	}

	c.JSON(http.StatusOK, dtos.MemeFeedResponse{
		Items:  feedItems,
		Cursor: next,
	})
}

// @Summary Get User Memes
// @Description Fetch paginated memes for authenticated user with optional status filter
// @Tags Memes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param status query string false "Meme status filter"
// @Param limit query int false "Limit" default(15)
// @Param cursor query string false "Cursor"
// @Success 200 {object} dtos.UserMemesResponse
// @Router /memes/user [get]
func (h *MemeHandler) GetUserMemes(c *gin.Context) {
	var req dtos.UserMemesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.Error(errs.NewBadRequestError("Invalid query parameters", err))
		return
	}

	user, exists := c.Get("user")
	if !exists {
		c.Error(errs.NewUnauthorizedError("User not authenticated", nil))
		return
	}
	userID := user.(*entities.User).ID

	limit := 15
	if req.Limit != nil {
		limit = *req.Limit
	}

	results, next, err := h.service.GetUserMemes(c, userID, req.Status, limit, req.Cursor)
	if err != nil {
		c.Error(err)
		return
	}

	memeItems := make([]dtos.UserMemesItem, len(results))
	for i, result := range results {
		memeItems[i] = dtos.UserMemesItem{
			ID:        result.Meme.ID,
			AssetURL:  result.AssetURL,
			Language:  result.LanguageCode,
			Status:    result.Meme.Status,
			ViewCount: result.Meme.ViewCount,
			ReactionCounts: dtos.ReactionCounts{
				Happy:   result.ReactionCounts.Happy,
				Sad:     result.ReactionCounts.Sad,
				Neutral: result.ReactionCounts.Neutral,
			},
			ReactionClick: dtos.ReactionClick{
				Happy:   result.ReactionClick.Happy,
				Sad:     result.ReactionClick.Sad,
				Neutral: result.ReactionClick.Neutral,
			},
			HasVoted:    result.HasVoted,
			CreatedAt:   result.Meme.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			Username:    result.Username,
			AvatarImage: result.AvatarImage,
		}
	}

	c.JSON(http.StatusOK, dtos.UserMemesResponse{
		Items:  memeItems,
		Cursor: next,
	})
}

// @Summary Fetch Meme Detail
// @Description Public detail for sharing; includes reactions
// @Tags Memes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param meme_id path string true "Meme ID"
// @Success 200 {object} dtos.MemeDetailResponse
// @Router /memes/{meme_id} [get]
func (h *MemeHandler) Detail(c *gin.Context) {
	memeID := c.Param("meme_id")

	userID := ""
	if user, exists := c.Get("user"); exists {
		userID = user.(*entities.User).ID
	}

	result, err := h.service.Detail(c, userID, memeID)
	if err != nil {
		c.Error(err)
		return
	}

	detail := dtos.MemeDetailResponse{
		ID:        result.Meme.ID,
		Status:    result.Meme.Status,
		AssetURL:  result.AssetURL,
		Language:  result.LanguageCode,
		ViewCount: result.Meme.ViewCount,
		ReactionCounts: dtos.ReactionCounts{
			Happy:   result.ReactionCounts.Happy,
			Sad:     result.ReactionCounts.Sad,
			Neutral: result.ReactionCounts.Neutral,
		},
		ReactionClick: dtos.ReactionClick{
			Happy:   result.ReactionClick.Happy,
			Sad:     result.ReactionClick.Sad,
			Neutral: result.ReactionClick.Neutral,
		},
		HasVoted: result.HasVoted,
	}

	c.JSON(http.StatusOK, detail)
}

// @Summary Vote on Meme
// @Description Enforce unique vote, award points
// @Tags Memes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dtos.MemeVoteRequest true "Vote payload"
// @Success 200 {object} dtos.IncrementCoinCountResponseDTO
// @Router /memes/vote [post]
func (h *MemeHandler) Vote(c *gin.Context) {
	userValue, _ := c.Get("user")
	userID := userValue.(*entities.User).ID

	var req dtos.MemeVoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(errs.NewBadRequestError("invalid payload", err))
		return
	}

	resp, err := h.service.Vote(c, userID, req.AssetID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, *resp)
}

// @Summary React on Meme
// @Description Enforce unique reaction, award points
// @Tags Memes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param meme_id path string true "Meme ID"
// @Param request body dtos.MemeReactRequest true "Reaction payload"
// @Success 200 {object} dtos.IncrementCoinCountResponseDTO
// @Router /memes/react/{meme_id} [post]
func (h *MemeHandler) React(c *gin.Context) {
	userValue, _ := c.Get("user")
	userID := userValue.(*entities.User).ID
	memeID := c.Param("meme_id")

	var req dtos.MemeReactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(errs.NewBadRequestError("invalid payload", err))
		return
	}

	resp, err := h.service.React(c, userID, memeID, req.Reaction)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, *resp)
}

// @Summary Increment View Count
// @Description Increment view count for a meme
// @Tags Memes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param meme_id path string true "Meme ID"
// @Success 200 {object} dtos.MemeIncrementViewResponse
// @Router /memes/{meme_id}/view [post]
func (h *MemeHandler) IncrementView(c *gin.Context) {
	memeID := c.Param("meme_id")

	userID := ""
	if user, exists := c.Get("user"); exists {
		userID = user.(*entities.User).ID
	}

	resp, err := h.service.IncrementView(c, userID, memeID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, *resp)
}
