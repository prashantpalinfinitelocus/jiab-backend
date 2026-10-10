package services

import (
	"context"
	"time"

	"github.com/Infinite-Locus-Product/coca_cola_backend/constants"
	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	"github.com/Infinite-Locus-Product/coca_cola_backend/repository"
	"github.com/Infinite-Locus-Product/coca_cola_backend/utils"
)

type ReactionService interface {
	AddReelReaction(ctx context.Context, userID string, reelID string, reaction string) error
}

type reactionService struct {
	txnManager                 *utils.TransactionManager
	userReelReactionRepository repository.UserReelReactionRepository
	comicCoinService           ComicCoinService
	notificationService        NotificationService
}

func NewReactionService(txnManager *utils.TransactionManager, userReelReactionRepository repository.UserReelReactionRepository, comicCoinService ComicCoinService, notificationService NotificationService) ReactionService {
	return &reactionService{
		txnManager:                 txnManager,
		userReelReactionRepository: userReelReactionRepository,
		comicCoinService:           comicCoinService,
		notificationService:        notificationService,
	}
}

func (s *reactionService) AddReelReaction(ctx context.Context, userID string, assetID string, reaction string) error {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	count, err := s.userReelReactionRepository.GetUserReactionCount(ctx, tx, userID)
	if err != nil {
		return err
	}

	if count < constants.SCROLL_AND_MAX_COIN_COUNT {
		if _, err := s.comicCoinService.UpdateBalance(ctx, tx, userID, 1); err != nil {
			s.txnManager.AbortTxn(tx)
			return err
		}
		coinCountIncrementValue := constants.COIN_COUNT_INCREMENT_VALUE
		if err := s.notificationService.SendPushNotification(ctx, tx, constants.NOTIFICATION_WIN_COIN, userID, &coinCountIncrementValue); err != nil {
			s.txnManager.AbortTxn(tx)
			return err
		}
	}

	userReelReaction := &entities.UserCreatorCollabReaction{
		UserID:       userID,
		JokeID:       assetID,
		Reaction:     reaction,
		ReactionDate: time.Now(),
		ReactionDttm: time.Now(),
	}

	err = s.userReelReactionRepository.Create(ctx, tx, userReelReaction)
	if err != nil {
		return err
	}

	s.txnManager.CommitTxn(tx)
	return nil
}
