package services

import (
	"context"
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

type ComicCoinService interface {
	GetUserWallet(ctx context.Context, tx *gorm.DB, userId string) (*entities.UserWallet, error)
	GetUserWalletWithoutTx(ctx context.Context, userId string) (*entities.UserWallet, error)
	UpdateBalance(ctx context.Context, tx *gorm.DB, userId string, balance int) (*entities.UserWallet, error)
	CreateUserWallet(ctx context.Context, tx *gorm.DB, userId string) (*entities.UserWallet, error)
	ResetUserWallet(ctx context.Context, tx *gorm.DB, userId string) (*entities.UserWallet, error)
	IncrementCoinCount(ctx context.Context, userId string, count int, req dtos.IncrementCoinCountRequestDTO) (*entities.UserWallet, error)
	IncrementCoinCountForMeme(ctx context.Context, userId string, count int, assetID string) (*entities.UserWallet, error)
	WACoinIncrement(ctx context.Context, userId string, count int, req dtos.AddUserReactPointRequest) (*entities.UserWallet, error)
}

type comicCoinService struct {
	txnManager                *utils.TransactionManager
	userWalletRepository      repository.UserWalletRepository
	notificationService       NotificationService
	voteHistoryRepository     repository.VoteHistoryRepository
	memeVoteHistoryRepository repository.MemeVoteHistoryRepository
	ugcMasterRepository       repository.UGCMasterRepository
	memeRepository            repository.MemeRepository
	languageRepository        repository.LanguageRepository
	genreRepository           repository.GenreRepository
	waAssetActivityRepository repository.WAAssetActivityRepository
}

func NewComicCoinService(txnManager *utils.TransactionManager, userWalletRepository repository.UserWalletRepository, notificationService NotificationService, voteHistoryRepository repository.VoteHistoryRepository, memeVoteHistoryRepository repository.MemeVoteHistoryRepository, ugcMasterRepository repository.UGCMasterRepository, memeRepository repository.MemeRepository, languageRepository repository.LanguageRepository, genreRepository repository.GenreRepository, waAssetActivityRepository repository.WAAssetActivityRepository) ComicCoinService {
	return &comicCoinService{
		txnManager:                txnManager,
		userWalletRepository:      userWalletRepository,
		notificationService:       notificationService,
		voteHistoryRepository:     voteHistoryRepository,
		memeVoteHistoryRepository: memeVoteHistoryRepository,
		ugcMasterRepository:       ugcMasterRepository,
		memeRepository:            memeRepository,
		languageRepository:        languageRepository,
		genreRepository:           genreRepository,
		waAssetActivityRepository: waAssetActivityRepository,
	}
}

func (s *comicCoinService) GetUserWallet(ctx context.Context, tx *gorm.DB, userId string) (*entities.UserWallet, error) {
	userWallet, err := s.userWalletRepository.FindByField(ctx, tx, "user_id", userId)
	if err != nil {
		return nil, err
	}
	return userWallet, nil
}

func (s *comicCoinService) GetUserWalletWithoutTx(ctx context.Context, userId string) (*entities.UserWallet, error) {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return nil, err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	userWallet, err := s.userWalletRepository.FindByField(ctx, tx, "user_id", userId)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}
	logrus.Infof("User wallet from DB: %+v", userWallet)

	s.txnManager.CommitTxn(tx)
	return userWallet, nil
}

func (s *comicCoinService) UpdateBalance(ctx context.Context, tx *gorm.DB, userId string, balance int) (*entities.UserWallet, error) {
	userWallet, err := s.userWalletRepository.FindByField(ctx, tx, "user_id", userId)
	if err != nil {
		return nil, err
	}
	if userWallet == nil {
		return nil, nil
	}

	newBalance := userWallet.CurrentBalance + balance
	totalCoinsEarned := userWallet.TotalCoinsEarned + balance

	err = s.userWalletRepository.Update(ctx, tx, userWallet.ID, &entities.UserWallet{
		CurrentBalance:   newBalance,
		WalletUpdatedOn:  time.Now(),
		TotalCoinsEarned: totalCoinsEarned,
	})
	if err != nil {
		return nil, err
	}

	return userWallet, nil
}

func (s *comicCoinService) CreateUserWallet(ctx context.Context, tx *gorm.DB, userId string) (*entities.UserWallet, error) {
	userWallet := &entities.UserWallet{
		UserID:           userId,
		CurrentBalance:   0,
		WalletUpdatedOn:  time.Now(),
		TotalCoinsEarned: 0,
		CreatedOn:        time.Now(),
	}

	err := s.userWalletRepository.Create(ctx, tx, userWallet)
	if err != nil {
		return nil, err
	}

	return userWallet, nil
}

func (s *comicCoinService) ResetUserWallet(ctx context.Context, tx *gorm.DB, userId string) (*entities.UserWallet, error) {
	userWallet, err := s.userWalletRepository.FindByField(ctx, tx, "user_id", userId)
	if err != nil {
		return nil, err
	}

	err = s.userWalletRepository.Update(ctx, tx, userWallet.ID, &entities.UserWallet{
		CurrentBalance:  0,
		WalletUpdatedOn: time.Now(),
	})
	if err != nil {
		return nil, err
	}

	return userWallet, nil
}

func (s *comicCoinService) IncrementCoinCount(ctx context.Context, userId string, count int, req dtos.IncrementCoinCountRequestDTO) (*entities.UserWallet, error) {
	addComicCoin := true
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return nil, err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	if req.Type != nil && *req.Type == "vote" {
		if req.AssetId == "" {
			s.txnManager.AbortTxn(tx)
			return nil, errs.NewBadRequestError("AssetId is required", nil)
		}
		var ugcMaster []entities.UGCMaster
		ugcMaster, err = s.ugcMasterRepository.GetUGCContentByIDs(ctx, tx, []string{req.AssetId}, false)
		if err != nil {
			s.txnManager.AbortTxn(tx)
			return nil, err
		}
		var languageKey string
		var genreValue string
		if len(ugcMaster) > 0 {
			language, err := s.languageRepository.FindByID(ctx, tx, ugcMaster[0].LanguageID)
			if err != nil {
				s.txnManager.AbortTxn(tx)
				return nil, errs.NewInternalServerError("Failed to fetch language", err)
			}
			if language == nil {
				s.txnManager.AbortTxn(tx)
				return nil, errs.NewBadRequestError("Language not found", nil)
			}
			languageKey = language.LanguageKey

			if ugcMaster[0].JokeGenreID != nil {
				genre, err := s.genreRepository.FindById(ctx, tx, *ugcMaster[0].JokeGenreID)
				if err != nil {
					s.txnManager.AbortTxn(tx)
					return nil, errs.NewInternalServerError("Failed to fetch genre", err)
				}
				if genre == nil {
					genreValue = constants.MISSING_GENRE
				} else {
					genreValue = genre.Genre
				}
			}
		} else {
			s.txnManager.AbortTxn(tx)
			return nil, errs.NewBadRequestError("Asset not found", nil)
		}
		voteHistory, err := s.voteHistoryRepository.GetVoteHistory(ctx, tx, userId, languageKey)
		if err != nil {
			s.txnManager.AbortTxn(tx)
			return nil, err
		}

		if voteHistory != nil {
			oneWeekAgo := time.Now().AddDate(0, 0, -7)
			if voteHistory.VotedAt.After(oneWeekAgo) {
				addComicCoin = false
			}
		}

		if addComicCoin {
			err = s.voteHistoryRepository.Create(ctx, tx, &entities.VoteHistory{
				UserID:   userId,
				Genre:    genreValue,
				Language: languageKey,
				VotedAt:  time.Now(),
			})
			if err != nil {
				s.txnManager.AbortTxn(tx)
				return nil, err
			}
		}
	}

	if !addComicCoin {
		s.txnManager.CommitTxn(tx)
		return nil, nil
	}

	userWallet, err := s.userWalletRepository.FindByField(ctx, tx, "user_id", userId)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	userWallet.CurrentBalance += count
	userWallet.TotalCoinsEarned += count
	userWallet.WalletUpdatedOn = time.Now()

	err = s.userWalletRepository.Update(ctx, tx, userWallet.ID, &entities.UserWallet{
		CurrentBalance:   userWallet.CurrentBalance,
		TotalCoinsEarned: userWallet.TotalCoinsEarned,
		WalletUpdatedOn:  userWallet.WalletUpdatedOn,
	})
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	err = s.notificationService.SendPushNotification(ctx, tx, constants.NOTIFICATION_WIN_COIN, userId, &count)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	s.txnManager.CommitTxn(tx)
	return userWallet, nil
}

func (s *comicCoinService) IncrementCoinCountForMeme(ctx context.Context, userId string, count int, assetID string) (*entities.UserWallet, error) {
	addComicCoin := true
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return nil, err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	if assetID == "" {
		s.txnManager.AbortTxn(tx)
		return nil, errs.NewBadRequestError("AssetId is required", nil)
	}

	meme, err := s.memeRepository.FindById(ctx, tx, assetID)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}
	if meme == nil {
		s.txnManager.AbortTxn(tx)
		return nil, errs.NewBadRequestError("Meme not found", nil)
	}

	language, err := s.languageRepository.FindByID(ctx, tx, meme.LanguageID)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, errs.NewInternalServerError("Failed to fetch language", err)
	}
	if language == nil {
		s.txnManager.AbortTxn(tx)
		return nil, errs.NewBadRequestError("Language not found", nil)
	}
	languageKey := language.LanguageKey

	voteHistory, err := s.memeVoteHistoryRepository.GetMemeVoteHistory(ctx, tx, userId, languageKey)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	if voteHistory != nil {
		oneWeekAgo := time.Now().AddDate(0, 0, -7)
		if voteHistory.VotedAt.After(oneWeekAgo) {
			addComicCoin = false
		}
	}

	err = s.memeVoteHistoryRepository.Create(ctx, tx, &entities.MemeVoteHistory{
		UserID:   userId,
		AssetID:  assetID,
		Language: languageKey,
		VotedAt:  time.Now(),
	})

	if addComicCoin {
		if err != nil {
			s.txnManager.AbortTxn(tx)
			return nil, err
		}
	}

	if !addComicCoin {
		s.txnManager.CommitTxn(tx)
		return nil, nil
	}

	userWallet, err := s.userWalletRepository.FindByField(ctx, tx, "user_id", userId)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	userWallet.CurrentBalance += count
	userWallet.TotalCoinsEarned += count
	userWallet.WalletUpdatedOn = time.Now()

	err = s.userWalletRepository.Update(ctx, tx, userWallet.ID, &entities.UserWallet{
		CurrentBalance:   userWallet.CurrentBalance,
		TotalCoinsEarned: userWallet.TotalCoinsEarned,
		WalletUpdatedOn:  userWallet.WalletUpdatedOn,
	})
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	err = s.notificationService.SendPushNotification(ctx, tx, constants.NOTIFICATION_WIN_COIN, userId, &count)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	s.txnManager.CommitTxn(tx)
	return userWallet, nil
}

func (s *comicCoinService) WACoinIncrement(ctx context.Context, userId string, count int, req dtos.AddUserReactPointRequest) (*entities.UserWallet, error) {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return nil, err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	userWallet, err := s.userWalletRepository.FindByField(ctx, tx, "user_id", userId)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	waAssetActivity := &entities.WAAssetActivity{
		AssetID:      req.AssetID,
		ActivityType: req.ActivityType,
		CreatedOn:    time.Now(),
	}

	err = s.waAssetActivityRepository.Create(ctx, tx, waAssetActivity)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	userWallet.CurrentBalance += count
	userWallet.TotalCoinsEarned += count
	userWallet.WalletUpdatedOn = time.Now()

	err = s.userWalletRepository.Update(ctx, tx, userWallet.ID, &entities.UserWallet{
		CurrentBalance:   userWallet.CurrentBalance,
		TotalCoinsEarned: userWallet.TotalCoinsEarned,
		WalletUpdatedOn:  userWallet.WalletUpdatedOn,
	})
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	err = s.notificationService.SendPushNotification(ctx, tx, constants.NOTIFICATION_WIN_COIN, userId, &count)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	s.txnManager.CommitTxn(tx)
	return userWallet, nil
}
