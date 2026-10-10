package services

import (
	"context"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Infinite-Locus-Product/coca_cola_backend/constants"
	"github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"github.com/Infinite-Locus-Product/coca_cola_backend/repository"
	"github.com/Infinite-Locus-Product/coca_cola_backend/utils"
)

type JokeStorageService interface {
	GetSurpriseMeJokeCount(ctx context.Context, tx *gorm.DB, userId string, platform int) (int, error)
	StoreSurpriseMeJoke(ctx context.Context, tx *gorm.DB, userId string, joke map[string]interface{}, jokeDTO *dtos.JokeDTO, genre string, language string, platform int) (map[string]interface{}, error)
	StoreUGCJoke(ctx context.Context, tx *gorm.DB, userId string, joke dtos.JokeIngestionDTO, formatID int, contentURL string, assetID string, platformID int, createdBy string, mediaID string) error
	GetUGCContentByIDs(ctx context.Context, tx *gorm.DB, ids []string) ([]dtos.UGCContentByIDsResponseDTO, error)
	GetProfileUGCContent(ctx context.Context, tx *gorm.DB, userId string) ([]dtos.UGCContentResponseDTO, error)
	GetUGCContentArtist(ctx context.Context, tx *gorm.DB, ids []string) ([]dtos.UGCContentArtistDTO, error)
	GetCountOfUGCContent(ctx context.Context, tx *gorm.DB, userId string) (int, error)
	GetScrollAndLolJokeCount(ctx context.Context, tx *gorm.DB, userId string) (int, error)
	StoreScrollAndLolJoke(ctx context.Context, tx *gorm.DB, userId string, jokes []map[string]interface{}, jokeDTO *dtos.JokeScrollDTO, language string) ([]map[string]interface{}, error)
	FetchScrollAndLolJokeByLanguage(ctx context.Context, tx *gorm.DB, languageID int, userId string) ([]entities.JokeConsumptionByUser, error)
	UpdateUGCStatus(ctx context.Context, tx *gorm.DB, assetID string, statusID int, isActive bool) error
	GetUGCContentByAssetID(ctx context.Context, tx *gorm.DB, assetID string, fetchOnlyInactive bool) (*entities.UGCMaster, error)
	GetUserTotalJokes(ctx context.Context, tx *gorm.DB, userId string) ([]entities.JokeConsumptionByUser, error)
	ActivateJokeConsumption(ctx context.Context, tx *gorm.DB, userId string, jokeId string) error
	GetRefreshRingJokeCount(ctx context.Context, tx *gorm.DB, userId string) (int, error)
	StoreRefreshRingJoke(ctx context.Context, tx *gorm.DB, userId string, jokes []map[string]interface{}, jokeDTO *dtos.JokeScrollDTO, language string) ([]map[string]interface{}, error)
	StoreRefreshRingJokesWithLanguage(ctx context.Context, tx *gorm.DB, userId string, jokesWithLanguage []dtos.JokeWithLanguage, jokeDTO *dtos.JokeScrollDTO) error
	FetchRefreshRingJokeByLanguage(ctx context.Context, tx *gorm.DB, languageID int, userId string) ([]entities.JokeConsumptionByUser, error)
	GetUserTotalRefreshRingJokes(ctx context.Context, tx *gorm.DB, userId string) ([]entities.JokeConsumptionByUser, error)
	GetTodaysUGCJokesByUserPlatformFormat(ctx context.Context, tx *gorm.DB, userId string, platformID int, formatID int) ([]entities.UGCMaster, error)
	GetUGCAssessmentContent(ctx context.Context, tx *gorm.DB, filter dtos.UGCAssessmentFilter) ([]entities.UGCMaster, int64, error)

	GetSpicyTrayJokeCount(ctx context.Context, tx *gorm.DB, userId string) (int, error)
	StoreSpicyTrayJokes(ctx context.Context, tx *gorm.DB, userId string, jokes []map[string]interface{}, jokeDTO *dtos.JokeScrollDTO, language string) ([]map[string]interface{}, error)
	FetchSpicyTrayJokeByLanguage(ctx context.Context, tx *gorm.DB, languageID int, userId string) ([]entities.JokeConsumptionByUser, error)
}

type jokeStorageService struct {
	jokeConsumptionRepository repository.JokeConsumptionRepository
	jokeDetailsRepository     repository.JokeDetailsRepository
	userVideoRepository       repository.UserVideoRepository
	ugcMasterRepository       repository.UGCMasterRepository
	gcsService                utils.GCSService
}

func NewJokeStorageService(jokeConsumptionRepository repository.JokeConsumptionRepository, jokeDetailsRepository repository.JokeDetailsRepository, userVideoRepository repository.UserVideoRepository, ugcMasterRepository repository.UGCMasterRepository, gcsService utils.GCSService) JokeStorageService {
	return &jokeStorageService{
		jokeConsumptionRepository: jokeConsumptionRepository,
		jokeDetailsRepository:     jokeDetailsRepository,
		userVideoRepository:       userVideoRepository,
		ugcMasterRepository:       ugcMasterRepository,
		gcsService:                gcsService,
	}
}

func (s *jokeStorageService) GetSurpriseMeJokeCount(ctx context.Context, tx *gorm.DB, userId string, platform int) (int, error) {
	jokeCount, err := s.jokeConsumptionRepository.GetSupriseMeJokeConsumptionByUserID(ctx, tx, userId, platform)
	if err != nil {
		return 0, err
	}
	return int(jokeCount), nil
}

func (s *jokeStorageService) StoreSurpriseMeJoke(ctx context.Context, tx *gorm.DB, userId string, joke map[string]interface{}, jokeDTO *dtos.JokeDTO, genre string, language string, platform int) (map[string]interface{}, error) {
	userVideo := &entities.UserVideo{
		UserID:       userId,
		AssetID:      joke["id"].(string),
		VideoWatchOn: time.Now(),
		CreatedBy:    userId,
	}

	if err := s.userVideoRepository.Create(ctx, tx, userVideo); err != nil {
		return nil, err
	}
	jokeDetail := &entities.JokeDetailsRecieved{
		JokeTitle:       joke["title"].(string),
		JokeContentType: joke["contentType"].(string),
		JokeDescription: joke["description"].(string),
		JokeLanguage:    language,
		JokeGenre:       genre,
		JokeCreator:     joke["user"].(map[string]interface{})["userName"].(string),
		CreatedBy:       userId,
		CreatedOn:       time.Now(),
	}
	if err := s.jokeDetailsRepository.Create(ctx, tx, jokeDetail); err != nil {
		return nil, err
	}

	jokeConsumption := &entities.JokeConsumptionByUser{
		UserID:           userId,
		JokeID:           joke["id"].(string),
		JokeDetailID:     jokeDetail.ID,
		ArtistID:         joke["user"].(map[string]interface{})["userId"].(string),
		Platform:         platform,
		SimilarJoke:      0,
		SelectedGenre:    *jokeDTO.Genre,
		SelectedLanguage: *jokeDTO.Language,
		JokeGeneratedAt:  time.Now(),
		CreatedBy:        userId,
		CreatedOn:        time.Now(),
		JokeType:         entities.SupriseMeJoke,
	}
	if err := s.jokeConsumptionRepository.Create(ctx, tx, jokeConsumption); err != nil {
		return nil, err
	}

	return joke, nil
}

func (s *jokeStorageService) GetScrollAndLolJokeCount(ctx context.Context, tx *gorm.DB, userId string) (int, error) {
	jokeCount, err := s.jokeConsumptionRepository.GetScrollAndLolJokeConsumptionByUserID(ctx, tx, userId)
	if err != nil {
		return 0, err
	}
	return int(jokeCount), nil
}

func (s *jokeStorageService) GetSpicyTrayJokeCount(ctx context.Context, tx *gorm.DB, userId string) (int, error) {
	jokeCount, err := s.jokeConsumptionRepository.GetSpicyTrayJokeConsumptionByUserID(ctx, tx, userId)
	if err != nil {
		return 0, err
	}
	return int(jokeCount), nil
}

func (s *jokeStorageService) StoreScrollAndLolJoke(ctx context.Context, tx *gorm.DB, userId string, jokes []map[string]interface{}, jokeDTO *dtos.JokeScrollDTO, language string) ([]map[string]interface{}, error) {
	jokeDetails := make([]entities.JokeDetailsRecieved, 0)
	for _, joke := range jokes {
		jokeDetails = append(jokeDetails, entities.JokeDetailsRecieved{
			JokeTitle:       joke["title"].(string),
			JokeContentType: joke["contentType"].(string),
			JokeDescription: joke["description"].(string),
			JokeLanguage:    language,
			JokeCreator:     joke["user"].(map[string]interface{})["userName"].(string),
			CreatedBy:       userId,
			CreatedOn:       time.Now(),
		})
	}

	if err := s.jokeDetailsRepository.CreateMany(ctx, tx, jokeDetails); err != nil {
		return nil, err
	}

	jokeConsumption := make([]entities.JokeConsumptionByUser, 0)
	for i, joke := range jokes {
		jokeConsumption = append(jokeConsumption, entities.JokeConsumptionByUser{
			UserID:           userId,
			JokeID:           joke["videoId"].(string),
			JokeDetailID:     jokeDetails[i].ID,
			ArtistID:         joke["user"].(map[string]interface{})["userId"].(string),
			SimilarJoke:      0,
			SelectedLanguage: *jokeDTO.Language,
			JokeGeneratedAt:  time.Now(),
			CreatedBy:        userId,
			CreatedOn:        time.Now(),
			JokeType:         entities.ScrollAndLolJoke,
		})
	}

	if err := s.jokeConsumptionRepository.CreateMany(ctx, tx, jokeConsumption); err != nil {
		return nil, err
	}

	return jokes, nil
}

func (s *jokeStorageService) StoreSpicyTrayJokes(ctx context.Context, tx *gorm.DB, userId string, jokes []map[string]interface{}, jokeDTO *dtos.JokeScrollDTO, language string) ([]map[string]interface{}, error) {
	if len(jokes) == 0 {
		return jokes, nil
	}
	jokeDetails := make([]entities.JokeDetailsRecieved, 0)
	for _, joke := range jokes {
		jokeDetails = append(jokeDetails, entities.JokeDetailsRecieved{
			JokeTitle:       joke["title"].(string),
			JokeContentType: joke["contentType"].(string),
			JokeDescription: joke["description"].(string),
			JokeLanguage:    language,
			JokeCreator:     joke["user"].(map[string]interface{})["userName"].(string),
			CreatedBy:       userId,
			CreatedOn:       time.Now(),
		})
	}

	if err := s.jokeDetailsRepository.CreateMany(ctx, tx, jokeDetails); err != nil {
		return nil, err
	}

	jokeConsumption := make([]entities.JokeConsumptionByUser, 0)
	for i, joke := range jokes {
		jokeConsumption = append(jokeConsumption, entities.JokeConsumptionByUser{
			UserID:           userId,
			JokeID:           joke["videoId"].(string),
			JokeDetailID:     jokeDetails[i].ID,
			ArtistID:         joke["user"].(map[string]interface{})["userId"].(string),
			SimilarJoke:      0,
			SelectedLanguage: *jokeDTO.Language,
			JokeGeneratedAt:  time.Now(),
			CreatedBy:        userId,
			CreatedOn:        time.Now(),
			JokeType:         entities.SpicyTrayJoke,
		})
	}

	if err := s.jokeConsumptionRepository.CreateMany(ctx, tx, jokeConsumption); err != nil {
		return nil, err
	}

	return jokes, nil
}

func (s *jokeStorageService) FetchScrollAndLolJokeByLanguage(ctx context.Context, tx *gorm.DB, languageID int, userId string) ([]entities.JokeConsumptionByUser, error) {
	jokes, err := s.jokeConsumptionRepository.FetchScrollAndLolJokeByLanguage(ctx, tx, languageID, userId)
	if err != nil {
		return nil, err
	}

	return jokes, nil
}

func (s *jokeStorageService) FetchSpicyTrayJokeByLanguage(ctx context.Context, tx *gorm.DB, languageID int, userId string) ([]entities.JokeConsumptionByUser, error) {
	jokes, err := s.jokeConsumptionRepository.FetchSpicyTrayJokeByLanguage(ctx, tx, languageID, userId)
	if err != nil {
		return nil, err
	}
	return jokes, nil
}

func (s *jokeStorageService) StoreUGCJoke(ctx context.Context, tx *gorm.DB, userId string, joke dtos.JokeIngestionDTO, formatID int, contentURL string, assetID string, platformID int, createdBy string, mediaID string) error {
	now := time.Now()
	defaultGenreID := 1 // Default genre ID

	filename := strings.Split(contentURL, "/")[len(strings.Split(contentURL, "/"))-1]
	if len(mediaID) != 0 {
		filename = mediaID
	}
	ugcMaster := &entities.UGCMaster{
		UserID:         userId,
		JokeGenreID:    &defaultGenreID, // Use default genre ID if none provided
		LanguageID:     joke.LanguageID,
		FormatID:       formatID,
		AssetID:        assetID,
		Title:          joke.Title,
		ContentURL:     contentURL,
		FileName:       filename,
		GluedinFill:    true,
		PlatformID:     platformID,
		StatusID:       1,
		IsActive:       false,
		IsDeleted:      false,
		CreatedBy:      createdBy,
		CreatedOn:      now,
		LastModifiedOn: &now,
	}
	if err := s.ugcMasterRepository.Create(ctx, tx, ugcMaster); err != nil {
		return err
	}
	return nil
}

func (s *jokeStorageService) GetUGCContentByIDs(ctx context.Context, tx *gorm.DB, ids []string) ([]dtos.UGCContentByIDsResponseDTO, error) {
	ugcContent, err := s.ugcMasterRepository.GetUGCContentByIDs(ctx, tx, ids, false)
	if err != nil {
		return nil, err
	}

	if len(ugcContent) == 0 {
		return []dtos.UGCContentByIDsResponseDTO{}, nil
	}

	// Create a map of normalized asset IDs to their original request IDs for case-insensitive matching
	requestIdMap := make(map[string]string)
	for _, id := range ids {
		requestIdMap[strings.ToLower(id)] = id
	}

	ugcContentDTOs := make([]dtos.UGCContentByIDsResponseDTO, 0)
	for _, content := range ugcContent {
		// Normalize the database asset ID to lowercase for comparison
		normalizedAssetID := strings.ToLower(content.AssetID)

		// Find the original request ID that matches this content
		if originalRequestID, exists := requestIdMap[normalizedAssetID]; exists {
			var responseContent string
			// Handle different content types
			if content.Format.Title == "Text" {
				// For text files, read the content directly
				textContent, err := s.gcsService.ReadFileContent(ctx, content.ContentURL)
				if err != nil {
					logrus.Debug("Error reading file content from GCS: ", err)
				}
				responseContent = textContent
			} else {
				// For media files (Image, Audio, Video), generate signed URL
				signedURL, err := s.gcsService.GetFileSignedURL(ctx, content.ContentURL, 60*time.Minute)
				if err != nil {
					logrus.Debug("Error generating signed URL: ", err)
				}
				responseContent = signedURL
			}

			ugcContentDTOs = append(ugcContentDTOs, dtos.UGCContentByIDsResponseDTO{
				ID:      originalRequestID, // Use the original request ID
				Content: responseContent,
			})
		}
	}
	return ugcContentDTOs, nil
}

func (s *jokeStorageService) GetProfileUGCContent(ctx context.Context, tx *gorm.DB, userId string) ([]dtos.UGCContentResponseDTO, error) {
	ugcContent, err := s.ugcMasterRepository.GetUGCContentByUserID(ctx, tx, userId)
	if err != nil {
		return nil, err
	}

	if len(ugcContent) == 0 {
		return []dtos.UGCContentResponseDTO{}, nil
	}

	ugcContentDTOs := make([]dtos.UGCContentResponseDTO, len(ugcContent))
	signedURL, _ := s.gcsService.GetFileSignedURL(ctx, constants.GLUEDIN_TEXT_THUMBNAIL_LINK, 24*time.Hour)
	for i, content := range ugcContent {
		ugcContentDTOs[i] = dtos.UGCContentResponseDTO{
			ID:           content.AssetID,
			Title:        content.Title,
			ThumbnailUrl: signedURL,
			Format:       content.Format.Title,
			Date:         content.CreatedOn.Format(time.RFC3339),
		}
		switch content.Status.Status {
		case "Pending":
			ugcContentDTOs[i].Status = "Pending"
		case "Successful":
			ugcContentDTOs[i].Status = "Accepted"
		case "Rejected":
			ugcContentDTOs[i].Status = "Rejected"
		}
	}
	return ugcContentDTOs, nil
}

func (s *jokeStorageService) GetUGCContentArtist(ctx context.Context, tx *gorm.DB, ids []string) ([]dtos.UGCContentArtistDTO, error) {
	ugcContent, err := s.ugcMasterRepository.GetUGCContentWithArtist(ctx, tx, ids)
	if err != nil {
		return nil, err
	}

	if len(ugcContent) == 0 {
		return []dtos.UGCContentArtistDTO{}, nil
	}

	ugcContentDTOs := make([]dtos.UGCContentArtistDTO, len(ugcContent))
	for i, content := range ugcContent {
		ugcContentDTOs[i] = dtos.UGCContentArtistDTO{
			ID:         content.AssetID,
			ArtistName: *content.User.Name,
		}
	}
	return ugcContentDTOs, nil
}

func (s *jokeStorageService) GetCountOfUGCContent(ctx context.Context, tx *gorm.DB, userId string) (int, error) {
	count, err := s.ugcMasterRepository.GetCountOfUGCContent(ctx, tx, userId)
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

func (s *jokeStorageService) UpdateUGCStatus(ctx context.Context, tx *gorm.DB, assetID string, statusID int, isActive bool) error {
	err := s.ugcMasterRepository.UpdateUGCStatus(ctx, tx, assetID, statusID, isActive)
	if err != nil {
		return err
	}
	return nil
}

func (s *jokeStorageService) GetUGCContentByAssetID(ctx context.Context, tx *gorm.DB, assetID string, fetchOnlyInactive bool) (*entities.UGCMaster, error) {
	ugcContent, err := s.ugcMasterRepository.GetUGCContentByIDs(ctx, tx, []string{assetID}, fetchOnlyInactive)
	if err != nil {
		return nil, err
	}
	if len(ugcContent) == 0 {
		return nil, errs.NewNotFoundError("UGC content not found", nil)
	}
	return &ugcContent[0], nil
}

func (s *jokeStorageService) GetUserTotalJokes(ctx context.Context, tx *gorm.DB, userId string) ([]entities.JokeConsumptionByUser, error) {
	jokes, err := s.jokeConsumptionRepository.GetUserTotalJokes(ctx, tx, userId)
	if err != nil {
		return nil, err
	}
	return jokes, nil
}

func (s *jokeStorageService) ActivateJokeConsumption(ctx context.Context, tx *gorm.DB, userId string, jokeId string) error {
	err := s.jokeConsumptionRepository.ActivateJokeConsumption(ctx, tx, userId, jokeId)
	if err != nil {
		return err
	}
	return nil
}

func (s *jokeStorageService) GetRefreshRingJokeCount(ctx context.Context, tx *gorm.DB, userId string) (int, error) {
	jokeCount, err := s.jokeConsumptionRepository.GetRefreshRingJokeConsumptionByUserID(ctx, tx, userId)
	if err != nil {
		return 0, err
	}
	return int(jokeCount), nil
}

func (s *jokeStorageService) StoreRefreshRingJoke(ctx context.Context, tx *gorm.DB, userId string, jokes []map[string]interface{}, jokeDTO *dtos.JokeScrollDTO, language string) ([]map[string]interface{}, error) {
	if len(jokes) == 0 {
		return jokes, nil
	}
	jokeDetails := make([]entities.JokeDetailsRecieved, 0)
	for _, joke := range jokes {
		jokeDetails = append(jokeDetails, entities.JokeDetailsRecieved{
			JokeTitle:       joke["title"].(string),
			JokeContentType: joke["contentType"].(string),
			JokeDescription: joke["description"].(string),
			JokeLanguage:    language,
			JokeCreator:     joke["user"].(map[string]interface{})["userName"].(string),
			CreatedBy:       userId,
			CreatedOn:       time.Now(),
		})
	}

	if err := s.jokeDetailsRepository.CreateMany(ctx, tx, jokeDetails); err != nil {
		return nil, err
	}

	jokeConsumption := make([]entities.JokeConsumptionByUser, 0)
	for i, joke := range jokes {
		jokeConsumption = append(jokeConsumption, entities.JokeConsumptionByUser{
			UserID:           userId,
			JokeID:           joke["videoId"].(string),
			JokeDetailID:     jokeDetails[i].ID,
			ArtistID:         joke["user"].(map[string]interface{})["userId"].(string),
			SimilarJoke:      0,
			SelectedLanguage: *jokeDTO.Language,
			JokeGeneratedAt:  time.Now(),
			CreatedBy:        userId,
			CreatedOn:        time.Now(),
			JokeType:         entities.RefreshRingJoke,
		})
	}

	if err := s.jokeConsumptionRepository.CreateMany(ctx, tx, jokeConsumption); err != nil {
		return nil, err
	}

	return jokes, nil
}

func (s *jokeStorageService) FetchRefreshRingJokeByLanguage(ctx context.Context, tx *gorm.DB, languageID int, userId string) ([]entities.JokeConsumptionByUser, error) {
	jokes, err := s.jokeConsumptionRepository.FetchRefreshRingJokeByLanguage(ctx, tx, languageID, userId)
	if err != nil {
		return nil, err
	}
	return jokes, nil
}

func (s *jokeStorageService) StoreRefreshRingJokesWithLanguage(ctx context.Context, tx *gorm.DB, userId string, jokesWithLanguage []dtos.JokeWithLanguage, jokeDTO *dtos.JokeScrollDTO) error {
	if len(jokesWithLanguage) == 0 {
		return nil
	}

	jokeDetails := make([]entities.JokeDetailsRecieved, 0)
	for _, jokeWithLang := range jokesWithLanguage {
		joke := jokeWithLang.Joke
		jokeDetails = append(jokeDetails, entities.JokeDetailsRecieved{
			JokeTitle:       joke["title"].(string),
			JokeContentType: joke["contentType"].(string),
			JokeDescription: joke["description"].(string),
			JokeLanguage:    jokeWithLang.Language,
			JokeCreator:     joke["user"].(map[string]interface{})["userName"].(string),
			CreatedBy:       userId,
			CreatedOn:       time.Now(),
		})
	}

	if err := s.jokeDetailsRepository.CreateMany(ctx, tx, jokeDetails); err != nil {
		return err
	}

	jokeConsumption := make([]entities.JokeConsumptionByUser, 0)
	for i, jokeWithLang := range jokesWithLanguage {
		joke := jokeWithLang.Joke
		jokeConsumption = append(jokeConsumption, entities.JokeConsumptionByUser{
			UserID:           userId,
			JokeID:           joke["videoId"].(string),
			JokeDetailID:     jokeDetails[i].ID,
			ArtistID:         joke["user"].(map[string]interface{})["userId"].(string),
			SimilarJoke:      0,
			SelectedLanguage: jokeWithLang.LanguageID,
			JokeGeneratedAt:  time.Now(),
			CreatedBy:        userId,
			CreatedOn:        time.Now(),
			JokeType:         entities.RefreshRingJoke,
		})
	}

	if err := s.jokeConsumptionRepository.CreateMany(ctx, tx, jokeConsumption); err != nil {
		return err
	}

	return nil
}

func (s *jokeStorageService) GetUserTotalRefreshRingJokes(ctx context.Context, tx *gorm.DB, userId string) ([]entities.JokeConsumptionByUser, error) {
	jokes, err := s.jokeConsumptionRepository.GetUserTotalRefreshRingJokes(ctx, tx, userId)
	if err != nil {
		return nil, err
	}
	return jokes, nil
}

func (s *jokeStorageService) GetTodaysUGCJokesByUserPlatformFormat(ctx context.Context, tx *gorm.DB, userId string, platformID int, formatID int) ([]entities.UGCMaster, error) {
	jokes, err := s.ugcMasterRepository.GetUGCContent(ctx, tx, userId, platformID, formatID)
	if err != nil {
		return nil, err
	}
	return jokes, nil
}

func (s *jokeStorageService) GetUGCAssessmentContent(ctx context.Context, tx *gorm.DB, filter dtos.UGCAssessmentFilter) ([]entities.UGCMaster, int64, error) {
	ugcContent, totalCount, err := s.ugcMasterRepository.GetUGCAssessmentContent(ctx, tx, filter)
	if err != nil {
		return nil, 0, err
	}
	return ugcContent, totalCount, nil
}
