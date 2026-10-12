package services

import (
	"context"
	"fmt"
	"mime/multipart"
	"strings"
	"time"

	"github.com/Infinite-Locus-Product/coca_cola_backend/constants"
	"github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"github.com/Infinite-Locus-Product/coca_cola_backend/repository"
	"github.com/Infinite-Locus-Product/coca_cola_backend/utils"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const (
	MEME_NOT_FOUND_ERROR = "Meme not found"
)

type MemeFeedResult struct {
	Meme           entities.Meme
	AssetURL       string
	LanguageCode   string
	ReactionCounts dtos.ReactionCount
	ReactionClick  dtos.ReactionClick
	HasVoted       bool
	Username       *string
	AvatarImage    *string
}

type MemeService interface {
	Submit(ctx context.Context, userID string, languageID int, templateID string, text1 string, text2 string, asset *multipart.FileHeader) (*entities.Meme, error)
	Feed(ctx context.Context, userID string, languageID *int, q *string, limit int, cursor *string, trending *bool) ([]MemeFeedResult, *string, error)
	Detail(ctx context.Context, userID string, memeID string) (*MemeFeedResult, error)
	Vote(ctx context.Context, userID string, assetID string) (*dtos.IncrementCoinCountResponseDTO, error)
	React(ctx context.Context, userID string, memeID string, reaction string) (*dtos.IncrementCoinCountResponseDTO, error)
	IncrementView(ctx context.Context, userID string, memeID string) (*dtos.MemeIncrementViewResponse, error)
	AdminListPending(ctx context.Context, limit int, cursor *string) (*dtos.AdminPendingMemesResponse, error)
	AdminApprove(ctx context.Context, memeID string, languageID *int, notes *string) (*dtos.ApproveMemeResponse, error)
	AdminReject(ctx context.Context, memeID string, reason string) (*dtos.RejectMemeResponse, error)
	GetUserMemes(ctx context.Context, userID string, status *int, limit int, cursor *string) ([]MemeFeedResult, *string, error)
	// Delete permanently removes one of the caller's own memes. Anyone else gets a 403.
	Delete(ctx context.Context, userID string, memeID string) error
}

type memeService struct {
	txnManager                *utils.TransactionManager
	memeRepository            repository.MemeRepository
	memeTextRepository        repository.MemeTextRepository
	reactionRepository        repository.MemeReactionRepository
	memeVoteHistoryRepository repository.MemeVoteHistoryRepository
	memeViewHistoryRepository repository.MemeViewHistoryRepository
	languageRepository        repository.LanguageRepository
	userRepository            repository.UserRepository
	userWalletService         ComicCoinService
	gcsService                utils.GCSService
	notificationService       NotificationService
}

func NewMemeService(txnManager *utils.TransactionManager, memeRepository repository.MemeRepository, memeTextRepository repository.MemeTextRepository, reactionRepository repository.MemeReactionRepository, memeVoteHistoryRepository repository.MemeVoteHistoryRepository, memeViewHistoryRepository repository.MemeViewHistoryRepository, languageRepository repository.LanguageRepository, userRepository repository.UserRepository, userWalletService ComicCoinService, gcsService utils.GCSService, notificationService NotificationService) MemeService {
	return &memeService{
		txnManager:                txnManager,
		memeRepository:            memeRepository,
		memeTextRepository:        memeTextRepository,
		reactionRepository:        reactionRepository,
		memeVoteHistoryRepository: memeVoteHistoryRepository,
		memeViewHistoryRepository: memeViewHistoryRepository,
		languageRepository:        languageRepository,
		userRepository:            userRepository,
		userWalletService:         userWalletService,
		gcsService:                gcsService,
		notificationService:       notificationService,
	}
}

func (s *memeService) Submit(ctx context.Context, userID string, languageID int, templateID string, text1 string, text2 string, asset *multipart.FileHeader) (*entities.Meme, error) {
	if userID == "" || templateID == "" || asset == nil {
		return nil, errs.NewBadRequestError("Missing required fields", nil)
	}

	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return nil, err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	language, err := s.languageRepository.FindByID(ctx, tx, languageID)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}
	if language == nil {
		s.txnManager.AbortTxn(tx)
		return nil, errs.NewBadRequestError("Invalid language_id", nil)
	}
	randomID := uuid.New().String()

	filePath := fmt.Sprintf("%s/%s/%s", constants.MEME_CONTENT_FOLDER, userID, randomID)
	_, gcsPath, err := s.gcsService.UploadFile(ctx, asset, filePath)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	meme := &entities.Meme{
		UserID:     userID,
		Status:     constants.UGC_STATUS_PENDING,
		LanguageID: languageID,
		TemplateID: templateID,
		AssetURL:   gcsPath,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if err := s.memeRepository.Create(ctx, tx, meme); err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	// Store text against meme using batch insert
	var t1 *string
	var t2 *string
	if text1 != "" {
		t1 = &text1
	}
	if text2 != "" {
		t2 = &text2
	}
	memeTexts := []entities.MemeText{{
		MemeID:    meme.ID,
		Text1:     t1,
		Text2:     t2,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}}
	if err := s.memeTextRepository.CreateMany(ctx, tx, memeTexts); err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	s.txnManager.CommitTxn(tx)

	_, err = s.userWalletService.IncrementCoinCountForMeme(ctx, userID, constants.COIN_COUNT_INCREMENT_VALUE, meme.ID)
	if err != nil {
		return nil, err
	}

	return meme, nil
}

func (s *memeService) Feed(ctx context.Context, userID string, languageID *int, q *string, limit int, cursor *string, trending *bool) ([]MemeFeedResult, *string, error) {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return nil, nil, err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	memes, next, err := s.memeRepository.FindApprovedFeed(ctx, tx, userID, languageID, q, limit, cursor, trending)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, nil, err
	}

	if len(memes) == 0 {
		s.txnManager.CommitTxn(tx)
		return []MemeFeedResult{}, next, nil
	}

	memeIDs := make([]string, len(memes))
	uniqueLanguageIDs := make(map[int]bool)
	uniqueUserIDs := make(map[string]bool)
	for i, meme := range memes {
		memeIDs[i] = meme.ID
		uniqueLanguageIDs[meme.LanguageID] = true
		uniqueUserIDs[meme.UserID] = true
	}

	languageIDsList := make([]int, 0, len(uniqueLanguageIDs))
	userIDsList := make([]string, 0, len(uniqueUserIDs))
	for langID := range uniqueLanguageIDs {
		languageIDsList = append(languageIDsList, langID)
	}
	for userID := range uniqueUserIDs {
		userIDsList = append(userIDsList, userID)
	}

	reactionCountsMap, err := s.reactionRepository.GetReactionCountsByMemeIDs(ctx, tx, memeIDs)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, nil, err
	}

	languageMap, err := s.languageRepository.FindLanguagesByIDs(ctx, tx, languageIDsList)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, nil, err
	}

	userMap, err := s.userRepository.FindUsersByIDs(ctx, tx, userIDsList)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, nil, err
	}

	votedMemesMap := make(map[string]bool)
	if userID != "" {
		votedMemesMap, err = s.memeVoteHistoryRepository.CheckUserVotedMemes(ctx, tx, userID, memeIDs)
		fmt.Println("votedMemesMap", votedMemesMap)
		if err != nil {
			s.txnManager.AbortTxn(tx)
			return nil, nil, err
		}
	}

	results := make([]MemeFeedResult, len(memes))
	for i, meme := range memes {
		reactionCount := dtos.ReactionCount{Happy: 0, Sad: 0, Neutral: 0}
		if rc, exists := reactionCountsMap[meme.ID]; exists {
			reactionCount = *rc
		}

		// Get user's reaction click data
		reactionClick, err := s.getUserReactionClick(ctx, tx, userID, meme.ID)
		if err != nil {
			s.txnManager.AbortTxn(tx)
			return nil, nil, err
		}

		// Convert asset path to GCS URL
		assetURL, err := s.getAssetURL(ctx, meme.AssetURL)
		if err != nil {
			s.txnManager.AbortTxn(tx)
			return nil, nil, err
		}

		languageCode := "en"
		if lang, exists := languageMap[meme.LanguageID]; exists && lang != nil {
			languageCode = lang.LanguageKey
		}

		var username *string
		var avatarImage *string
		if user, exists := userMap[meme.UserID]; exists && user != nil {
			username = utils.DisplayName(user.Name)
			avatarImage = user.ProfilePicture
		}

		hasVoted := votedMemesMap[meme.ID]

		results[i] = MemeFeedResult{
			Meme:           meme,
			AssetURL:       assetURL,
			LanguageCode:   languageCode,
			ReactionCounts: reactionCount,
			ReactionClick:  reactionClick,
			HasVoted:       hasVoted,
			Username:       username,
			AvatarImage:    avatarImage,
		}
	}

	s.txnManager.CommitTxn(tx)
	return results, next, nil
}

func (s *memeService) GetUserMemes(ctx context.Context, userID string, status *int, limit int, cursor *string) ([]MemeFeedResult, *string, error) {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return nil, nil, err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	memes, next, err := s.memeRepository.FindByUserID(ctx, tx, userID, status, limit, cursor)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, nil, err
	}

	if len(memes) == 0 {
		s.txnManager.CommitTxn(tx)
		return []MemeFeedResult{}, next, nil
	}

	memeIDs := make([]string, len(memes))
	uniqueLanguageIDs := make(map[int]bool)
	uniqueUserIDs := make(map[string]bool)
	for i, meme := range memes {
		memeIDs[i] = meme.ID
		uniqueLanguageIDs[meme.LanguageID] = true
		uniqueUserIDs[meme.UserID] = true
	}

	languageIDsList := make([]int, 0, len(uniqueLanguageIDs))
	userIDsList := make([]string, 0, len(uniqueUserIDs))
	for langID := range uniqueLanguageIDs {
		languageIDsList = append(languageIDsList, langID)
	}
	for userID := range uniqueUserIDs {
		userIDsList = append(userIDsList, userID)
	}

	reactionCountsMap, err := s.reactionRepository.GetReactionCountsByMemeIDs(ctx, tx, memeIDs)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, nil, err
	}

	languageMap, err := s.languageRepository.FindLanguagesByIDs(ctx, tx, languageIDsList)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, nil, err
	}

	userMap, err := s.userRepository.FindUsersByIDs(ctx, tx, userIDsList)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, nil, err
	}

	votedMemesMap := make(map[string]bool)
	if userID != "" {
		votedMemesMap, err = s.memeVoteHistoryRepository.CheckUserVotedMemes(ctx, tx, userID, memeIDs)
		if err != nil {
			s.txnManager.AbortTxn(tx)
			return nil, nil, err
		}
	}

	results := make([]MemeFeedResult, len(memes))
	for i, meme := range memes {
		reactionCount := dtos.ReactionCount{Happy: 0, Sad: 0, Neutral: 0}
		if rc, exists := reactionCountsMap[meme.ID]; exists {
			reactionCount = *rc
		}

		// Get user's reaction click data
		reactionClick, err := s.getUserReactionClick(ctx, tx, userID, meme.ID)
		if err != nil {
			s.txnManager.AbortTxn(tx)
			return nil, nil, err
		}

		// Convert asset path to GCS URL
		assetURL, err := s.getAssetURL(ctx, meme.AssetURL)
		if err != nil {
			s.txnManager.AbortTxn(tx)
			return nil, nil, err
		}

		languageCode := "en"
		if lang, exists := languageMap[meme.LanguageID]; exists && lang != nil {
			languageCode = lang.LanguageKey
		}

		var username *string
		var avatarImage *string
		if user, exists := userMap[meme.UserID]; exists && user != nil {
			username = utils.DisplayName(user.Name)
			avatarImage = user.ProfilePicture
		}

		hasVoted := votedMemesMap[meme.ID]

		results[i] = MemeFeedResult{
			Meme:           meme,
			AssetURL:       assetURL,
			LanguageCode:   languageCode,
			ReactionCounts: reactionCount,
			ReactionClick:  reactionClick,
			HasVoted:       hasVoted,
			Username:       username,
			AvatarImage:    avatarImage,
		}
	}

	s.txnManager.CommitTxn(tx)
	return results, next, nil
}

func (s *memeService) Detail(ctx context.Context, userID string, memeID string) (*MemeFeedResult, error) {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return nil, err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	meme, err := s.memeRepository.FindById(ctx, tx, memeID)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}
	if meme == nil {
		s.txnManager.AbortTxn(tx)
		return nil, errs.NewNotFoundError(MEME_NOT_FOUND_ERROR, nil)
	}

	reactionCountsMap, err := s.reactionRepository.GetReactionCountsByMemeIDs(ctx, tx, []string{meme.ID})
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	reactionCount := dtos.ReactionCount{Happy: 0, Sad: 0, Neutral: 0}
	if rc, exists := reactionCountsMap[meme.ID]; exists {
		reactionCount = *rc
	}

	languageMap, err := s.languageRepository.FindLanguagesByIDs(ctx, tx, []int{meme.LanguageID})
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	languageCode := "en"
	if lang, exists := languageMap[meme.LanguageID]; exists && lang != nil {
		languageCode = lang.LanguageKey
	}

	hasVoted := false
	if userID != "" {
		voteHistory, err := s.memeVoteHistoryRepository.GetMemeVoteHistoryByAsset(ctx, tx, userID, memeID)
		if err != nil {
			s.txnManager.AbortTxn(tx)
			return nil, err
		}
		hasVoted = voteHistory != nil
	}

	// Get user's reaction click data
	reactionClick, err := s.getUserReactionClick(ctx, tx, userID, memeID)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	// Convert asset path to GCS URL
	assetURL, err := s.getAssetURL(ctx, meme.AssetURL)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	result := &MemeFeedResult{
		Meme:           *meme,
		AssetURL:       assetURL,
		LanguageCode:   languageCode,
		ReactionCounts: reactionCount,
		ReactionClick:  reactionClick,
		HasVoted:       hasVoted,
	}

	s.txnManager.CommitTxn(tx)
	return result, nil
}

func (s *memeService) Vote(ctx context.Context, userID string, assetID string) (*dtos.IncrementCoinCountResponseDTO, error) {
	if assetID == "" {
		return nil, errs.NewBadRequestError("asset_id is required", nil)
	}
	wallet, err := s.userWalletService.IncrementCoinCountForMeme(ctx, userID, constants.COIN_COUNT_INCREMENT_VALUE, assetID)
	if err != nil {
		return nil, err
	}
	show := wallet != nil
	return &dtos.IncrementCoinCountResponseDTO{Message: "Vote recorded", ShowCoinAnimation: show}, nil
}

func (s *memeService) React(ctx context.Context, userID string, memeID string, reaction string) (*dtos.IncrementCoinCountResponseDTO, error) {
	if memeID == "" || reaction == "" {
		return nil, errs.NewBadRequestError("meme_id and reaction are required", nil)
	}
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return nil, err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	already, err := s.reactionRepository.HasReacted(ctx, tx, userID, memeID)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}
	if already {
		s.txnManager.CommitTxn(tx)
		return &dtos.IncrementCoinCountResponseDTO{Message: "Already recorded", ShowCoinAnimation: false}, nil
	}

	mr := &entities.MemeReaction{MemeID: &memeID, UserID: userID, Reaction: reaction, CreatedAt: time.Now()}
	if err := s.reactionRepository.Create(ctx, tx, mr); err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	dailyCount, err := s.reactionRepository.CountUserDailyReactions(ctx, tx, userID, time.Now())
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}
	showAnim := false
	if dailyCount < constants.SCROLL_AND_MAX_COIN_COUNT {
		_, _ = s.userWalletService.UpdateBalance(ctx, tx, userID, constants.COIN_COUNT_INCREMENT_VALUE)
		showAnim = true
	}

	s.txnManager.CommitTxn(tx)
	return &dtos.IncrementCoinCountResponseDTO{Message: "Reaction recorded", ShowCoinAnimation: showAnim}, nil
}

func (s *memeService) IncrementView(ctx context.Context, userID string, memeID string) (*dtos.MemeIncrementViewResponse, error) {
	if memeID == "" {
		return nil, errs.NewBadRequestError("meme_id is required", nil)
	}

	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return nil, err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	// Check if user has already viewed this meme
	hasViewed := false
	if userID != "" {
		hasViewed, err = s.memeViewHistoryRepository.HasViewed(ctx, tx, userID, memeID)
		if err != nil {
			s.txnManager.AbortTxn(tx)
			return nil, err
		}
	}

	// Only increment view count if user hasn't viewed it before
	if !hasViewed {
		err = s.memeRepository.IncrementViewCount(ctx, tx, memeID)
		if err != nil {
			s.txnManager.AbortTxn(tx)
			return nil, err
		}

		// Record the view for authenticated users
		if userID != "" {
			viewHistory := &entities.MemeViewHistory{
				UserID:   userID,
				MemeID:   memeID,
				ViewedAt: time.Now(),
			}
			if err := s.memeViewHistoryRepository.Create(ctx, tx, viewHistory); err != nil {
				s.txnManager.AbortTxn(tx)
				return nil, err
			}
		}
	}

	meme, err := s.memeRepository.FindById(ctx, tx, memeID)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}
	if meme == nil {
		s.txnManager.AbortTxn(tx)
		return nil, errs.NewNotFoundError(MEME_NOT_FOUND_ERROR, nil)
	}

	s.txnManager.CommitTxn(tx)
	return &dtos.MemeIncrementViewResponse{
		MemeID:    meme.ID,
		ViewCount: meme.ViewCount,
	}, nil
}

func (s *memeService) AdminListPending(ctx context.Context, limit int, cursor *string) (*dtos.AdminPendingMemesResponse, error) {
	if limit <= 0 {
		limit = constants.DEFAULT_LIMIT
	}

	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return nil, err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	memes, next, err := s.memeRepository.FindPending(ctx, tx, limit, cursor)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	// Fetch meme texts in batch
	memeIDs := make([]string, 0, len(memes))
	for _, m := range memes {
		memeIDs = append(memeIDs, m.ID)
	}
	texts, err := s.memeTextRepository.FindByMemeIDs(ctx, tx, memeIDs)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}
	textMap := make(map[string]entities.MemeText, len(texts))
	for _, t := range texts {
		textMap[t.MemeID] = t
	}

	items := make([]dtos.AdminPendingMemeItem, 0, len(memes))
	for _, m := range memes {
		mt := textMap[m.ID]
		var text1Val string
		var text2Val string
		if mt.Text1 != nil {
			text1Val = *mt.Text1
		}
		if mt.Text2 != nil {
			text2Val = *mt.Text2
		}
		items = append(items, dtos.AdminPendingMemeItem{
			ID:          m.ID,
			AssetURL:    m.AssetURL,
			LanguageID:  fmt.Sprintf("%d", m.LanguageID),
			Category:    m.TemplateID,
			Text1:       text1Val,
			Text2:       text2Val,
			SubmittedBy: m.UserID,
			SubmittedAt: m.CreatedAt.Format(time.RFC3339),
		})
	}

	s.txnManager.CommitTxn(tx)
	return &dtos.AdminPendingMemesResponse{Items: items, Cursor: next}, nil
}

func (s *memeService) AdminApprove(ctx context.Context, memeID string, languageID *int, notes *string) (*dtos.ApproveMemeResponse, error) {
	if memeID == "" {
		return nil, errs.NewBadRequestError("id is required", nil)
	}

	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return nil, err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	meme, err := s.memeRepository.FindById(ctx, tx, memeID)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}
	if meme == nil {
		s.txnManager.AbortTxn(tx)
		return nil, errs.NewNotFoundError(MEME_NOT_FOUND_ERROR, nil)
	}

	update := map[string]interface{}{
		"status":     constants.UGC_STATUS_APPROVED,
		"updated_at": time.Now(),
	}
	if languageID != nil {
		update["language_id"] = *languageID
	}
	if err := s.memeRepository.UpdateFields(ctx, tx, memeID, update); err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	// Award submission points and notify publisher
	_, _ = s.userWalletService.UpdateBalance(ctx, tx, meme.UserID, constants.REWARD_UNIQUE_CODE)
	_ = s.notificationService.SendPushNotification(ctx, tx, constants.NOTIFICATION_UGC_APPROVED, meme.UserID, nil)

	s.txnManager.CommitTxn(tx)
	return &dtos.ApproveMemeResponse{Status: "OK", Message: "Meme approved"}, nil
}

func (s *memeService) AdminReject(ctx context.Context, memeID string, reason string) (*dtos.RejectMemeResponse, error) {
	if memeID == "" || reason == "" {
		return nil, errs.NewBadRequestError("id and reason are required", nil)
	}

	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return nil, err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	meme, err := s.memeRepository.FindById(ctx, tx, memeID)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}
	if meme == nil {
		s.txnManager.AbortTxn(tx)
		return nil, errs.NewNotFoundError(MEME_NOT_FOUND_ERROR, nil)
	}

	if err := s.memeRepository.UpdateFields(ctx, tx, memeID, map[string]interface{}{
		"status":     constants.UGC_STATUS_REJECTED,
		"updated_at": time.Now(),
	}); err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	// Notify publisher of rejection
	_ = s.notificationService.SendPushNotification(ctx, tx, constants.NOTIFICATION_UGC_REMOVED, meme.UserID, nil)

	s.txnManager.CommitTxn(tx)
	return &dtos.RejectMemeResponse{Status: "OK"}, nil
}

// getUserReactionClick returns the user's reaction click data for a specific meme
func (s *memeService) getUserReactionClick(ctx context.Context, tx *gorm.DB, userID string, memeID string) (dtos.ReactionClick, error) {
	if userID == "" {
		return dtos.ReactionClick{Happy: 0, Sad: 0, Neutral: 0}, nil
	}

	reaction, err := s.reactionRepository.FindByUserAndMeme(ctx, tx, userID, memeID)
	if err != nil {
		return dtos.ReactionClick{Happy: 0, Sad: 0, Neutral: 0}, err
	}

	if reaction == nil {
		return dtos.ReactionClick{Happy: 0, Sad: 0, Neutral: 0}, nil
	}

	reactionClick := dtos.ReactionClick{Happy: 0, Sad: 0, Neutral: 0}
	switch reaction.Reaction {
	case "happy":
		reactionClick.Happy = 1
	case "sad":
		reactionClick.Sad = 1
	case "neutral":
		reactionClick.Neutral = 1
	}

	return reactionClick, nil
}

// getAssetURL converts the stored asset path to a GCS signed URL
func (s *memeService) Delete(ctx context.Context, userID string, memeID string) error {
	if userID == "" {
		return errs.NewUnauthorizedError("User not authenticated", nil)
	}
	// An id that is not a UUID cannot exist; answering 404 here also keeps it from reaching the database.
	if _, err := uuid.Parse(memeID); err != nil {
		return errs.NewNotFoundError(MEME_NOT_FOUND_ERROR, nil)
	}

	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	meme, err := s.memeRepository.FindById(ctx, tx, memeID)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return err
	}
	if meme == nil {
		s.txnManager.AbortTxn(tx)
		return errs.NewNotFoundError(MEME_NOT_FOUND_ERROR, nil)
	}
	if meme.UserID != userID {
		s.txnManager.AbortTxn(tx)
		return errs.NewForbiddenError("You can only delete your own memes", nil)
	}

	if err := s.memeRepository.DeleteWithDependents(ctx, tx, memeID); err != nil {
		s.txnManager.AbortTxn(tx)
		return err
	}
	s.txnManager.CommitTxn(tx)

	// The row is already gone. Removing the stored file is housekeeping, so a failure there is logged
	// and does not turn a successful delete into an error.
	if meme.AssetURL != "" && !strings.HasPrefix(meme.AssetURL, "http") {
		if err := s.gcsService.DeleteObject(ctx, meme.AssetURL); err != nil {
			logrus.Warnf("meme %s deleted but its file could not be removed: %v", utils.SanitizeForLog(memeID), err)
		}
	}
	return nil
}

func (s *memeService) getAssetURL(ctx context.Context, assetPath string) (string, error) {
	if strings.HasPrefix(assetPath, "http") {
		return assetPath, nil
	}

	return s.gcsService.GetFileSignedURL(ctx, assetPath, 24*time.Hour)
}
