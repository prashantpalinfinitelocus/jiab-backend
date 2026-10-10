package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Infinite-Locus-Product/coca_cola_backend/constants"
	"github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"github.com/Infinite-Locus-Product/coca_cola_backend/utils"
)

type JokeService interface {
	GetSurpriseJoke(ctx context.Context, userId string, joke *dtos.JokeDTO, platform int) (*dtos.JokeResponseDTO, int, error)
	GetScrollAndLolJokes(ctx context.Context, userId *string, jokeDTO *dtos.JokeScrollDTO) ([]dtos.JokeResponseDTO, error)
	GetSpicyTrayJokes(ctx context.Context, userId *string, jokeDTO *dtos.JokeScrollDTO) ([]dtos.JokeResponseDTO, error)
	SubmitUGC(ctx context.Context, userId string, joke dtos.JokeIngestionDTO, platform int) error
	GetUGCContentByIDs(ctx context.Context, ids []string) ([]dtos.UGCContentByIDsResponseDTO, error)
	GetProfileUGCContent(ctx context.Context, userId string) ([]dtos.UGCContentResponseDTO, error)
	GetUGCContentArtist(ctx context.Context, ids []string) ([]dtos.UGCContentArtistDTO, error)
	GetJokeGenres(ctx context.Context, userID string) ([]dtos.JokeGenreLanguageResponseDTO, error)
	SubmitWAUGCJoke(ctx context.Context, userId string, joke dtos.UGCJokeUploadRequest, platform int) (string, string, error)
	ChangeUGCStatus(ctx context.Context, videoID string, status int) error
	ActivateJokeConsumption(ctx context.Context, userId string, jokeId string) error
	GetRefreshRingJourney(ctx context.Context, userId *string, jokeDTO *dtos.JokeScrollDTO) ([]dtos.JokeResponseDTO, error)
	CheckForDuplicateContent(ctx context.Context, currentContent []byte, existingJokes []entities.UGCMaster) (bool, error)
	GetUGCAssessmentContent(ctx context.Context, request dtos.UGCAssessmentRequest, page, limit int) (*dtos.UGCAssessmentListResponse, error)
}

type jokeService struct {
	txnManager               *utils.TransactionManager
	gluedinService           GluedinService
	jokeStorageService       JokeStorageService
	jokeReferenceDataService JokeReferenceDataService
	gcsService               utils.GCSService
	infobipService           InfobipService
	comicCoinService         ComicCoinService
	notificationService      NotificationService
	userService              UserService
	chatbotGluedinUserID     string
}

func NewJokeService(
	txnManager *utils.TransactionManager,
	gluedinService GluedinService,
	jokeStorageService JokeStorageService,
	jokeReferenceDataService JokeReferenceDataService,
	gcsService utils.GCSService,
	infobipService InfobipService,
	comicCoinService ComicCoinService,
	notificationService NotificationService,
	userService UserService,
	chatbotGluedinUserID string,
) JokeService {
	return &jokeService{
		txnManager:               txnManager,
		gluedinService:           gluedinService,
		jokeStorageService:       jokeStorageService,
		jokeReferenceDataService: jokeReferenceDataService,
		gcsService:               gcsService,
		infobipService:           infobipService,
		comicCoinService:         comicCoinService,
		notificationService:      notificationService,
		userService:              userService,
		chatbotGluedinUserID:     chatbotGluedinUserID,
	}
}

func (s *jokeService) GetSurpriseJoke(ctx context.Context, userId string, jokeDTO *dtos.JokeDTO, platform int) (*dtos.JokeResponseDTO, int, error) {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		logrus.Info("error starting transaction", err)
		return nil, 0, errs.NewInternalServerError("failed to start transaction", err)
	}
	defer s.txnManager.RollbackOnPanic(tx)

	jokeCount, err := s.jokeStorageService.GetSurpriseMeJokeCount(ctx, tx, userId, platform)
	if err != nil {
		logrus.Info("getting suprise me joke count", err)
		s.txnManager.AbortTxn(tx)
		return nil, 0, err
	}

	if jokeCount >= constants.MAX_JOKE_COUNT {
		logrus.Info("maximum limit reached", nil)
		s.txnManager.AbortTxn(tx)
		return nil, 0, errs.NewBadRequestError("Maximum Joke Limit Reached for today.", nil)
	}

	genre, language, err := s.jokeReferenceDataService.GetJokeReferenceData(ctx, tx, jokeDTO.Genre, jokeDTO.Language)
	if err != nil {
		logrus.Info("error getting joke reference data", err)
		s.txnManager.AbortTxn(tx)
		return nil, 0, err
	}

	languageKey, err := s.jokeReferenceDataService.GetJokeLanguageKey(ctx, tx, *jokeDTO.Language)
	if err != nil {
		logrus.Info("error getting joke language key", err)
		s.txnManager.AbortTxn(tx)
		return nil, 0, err
	}

	joke, err := s.gluedinService.GetSupriseMeJokeFromGluedin(userId, genre, language, jokeDTO.Exclude, platform)
	if err != nil {
		logrus.Info("error storing suprise me joke", err)
		s.txnManager.AbortTxn(tx)
		return nil, 0, err
	}

	joke, err = s.jokeStorageService.StoreSurpriseMeJoke(ctx, tx, userId, joke, jokeDTO, genre, language, platform)
	if err != nil {
		logrus.Info("error storing suprise me joke", err)
		s.txnManager.AbortTxn(tx)
		return nil, 0, err
	}

	user := joke["user"].(map[string]interface{})
	name := user["userName"].(string)
	artistId := user["userId"].(string)
	var userReaction dtos.UserReaction
	reactions, ok := joke["reactions"].(map[string]interface{})

	if ok {
		if laugh, ok := reactions["laugh"].(float64); ok {
			userReaction.Laugh = int(laugh)
		} else {
			userReaction.Laugh = 0
		}
		if neutral, ok := reactions["neutral"].(float64); ok {
			userReaction.Neutral = int(neutral)
		} else {
			userReaction.Neutral = 0
		}
		if sad, ok := reactions["sad"].(float64); ok {
			userReaction.Sad = int(sad)
		} else {
			userReaction.Sad = 0
		}
	} else {
		userReaction.Laugh = 0
		userReaction.Neutral = 0
		userReaction.Sad = 0
	}

	title := s.getLocalizedTitle(joke, languageKey)

	jokeResponseDTO := &dtos.JokeResponseDTO{
		ID:           joke["id"].(string),
		Type:         joke["contentType"].(string),
		Title:        title,
		ThumbnailUrl: joke["thumbnailUrl"].(string),
		ViewCount:    int(joke["viewsCount"].(float64)),
		Creator:      name,
		ArtistId:     artistId,
		JokeLanguage: language,
		JokeGenre:    genre,
		UserReaction: userReaction,
	}

	contentUrls := joke["contentUrls"].([]interface{})[0].(map[string]interface{})["urls"].([]interface{})
	if len(contentUrls) > 0 {
		var fallbackURL string
		for _, url := range contentUrls {
			urlString, ok := url.(string)
			if !ok {
				continue
			}
			if strings.Contains(urlString, constants.WA_FORMAT_HIGH_RES) {
				jokeResponseDTO.URL = urlString
				break
			}
			if strings.Contains(urlString, constants.WA_FORMAT_LOW_RES) && fallbackURL == "" {
				fallbackURL = urlString
			}
		}
		if jokeResponseDTO.URL == "" && fallbackURL != "" {
			jokeResponseDTO.URL = fallbackURL
		}
	}
	if jokeResponseDTO.URL == "" {
		jokeResponseDTO.URL = joke["downloadUrl"].(string)
	}

	s.txnManager.CommitTxn(tx)
	return jokeResponseDTO, jokeCount, nil
}

func (s *jokeService) GetScrollAndLolJokes(ctx context.Context, userId *string, jokeDTO *dtos.JokeScrollDTO) ([]dtos.JokeResponseDTO, error) {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		logrus.Info("error starting transaction", err)
		return nil, errs.NewInternalServerError("failed to start transaction", err)
	}
	defer s.txnManager.RollbackOnPanic(tx)

	userOldJokes := []entities.JokeConsumptionByUser{}
	if userId != nil && jokeDTO.Type != nil && *jokeDTO.Type != "home" {
		fetchedJokes, err := s.jokeStorageService.FetchScrollAndLolJokeByLanguage(ctx, tx, *jokeDTO.Language, *userId)
		if err != nil {
			s.txnManager.AbortTxn(tx)
			return nil, errs.NewInternalServerError("failed to get joke reference data", err)
		}
		userOldJokes = fetchedJokes
	}

	language, err := s.jokeReferenceDataService.GetJokeLanguage(ctx, tx, *jokeDTO.Language)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, errs.NewInternalServerError("failed to get joke reference data", err)
	}

	languageKey, err := s.jokeReferenceDataService.GetJokeLanguageKey(ctx, tx, *jokeDTO.Language)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, errs.NewInternalServerError("failed to get joke reference data", err)
	}

	var remainingJokeCount *int
	if userId != nil && len(userOldJokes) == 0 && jokeDTO.Type != nil && *jokeDTO.Type != "home" {
		jokeCount, err := s.jokeStorageService.GetScrollAndLolJokeCount(ctx, tx, *userId)
		if err != nil {
			s.txnManager.AbortTxn(tx)
			return nil, err
		}
		count := constants.MAX_SCROLL_AND_LOL_JOKE_COUNT - jokeCount
		if count == 0 {
			s.txnManager.AbortTxn(tx)
			return nil, errs.ErrMaximumJokeLimitReached
		}
		remainingJokeCount = &count
	} else {
		remainingJokeCount = nil
	}

	if len(userOldJokes) > 0 {
		jokeIDs := make([]string, 0)
		for _, joke := range userOldJokes {
			jokeIDs = append(jokeIDs, joke.JokeID)
		}
		preferredJokes := strings.Join(jokeIDs, ",")
		jokeDTO.PreferredJokes = &preferredJokes
		logrus.Info("preferredJokes", preferredJokes)
	}

	userTotalActiveJokes := []entities.JokeConsumptionByUser{}
	if userId != nil && jokeDTO.Type != nil && *jokeDTO.Type != "home" {
		var fetchedUserTotalJokes []entities.JokeConsumptionByUser
		fetchedUserTotalJokes, err = s.jokeStorageService.GetUserTotalJokes(ctx, tx, *userId)
		if err != nil {
			s.txnManager.AbortTxn(tx)
			return nil, errs.NewInternalServerError("failed to get user total jokes", err)
		}
		userTotalActiveJokes = fetchedUserTotalJokes
	}

	jokes, err := s.gluedinService.GetScrollAndLolJokesFromGluedin(userId, jokeDTO, remainingJokeCount, language, len(userOldJokes))
	if err != nil {
		logrus.Info("error getting scroll and lol jokes", err)
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	if userId != nil && len(userOldJokes) == 0 && jokeDTO.Type != nil && *jokeDTO.Type != "home" {
		jokes, err = s.jokeStorageService.StoreScrollAndLolJoke(ctx, tx, *userId, jokes, jokeDTO, language)
		if err != nil {
			logrus.Info("error storing scroll and lol jokes", err)
			s.txnManager.AbortTxn(tx)
			return nil, err
		}
	}

	// Calculate remaining slots after considering all active jokes
	maxAllowed := constants.MAX_SCROLL_AND_LOL_JOKE_COUNT
	remainingSlots := maxAllowed - len(userTotalActiveJokes)
	activeLanguageJokes := 0
	if jokeDTO.Type != nil && *jokeDTO.Type != "home" {
		// If we have userOldJokes, prioritize active jokes first
		if len(userOldJokes) > 0 {
			var activeOldJokes []entities.JokeConsumptionByUser
			var inactiveOldJokes []entities.JokeConsumptionByUser

			// Separate active and inactive jokes from userOldJokes
			for _, joke := range userOldJokes {
				if joke.IsActive {
					activeOldJokes = append(activeOldJokes, joke)
				} else {
					inactiveOldJokes = append(inactiveOldJokes, joke)
				}
			}
			activeLanguageJokes += len(activeOldJokes)

			// Send active jokes first (up to remaining slots)
			activeToSend := len(activeOldJokes)

			// Send inactive jokes next (fill remaining slots)
			inactiveToSend := remainingSlots - activeToSend
			if inactiveToSend > len(inactiveOldJokes) {
				inactiveToSend = len(inactiveOldJokes)
			}
			if inactiveToSend < 0 {
				inactiveToSend = 0
			}

			// Now, reconstruct the jokes array to prioritize active then inactive
			var prioritizedJokes []map[string]interface{}
			// Map for quick lookup
			activeJokeIDs := make(map[string]struct{})
			for _, joke := range activeOldJokes {
				activeJokeIDs[joke.JokeID] = struct{}{}
			}
			inactiveJokeIDs := make(map[string]struct{})
			for _, joke := range inactiveOldJokes {
				inactiveJokeIDs[joke.JokeID] = struct{}{}
			}
			// Add active jokes from API response
			logrus.Info("activeToSend ", activeToSend)
			logrus.Info("inactiveToSend ", inactiveToSend)
			added := 0
			for _, joke := range jokes {
				if _, ok := activeJokeIDs[joke["videoId"].(string)]; ok && added < activeToSend {
					prioritizedJokes = append(prioritizedJokes, joke)
					added++
				}
				if added >= activeToSend {
					break
				}
			}
			// Add inactive jokes from API response
			added = 0
			for _, joke := range jokes {
				if _, ok := inactiveJokeIDs[joke["videoId"].(string)]; ok && added < inactiveToSend {
					prioritizedJokes = append(prioritizedJokes, joke)
					added++
				}
				if added >= inactiveToSend {
					break
				}
			}
			jokes = prioritizedJokes
			remainingSlots = activeToSend + inactiveToSend
		}
		logrus.Info("remainingSlots", remainingSlots)
		logrus.Info("length of jokes", len(jokes))
		// If still more jokes than allowed, trim
		if len(jokes) > remainingSlots {
			jokes = jokes[:remainingSlots]
		}
	}

	isExhausted := false
	if len(userTotalActiveJokes)+len(jokes)-activeLanguageJokes >= maxAllowed {
		isExhausted = true
	}

	// Fetch all active genres once to create a lookup map
	genreLookup := make(map[string]string)
	isActive := true
	allGenres, _, err := s.jokeReferenceDataService.GetAllGenres(ctx, tx, dtos.GenreFilter{IsActive: &isActive})
	if err != nil {
		logrus.Errorf("error fetching genres for lookup: %v", err)
	} else {
		for _, genre := range allGenres {
			imageURL := ""
			if genre.ImageURL != "" {
				signedURL, err := s.gcsService.GetFileSignedURL(ctx, genre.ImageURL, 24*time.Hour)
				if err != nil {
					imageURL = genre.ImageURL // fallback to raw path if signing fails
				} else {
					imageURL = signedURL
				}
			}
			genreLookup[genre.Genre] = imageURL
		}
	}

	var jokeResponseDTOs []dtos.JokeResponseDTO
	for _, joke := range jokes {
		reactions, ok := joke["reactions"].(map[string]interface{})
		var userReaction dtos.UserReaction
		if ok {
			if laugh, ok := reactions["laugh"].(float64); ok {
				userReaction.Laugh = int(laugh)
			}
			if neutral, ok := reactions["neutral"].(float64); ok {
				userReaction.Neutral = int(neutral)
			}
			if sad, ok := reactions["sad"].(float64); ok {
				userReaction.Sad = int(sad)
			}
		}

		// Process labels to find matching genre and image
		labelsInterface, ok := joke["labels"]
		var genre string
		var genreImage string
		if ok && labelsInterface != nil {
			// Handle different possible types for labels
			var labels []string
			switch v := labelsInterface.(type) {
			case []string:
				labels = v
			case []interface{}:
				// Convert []interface{} to []string
				for _, item := range v {
					if str, ok := item.(string); ok {
						labels = append(labels, str)
					}
				}
			}
			if len(labels) > 0 {
				// Check all labels for matching genres using the lookup map
				for _, label := range labels {
					logrus.Info("label", label)
					if imageURL, exists := genreLookup[label]; exists {
						genre = label
						genreImage = imageURL
						break // Use the first matching genre
					}
				}
				// If no matching genre found, use the first label as before
				if genre == "" && len(labels) > 2 {
					genre = labels[0]
				}
			}
		}

		title := s.getLocalizedTitle(joke, languageKey)

		jokeResponseDTO := dtos.JokeResponseDTO{
			ID:           joke["videoId"].(string),
			URL:          joke["s3Url"].(string),
			Type:         joke["contentType"].(string),
			Title:        title,
			ThumbnailUrl: joke["thumbnailUrl"].(string),
			ViewCount:    int(joke["viewsCount"].(float64)),
			Creator:      joke["user"].(map[string]interface{})["userName"].(string),
			ArtistId:     joke["user"].(map[string]interface{})["userId"].(string),
			JokeLanguage: language,
			JokeGenre:    genre,
			GenreImage:   genreImage,
			UserReaction: userReaction,
			IsExhausted:  isExhausted,
		}
		jokeResponseDTOs = append(jokeResponseDTOs, jokeResponseDTO)
	}
	s.txnManager.CommitTxn(tx)

	return jokeResponseDTOs, nil
}

func (s *jokeService) GetSpicyTrayJokes(ctx context.Context, userID *string, jokeDTO *dtos.JokeScrollDTO) ([]dtos.JokeResponseDTO, error) {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return nil, errs.NewInternalServerError("failed to start transaction", err)
	}
	defer s.txnManager.RollbackOnPanic(tx)

	userOldJokes := []entities.JokeConsumptionByUser{}
	if userID != nil && jokeDTO.Type != nil && *jokeDTO.Type != "home" {
		fetchedJokes, err := s.jokeStorageService.FetchSpicyTrayJokeByLanguage(ctx, tx, *jokeDTO.Language, *userID)
		if err != nil {
			s.txnManager.AbortTxn(tx)
			return nil, errs.NewInternalServerError("failed to get joke reference data", err)
		}
		userOldJokes = fetchedJokes
	}

	language, err := s.jokeReferenceDataService.GetJokeLanguage(ctx, tx, *jokeDTO.Language)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, errs.NewInternalServerError("failed to get joke reference data", err)
	}

	languageKey, err := s.jokeReferenceDataService.GetJokeLanguageKey(ctx, tx, *jokeDTO.Language)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, errs.NewInternalServerError("failed to get joke reference data", err)
	}

	var remainingJokeCount *int
	if userID != nil && len(userOldJokes) == 0 && jokeDTO.Type != nil && *jokeDTO.Type != "home" {
		jokeCount, err := s.jokeStorageService.GetSpicyTrayJokeCount(ctx, tx, *userID)
		if err != nil {
			s.txnManager.AbortTxn(tx)
			return nil, err
		}
		count := constants.MAX_SPICY_TRAY_JOKE_COUNT - jokeCount
		if count == 0 {
			s.txnManager.AbortTxn(tx)
			return nil, errs.ErrMaximumJokeLimitReached
		}
		remainingJokeCount = &count
	}

	if len(userOldJokes) > 0 {
		jokeIDs := make([]string, 0)
		for _, joke := range userOldJokes {
			jokeIDs = append(jokeIDs, joke.JokeID)
		}
		preferredJokes := strings.Join(jokeIDs, ",")
		jokeDTO.PreferredJokes = &preferredJokes
	}

	userTotalActiveJokes := []entities.JokeConsumptionByUser{}
	if userID != nil && jokeDTO.Type != nil && *jokeDTO.Type != "home" {
		var fetchedUserTotalJokes []entities.JokeConsumptionByUser
		// Reuse GetUserTotalJokes for active spicy-tray as we mark is_active when needed
		fetchedUserTotalJokes, err = s.jokeStorageService.GetUserTotalJokes(ctx, tx, *userID)
		if err != nil {
			s.txnManager.AbortTxn(tx)
			return nil, errs.NewInternalServerError("failed to get user total jokes", err)
		}
		userTotalActiveJokes = fetchedUserTotalJokes
	}

	jokes, err := s.gluedinService.GetSpicyTrayJokesFromGluedin(jokeDTO, remainingJokeCount, language, len(userOldJokes))
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	if userID != nil && len(userOldJokes) == 0 && jokeDTO.Type != nil && *jokeDTO.Type != "home" {
		jokes, err = s.jokeStorageService.StoreSpicyTrayJokes(ctx, tx, *userID, jokes, jokeDTO, language)
		if err != nil {
			s.txnManager.AbortTxn(tx)
			return nil, err
		}
	}

	maxAllowed := constants.MAX_SPICY_TRAY_JOKE_COUNT
	remainingSlots := maxAllowed - len(userTotalActiveJokes)
	activeLanguageJokes := 0
	if jokeDTO.Type != nil && *jokeDTO.Type != "home" {
		if len(userOldJokes) > 0 {
			var activeOldJokes []entities.JokeConsumptionByUser
			var inactiveOldJokes []entities.JokeConsumptionByUser
			for _, joke := range userOldJokes {
				if joke.IsActive {
					activeOldJokes = append(activeOldJokes, joke)
				} else {
					inactiveOldJokes = append(inactiveOldJokes, joke)
				}
			}
			activeLanguageJokes += len(activeOldJokes)

			activeToSend := len(activeOldJokes)
			inactiveToSend := remainingSlots - activeToSend
			if inactiveToSend > len(inactiveOldJokes) {
				inactiveToSend = len(inactiveOldJokes)
			}
			if inactiveToSend < 0 {
				inactiveToSend = 0
			}

			var prioritizedJokes []map[string]interface{}
			activeJokeIDs := make(map[string]struct{})
			for _, joke := range activeOldJokes {
				activeJokeIDs[joke.JokeID] = struct{}{}
			}
			inactiveJokeIDs := make(map[string]struct{})
			for _, joke := range inactiveOldJokes {
				inactiveJokeIDs[joke.JokeID] = struct{}{}
			}
			added := 0
			for _, joke := range jokes {
				if _, ok := activeJokeIDs[joke["videoId"].(string)]; ok && added < activeToSend {
					prioritizedJokes = append(prioritizedJokes, joke)
					added++
				}
				if added >= activeToSend {
					break
				}
			}
			added = 0
			for _, joke := range jokes {
				if _, ok := inactiveJokeIDs[joke["videoId"].(string)]; ok && added < inactiveToSend {
					prioritizedJokes = append(prioritizedJokes, joke)
					added++
				}
				if added >= inactiveToSend {
					break
				}
			}
			jokes = prioritizedJokes
			remainingSlots = activeToSend + inactiveToSend
		}
		if len(jokes) > remainingSlots {
			jokes = jokes[:remainingSlots]
		}
	}

	isExhausted := false
	if len(userTotalActiveJokes)+len(jokes)-activeLanguageJokes >= maxAllowed {
		isExhausted = true
	}

	genreLookup := make(map[string]string)
	isActive := true
	allGenres, _, err := s.jokeReferenceDataService.GetAllGenres(ctx, tx, dtos.GenreFilter{IsActive: &isActive})
	if err == nil {
		for _, genre := range allGenres {
			genreLookup[genre.Genre] = genre.ImageURL
		}
	}

	var jokeResponseDTOs []dtos.JokeResponseDTO
	for _, joke := range jokes {
		jokeResponseDTO := s.convertJokeToResponseDTO(joke, language, languageKey, genreLookup)
		jokeResponseDTO.IsExhausted = isExhausted
		jokeResponseDTOs = append(jokeResponseDTOs, jokeResponseDTO)
	}

	s.txnManager.CommitTxn(tx)
	return jokeResponseDTOs, nil
}

func (s *jokeService) getLocalizedTitle(joke map[string]interface{}, languageKey string) string {
	defaultTitle := ""
	if title, ok := joke["title"].(string); ok {
		defaultTitle = title
	}

	localisedTitleInterface, ok := joke["localisedTitle"]
	if !ok || localisedTitleInterface == nil {
		return defaultTitle
	}

	localisedTitleMap, ok := localisedTitleInterface.(map[string]interface{})
	if !ok {
		return defaultTitle
	}

	if localizedTitle, exists := localisedTitleMap[languageKey]; exists {
		if titleStr, ok := localizedTitle.(string); ok && titleStr != "" {
			return titleStr
		}
	}

	return defaultTitle
}

func (s *jokeService) GetJokeReferenceData(ctx context.Context, tx *gorm.DB, genreID *int, languageID int, format string) (string, string, int, error) {
	genre, err := s.jokeReferenceDataService.GetJokeGenre(ctx, tx, genreID)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return "", "", 0, errs.NewInternalServerError("failed to get joke reference data", err)
	}

	language, err := s.jokeReferenceDataService.GetJokeLanguage(ctx, tx, languageID)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return "", "", 0, errs.NewInternalServerError("failed to get joke reference data", err)
	}

	formatID, err := s.jokeReferenceDataService.GetJokeFormat(ctx, tx, format)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return "", "", 0, errs.NewInternalServerError("failed to get joke format", err)
	}

	return genre, language, formatID, nil
}

func (s *jokeService) SubmitUGC(ctx context.Context, userId string, joke dtos.JokeIngestionDTO, platform int) error {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	ugcContentCount, err := s.jokeStorageService.GetCountOfUGCContent(ctx, tx, userId)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return err
	}

	if ugcContentCount >= constants.MAX_UGC_CONTENT_COUNT {
		s.txnManager.AbortTxn(tx)
		return errs.ErrMaximumUGCContentLimitReached
	}

	filePath := fmt.Sprintf("%s/%s/%s", constants.UGC_CONTENT_FOLDER, userId, joke.Format)
	gcsURL, contentURL, err := s.gcsService.UploadFile(ctx, joke.File, filePath)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return err
	}

	genre, language, formatID, err := s.GetJokeReferenceData(ctx, tx, joke.JokeGenreID, joke.LanguageID, joke.Format)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return err
	}

	var labels []string
	if genre != "" {
		labels = append(labels, genre)
	}
	labels = append(labels, language)
	assetID, err := s.gluedinService.SubmitUGCContent(userId, joke, gcsURL, labels)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return err
	}

	err = s.jokeStorageService.StoreUGCJoke(ctx, tx, userId, joke, formatID, contentURL, assetID, platform, userId, "")
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return err
	}

	s.txnManager.CommitTxn(tx)
	return nil
}

func (s *jokeService) GetUGCContentByIDs(ctx context.Context, ids []string) ([]dtos.UGCContentByIDsResponseDTO, error) {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return nil, err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	ugcContent, err := s.jokeStorageService.GetUGCContentByIDs(ctx, tx, ids)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	s.txnManager.CommitTxn(tx)
	return ugcContent, nil
}

func (s *jokeService) GetProfileUGCContent(ctx context.Context, userId string) ([]dtos.UGCContentResponseDTO, error) {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return nil, err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	ugcContent, err := s.jokeStorageService.GetProfileUGCContent(ctx, tx, userId)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	s.txnManager.CommitTxn(tx)
	return ugcContent, nil
}

func (s *jokeService) GetUGCContentArtist(ctx context.Context, ids []string) ([]dtos.UGCContentArtistDTO, error) {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return nil, err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	ugcContent, err := s.jokeStorageService.GetUGCContentArtist(ctx, tx, ids)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	s.txnManager.CommitTxn(tx)
	return ugcContent, nil
}

func (s *jokeService) GetJokeGenres(ctx context.Context, userID string) ([]dtos.JokeGenreLanguageResponseDTO, error) {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return nil, err
	}
	defer s.txnManager.AbortTxn(tx)

	result, err := s.jokeReferenceDataService.GetJokeGenres(ctx, tx, userID)
	if err != nil {
		return nil, err
	}

	s.txnManager.CommitTxn(tx)
	return result, nil
}

func (s *jokeService) SubmitWAUGCJoke(ctx context.Context, userId string, joke dtos.UGCJokeUploadRequest, platform int) (string, string, error) {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return "", "", err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	ugcContentCount, err := s.jokeStorageService.GetCountOfUGCContent(ctx, tx, userId)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return "", "", err
	}

	if ugcContentCount >= constants.MAX_UGC_CONTENT_COUNT {
		s.txnManager.AbortTxn(tx)
		return "", "", errs.NewAppError(1001, "Maximum UGC Limit Reached for today.", 429, nil)
	}

	var file io.ReadCloser
	var currentContent []byte

	if joke.Format == "Text" {
		if joke.JokeText == nil {
			s.txnManager.AbortTxn(tx)
			return "", "", errs.NewBadRequestError("jokeText is required for this format", nil)
		}
		currentContent = []byte(*joke.JokeText)
		file = io.NopCloser(bytes.NewReader(currentContent))
	} else {
		if joke.MediaID == nil {
			s.txnManager.AbortTxn(tx)
			return "", "", errs.NewBadRequestError("mediaId is required for this format", nil)
		}
		file, err = s.infobipService.FetchMedia(ctx, *joke.MediaID)
		if err != nil {
			s.txnManager.AbortTxn(tx)
			return "", "", err
		}

		currentContent, err = io.ReadAll(file)
		_ = file.Close()
		if err != nil {
			s.txnManager.AbortTxn(tx)
			return "", "", err
		}

		file = io.NopCloser(bytes.NewReader(currentContent))
	}

	genre, language, formatID, err := s.GetJokeReferenceData(ctx, tx, nil, int(joke.LanguageID), joke.Format)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return "", "", err
	}

	/*	todaysJokes, err := s.jokeStorageService.GetTodaysUGCJokesByUserPlatformFormat(ctx, tx, userId, platform, formatID)
		if err != nil {
			s.txnManager.AbortTxn(tx)
			return "", "", err
		}

		// Check for duplicate content
		isDuplicate, err := s.CheckForDuplicateContent(ctx, currentContent, todaysJokes)
		if err != nil {
			s.txnManager.AbortTxn(tx)
			return "", "", err
		}

		if isDuplicate {
			s.txnManager.AbortTxn(tx)
			return "", "", errs.NewAppError(1002, "The same joke content has already been submitted today.", 409, nil)
		}*/

	filePath := fmt.Sprintf("%s/%s/%s", constants.UGC_CONTENT_FOLDER, userId, joke.Format)
	gcsURL, contentURL, err := s.gcsService.UploadFileFromReader(ctx, file, filePath)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return "", "", err
	}

	var labels []string
	if genre != "" {
		labels = append(labels, genre)
	}
	labels = append(labels, language)
	jokeIngestionDTO := dtos.JokeIngestionDTO{
		Format:     joke.Format,
		LanguageID: int(joke.LanguageID),
		Title:      joke.Title,
	}
	assetID, err := s.gluedinService.SubmitUGCContent(s.chatbotGluedinUserID, jokeIngestionDTO, gcsURL, labels)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return "", "", err
	}

	mediaID := ""
	if joke.Format == "Text" {
		mediaID = string(currentContent) // For text format
	} else {
		mediaID = *joke.MediaID // For other formats
	}
	err = s.jokeStorageService.StoreUGCJoke(ctx, tx, userId, jokeIngestionDTO, formatID, contentURL, assetID, platform, s.chatbotGluedinUserID, mediaID)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return "", "", err
	}

	s.txnManager.CommitTxn(tx)
	return contentURL, assetID, nil
}

func GetMessageID(ctx context.Context) string {
	messageID, ok := ctx.Value("requestID").(string)
	if !ok {
		uid, _ := uuid.NewUUID()
		messageID = uid.String()
	}
	return messageID
}

func (s *jokeService) ChangeUGCStatus(ctx context.Context, videoID string, status int) error {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	ugcContent, err := s.jokeStorageService.GetUGCContentByAssetID(ctx, tx, videoID, true)
	if err != nil {
		return err
	}

	switch status {
	case constants.UGC_STATUS_ACCEPTED:
		err = s.handleUGCSuccess(ctx, tx, ugcContent)
	case constants.UGC_STATUS_REJECTED:
		err = s.handleUGCRejected(ctx, tx, ugcContent)
	case constants.UGC_STATUS_PLAGIARIZED:
		err = s.handleUGCPlagiarized(ctx, tx, ugcContent)
	default:
		return errs.NewBadRequestError("Invalid status value", nil)
	}

	if err != nil {
		return err
	}

	s.txnManager.CommitTxn(tx)
	return nil
}

func (s *jokeService) handleUGCSuccess(ctx context.Context, tx *gorm.DB, ugcContent *entities.UGCMaster) error {
	userID := ugcContent.UserID

	err := s.gluedinService.ChangeUGCStatus(ugcContent.AssetID, "ACTIVE")
	if err != nil {
		return err
	}

	err = s.jokeStorageService.UpdateUGCStatus(ctx, tx, ugcContent.AssetID, constants.UGC_STATUS_APPROVED, true)
	if err != nil {
		return err
	}

	_, err = s.comicCoinService.UpdateBalance(ctx, tx, userID, constants.REWARD_UNIQUE_CODE)
	if err != nil {
		return err
	}

	if err := s.notificationService.SendPushNotification(ctx, tx, constants.NOTIFICATION_UGC_APPROVED, userID, nil); err != nil {
		logrus.Error("error sending UGC approved notification", err)
	}
	uniqueCodeReward := constants.REWARD_UNIQUE_CODE
	if err := s.notificationService.SendPushNotification(ctx, tx, constants.NOTIFICATION_WIN_COIN, userID, &uniqueCodeReward); err != nil {
		logrus.Error("error sending UGC approval coin-win notification", err)
	}

	err = s.sendWhatsAppSuccessMessage(ctx, tx, ugcContent)
	if err != nil {
		logrus.Error("error sending WhatsApp success message", err)
	}

	return nil
}

func (s *jokeService) handleUGCRejected(ctx context.Context, tx *gorm.DB, ugcContent *entities.UGCMaster) error {

	err := s.jokeStorageService.UpdateUGCStatus(ctx, tx, ugcContent.AssetID, constants.UGC_STATUS_REJECTED, false)
	if err != nil {
		return err
	}

	if err := s.notificationService.SendPushNotification(ctx, tx, constants.NOTIFICATION_UGC_REJECTED, ugcContent.UserID, nil); err != nil {
		logrus.Error("error sending UGC rejected notification", err)
	}

	err = s.sendWhatsAppRejectedMessage(ctx, ugcContent)
	if err != nil {
		logrus.Error("error sending WhatsApp rejected message", err)
	}
	return nil
}

func (s *jokeService) handleUGCPlagiarized(ctx context.Context, tx *gorm.DB, ugcContent *entities.UGCMaster) error {

	err := s.jokeStorageService.UpdateUGCStatus(ctx, tx, ugcContent.AssetID, constants.UGC_STATUS_PLAGIARIZED, false)
	if err != nil {
		return err
	}

	if err := s.notificationService.SendPushNotification(ctx, tx, constants.NOTIFICATION_UGC_DUPLICATE, ugcContent.UserID, nil); err != nil {
		logrus.Error("error sending UGC duplicate notification", err)
	}

	err = s.sendWhatsAppPlagiarizedMessage(ctx, ugcContent)
	if err != nil {
		logrus.Error("error sending WhatsApp plagiarized message", err)
	}
	return nil
}

func (s *jokeService) sendWhatsAppSuccessMessage(ctx context.Context, tx *gorm.DB, ugcContent *entities.UGCMaster) error {
	user, err := s.userService.GetUser(ctx, ugcContent.UserID)
	if err != nil {
		return err
	}

	genre, language, _, err := s.GetJokeReferenceData(ctx, tx, ugcContent.JokeGenreID, ugcContent.LanguageID, "Text")
	if err != nil {
		return err
	}

	placeholders := []string{
		*user.Name,
		language,
		genre,
	}

	request := dtos.WhatsAppTemplateRequest{
		To:           utils.NumberWithCountryCode(user.PhoneNumber),
		MessageID:    GetMessageID(ctx),
		Placeholders: placeholders,
	}

	return s.infobipService.SendWhatsAppSuccessTemplate(request)
}

func (s *jokeService) sendWhatsAppRejectedMessage(ctx context.Context, ugcContent *entities.UGCMaster) error {
	user, err := s.userService.GetUser(ctx, ugcContent.UserID)
	if err != nil {
		return err
	}

	request := dtos.WhatsAppRejectedTemplateRequest{
		To:        utils.NumberWithCountryCode(user.PhoneNumber),
		MessageID: GetMessageID(ctx),
	}

	return s.infobipService.SendWhatsAppRejectedTemplate(request)
}

func (s *jokeService) sendWhatsAppPlagiarizedMessage(ctx context.Context, ugcContent *entities.UGCMaster) error {
	user, err := s.userService.GetUser(ctx, ugcContent.UserID)
	if err != nil {
		return err
	}

	request := dtos.WhatsAppPlagiarizedTemplateRequest{
		To:        utils.NumberWithCountryCode(user.PhoneNumber),
		MessageID: GetMessageID(ctx),
	}

	return s.infobipService.SendWhatsAppPlagiarizedTemplate(request)
}

func (s *jokeService) ActivateJokeConsumption(ctx context.Context, userId string, jokeId string) error {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return errs.NewInternalServerError("failed to start transaction", err)
	}
	defer s.txnManager.RollbackOnPanic(tx)

	err = s.jokeStorageService.ActivateJokeConsumption(ctx, tx, userId, jokeId)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return err
	}

	s.txnManager.CommitTxn(tx)
	return nil
}

func (s *jokeService) GetRefreshRingJourney(ctx context.Context, userId *string, jokeDTO *dtos.JokeScrollDTO) ([]dtos.JokeResponseDTO, error) {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		logrus.Info("error starting transaction", err)
		return nil, errs.NewInternalServerError("failed to start transaction", err)
	}
	defer s.txnManager.RollbackOnPanic(tx)

	// Initialize genre lookup map early
	genreLookup := s.initializeGenreLookup(ctx, tx)

	// 1. Fetch all user refresh_ring jokes (not filtered by language)
	userAllRefreshRingJokes := []entities.JokeConsumptionByUser{}
	logrus.Info("user id is", *userId)
	if userId != nil {
		userAllRefreshRingJokes, err = s.jokeStorageService.GetUserTotalRefreshRingJokes(ctx, tx, *userId)
		if err != nil {
			s.txnManager.AbortTxn(tx)
			return nil, errs.NewInternalServerError("failed to get user total refresh ring jokes", err)
		}
	}
	logrus.Info("total joke is ", len(userAllRefreshRingJokes))

	// 2. If any exist, fetch fresh data from GluedIn and match with stored IDs
	if len(userAllRefreshRingJokes) > 0 {
		matchedJokes, err := s.getMatchedRefreshRingJokes(ctx, tx, userId, jokeDTO, userAllRefreshRingJokes, genreLookup)
		if err != nil {
			s.txnManager.AbortTxn(tx)
			return nil, err
		}
		s.txnManager.CommitTxn(tx)
		return matchedJokes, nil
	}

	// 3. If none exist, fetch fresh jokes and store them
	jokesWithLanguage, err := s.getFreshRefreshRingJokes(ctx, tx, userId, jokeDTO)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	// Convert jokes to response DTOs
	jokeResponseDTOs := s.convertJokesToResponseDTOs(jokesWithLanguage, genreLookup)

	s.txnManager.CommitTxn(tx)
	return jokeResponseDTOs, nil
}

func (s *jokeService) initializeGenreLookup(ctx context.Context, tx *gorm.DB) map[string]string {
	genreLookup := make(map[string]string)
	isActive := true
	allGenres, _, err := s.jokeReferenceDataService.GetAllGenres(ctx, tx, dtos.GenreFilter{IsActive: &isActive})
	if err != nil {
		logrus.Errorf("error fetching genres for lookup: %v", err)
	} else {
		for _, genre := range allGenres {
			genreLookup[genre.Genre] = genre.ImageURL
		}
	}
	return genreLookup
}

func (s *jokeService) getMatchedRefreshRingJokes(ctx context.Context, tx *gorm.DB, userId *string, jokeDTO *dtos.JokeScrollDTO, userAllRefreshRingJokes []entities.JokeConsumptionByUser, genreLookup map[string]string) ([]dtos.JokeResponseDTO, error) {
	// Get unique language IDs from stored jokes
	languageIDSet := make(map[int]bool)
	for _, joke := range userAllRefreshRingJokes {
		languageIDSet[joke.SelectedLanguage] = true
	}

	// Create a set of stored joke IDs for quick lookup
	storedJokeIDs := make(map[string]bool)
	for _, joke := range userAllRefreshRingJokes {
		storedJokeIDs[joke.JokeID] = true
	}

	var allMatchedJokes []dtos.JokeResponseDTO

	// Fetch fresh data from Gluedin for each unique language
	for languageID := range languageIDSet {
		language, err := s.jokeReferenceDataService.GetJokeLanguage(ctx, tx, languageID)
		if err != nil {
			logrus.Errorf("error getting language for ID %d: %v", languageID, err)
			continue
		}

		languageKey, err := s.jokeReferenceDataService.GetJokeLanguageKey(ctx, tx, languageID)
		if err != nil {
			logrus.Errorf("error getting language key for ID %d: %v", languageID, err)
			continue
		}
		// Create a temporary jokeDTO for this language
		tempJokeDTO := &dtos.JokeScrollDTO{
			Language: &languageID,
			Type:     jokeDTO.Type,
		}

		// Fetch jokes from Gluedin for this language
		freshJokes, err := s.gluedinService.GetRefreshRingJokesFromGluedin(userId, tempJokeDTO, nil, language, 0)
		if err != nil {
			logrus.Errorf("error fetching jokes from Gluedin for language %s: %v", language, err)
			continue
		}

		// Match fresh jokes with stored joke IDs
		matchedJokes := s.matchJokesWithStoredIDs(freshJokes, storedJokeIDs, language, languageKey, genreLookup)
		allMatchedJokes = append(allMatchedJokes, matchedJokes...)
	}

	return allMatchedJokes, nil
}

func (s *jokeService) matchJokesWithStoredIDs(freshJokes []map[string]interface{}, storedJokeIDs map[string]bool, language string, languageKey string, genreLookup map[string]string) []dtos.JokeResponseDTO {
	var matchedJokes []dtos.JokeResponseDTO

	for _, freshJoke := range freshJokes {
		if videoId, ok := freshJoke["videoId"].(string); ok {
			if storedJokeIDs[videoId] {
				// This joke exists in our stored list, add it to response
				jokeResponseDTO := s.convertJokeToResponseDTO(freshJoke, language, languageKey, genreLookup)
				matchedJokes = append(matchedJokes, jokeResponseDTO)
			}
		}
	}

	return matchedJokes
}

func (s *jokeService) getFreshRefreshRingJokes(ctx context.Context, tx *gorm.DB, userId *string, jokeDTO *dtos.JokeScrollDTO) ([]dtos.JokeWithLanguage, error) {
	language, err := s.jokeReferenceDataService.GetJokeLanguage(ctx, tx, *jokeDTO.Language)
	if err != nil {
		return nil, errs.NewInternalServerError("failed to get joke reference data", err)
	}

	languageKey, err := s.jokeReferenceDataService.GetJokeLanguageKey(ctx, tx, *jokeDTO.Language)
	if err != nil {
		logrus.Errorf("error getting language key for ID %s: %v", utils.SanitizeForLog(fmt.Sprintf("%d", *jokeDTO.Language)), err)
		languageKey = "en"
	}

	var jokes []map[string]interface{}
	jokes, err = s.gluedinService.GetRefreshRingJokesFromGluedin(userId, jokeDTO, nil, language, 0)
	if err != nil {
		logrus.Info("error getting refresh ring jokes", err)
		return nil, err
	}

	// Track jokes with their language information
	type JokeWithLanguage struct {
		Joke        map[string]interface{}
		Language    string
		LanguageID  int
		LanguageKey string
	}

	jokesWithLanguage := make([]dtos.JokeWithLanguage, 0)

	// Add primary language jokes
	for _, joke := range jokes {
		jokesWithLanguage = append(jokesWithLanguage, dtos.JokeWithLanguage{
			Joke:        joke,
			Language:    language,
			LanguageID:  *jokeDTO.Language,
			LanguageKey: languageKey,
		})
	}

	// If language is English and count < 15, also fetch Hindi jokes
	if (language == "English" || language != "Hindi") && len(jokes) < constants.MAX_SCROLL_AND_LOL_JOKE_COUNT {
		if language == "English" || language != "Hindi" {
			// Fetch Hindi jokes
			hindiLangID := 2
			hindiLang, err := s.jokeReferenceDataService.GetJokeLanguage(ctx, tx, hindiLangID)
			if err == nil {
				hindiLanguageKey, err := s.jokeReferenceDataService.GetJokeLanguageKey(ctx, tx, hindiLangID)
				if err != nil {
					hindiLanguageKey = "hi"
				}
				jokesHindi, err := s.gluedinService.GetRefreshRingJokesFromGluedin(userId, jokeDTO, nil, hindiLang, 0)
				if err == nil {
					for _, joke := range jokesHindi {
						jokesWithLanguage = append(jokesWithLanguage, dtos.JokeWithLanguage{
							Joke:        joke,
							Language:    hindiLang,
							LanguageID:  hindiLangID,
							LanguageKey: hindiLanguageKey,
						})
					}
				}
			}
		}
	}

	// Limit to 15 jokes
	if len(jokesWithLanguage) > constants.MAX_SCROLL_AND_LOL_JOKE_COUNT {
		jokesWithLanguage = jokesWithLanguage[:constants.MAX_SCROLL_AND_LOL_JOKE_COUNT]
	}

	// Save jokes for user if userId is present
	if userId != nil && len(jokesWithLanguage) > 0 {
		err = s.jokeStorageService.StoreRefreshRingJokesWithLanguage(ctx, tx, *userId, jokesWithLanguage, jokeDTO)
		if err != nil {
			logrus.Info("error storing refresh ring jokes", err)
			return nil, err
		}
	}

	return jokesWithLanguage, nil
}

func (s *jokeService) convertJokesToResponseDTOs(jokesWithLanguage []dtos.JokeWithLanguage, genreLookup map[string]string) []dtos.JokeResponseDTO {
	var jokeResponseDTOs []dtos.JokeResponseDTO
	for _, jokeWithLang := range jokesWithLanguage {
		jokeResponseDTO := s.convertJokeToResponseDTO(jokeWithLang.Joke, jokeWithLang.Language, jokeWithLang.LanguageKey, genreLookup)
		jokeResponseDTOs = append(jokeResponseDTOs, jokeResponseDTO)
	}
	return jokeResponseDTOs
}

func (s *jokeService) convertJokeToResponseDTO(joke map[string]interface{}, language string, languageKey string, genreLookup map[string]string) dtos.JokeResponseDTO {
	reactions, ok := joke["reactions"].(map[string]interface{})
	var userReaction dtos.UserReaction
	if ok {
		if laugh, ok := reactions["laugh"].(float64); ok {
			userReaction.Laugh = int(laugh)
		}
		if neutral, ok := reactions["neutral"].(float64); ok {
			userReaction.Neutral = int(neutral)
		}
		if sad, ok := reactions["sad"].(float64); ok {
			userReaction.Sad = int(sad)
		}
		if like, ok := reactions["like"].(float64); ok {
			userReaction.Like = int(like)
		}
		if share, ok := reactions["share"].(float64); ok {
			userReaction.Share = int(share)
		}
	}

	// Process labels to find matching genre and image
	labelsInterface, ok := joke["labels"]
	var genre string
	var genreImage string
	if ok && labelsInterface != nil {
		var labels []string
		switch v := labelsInterface.(type) {
		case []string:
			labels = v
		case []interface{}:
			// Convert []interface{} to []string
			for _, item := range v {
				if str, ok := item.(string); ok {
					labels = append(labels, str)
				}
			}
		}
		if len(labels) > 0 {
			// Check all labels for matching genres using the lookup map
			for _, label := range labels {
				logrus.Info("label", label)
				if imageURL, exists := genreLookup[label]; exists {
					genre = label
					genreImage = imageURL
					break // Use the first matching genre
				}
			}
			// If no matching genre found, use the first label as before
			if genre == "" && len(labels) > 2 {
				genre = labels[0]
			}
		}
	}

	// Safe type assertions
	videoId, _ := joke["videoId"].(string)
	s3Url, _ := joke["s3Url"].(string)
	contentType, _ := joke["contentType"].(string)
	thumbnailUrl, _ := joke["thumbnailUrl"].(string)
	viewsCount, _ := joke["viewsCount"].(float64)

	title := s.getLocalizedTitle(joke, languageKey)

	user, _ := joke["user"].(map[string]interface{})
	userName := ""
	userId := ""
	if user != nil {
		if un, ok := user["userName"].(string); ok {
			userName = un
		}
		if uid, ok := user["userId"].(string); ok {
			userId = uid
		}
	}

	return dtos.JokeResponseDTO{
		ID:           videoId,
		URL:          s3Url,
		Type:         contentType,
		Title:        title,
		ThumbnailUrl: thumbnailUrl,
		ViewCount:    int(viewsCount),
		Creator:      userName,
		ArtistId:     userId,
		JokeLanguage: language,
		JokeGenre:    genre,
		GenreImage:   genreImage,
		UserReaction: userReaction,
	}
}

func (s *jokeService) CheckForDuplicateContent(ctx context.Context, currentContent []byte, existingJokes []entities.UGCMaster) (bool, error) {
	for _, joke := range existingJokes {
		if joke.FileName == "" {
			continue
		}

		existingFile, err := s.infobipService.FetchMedia(ctx, joke.FileName)
		if err != nil {
			continue
		}

		existingContent, err := io.ReadAll(existingFile)
		_ = existingFile.Close()
		if err != nil {
			continue
		}

		if joke.FormatID == constants.JOKE_FORMAT_TEXT {
			if string(currentContent) == joke.FileName {
				return true, nil
			} else {
				continue
			}
		}

		if bytes.Equal(currentContent, existingContent) {
			return true, nil
		}
	}
	return false, nil
}

func isDirectURL(url string) bool {
	return strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")
}

func (s *jokeService) GetUGCAssessmentContent(ctx context.Context, request dtos.UGCAssessmentRequest, page, limit int) (*dtos.UGCAssessmentListResponse, error) {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		logrus.Info("error starting transaction", err)
		return nil, errs.NewInternalServerError("failed to start transaction", err)
	}
	defer s.txnManager.RollbackOnPanic(tx)
	filter := dtos.NewUGCAssessmentFilter(request, page, limit)
	ugcContent, totalCount, err := s.jokeStorageService.GetUGCAssessmentContent(ctx, tx, filter)
	if err != nil {
		logrus.Info("error getting UGC assessment content", err)
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	var responses []dtos.UGCAssessmentResponse
	for _, content := range ugcContent {
		var processedContentURL string

		if isDirectURL(content.ContentURL) {
			processedContentURL = content.ContentURL
		} else {
			if content.Format.Title == "Text" {
				textContent, err := s.gcsService.ReadFileContent(ctx, content.ContentURL)
				if err != nil {
					logrus.Debug("Error reading file content from GCS: ", err)
					processedContentURL = content.ContentURL
				} else {
					processedContentURL = textContent
				}
			} else {
				signedURL, err := s.gcsService.GetFileSignedURL(ctx, content.ContentURL, 60*time.Minute)
				if err != nil {
					logrus.Debug("Error generating signed URL: ", err)
					processedContentURL = content.ContentURL
				} else {
					processedContentURL = signedURL
				}
			}
		}

		response := dtos.UGCAssessmentResponse{
			ContentID:       content.AssetID,
			ContentURL:      processedContentURL,
			Title:           content.Title,
			UserID:          content.UserID,
			CreatedOn:       content.CreatedOn,
			ContentLanguage: content.Language.Name,
			Status:          getJokesStatus(content.StatusID),
		}

		if content.Genre.ID != 0 {
			response.ContentGenre = &content.Genre.Genre
		}

		responses = append(responses, response)
	}

	s.txnManager.CommitTxn(tx)

	return &dtos.UGCAssessmentListResponse{
		Data:       responses,
		TotalCount: totalCount,
		Page:       page,
		Limit:      limit,
	}, nil
}

func getJokesStatus(status int) string {
	switch status {
	case constants.UGC_STATUS_PENDING:
		return constants.UGC_STATUS_PENDING_STRING
	case constants.UGC_STATUS_APPROVED:
		return constants.UGC_STATUS_ACCEPTED_STRING
	case constants.UGC_STATUS_REJECTED:
		return constants.UGC_STATUS_REJECTED_STRING
	case constants.UGC_STATUS_PLAGIARIZED:
		return constants.UGC_STATUS_PLAGIARIZED_STRING
	}
	return constants.UGC_STATUS_PENDING_STRING
}
