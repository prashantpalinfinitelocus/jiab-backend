package services

import (
	"context"
	"fmt"
	"time"

	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	"github.com/Infinite-Locus-Product/coca_cola_backend/repository"
	"github.com/Infinite-Locus-Product/coca_cola_backend/utils"
)

type AvatarService interface {
	GetAvatar(ctx context.Context, id int) (*entities.Avatar, error)
	GetAvatars(ctx context.Context, ids []int) ([]entities.Avatar, error)
	GetAllAvatars(ctx context.Context) ([]entities.Avatar, error)
}

type avatarService struct {
	txnManager *utils.TransactionManager
	avatarRepo *repository.AvatarRepository
	gcsService utils.GCSService
}

func NewAvatarService(txnManager *utils.TransactionManager, avatarRepo *repository.AvatarRepository, gcsService utils.GCSService) AvatarService {
	return &avatarService{
		txnManager: txnManager,
		avatarRepo: avatarRepo,
		gcsService: gcsService,
	}
}

func (s *avatarService) GetAvatar(ctx context.Context, id int) (*entities.Avatar, error) {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return nil, err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	avatar, err := s.avatarRepo.FindByID(ctx, tx, id)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, fmt.Errorf("failed to get avatar: %v", err)
	}

	if avatar == nil || !avatar.IsActive || avatar.IsDeleted {
		s.txnManager.AbortTxn(tx)
		return nil, fmt.Errorf("avatar not found or inactive")
	}

	s.txnManager.CommitTxn(tx)
	return avatar, nil
}

func (s *avatarService) GetAvatars(ctx context.Context, ids []int) ([]entities.Avatar, error) {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return nil, err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	avatars, err := s.avatarRepo.FindByIDs(ctx, tx, ids)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, fmt.Errorf("failed to get avatars: %v", err)
	}

	// Filter out inactive or deleted avatars
	activeAvatars := make([]entities.Avatar, 0, len(avatars))
	for _, avatar := range avatars {
		if avatar.IsActive && !avatar.IsDeleted {
			activeAvatars = append(activeAvatars, avatar)
		}
	}

	s.txnManager.CommitTxn(tx)
	return activeAvatars, nil
}

func (s *avatarService) GetAllAvatars(ctx context.Context) ([]entities.Avatar, error) {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return nil, err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	avatars, err := s.avatarRepo.FindAllActive(ctx, tx)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, fmt.Errorf("failed to get avatars: %w", err)
	}

	// Generate signed URLs for each avatar
	for i := range avatars {
		signedURL, err := s.gcsService.GetFileSignedURL(ctx, avatars[i].ImageURL, 24*time.Hour)
		if err != nil {
			// Log the error but continue with the original URL
			fmt.Printf("Failed to generate signed URL for avatar %d: %v\n", avatars[i].ID, err)
		} else {
			avatars[i].ImageURL = signedURL
		}
	}

	s.txnManager.CommitTxn(tx)
	return avatars, nil
}
