package services

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Infinite-Locus-Product/coca_cola_backend/constants"
	"github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"github.com/Infinite-Locus-Product/coca_cola_backend/utils"
	"github.com/Infinite-Locus-Product/coca_cola_backend/vendors"
)

type GluedinService interface {
	GetSupriseMeJokeFromGluedin(userId string, genre string, language string, exclude *string, platform int) (map[string]interface{}, error)
	GetScrollAndLolJokesFromGluedin(userId *string, jokeDTO *dtos.JokeScrollDTO, remainingJokeCount *int, language string, userOldJokesCount int) ([]map[string]interface{}, error)
	GetSpicyTrayJokesFromGluedin(jokeDTO *dtos.JokeScrollDTO, remainingJokeCount *int, language string, userOldJokesCount int) ([]map[string]interface{}, error)
	SubmitUGCContent(userID string, joke dtos.JokeIngestionDTO, contentURL string, labels []string) (string, error)
	ChangeUGCStatus(videoID string, status string) error
	GetRefreshRingJokesFromGluedin(userId *string, jokeDTO *dtos.JokeScrollDTO, remainingJokeCount *int, language string, userOldJokesCount int) ([]map[string]interface{}, error)
}

type gluedinService struct {
	gluedinClient      *vendors.APIClient
	projectID          string
	gcsService         utils.GCSService
	gluedInUGCEndpoint string
}

func NewGluedinService(
	gluedinClient *vendors.APIClient,
	projectID string,
	gcsService utils.GCSService,
	gluedInUGCBaseURL string,
) GluedinService {
	return &gluedinService{
		gluedinClient:      gluedinClient,
		projectID:          projectID,
		gcsService:         gcsService,
		gluedInUGCEndpoint: gluedInUGCBaseURL,
	}
}

func (s *gluedinService) GetSupriseMeJokeFromGluedin(userId string, genre string, language string, exclude *string, platform int) (map[string]interface{}, error) {
	queryParams := map[string]string{}

	var labels []string
	if genre != "" {
		labels = append(labels, genre)
	}
	if language != "" {
		labels = append(labels, language)
	}
	if len(labels) > 0 {
		queryParams["labels"] = strings.Join(labels, ",")
	}

	if exclude != nil && *exclude != "" {
		queryParams["exclude"] = *exclude
	}
	queryParams["userId"] = userId

	if platform == constants.WA_PLATFORM {
		queryParams["source"] = "bot"
	} else {
		queryParams["source"] = "microsite"
	}

	logrus.Infof("Making Gluedin API request with params: %s", utils.SanitizeForLog(fmt.Sprintf("%+v", queryParams)))

	response, err := s.gluedinClient.Get(constants.GET_SUPRISE_ME_JOKE_ENDPOINT, queryParams)
	if err != nil {
		logrus.Infof("error fetching joke from gluedin: %v", err)
		return nil, errs.NewInternalServerError("error fetching joke from gluedin", err)
	}

	logrus.Info("Gluedin API response status:", response.StatusCode)
	logrus.Info("Gluedin API response body:", string(response.Body))

	var joke map[string]interface{}
	if err := response.ParseJSON(&joke); err != nil {
		return nil, errs.NewInternalServerError("error parsing joke from gluedin", err)
	}

	jokeDataInterface, ok := joke["data"].([]interface{})
	logrus.Info("jokeDataInterface", jokeDataInterface)
	if !ok || len(jokeDataInterface) == 0 {
		logrus.Info("error fetching joke from gluedin - no data or empty data array", jokeDataInterface)

		// Try without labels as fallback
		if len(labels) > 0 && platform != constants.WA_PLATFORM {
			logrus.Info("Trying fallback request without labels")
			fallbackParams := map[string]string{
				"userId": userId,
			}
			if exclude != nil && *exclude != "" {
				fallbackParams["exclude"] = *exclude
			}

			fallbackParams["source"] = "microsite"

			fallbackLabels := []string{}

			if genre != "" {
				fallbackLabels = append(fallbackLabels, genre)
			}

			fallbackLabels = append(fallbackLabels, "Hindi")
			fallbackParams["labels"] = strings.Join(fallbackLabels, ",")

			logrus.Infof("Making fallback Gluedin API request with params: %s", utils.SanitizeForLog(fmt.Sprintf("%+v", fallbackParams)))

			fallbackResponse, fallbackErr := s.gluedinClient.Get(constants.GET_SUPRISE_ME_JOKE_ENDPOINT, fallbackParams)
			if fallbackErr != nil {
				logrus.Info("Fallback request also failed:", fallbackErr)
				return nil, errs.NewInternalServerError("error fetching joke from gluedin", nil)
			}

			logrus.Info("Fallback response status:", fallbackResponse.StatusCode)
			logrus.Info("Fallback response body:", string(fallbackResponse.Body))

			var fallbackJoke map[string]interface{}
			if err := fallbackResponse.ParseJSON(&fallbackJoke); err != nil {
				return nil, errs.NewInternalServerError("error parsing fallback joke from gluedin", err)
			}

			fallbackDataInterface, fallbackOk := fallbackJoke["data"].([]interface{})
			if fallbackOk && len(fallbackDataInterface) > 0 {
				if fallbackJokeData, fallbackDataOk := fallbackDataInterface[0].(map[string]interface{}); fallbackDataOk {
					logrus.Info("Fallback request successful")
					return fallbackJokeData, nil
				}
			}

			// Fallback If joke is not found for specific genre even in hindi (Finding joke in any genre for hindi language)
			if genre != "" && fallbackOk && len(fallbackDataInterface) == 0 && platform != constants.WA_PLATFORM {
				newFallbackParams := map[string]string{
					"userId": userId,
				}

				if exclude != nil && *exclude != "" {
					newFallbackParams["exclude"] = *exclude
				}

				newFallbackParams["source"] = "microsite"

				newFallbackParams["labels"] = "Hindi"

				newFallbackResponse, newFallbackErr := s.gluedinClient.Get(constants.GET_SUPRISE_ME_JOKE_ENDPOINT, newFallbackParams)
				if newFallbackErr != nil {
					logrus.Info("Fallback request also failed:", newFallbackErr)
					return nil, errs.NewInternalServerError("error fetching joke from gluedin", nil)
				}

				var newFallbackJoke map[string]interface{}
				if err := newFallbackResponse.ParseJSON(&newFallbackJoke); err != nil {
					return nil, errs.NewInternalServerError("error parsing new fallback joke from gluedin", err)
				}

				newFallbackDataInterface, newFallbackOk := newFallbackJoke["data"].([]interface{})
				if newFallbackOk && len(newFallbackDataInterface) > 0 {
					if newFallbackJokeData, newFallbackDataOk := newFallbackDataInterface[0].(map[string]interface{}); newFallbackDataOk {
						logrus.Info("New fallback request successful")
						return newFallbackJokeData, nil
					}
				}
			}
		}

		return nil, errs.NewAppError(1021, "No Jokes in this Criteria", 204, nil)
	}

	jokeData, ok := jokeDataInterface[0].(map[string]interface{})
	if !ok {
		logrus.Info("error parsing joke data structure", jokeDataInterface)
		return nil, errs.NewInternalServerError("error parsing joke data structure", nil)
	}

	fallbackAtGluedIn, _ := jokeData["fallbackResult"].(bool)

	if fallbackAtGluedIn && platform == constants.WA_PLATFORM {
		logrus.Info("Fallback at Gluedin : ", fallbackAtGluedIn)
		return nil, errs.NewAppError(1021, "No Jokes in this Criteria", 204, nil)
	}

	return jokeData, nil
}

func (s *gluedinService) GetScrollAndLolJokesFromGluedin(userId *string, jokeDTO *dtos.JokeScrollDTO, remainingJokeCount *int, language string, userOldJokesCount int) ([]map[string]interface{}, error) {
	queryParams := map[string]string{}
	if jokeDTO.Limit != nil && *jokeDTO.Limit > 0 {
		queryParams["pageSize"] = fmt.Sprintf("%d", *jokeDTO.Limit)
	} else {
		queryParams["pageSize"] = fmt.Sprintf("%d", constants.DEFAULT_LIMIT)
	}
	if jokeDTO.SelectedJoke != nil {
		queryParams["selectedJoke"] = *jokeDTO.SelectedJoke
	}
	if jokeDTO.PreferredJokes != nil {
		queryParams["preferredJokes"] = *jokeDTO.PreferredJokes
	}
	if remainingJokeCount != nil {
		queryParams["pageSize"] = fmt.Sprintf("%d", *remainingJokeCount)
	}
	if userOldJokesCount > 0 {
		queryParams["pageSize"] = fmt.Sprintf("%d", userOldJokesCount)
	}

	labels := []string{}

	labels = append(labels, constants.LANGUAGE_MNEMONICS[language]+constants.SCROLL_AND_LOL_LABEL)

	queryParams["labels"] = strings.Join(labels, ",")

	logrus.Infof("queryParams::: %s", utils.SanitizeForLog(fmt.Sprintf("%+v", queryParams)))

	response, err := s.gluedinClient.Get(constants.GET_SCROLL_AND_LOL_JOKE_ENDPOINT, queryParams)
	if err != nil {
		return nil, errs.NewInternalServerError("error fetching joke from gluedin", err)
	}

	var jokes map[string]interface{}
	if err := response.ParseJSON(&jokes); err != nil {
		logrus.Info("error parsing joke from gluedin", response)
		return nil, errs.NewInternalServerError("error parsing joke from gluedin", err)
	}

	jokeDataInterface, ok := jokes["data"].([]interface{})
	if !ok {
		logrus.Info("error fetching joke from gluedin", jokes)
		return nil, errs.NewInternalServerError("error fetching joke from gluedin", nil)
	}

	var jokeData []map[string]interface{}
	for _, jokeInterface := range jokeDataInterface {
		joke, ok := jokeInterface.(map[string]interface{})
		if !ok {
			logrus.Info("error parsing joke from gluedin", jokeInterface)
			continue
		}
		jokeData = append(jokeData, joke)
	}

	return jokeData, nil
}

func (s *gluedinService) GetSpicyTrayJokesFromGluedin(jokeDTO *dtos.JokeScrollDTO, remainingJokeCount *int, language string, userOldJokesCount int) ([]map[string]interface{}, error) {
	queryParams := map[string]string{}
	if jokeDTO.Limit != nil && *jokeDTO.Limit > 0 {
		if *jokeDTO.Limit > constants.MAX_SPICY_TRAY_JOKE_COUNT {
			queryParams["pageSize"] = fmt.Sprintf("%d", constants.MAX_SPICY_TRAY_JOKE_COUNT)
		} else {
			queryParams["pageSize"] = fmt.Sprintf("%d", *jokeDTO.Limit)
		}
	} else {
		queryParams["pageSize"] = fmt.Sprintf("%d", constants.DEFAULT_LIMIT)
	}
	if remainingJokeCount != nil {
		queryParams["pageSize"] = fmt.Sprintf("%d", *remainingJokeCount)
	}
	if userOldJokesCount > 0 {
		queryParams["pageSize"] = fmt.Sprintf("%d", userOldJokesCount)
	}

	labels := []string{}
	if jokeDTO.SelectedJoke != nil {
		queryParams["selectedJoke"] = *jokeDTO.SelectedJoke
	}
	if jokeDTO.PreferredJokes != nil {
		queryParams["preferredJokes"] = *jokeDTO.PreferredJokes
	}
	labels = append(labels, constants.SPICY_TRAY_LABEL+"_"+constants.LANGUAGE_MNEMONICS[language])
	queryParams["labels"] = strings.Join(labels, ",")

	response, err := s.gluedinClient.Get(constants.GET_SCROLL_AND_LOL_JOKE_ENDPOINT, queryParams)
	if err != nil {
		return nil, errs.NewInternalServerError("error fetching spicy tray jokes from gluedin", err)
	}

	var jokes map[string]interface{}
	if err := response.ParseJSON(&jokes); err != nil {
		return nil, errs.NewInternalServerError("error parsing spicy tray jokes from gluedin", err)
	}

	jokeDataInterface, ok := jokes["data"].([]interface{})
	if !ok {
		return nil, errs.NewInternalServerError("error fetching spicy tray jokes from gluedin", nil)
	}

	var jokeData []map[string]interface{}
	for _, jokeInterface := range jokeDataInterface {
		joke, ok := jokeInterface.(map[string]interface{})
		if !ok {
			continue
		}
		jokeData = append(jokeData, joke)
	}

	if len(jokeData) > constants.MAX_SCROLL_AND_LOL_JOKE_COUNT {
		jokeData = jokeData[:constants.MAX_SCROLL_AND_LOL_JOKE_COUNT]
	}

	return jokeData, nil
}

func (s *gluedinService) GetRefreshRingJokesFromGluedin(userId *string, jokeDTO *dtos.JokeScrollDTO, remainingJokeCount *int, language string, userOldJokesCount int) ([]map[string]interface{}, error) {
	queryParams := map[string]string{}
	if jokeDTO.Limit != nil && *jokeDTO.Limit > 0 {
		queryParams["pageSize"] = fmt.Sprintf("%d", *jokeDTO.Limit)
	} else {
		queryParams["pageSize"] = fmt.Sprintf("%d", constants.DEFAULT_LIMIT)
	}
	if jokeDTO.SelectedJoke != nil {
		queryParams["selectedJoke"] = *jokeDTO.SelectedJoke
	}
	if jokeDTO.PreferredJokes != nil {
		queryParams["preferredJokes"] = *jokeDTO.PreferredJokes
	}
	if remainingJokeCount != nil {
		queryParams["pageSize"] = fmt.Sprintf("%d", *remainingJokeCount)
	}
	if userOldJokesCount > 0 {
		queryParams["pageSize"] = fmt.Sprintf("%d", userOldJokesCount)
	}

	labels := []string{}
	labels = append(labels, constants.REFRESH_RING_LABEL+language)
	queryParams["labels"] = strings.Join(labels, ",")
	queryParams["rr_feed"] = "true"

	logrus.Infof("queryParams %s", utils.SanitizeForLog(fmt.Sprintf("%+v", queryParams)))

	response, err := s.gluedinClient.Get(constants.GET_SCROLL_AND_LOL_JOKE_ENDPOINT, queryParams)
	if err != nil {
		return nil, errs.NewInternalServerError("error fetching refresh ring joke from gluedin", err)
	}

	var jokes map[string]interface{}
	if err := response.ParseJSON(&jokes); err != nil {
		logrus.Info("error parsing joke from gluedin", response)
		return nil, errs.NewInternalServerError("error parsing joke from gluedin", err)
	}

	jokeDataInterface, ok := jokes["data"].([]interface{})
	if !ok {
		logrus.Info("error fetching joke from gluedin", jokes)
		return nil, errs.NewInternalServerError("error fetching joke from gluedin", nil)
	}

	var jokeData []map[string]interface{}
	for _, jokeInterface := range jokeDataInterface {
		joke, ok := jokeInterface.(map[string]interface{})
		if !ok {
			logrus.Info("error parsing joke from gluedin", jokeInterface)
			continue
		}
		jokeData = append(jokeData, joke)
	}

	return jokeData, nil
}

func (s *gluedinService) SubmitUGCContent(userID string, joke dtos.JokeIngestionDTO, contentURL string, labels []string) (string, error) {
	artistId := fmt.Sprintf("%s_%s", userID, s.projectID)
	assetID := uuid.New().String()
	body := map[string]interface{}{
		"id":               assetID,
		"title":            joke.Title,
		"description":      joke.Title,
		"userId":           userID,
		"status":           "INACTIVE",
		"contentType":      strings.ToLower(joke.Format),
		"createdTimestamp": time.Now().UnixMilli(),
		"updatedTimestamp": time.Now().UnixMilli(),
		"projectId":        s.projectID,
		"labels":           labels,
		"updateRequest":    false,
		"artists":          []string{artistId},
		"socialType":       "UGC",
		"jokeGenreId":      joke.JokeGenreID,
	}

	gluedInTextThumbnailLink, _ := s.gcsService.GetFileSignedURL(context.Background(), constants.GLUEDIN_TEXT_THUMBNAIL_LINK, 60*time.Minute)
	if strings.ToLower(joke.Format) == "text" {
		body["thumbnailUrl"] = gluedInTextThumbnailLink
		body["thumbnails"] = map[string]interface{}{
			"desktop": gluedInTextThumbnailLink,
			"mobile":  gluedInTextThumbnailLink,
			"seeall":  gluedInTextThumbnailLink,
		}
	}

	response, err := s.gluedinClient.Post(s.gluedInUGCEndpoint, body, nil, false)
	if err != nil {
		logrus.Info("submit ugc content to gluedin", err)
		return "", errs.NewInternalServerError("error submitting ugc content to gluedin", err)
	}

	if response.StatusCode != http.StatusCreated {
		logrus.Info("error submitting ugc content to gluedin", response.Body)
		return "", errs.NewInternalServerError("error submitting ugc content to gluedin", nil)
	}

	var responseBody map[string]interface{}
	if err := response.ParseJSON(&responseBody); err != nil {
		logrus.Info("error parsing response from gluedin", response.Body)
		return "", errs.NewInternalServerError("error parsing response from gluedin", err)
	}

	if responseBody["success"] != true {
		return "", errs.NewInternalServerError("error submitting ugc content to gluedin", nil)
	}

	return assetID, nil
}

func (s *gluedinService) ChangeUGCStatus(videoID string, status string) error {
	body := map[string]interface{}{
		"videoId": videoID,
		"status":  status,
	}

	response, err := s.gluedinClient.Post(fmt.Sprintf(constants.CHANGE_UGC_STATUS_ENDPOINT, s.gluedInUGCEndpoint), body, nil, false)
	if err != nil {
		logrus.Info("error changing ugc status to gluedin", err)
		return errs.NewInternalServerError("error changing ugc status to gluedin", err)
	}

	if response.StatusCode != http.StatusCreated {
		logrus.Info("error changing ugc status to gluedin", response.StatusCode)
		return errs.NewInternalServerError("error changing ugc status to gluedin", nil)
	}

	var responseBody map[string]interface{}
	if err := response.ParseJSON(&responseBody); err != nil {
		logrus.Info("error parsing response from gluedin", responseBody)
		return errs.NewInternalServerError("error parsing response from gluedin", err)
	}

	logrus.Info("responseBody", responseBody)

	if responseBody["success"] != true {
		return errs.NewInternalServerError("error changing ugc status to gluedin", nil)
	}

	return nil
}
