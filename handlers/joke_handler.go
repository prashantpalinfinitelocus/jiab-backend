package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"path/filepath"
	"slices"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/Infinite-Locus-Product/coca_cola_backend/constants"
	"github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"github.com/Infinite-Locus-Product/coca_cola_backend/services"
	"github.com/Infinite-Locus-Product/coca_cola_backend/utils"
)

var imageFileExtensions = []string{".jpg", ".jpeg", ".png"}
var audioFileExtensions = []string{".mp3", ".wav", ".webm"}
var videoFileExtensions = []string{".mp4", ".mov"}

type JokeHandler interface {
	GetSupriseJoke(c *gin.Context)
	GetJokes(c *gin.Context)
	GetSpicyTrayJokes(c *gin.Context)
	SubmitUGC(c *gin.Context)
	GetUGCContentByIDs(c *gin.Context)
	GetProfileUGCContent(c *gin.Context)
	GetUGCContentArtist(c *gin.Context)
	WAGetJoke(c *gin.Context)
	WAGetJokeGenres(c *gin.Context)
	WAUGCUpload(c *gin.Context)
	AddUserReactPoint(c *gin.Context)
	AddUserReelReaction(c *gin.Context)
	ChangeUGCStatus(c *gin.Context)
	ActivateJokeConsumption(c *gin.Context)
	GetUGCAssessmentContent(c *gin.Context)
}

type jokeHandler struct {
	jokeService      services.JokeService
	comicCoinService services.ComicCoinService
	reactionService  services.ReactionService
}

func NewJokeHandler(jokeService services.JokeService, comicCoinService services.ComicCoinService, reactionService services.ReactionService) JokeHandler {
	return &jokeHandler{
		jokeService:      jokeService,
		comicCoinService: comicCoinService,
		reactionService:  reactionService,
	}
}

// @Summary Get surprise me jokes (Limit 3 per day)
// @Description Get a joke from the Gluedin API
// @Tags Joke
// @Accept json
// @Produce json
// @Param genre query int false "Genre ID"
// @Param language query int false "Language ID"
// @Param exclude query string false "Exclude joke IDs"
// @Success 200 {object} utils.ResponseType[dtos.JokeResponseDTO] "Joke"
// @Security BearerAuth
// @Router /joke/surprise-me [get]
func (h *jokeHandler) GetSupriseJoke(c *gin.Context) {
	var jokeDTO dtos.JokeDTO
	if err := c.ShouldBindQuery(&jokeDTO); err != nil {
		c.Error(errs.NewBadRequestError("Invalid request", err))
		return
	}

	userValue, exists := c.Get("user")
	if !exists {
		c.Error(errs.NewUnauthorizedError("Unauthorized", nil))
		return
	}
	user := userValue.(*entities.User)

	jokeReponse, _, err := h.jokeService.GetSurpriseJoke(c.Request.Context(), user.ID, &jokeDTO, constants.WEB_PLATFORM)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, jokeReponse)
}

// @Summary Get jokes
// @Description Get jokes
// @Tags Joke
// @Accept json
// @Produce json
// @Param limit query int false "Limit"
// @Param selected_joke query string false "Selected joke ID"
// @Param preferred_jokes query string false "Preferred jokes IDs"
// @Param language query int true "Language ID"
// @Param type query string false "Type"
// @Success 200 {object} utils.ResponseType[[]dtos.JokeResponseDTO] "Joke"
// @Security BearerAuth
// @Router /joke/jokes [get]
func (h *jokeHandler) GetJokes(c *gin.Context) {
	var jokeDTO dtos.JokeScrollDTO
	if err := c.ShouldBindQuery(&jokeDTO); err != nil {
		c.Error(errs.NewBadRequestError("Invalid request", err))
		return
	}

	var userID *string
	userValue, exists := c.Get("user")
	if exists {
		if user, ok := userValue.(*entities.User); ok && user != nil {
			userID = &user.ID
		}
	}

	var jokeResponse []dtos.JokeResponseDTO
	var err error
	if jokeDTO.IsRefreshRing != nil && *jokeDTO.IsRefreshRing {
		jokeResponse, err = h.jokeService.GetRefreshRingJourney(c.Request.Context(), userID, &jokeDTO)
	} else {
		jokeResponse, err = h.jokeService.GetScrollAndLolJokes(c.Request.Context(), userID, &jokeDTO)
	}
	if err != nil {
		c.Error(err)
		return
	}

	if len(jokeResponse) == 0 {
		c.JSON(http.StatusOK, []dtos.JokeResponseDTO{})
		return
	}

	c.JSON(http.StatusOK, jokeResponse)
}

// @Summary Get spicy tray jokes
// @Description Get spicy tray jokes
// @Tags Joke
// @Accept json
// @Produce json
// @Param limit query int false "Limit"
// @Param language query int true "Language ID"
// @Success 200 {object} utils.ResponseType[[]dtos.JokeResponseDTO] "Joke"
// @Security BearerAuth
// @Router /joke/spicy-tray [get]
func (h *jokeHandler) GetSpicyTrayJokes(c *gin.Context) {
	var jokeDTO dtos.JokeScrollDTO
	if err := c.ShouldBindQuery(&jokeDTO); err != nil {
		c.Error(errs.NewBadRequestError("Invalid request", err))
		return
	}

	var userID *string
	userValue, exists := c.Get("user")
	if exists {
		if user, ok := userValue.(*entities.User); ok && user != nil {
			userID = &user.ID
		}
	}

	jokes, err := h.jokeService.GetSpicyTrayJokes(c.Request.Context(), userID, &jokeDTO)
	if err != nil {
		c.Error(err)
		return
	}
	if len(jokes) == 0 {
		c.JSON(http.StatusOK, []dtos.JokeResponseDTO{})
		return
	}
	c.JSON(http.StatusOK, jokes)
}

// @Summary Submit UGC Jokes
// @Description Submit UGC Jokes
// @Tags Joke
// @Accept multipart/form-data
// @Produce json
// @Param jokes formData dtos.JokeIngestionDTO true "Jokes"
// @Success 200 {object} utils.ResponseType[string] "Joke"
// @Security BearerAuth
// @Router /joke/ugc/submit [post]
func (h *jokeHandler) SubmitUGC(c *gin.Context) {
	var joke dtos.JokeIngestionDTO
	if err := c.ShouldBind(&joke); err != nil {
		logrus.Errorf("validation error: %v", err)
		validationErrors := utils.FormatValidationErrors(err)
		c.Error(errs.NewValidationErrorWithFields("", validationErrors))
		return
	}

	userValue, _ := c.Get("user")
	user := userValue.(*entities.User)

	if joke.File != nil {
		fileExtension := filepath.Ext(joke.File.Filename)

		if joke.Format == "Image" && !slices.Contains(imageFileExtensions, fileExtension) {
			c.Error(errs.NewBadRequestError("Invalid file extension", nil))
			return
		} else if joke.Format == "Audio" && !slices.Contains(audioFileExtensions, fileExtension) {
			c.Error(errs.NewBadRequestError("Invalid file extension", nil))
			return
		} else if joke.Format == "Video" && !slices.Contains(videoFileExtensions, fileExtension) {
			c.Error(errs.NewBadRequestError("Invalid file extension", nil))
			return
		}
	}

	err := h.jokeService.SubmitUGC(c.Request.Context(), user.ID, joke, constants.WEB_PLATFORM)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Jokes submitted successfully"})
}

// @Summary Get UGC Content
// @Description Get UGC Content
// @Tags Joke
// @Accept json
// @Produce json
// @Param ids body dtos.UGCContentByIDsDTO true "UGC IDs"
// @Success 200 {object} utils.ResponseType[[]dtos.UGCContentByIDsResponseDTO] "UGC Content"
// @Security BearerAuth
// @Router /joke/ugc/content-by-ids [post]
func (h *jokeHandler) GetUGCContentByIDs(c *gin.Context) {
	var ugcContentByIDsDTO dtos.UGCContentByIDsDTO
	if err := c.ShouldBindJSON(&ugcContentByIDsDTO); err != nil {
		logrus.Errorf("validation error: %v", err)
		validationErrors := utils.FormatValidationErrors(err)
		c.Error(errs.NewValidationErrorWithFields("", validationErrors))
		return
	}

	ugcContent, err := h.jokeService.GetUGCContentByIDs(c.Request.Context(), ugcContentByIDsDTO.IDs)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, ugcContent)
}

// @Summary Get Profile UGC Content
// @Description Get Profile UGC Content
// @Tags Joke
// @Accept json
// @Produce json
// @Success 200 {object} utils.ResponseType[[]dtos.UGCContentResponseDTO] "UGC Content"
// @Security BearerAuth
// @Router /joke/ugc/profile-content [get]
func (h *jokeHandler) GetProfileUGCContent(c *gin.Context) {
	userValue, _ := c.Get("user")
	user := userValue.(*entities.User)

	ugcContent, err := h.jokeService.GetProfileUGCContent(c.Request.Context(), user.ID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, ugcContent)
}

// @Summary Get UGC Content Artist
// @Description Get UGC Content Artist
// @Tags Joke
// @Accept json
// @Produce json
// @Param ids body dtos.UGCContentByIDsDTO true "UGC IDs"
// @Success 200 {object} utils.ResponseType[[]dtos.UGCContentArtistDTO] "UGC Content Artist"
// @Security BearerAuth
// @Router /joke/ugc/content-artist [post]
func (h *jokeHandler) GetUGCContentArtist(c *gin.Context) {
	var ugcContentByIDsDTO dtos.UGCContentByIDsDTO
	if err := c.ShouldBindJSON(&ugcContentByIDsDTO); err != nil {
		logrus.Errorf("validation error: %v", err)
		validationErrors := utils.FormatValidationErrors(err)
		c.Error(errs.NewValidationErrorWithFields("", validationErrors))
		return
	}

	ugcContent, err := h.jokeService.GetUGCContentArtist(c.Request.Context(), ugcContentByIDsDTO.IDs)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, ugcContent)
}

func (h *jokeHandler) WAGetJoke(c *gin.Context) {
	var jokeDTO dtos.WAJokeDTO
	if err := c.ShouldBindQuery(&jokeDTO); err != nil {
		c.JSON(400, gin.H{
			"data":            nil,
			"failed":          true,
			"message":         "Invalid request",
			"succeeded":       false,
			"statusCode":      400,
			"page":            0,
			"totalPages":      0,
			"totalCount":      0,
			"hasPreviousPage": false,
			"hasNextPage":     false,
		})
		return
	}
	logrus.Infof("jokeDTO %s", utils.SanitizeForLog(fmt.Sprintf("%+v", jokeDTO)))

	userValue, exists := c.Get("user")
	if !exists {
		c.JSON(401, gin.H{
			"data":            nil,
			"failed":          true,
			"message":         "Unauthorized",
			"succeeded":       false,
			"statusCode":      401,
			"page":            0,
			"totalPages":      0,
			"totalCount":      0,
			"hasPreviousPage": false,
			"hasNextPage":     false,
		})
		return
	}
	user := userValue.(*entities.User)

	// Convert WAJokeDTO to JokeDTO for the service
	serviceJokeDTO := &dtos.JokeDTO{
		Genre:    jokeDTO.GenreId,
		Language: jokeDTO.LanguageID,
	}

	jokeResponse, jokeCount, err := h.jokeService.GetSurpriseJoke(c.Request.Context(), user.ID, serviceJokeDTO, constants.WA_PLATFORM)
	if err != nil {
		// Handle all errors by extracting message and status from service layer
		if appErr, ok := err.(*errs.AppError); ok {
			if appErr.Code == 400 {
				c.JSON(400, gin.H{
					"data":       nil,
					"failed":     true,
					"message":    appErr.Message,
					"succeeded":  false,
					"statusCode": 400,
				})
				return
			}
			if appErr.Status == 204 {
				c.JSON(appErr.Status, gin.H{
					"data":            nil,
					"failed":          true,
					"message":         appErr.Message,
					"succeeded":       false,
					"statusCode":      appErr.Status,
					"page":            0,
					"totalPages":      0,
					"totalCount":      0,
					"hasPreviousPage": false,
					"hasNextPage":     false,
				})
			}
			c.JSON(appErr.Status, gin.H{
				"data":            nil,
				"failed":          true,
				"message":         appErr.Message,
				"succeeded":       false,
				"statusCode":      appErr.Status,
				"page":            0,
				"totalPages":      0,
				"totalCount":      0,
				"hasPreviousPage": false,
				"hasNextPage":     false,
			})
			return
		}
		// Fallback for non-AppError errors
		c.JSON(500, gin.H{
			"data":            nil,
			"failed":          true,
			"message":         "Internal server error",
			"succeeded":       false,
			"statusCode":      500,
			"page":            0,
			"totalPages":      0,
			"totalCount":      0,
			"hasPreviousPage": false,
			"hasNextPage":     false,
		})
		return
	}

	// Transform JokeResponseDTO to WAJokeResponseDTO
	waJokeResponse := &dtos.WAJokeResponseDTO{
		JokeId:        jokeResponse.ID,
		Joke:          jokeResponse.URL,
		JokeType:      jokeResponse.Type,
		ArtistId:      jokeResponse.ArtistId,
		JokesConsumed: jokeCount + 1,
		JokeDetails: dtos.WAJokeDetailsDTO{
			Title:               jokeResponse.Title,
			JokeContentType:     jokeResponse.Type,
			ThumbnailUrl:        jokeResponse.ThumbnailUrl,
			JokeCreator:         jokeResponse.Creator,
			JokeLanguage:        jokeResponse.JokeLanguage,
			JokeGenre:           jokeResponse.JokeGenre,
			JokeReleaseDateTime: time.Now().Format("2006-01-02T15:04:05.000"),
			Reaction:            jokeResponse.UserReaction,
			ViewsCount:          jokeResponse.ViewCount,
		},
	}

	// Handle genre and language ID safely
	if jokeDTO.GenreId != nil {
		waJokeResponse.Genre = *jokeDTO.GenreId
	} else {
		waJokeResponse.Genre = 0
	}

	if jokeDTO.LanguageID != nil {
		waJokeResponse.LanguageId = *jokeDTO.LanguageID
	} else {
		waJokeResponse.LanguageId = 0
	}

	c.JSON(http.StatusOK, gin.H{
		"data":            waJokeResponse,
		"failed":          false,
		"message":         "Success",
		"succeeded":       true,
		"statusCode":      http.StatusOK,
		"page":            0,
		"totalPages":      0,
		"totalCount":      0,
		"hasPreviousPage": false,
		"hasNextPage":     false,
	})
}

func (h *jokeHandler) WAGetJokeGenres(c *gin.Context) {
	userID := c.Query("UserId")
	if userID == "" {
		c.JSON(400, gin.H{
			"data":            nil,
			"failed":          true,
			"message":         "UserId is required",
			"succeeded":       false,
			"statusCode":      400,
			"page":            0,
			"totalPages":      0,
			"totalCount":      0,
			"hasPreviousPage": false,
			"hasNextPage":     false,
		})
		return
	}

	genres, err := h.jokeService.GetJokeGenres(c.Request.Context(), userID)
	if err != nil {
		// Handle all errors by extracting message and status from service layer
		if appErr, ok := err.(*errs.AppError); ok {
			c.JSON(appErr.Status, gin.H{
				"data":            nil,
				"failed":          true,
				"message":         appErr.Message,
				"succeeded":       false,
				"statusCode":      appErr.Status,
				"page":            0,
				"totalPages":      0,
				"totalCount":      0,
				"hasPreviousPage": false,
				"hasNextPage":     false,
			})
			return
		}
		// Fallback for non-AppError errors
		c.JSON(500, gin.H{
			"data":            nil,
			"failed":          true,
			"message":         "Internal server error",
			"succeeded":       false,
			"statusCode":      500,
			"page":            0,
			"totalPages":      0,
			"totalCount":      0,
			"hasPreviousPage": false,
			"hasNextPage":     false,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":            genres,
		"failed":          false,
		"message":         "Success",
		"succeeded":       true,
		"statusCode":      http.StatusOK,
		"page":            0,
		"totalPages":      0,
		"totalCount":      0,
		"hasPreviousPage": false,
		"hasNextPage":     false,
	})
}

func (h *jokeHandler) WAUGCUpload(c *gin.Context) {
	// Read raw request body to handle unescaped newlines in jokeText
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		logrus.Errorf("error reading request body: %v", err)
		c.JSON(400, gin.H{
			"data":       nil,
			"failed":     true,
			"message":    "Failed to read request body",
			"succeeded":  false,
			"statusCode": 400,
		})
		return
	}

	logrus.Infof("WAUGCUpload Request Body: %s", utils.SanitizeForLog(string(body)))

	// Preprocess the JSON string to escape newlines in string literals
	bodyStr := string(body)
	bodyStr = utils.EscapeNewlinesInJSON(bodyStr)

	// Parse JSON manually to handle unescaped newlines
	var rawData map[string]interface{}
	if err := json.Unmarshal([]byte(bodyStr), &rawData); err != nil {
		validationErrors := utils.FormatValidationErrors(err)
		logrus.Errorf("error while parsing JSON: %v", err)
		logrus.Errorf("validation error while parsing JSON: %v", validationErrors)
		c.JSON(400, gin.H{
			"data":             nil,
			"failed":           true,
			"message":          "Invalid JSON format",
			"validationErrors": validationErrors,
			"succeeded":        false,
			"statusCode":       400,
		})
		return
	}

	// Convert botNumber from string to int64 if needed
	if botNumberVal, exists := rawData["botNumber"]; exists {
		switch v := botNumberVal.(type) {
		case string:
			// Convert string to int64
			botNumberInt, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				c.JSON(400, gin.H{
					"data":       nil,
					"failed":     true,
					"message":    "Invalid botNumber format",
					"succeeded":  false,
					"statusCode": 400,
				})
				return
			}
			rawData["botNumber"] = botNumberInt
		case float64:
			// Convert float64 to int64 (JSON numbers are parsed as float64)
			rawData["botNumber"] = int64(v)
		}
	}

	// Convert back to JSON and bind to struct
	jsonData, err := json.Marshal(rawData)
	if err != nil {
		c.JSON(400, gin.H{
			"data":       nil,
			"failed":     true,
			"message":    "Failed to process request data",
			"succeeded":  false,
			"statusCode": 400,
		})
		return
	}

	var req dtos.UGCJokeUploadRequest
	if err := json.Unmarshal(jsonData, &req); err != nil {
		validationErrors := utils.FormatValidationErrors(err)
		logrus.Errorf("Unmarshal error : %v", err)
		logrus.Errorf("validation error while unmarshal : %v", validationErrors)
		c.JSON(400, gin.H{
			"data":             nil,
			"failed":           true,
			"message":          "Invalid request",
			"validationErrors": validationErrors,
			"succeeded":        false,
			"statusCode":       400,
		})
		return
	}

	logrus.Infof("req: %v", req)

	userValue, _ := c.Get("user")
	userID := userValue.(*entities.User).ID

	filename, assetID, err := h.jokeService.SubmitWAUGCJoke(c.Request.Context(), userID, req, constants.WA_PLATFORM)
	if err != nil {
		if appErr, ok := err.(*errs.AppError); ok {
			c.JSON(appErr.Status, gin.H{
				"data":            nil,
				"failed":          true,
				"message":         appErr.Message,
				"succeeded":       false,
				"statusCode":      appErr.Status,
				"page":            0,
				"totalPages":      0,
				"totalCount":      0,
				"hasPreviousPage": false,
				"hasNextPage":     false,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"data":            nil,
			"failed":          true,
			"message":         "Internal server error",
			"succeeded":       false,
			"statusCode":      500,
			"page":            0,
			"totalPages":      0,
			"totalCount":      0,
			"hasPreviousPage": false,
			"hasNextPage":     false,
		})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"data": gin.H{
			"filePath":  filename,
			"isSuccess": true,
			"message":   "",
			"assetId":   assetID,
		},
		"failed":          false,
		"message":         "Success",
		"succeeded":       true,
		"statusCode":      http.StatusCreated,
		"page":            0,
		"totalPages":      0,
		"totalCount":      0,
		"hasPreviousPage": false,
		"hasNextPage":     false,
	})
}

func (h *jokeHandler) AddUserReactPoint(c *gin.Context) {
	// Print raw request body
	body, err := c.GetRawData()
	if err != nil {
		logrus.Errorf("Failed to read request body: %v", err)
	} else {
		logrus.Infof("Raw request body for AddUserReactPoint: %s", utils.SanitizeForLog(string(body)))
		// Restore the body so it can be read again
		c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
	}

	var req dtos.AddUserReactPointRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validationErrors := utils.FormatValidationErrors(err)
		logrus.Errorf("validation error for AddUserReactPoint: %v", validationErrors)
		c.JSON(400, gin.H{
			"data":             nil,
			"failed":           true,
			"message":          "Invalid request",
			"validationErrors": validationErrors,
			"succeeded":        false,
			"statusCode":       400,
		})
		return
	}

	userValue, _ := c.Get("user")
	userID := userValue.(*entities.User).ID

	_, err = h.comicCoinService.WACoinIncrement(c.Request.Context(), userID, constants.COIN_COUNT_INCREMENT_VALUE, req)
	if err != nil {
		c.Error(err)
		return
	}

	response := dtos.AddUserReactPointResponse{
		Success: true,
		Message: "Reaction added successfully",
	}

	c.JSON(200, gin.H{
		"data":            response,
		"failed":          false,
		"message":         "Success",
		"succeeded":       true,
		"statusCode":      http.StatusOK,
		"page":            0,
		"totalPages":      0,
		"totalCount":      0,
		"hasPreviousPage": false,
		"hasNextPage":     false,
	})
}

// @Summary Add user reel reaction
// @Description Add user reel reaction
// @Tags Joke
// @Accept json
// @Produce json
// @Param request body dtos.AddUserReelReactionRequest true "Add user reel reaction"
// @Security BearerAuth
// @Success 200 {object} utils.ResponseType[string] "Reaction added successfully"
// @Router /joke/reel-reaction [post]
func (h *jokeHandler) AddUserReelReaction(c *gin.Context) {
	userValue, _ := c.Get("user")
	userID := userValue.(*entities.User).ID

	var req dtos.AddUserReelReactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validationErrors := utils.FormatValidationErrors(err)
		c.Error(errs.NewValidationErrorWithFields("", validationErrors))
		return
	}
	err := h.reactionService.AddReelReaction(c.Request.Context(), userID, req.AssetID, req.Reaction)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Reaction added successfully",
	})
}

// @Summary Change UGC Status
// @Description Change UGC Status
// @Tags Joke
// @Accept json
// @Produce json
// @Param request body dtos.ChangeUGCStatusRequest true "Change UGC Status"
// @Security BearerAuth
// @Success 200 {object} utils.ResponseType[string] "UGC status changed successfully"
// @Router /joke/change-ugc-status [post]
func (h *jokeHandler) ChangeUGCStatus(c *gin.Context) {
	var req dtos.ChangeUGCStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validationErrors := utils.FormatValidationErrors(err)
		c.Error(errs.NewValidationErrorWithFields("", validationErrors))
		return
	}

	err := h.jokeService.ChangeUGCStatus(c.Request.Context(), req.VideoID, req.Status)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "UGC status changed successfully",
	})
}

// @Summary Activate joke consumption
// @Description Activate a joke consumption record by setting is_active to true
// @Tags Joke
// @Accept json
// @Produce json
// @Param request body dtos.ActivateJokeConsumptionRequest true "Activate joke consumption"
// @Success 200 {object} utils.ResponseType[string] "Joke consumption activated successfully"
// @Security BearerAuth
// @Router /joke/activate-consumption [patch]
func (h *jokeHandler) ActivateJokeConsumption(c *gin.Context) {
	var req dtos.ActivateJokeConsumptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validationErrors := utils.FormatValidationErrors(err)
		c.Error(errs.NewValidationErrorWithFields("", validationErrors))
		return
	}

	userValue, exists := c.Get("user")
	if !exists {
		c.Error(errs.NewUnauthorizedError("Unauthorized", nil))
		return
	}
	user := userValue.(*entities.User)

	err := h.jokeService.ActivateJokeConsumption(c.Request.Context(), user.ID, req.JokeID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Joke consumption activated successfully",
	})
}

// @Summary Get UGC content for assessment
// @Description Get UGC content for manual assessment with filters for date range, language, genre, and source
// @Tags Joke
// @Accept json
// @Produce json
// @Param request body dtos.UGCAssessmentRequest true "Assessment request with filters"
// @Param page query int false "Page number (default: 1)" example:"1"
// @Param limit query int false "Items per page (default: 10)" example:"10"
// @Security BearerAuth
//
//	@Success 200 {object} object "UGC assessment content with pagination" {
//	  "success": true,
//	  "status": 200,
//	  "message": "UGC assessment content retrieved successfully",
//	  "data": {
//	    "content": [
//	      {
//	        "content_id": "string",
//	        "content_url": "string",
//	        "content_genre": "string",
//	        "content_language": "string",
//	        "title": "string",
//	        "file_name": "string",
//	        "user_id": "string",
//	        "created_on": "string",
//	        "status": "string",
//	        "format": "string",
//	        "source": "string"
//	      }
//	    ],
//	    "pagination": {
//	      "page": 1,
//	      "limit": 10,
//	      "total_count": 100,
//	      "total_pages": 10
//	    }
//	  }
//	}
//
// @Router /joke/ugc/assessment [post]
func (h *jokeHandler) GetUGCAssessmentContent(c *gin.Context) {
	var req dtos.UGCAssessmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validationErrors := utils.FormatValidationErrors(err)
		c.Error(errs.NewValidationErrorWithFields("Invalid request", validationErrors))
		return
	}

	page := 1
	limit := 10
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	if req.EndDate.Before(req.StartDate) {
		c.Error(errs.NewBadRequestError("End date cannot be before start date", nil))
		return
	}

	response, err := h.jokeService.GetUGCAssessmentContent(c.Request.Context(), req, page, limit)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"content": response.Data,
		"pagination": gin.H{
			"page":        response.Page,
			"limit":       response.Limit,
			"total_count": response.TotalCount,
			"total_pages": int(math.Ceil(float64(response.TotalCount) / float64(response.Limit))),
		},
	})
}
