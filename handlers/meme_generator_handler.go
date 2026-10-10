package handlers

import (
	"fmt"
	"mime/multipart"
	"net/http"

	"github.com/Infinite-Locus-Product/coca_cola_backend/constants"
	"github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"github.com/Infinite-Locus-Product/coca_cola_backend/services"
	"github.com/Infinite-Locus-Product/coca_cola_backend/utils"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type MemeGeneratorHandler struct {
	service services.MemeGeneratorService
}

func NewMemeGeneratorHandler(service services.MemeGeneratorService) *MemeGeneratorHandler {
	return &MemeGeneratorHandler{service: service}
}

// resolveMemeFile prefers a client-uploaded "meme" file, falling back to
// fetching req.MemeURL server-side. This lets a client that already knows
// the template's URL (e.g. a Giphy CDN link the template picker just
// downloaded) skip re-uploading the same bytes to us - the common case that
// was making meme generation slow on large templates over mobile networks.
func resolveMemeFile(c *gin.Context, req dtos.MemeGeneratorRequest) (*multipart.FileHeader, error) {
	memeFile, err := c.FormFile("meme")
	if err == nil {
		return memeFile, nil
	}

	if req.MemeURL == "" {
		return nil, errs.NewBadRequestError("either meme file or meme_url is required", err)
	}

	return utils.FetchRemoteAssetAsFileHeader(req.MemeURL)
}

// @Summary Generate Meme
// @Description Process and generate a meme with text overlay. Returns a URL to the processed image. Backdrop and logo are loaded from static assets. Text is limited to 40 characters.
// @Tags Meme Generator
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param meme formData file false "Meme GIF or sticker (required unless meme_url is set)"
// @Param meme_url formData string false "URL of an already-hosted template (Giphy CDN or our GCS bucket) to fetch server-side instead of uploading the file"
// @Param top_text formData string false "Top text overlay (max 40 characters)"
// @Param bottom_text formData string false "Bottom text overlay (max 40 characters)"
// @Success 200 {object} dtos.MemeGeneratorResponse
// @Router /meme-generator/generate [post]
func (h *MemeGeneratorHandler) Generate(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.Error(errs.NewUnauthorizedError("User not authenticated", nil))
		return
	}
	userID := user.(*entities.User).ID

	var req dtos.MemeGeneratorRequest
	if err := c.ShouldBind(&req); err != nil {
		c.Error(errs.NewBadRequestError("Invalid form data", err))
		return
	}

	if len(req.TopText) > constants.MEME_TEXT_MAX_LENGTH {
		c.Error(errs.NewBadRequestError(fmt.Sprintf("Top text must not exceed %d characters", constants.MEME_TEXT_MAX_LENGTH), nil))
		return
	}

	if len(req.BottomText) > constants.MEME_TEXT_MAX_LENGTH {
		c.Error(errs.NewBadRequestError(fmt.Sprintf("Bottom text must not exceed %d characters", constants.MEME_TEXT_MAX_LENGTH), nil))
		return
	}

	memeFile, err := resolveMemeFile(c, req)
	if err != nil {
		c.Error(err)
		return
	}

	imageURL, err := h.service.GenerateMeme(c, userID, memeFile, req.TopText, req.BottomText)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, dtos.MemeGeneratorResponse{
		ImageURL: imageURL,
		Message:  "Meme generated successfully",
	})
}

// @Summary Generate Meme Stream
// @Description Process and generate a meme with text overlay. Returns the image directly. Backdrop and logo are loaded from static assets. Text is limited to 40 characters.
// @Tags Meme Generator
// @Accept multipart/form-data
// @Produce image/png
// @Produce image/gif
// @Security BearerAuth
// @Param meme formData file false "Meme GIF or sticker (required unless meme_url is set)"
// @Param meme_url formData string false "URL of an already-hosted template (Giphy CDN or our GCS bucket) to fetch server-side instead of uploading the file"
// @Param top_text formData string false "Top text overlay (max 40 characters)"
// @Param bottom_text formData string false "Bottom text overlay (max 40 characters)"
// @Success 200 {file} binary "Processed image"
// @Router /meme-generator/generate-stream [post]
func (h *MemeGeneratorHandler) GenerateStream(c *gin.Context) {
	var req dtos.MemeGeneratorRequest
	if err := c.ShouldBind(&req); err != nil {
		c.Error(errs.NewBadRequestError("Invalid form data", err))
		return
	}

	if len(req.TopText) > constants.MEME_TEXT_MAX_LENGTH {
		c.Error(errs.NewBadRequestError(fmt.Sprintf("Top text must not exceed %d characters", constants.MEME_TEXT_MAX_LENGTH), nil))
		return
	}

	if len(req.BottomText) > constants.MEME_TEXT_MAX_LENGTH {
		c.Error(errs.NewBadRequestError(fmt.Sprintf("Bottom text must not exceed %d characters", constants.MEME_TEXT_MAX_LENGTH), nil))
		return
	}

	logrus.WithFields(logrus.Fields{
		"meme_url": utils.SanitizeForLog(req.MemeURL),
		"top_text": utils.SanitizeForLog(req.TopText), "bottom_text": utils.SanitizeForLog(req.BottomText),
	}).Info("generate-stream request received")

	memeFile, err := resolveMemeFile(c, req)
	if err != nil {
		c.Error(err)
		return
	}

	logrus.WithFields(logrus.Fields{
		"filename": memeFile.Filename, "size_bytes": memeFile.Size,
	}).Info("generate-stream resolved meme file")

	imageData, contentType, err := h.service.GenerateMemeInMemory(c, memeFile, req.TopText, req.BottomText)
	if err != nil {
		c.Error(err)
		return
	}

	c.Data(http.StatusOK, contentType, imageData)
}
