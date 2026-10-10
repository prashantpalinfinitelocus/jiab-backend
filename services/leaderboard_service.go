package services

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Infinite-Locus-Product/coca_cola_backend/constants"
	"github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	"github.com/Infinite-Locus-Product/coca_cola_backend/repository"
	"github.com/Infinite-Locus-Product/coca_cola_backend/utils"
)

type LeaderboardService interface {
	GetLeaderboard(ctx *gin.Context, user *entities.User, date *string) (*dtos.LeaderboardResponseDTO, error)
}

type leaderboardService struct {
	txnManager     *utils.TransactionManager
	userRepo       repository.UserRepository
	userWinnerRepo repository.UserWinnerRepository
	avatarService  AvatarService
	userWalletRepo repository.UserWalletRepository
	gcsService     utils.GCSService
}

func NewLeaderboardService(txnManager *utils.TransactionManager, userRepo repository.UserRepository, userWinnerRepo repository.UserWinnerRepository, avatarService AvatarService, userWalletRepo repository.UserWalletRepository, gcsService utils.GCSService) LeaderboardService {
	return &leaderboardService{
		txnManager:     txnManager,
		userRepo:       userRepo,
		userWinnerRepo: userWinnerRepo,
		avatarService:  avatarService,
		userWalletRepo: userWalletRepo,
		gcsService:     gcsService,
	}
}

func (s *leaderboardService) GetLeaderboard(ctx *gin.Context, user *entities.User, date *string) (*dtos.LeaderboardResponseDTO, error) {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	var parsedDate *time.Time
	if date != nil && *date != "" {
		p, err := time.Parse("01/02/2006", *date)
		if err != nil {
			s.txnManager.AbortTxn(tx)
			return nil, fmt.Errorf("invalid date format: %w", err)
		}
		parsedDate = &p
	}

	// Get top users from repository
	topUsers, err := s.userWinnerRepo.GetTopUsers(ctx, tx, 10, parsedDate)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, fmt.Errorf("error fetching top users: %v", err)
	}

	// Convert to DTOs
	topUsersDTO := make([]dtos.LeaderboardEntryDTO, len(topUsers))
	for i, user := range topUsers {
		imagePath := constants.CONTEST_GCP_PREFIX + topUsers[i].WinnerImageURL
		signedURL, err := s.gcsService.GetFileSignedURL(ctx, imagePath, 60*time.Minute)
		if err != nil {
			signedURL = "" // fallback to empty string if error
		}
		entry := dtos.LeaderboardEntryDTO{
			UserID:   user.UserID,
			Mobile:   user.Mobile,
			Coins:    user.Coins,
			UserRank: user.UserRank,
			WinnerReward: dtos.WinnerRewardDTO{
				ImageURL:    signedURL,
				Description: user.WinnerDescription,
			},
		}

		// Get avatar for each top user
		if user.AvatarID != nil {
			avatar, err := s.avatarService.GetAvatar(ctx.Request.Context(), *user.AvatarID)
			if err == nil && avatar != nil {
				entry.Avatar, _ = s.gcsService.GetFileSignedURL(ctx, avatar.ImageURL, 60*time.Minute)
			}
		}

		topUsersDTO[i] = entry
	}

	// Fetch current user's rank
	myRank, err := s.userWinnerRepo.GetUserRank(ctx, tx, user.ID, parsedDate)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, fmt.Errorf("failed to get user rank: %w", err)
	}

	var myRankDTO dtos.LeaderboardEntryDTO

	if myRank != nil {
		imagePath := constants.CONTEST_GCP_PREFIX + myRank.WinnerImageURL
		signedURL, _ := s.gcsService.GetFileSignedURL(ctx, imagePath, 60*time.Minute)
		myRankDTO = dtos.LeaderboardEntryDTO{
			UserID:   user.ID,
			Mobile:   user.PhoneNumber,
			Coins:    myRank.Coins,
			UserRank: myRank.UserRank,
			WinnerReward: dtos.WinnerRewardDTO{
				ImageURL:    signedURL,
				Description: myRank.WinnerDescription,
			},
		}
		if myRank.AvatarID != nil {
			avatar, err := s.avatarService.GetAvatar(ctx.Request.Context(), *myRank.AvatarID)
			if err == nil && avatar != nil {
				myRankDTO.Avatar, _ = s.gcsService.GetFileSignedURL(ctx, avatar.ImageURL, 60*time.Minute)
			}
		}
	} else {
		var avatarURL string
		if user.AvatarID != nil {
			avatar, err := s.avatarService.GetAvatar(ctx.Request.Context(), *user.AvatarID)
			if err == nil && avatar != nil {
				avatarURL, _ = s.gcsService.GetFileSignedURL(ctx, avatar.ImageURL, 60*time.Minute)
			}
		}
		var coins int
		userWallet, _ := s.userWalletRepo.FindByField(ctx.Request.Context(), tx, "user_id", user.ID)
		if userWallet != nil {
			coins = userWallet.CurrentBalance
		} else {
			coins = 0
		}
		myRankDTO = dtos.LeaderboardEntryDTO{
			UserID:   user.ID,
			Mobile:   user.PhoneNumber,
			Avatar:   avatarURL,
			Coins:    coins,
			UserRank: 0,
			WinnerReward: dtos.WinnerRewardDTO{
				ImageURL:    "",
				Description: "",
			},
		}
	}

	response := dtos.LeaderboardResponseDTO{
		TopUsers: topUsersDTO,
		MyRank:   myRankDTO,
	}

	s.txnManager.CommitTxn(tx)

	return &response, nil
}
