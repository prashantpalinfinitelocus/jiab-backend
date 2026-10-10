package services

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Infinite-Locus-Product/coca_cola_backend/config"
	"github.com/Infinite-Locus-Product/coca_cola_backend/constants"
	"github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"github.com/Infinite-Locus-Product/coca_cola_backend/repository"
	"github.com/Infinite-Locus-Product/coca_cola_backend/utils"
)

type AuthService interface {
	SendOTP(ctx context.Context, mobileNumber string) error
	VerifyOTP(ctx context.Context, mobileNumber string, otp string) (*dtos.TokenResponse, error)
	SignUp(ctx context.Context, mobileNumber string, req dtos.SignUpRequest) (*dtos.SignUpResponse, error)
	RefreshToken(ctx context.Context, refreshToken string) (*dtos.TokenResponse, error)
	ValidateContactNo(ctx context.Context, mobileNumber string) (int, error)
	WARegister(ctx context.Context, req dtos.WARegisterRequest) (*dtos.WARegisterResponse, error)
	WAToken(ctx context.Context, mobileNumber string) (*dtos.WATokenResponse, error)
	SSOInitiate(ctx context.Context, req dtos.SSOInitiateRequest) (*dtos.SSOInitiateResponse, error)
	SSOAuthenticate(ctx context.Context, ssoToken string) (*dtos.SSOAuthenticateResponse, error)
}

type authService struct {
	txnManager          *utils.TransactionManager
	otpLogsRepo         repository.OtpLogsRepository
	refreshTokenRepo    repository.RefreshTokenRepository
	userRepo            repository.UserRepository
	referralRepo        repository.UserReferralRepository
	ssoTokenRepo        repository.SSOTokenRepository
	infobipService      InfobipService
	comicCoinService    ComicCoinService
	notificationService NotificationService
}

func NewAuthService(txnManager *utils.TransactionManager, otpLogsRepo repository.OtpLogsRepository, refreshTokenRepo repository.RefreshTokenRepository, userRepo repository.UserRepository, referralRepo repository.UserReferralRepository, ssoTokenRepo repository.SSOTokenRepository, infobipService InfobipService, comicCoinService ComicCoinService, notificationService NotificationService) AuthService {
	return &authService{txnManager: txnManager, otpLogsRepo: otpLogsRepo, refreshTokenRepo: refreshTokenRepo, userRepo: userRepo, referralRepo: referralRepo, ssoTokenRepo: ssoTokenRepo, infobipService: infobipService, comicCoinService: comicCoinService, notificationService: notificationService}
}

func (s *authService) SendOTP(ctx context.Context, mobileNumber string) error {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	recentAttempts, err := s.otpLogsRepo.CountRecentAttempts(ctx, tx, mobileNumber, time.Duration(constants.OTP_RATE_LIMIT_DURATION_MINUTES)*time.Minute)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return err
	}
	if recentAttempts >= int64(constants.OTP_MAX_ATTEMPTS) {
		s.txnManager.AbortTxn(tx)
		return errs.NewTooManyRequestsError(fmt.Sprintf("Too many attempts. Please try again after %d minutes.", constants.OTP_RATE_LIMIT_DURATION_MINUTES), nil)
	}

	otpNum, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return fmt.Errorf("failed to generate OTP: %w", err)
	}
	otp := fmt.Sprintf("%06d", otpNum.Int64())

	otpLog := &entities.OTPLogs{
		MobileNo:  mobileNumber,
		OTP:       otp,
		IsActive:  true,
		CreatedOn: time.Now(),
	}

	existingOTP, err := s.otpLogsRepo.FindOneWithConditions(ctx, tx, map[string]interface{}{
		"mobile_no":  mobileNumber,
		"is_active":  true,
		"is_deleted": false,
	})
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return err
	}

	if existingOTP != nil {
		if err := s.otpLogsRepo.UpdateByMobileNumber(ctx, tx, mobileNumber); err != nil {
			s.txnManager.AbortTxn(tx)
			return err
		}
	}
	if err := s.otpLogsRepo.Create(ctx, tx, otpLog); err != nil {
		s.txnManager.AbortTxn(tx)
		return err
	}

	s.txnManager.CommitTxn(tx)

	go func() {
		if err := s.infobipService.SendOTP(mobileNumber, otp); err != nil {
			logrus.Errorf("error sending OTP via Infobip for mobile %s: %v", utils.SanitizeForLog(mobileNumber), err)
		} else {
			logrus.Infof("OTP sent successfully via Infobip for mobile %s", utils.SanitizeForLog(mobileNumber))
		}
	}()

	return nil
}

func (s *authService) VerifyOTP(ctx context.Context, mobileNumber string, otp string) (*dtos.TokenResponse, error) {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return nil, err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	if err := s.otpLogsRepo.CheckVerificationRateLimit(ctx, tx, mobileNumber); err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	// Verify OTP
	isValid, err := s.otpLogsRepo.VerifyOTP(ctx, tx, mobileNumber, otp)
	if err != nil {
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			attemptsTx, err := s.txnManager.StartTxn()
			if err != nil {
				logrus.Errorf("Failed to start separate transaction for IncrementAttempts: %v", err)
				return
			}
			defer s.txnManager.RollbackOnPanic(attemptsTx)

			if err := s.otpLogsRepo.IncrementAttempts(ctx, attemptsTx, mobileNumber); err != nil {
				logrus.Errorf("Failed to increment attempts in separate transaction: %v", err)
				s.txnManager.AbortTxn(attemptsTx)
				return
			}

			s.txnManager.CommitTxn(attemptsTx)
			logrus.Infof("Successfully incremented attempts for mobile number: %s in separate transaction", utils.SanitizeForLog(mobileNumber))
		}()
		wg.Wait()
		s.txnManager.AbortTxn(tx)
		return nil, err
	}
	if !isValid {
		s.txnManager.AbortTxn(tx)
		return nil, errs.NewUnauthorizedError("Invalid OTP", nil)
	}

	if err := s.otpLogsRepo.UpdateByMobileNumber(ctx, tx, mobileNumber); err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	user, err := s.userRepo.FindByField(ctx, tx, "phone_number", mobileNumber)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	if user == nil {
		tempToken, err := utils.GenerateTempAccessToken(mobileNumber)
		if err != nil {
			s.txnManager.AbortTxn(tx)
			return nil, errs.NewInternalServerError("Failed to generate temp access token", err)
		}

		s.txnManager.CommitTxn(tx)

		return &dtos.TokenResponse{
			AccessToken: tempToken,
			ExpiresIn:   int64(5 * time.Minute),
			TokenType:   "temp",
		}, nil
	}

	// Generate JWT tokens
	tokenPair, err := utils.GenerateTokenPair(user.ID, user.PhoneNumber)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, errs.NewInternalServerError("Failed to generate tokens", err)
	}

	if err := s.refreshTokenRepo.Create(ctx, tx, &entities.UserRefreshToken{
		UserID:                 user.ID,
		RefreshToken:           &tokenPair.RefreshToken,
		RefreshTokenExpiryTime: &tokenPair.RefreshTokenExpiry,
		IsUsed:                 false,
		IsDeleted:              false,
		IsActive:               true,
		CreatedBy:              user.ID,
		CreatedOn:              time.Now(),
	}); err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	tokenResponse := &dtos.TokenResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.AccessTokenExpiry,
		TokenType:    tokenPair.TokenType,
		UserID:       user.ID,
		MobileNumber: user.PhoneNumber,
		Name:         derefString(user.Name),
		Email:        derefString(user.Email),
	}

	s.txnManager.CommitTxn(tx)

	return tokenResponse, nil
}

func (s *authService) SignUp(ctx context.Context, mobileNumber string, req dtos.SignUpRequest) (*dtos.SignUpResponse, error) {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return nil, err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	if mobileNumber != req.MobileNumber {
		s.txnManager.AbortTxn(tx)
		return nil, errs.NewUnauthorizedError("Invalid temp token", nil)
	}

	userByEmail, err := s.userRepo.FindByField(ctx, tx, "email", req.EmailID)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	if userByEmail != nil {
		s.txnManager.AbortTxn(tx)
		return nil, errs.NewConflictError("Email already in use", nil)
	}

	user, err := s.userRepo.FindByField(ctx, tx, "phone_number", mobileNumber)

	if user != nil {
		s.txnManager.AbortTxn(tx)
		return nil, errs.NewConflictError("user already exists", nil)
	}

	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	referralCodePresent := false
	var referral *entities.UserReferral
	var referrerUser *entities.User
	if req.ReferralCode != nil && *req.ReferralCode != "" {
		referral, err = s.referralRepo.ValidateReferralCode(ctx, tx, mobileNumber, *req.ReferralCode)
		if err != nil {
			s.txnManager.AbortTxn(tx)
			return nil, err
		}

		// Fetch the mobile number of the user who created the referral
		referrerUser, err = s.userRepo.FindByField(ctx, tx, "id", referral.CreatedBy)
		if err != nil {
			s.txnManager.AbortTxn(tx)
			return nil, err
		}
		if referrerUser == nil {
			s.txnManager.AbortTxn(tx)
			return nil, errs.NewInternalServerError("referrer user not found", fmt.Errorf("user with id %s not found", referral.CreatedBy))
		}

		if err := s.referralRepo.UpdateReferralStatus(ctx, tx, referral.ID, true); err != nil {
			s.txnManager.AbortTxn(tx)
			return nil, err
		}
		_, err = s.comicCoinService.UpdateBalance(ctx, tx, referral.UserID, constants.REWARD_REFERRER_ON_VERIFY)
		if err != nil {
			s.txnManager.AbortTxn(tx)
			return nil, err
		}

		rewardReferrerOnVerify := constants.REWARD_REFERRER_ON_VERIFY
		err = s.notificationService.SendPushNotification(ctx, tx, constants.NOTIFICATION_WIN_COIN, referral.UserID, &rewardReferrerOnVerify)
		if err != nil {
			s.txnManager.AbortTxn(tx)
			return nil, err
		}
		err = s.notificationService.SendPushNotification(ctx, tx, constants.NOTIFICATION_SIGNUP_REFERRAL, referral.UserID, nil)
		if err != nil {
			s.txnManager.AbortTxn(tx)
			return nil, err
		}
		referralCodePresent = true
	}

	profilePercentage := 0
	if req.FullName != "" {
		profilePercentage += 20
	}
	if req.MobileNumber != "" {
		profilePercentage += 25
	}

	user = &entities.User{
		PhoneNumber:           mobileNumber,
		Name:                  &req.FullName,
		Email:                 req.EmailID,
		RegisteredOn:          time.Now(),
		ReferralCode:          req.ReferralCode,
		IsReferral:            referralCodePresent,
		ProfilePercentage:     profilePercentage,
		LastLanguageSelection: 1,
	}
	if req.AvatarID != nil {
		user.AvatarID = req.AvatarID
	}
	if err := s.userRepo.Create(ctx, tx, user); err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	if req.DeviceToken != "" {
		err = s.notificationService.RegisterWithTxn(ctx, tx, user.ID, req.DeviceToken)
		if err != nil {
			s.txnManager.AbortTxn(tx)
			return nil, err
		}
	}

	_, err = s.comicCoinService.CreateUserWallet(ctx, tx, user.ID)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	if referralCodePresent {
		_, err = s.comicCoinService.UpdateBalance(ctx, tx, user.ID, constants.REWARD_REFEREE_ON_VERIFY)
		if err != nil {
			s.txnManager.AbortTxn(tx)
			return nil, err
		}

		rewardRefereeOnVerify := constants.REWARD_REFEREE_ON_VERIFY
		err = s.notificationService.SendPushNotification(ctx, tx, constants.NOTIFICATION_WIN_COIN, user.ID, &rewardRefereeOnVerify)
		if err != nil {
			s.txnManager.AbortTxn(tx)
			return nil, err
		}
	}

	tokenPair, err := utils.GenerateTokenPair(user.ID, mobileNumber)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, errs.NewInternalServerError("Failed to generate tokens", err)
	}

	if err = s.refreshTokenRepo.Create(ctx, tx, &entities.UserRefreshToken{
		UserID:                 user.ID,
		RefreshToken:           &tokenPair.RefreshToken,
		RefreshTokenExpiryTime: &tokenPair.RefreshTokenExpiry,
		IsUsed:                 false,
		IsActive:               true,
		IsDeleted:              false,
		CreatedBy:              user.ID,
		CreatedOn:              time.Now(),
	}); err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}
	referralID := ""
	referredBy := ""
	referralPhoneNumber := ""
	if referralCodePresent && referral != nil {
		referralID = fmt.Sprintf("%d", referral.ID)
		referredBy = referral.CreatedBy
		referralPhoneNumber = referrerUser.PhoneNumber
	}

	tokenResponse := &dtos.SignUpResponse{
		AccessToken:         tokenPair.AccessToken,
		RefreshToken:        tokenPair.RefreshToken,
		ExpiresIn:           tokenPair.AccessTokenExpiry,
		TokenType:           tokenPair.TokenType,
		UserID:              user.ID,
		Name:                derefString(user.Name),
		Email:               derefString(user.Email),
		PhoneNumber:         user.PhoneNumber,
		ReferralID:          referralID,
		ReferredBy:          referredBy,
		ReferralPhoneNumber: referralPhoneNumber,
	}

	err = s.notificationService.SendPushNotification(ctx, tx, constants.NOTIFICATION_SIGNUP, user.ID, nil)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	s.txnManager.CommitTxn(tx)

	return tokenResponse, nil
}

func (s *authService) RefreshToken(ctx context.Context, refreshToken string) (*dtos.TokenResponse, error) {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return nil, err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	userRefreshToken, err := s.refreshTokenRepo.FindOneWithConditions(ctx, tx, map[string]interface{}{
		"refresh_token": refreshToken,
		"is_active":     true,
		"is_deleted":    false,
	})
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	if userRefreshToken == nil {
		s.txnManager.AbortTxn(tx)
		return nil, errs.NewUnauthorizedError("Invalid refresh token", nil)
	}

	if userRefreshToken.RefreshTokenExpiryTime.Before(time.Now()) {
		s.txnManager.AbortTxn(tx)
		return nil, errs.NewUnauthorizedError("Refresh token expired", nil)
	}

	if err := s.refreshTokenRepo.Update(ctx, tx, userRefreshToken.ID, &entities.UserRefreshToken{
		IsUsed: true,
	}); err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	user, err := s.userRepo.FindByField(ctx, tx, "id", userRefreshToken.UserID)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	tokenPair, err := utils.RefreshAccessToken(refreshToken, userRefreshToken.UserID, user.PhoneNumber)

	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, errs.NewUnauthorizedError("Invalid refresh token", err)
	}

	if err := s.refreshTokenRepo.Create(ctx, tx, &entities.UserRefreshToken{
		UserID:                 userRefreshToken.UserID,
		RefreshToken:           &tokenPair.RefreshToken,
		RefreshTokenExpiryTime: &tokenPair.RefreshTokenExpiry,
		IsUsed:                 false,
		IsActive:               true,
	}); err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	tokenResponse := &dtos.TokenResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.AccessTokenExpiry,
		TokenType:    tokenPair.TokenType,
	}

	s.txnManager.CommitTxn(tx)

	return tokenResponse, nil
}

func (s *authService) ValidateContactNo(ctx context.Context, mobileNumber string) (int, error) {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return 0, errs.NewInternalServerError("Failed to start transaction", err)
	}
	defer s.txnManager.RollbackOnPanic(tx)

	user, err := s.userRepo.FindByField(ctx, tx, "phone_number", mobileNumber)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return 0, errs.NewInternalServerError("Failed to validate contact number", err)
	}

	s.txnManager.CommitTxn(tx)

	if user != nil {
		return 0, errs.NewAppError(1002, "Number already in use. Please enter a different one.", 202, nil)
	}

	return 1, nil
}

func (s *authService) WARegister(ctx context.Context, req dtos.WARegisterRequest) (*dtos.WARegisterResponse, error) {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	// Check if user already exists
	existingUser, err := s.userRepo.FindByField(ctx, tx, "phone_number", req.PhoneNumber)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}
	if existingUser != nil {
		s.txnManager.AbortTxn(tx)
		return nil, errs.NewConflictError("Number already in use. Please enter a different one.", nil)
	}

	// Handle referral code
	referralCodePresent := false
	if req.ReferralCode != "" {
		referral, err := s.referralRepo.ValidateReferralCode(ctx, tx, req.PhoneNumber, req.ReferralCode)
		if err != nil {
			s.txnManager.AbortTxn(tx)
			return nil, err
		}
		_, err = s.comicCoinService.UpdateBalance(ctx, tx, referral.UserID, constants.REWARD_REFERRER_ON_VERIFY)
		if err != nil {
			s.txnManager.AbortTxn(tx)
			return nil, err
		}
		referralCodePresent = true
	}

	if req.LanguageId == 0 {
		req.LanguageId = 1
	}

	// Create user
	user := &entities.User{
		PhoneNumber:           req.PhoneNumber,
		Name:                  &req.Name,
		Email:                 &req.Email,
		RegisteredOn:          time.Now(),
		ReferralCode:          nil,
		IsReferral:            referralCodePresent,
		ProfilePercentage:     45,
		LastLanguageSelection: req.LanguageId,
	}
	if req.ReferralCode != "" {
		user.ReferralCode = &req.ReferralCode
	}
	if err := s.userRepo.Create(ctx, tx, user); err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	// Create wallet
	_, err = s.comicCoinService.CreateUserWallet(ctx, tx, user.ID)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	if referralCodePresent {
		_, err = s.comicCoinService.UpdateBalance(ctx, tx, user.ID, constants.REWARD_REFEREE_ON_VERIFY)
		if err != nil {
			s.txnManager.AbortTxn(tx)
			return nil, err
		}
	}

	// Generate tokens
	tokenPair, err := utils.GenerateTokenPair(user.ID, req.PhoneNumber)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, errs.NewInternalServerError("Failed to generate tokens", err)
	}

	if err = s.refreshTokenRepo.Create(ctx, tx, &entities.UserRefreshToken{
		UserID:                 user.ID,
		RefreshToken:           &tokenPair.RefreshToken,
		RefreshTokenExpiryTime: &tokenPair.RefreshTokenExpiry,
		IsUsed:                 false,
		IsActive:               true,
	}); err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	// Get wallet points (assume 0 for new user)
	walletPoints := 0

	s.txnManager.CommitTxn(tx)

	waResp := &dtos.WARegisterResponse{
		ID:           user.ID,
		MobileNo:     user.PhoneNumber,
		Name:         derefString(user.Name),
		Email:        derefString(user.Email),
		Roles:        []string{"Basic"},
		IsVerified:   true,
		JwtToken:     tokenPair.AccessToken,
		IssuedOn:     time.Now().Format(time.RFC3339),
		ExpiresOn:    time.Now().Add(2 * time.Hour).Format(time.RFC3339),
		MainRole:     "Basic",
		WalletPoints: walletPoints,
		NewUser:      true,
		ReferralCode: derefString(user.ReferralCode),
		RefreshToken: tokenPair.RefreshToken,
	}
	return waResp, nil
}

func (s *authService) WAToken(ctx context.Context, mobileNumber string) (*dtos.WATokenResponse, error) {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return nil, err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	user, err := s.userRepo.FindByField(ctx, tx, "phone_number", mobileNumber)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}
	if user == nil {
		s.txnManager.AbortTxn(tx)
		return nil, errs.NewNotFoundError("No Accounts Registered with this number", nil)
	}

	tokenPair, err := utils.GenerateTokenPair(user.ID, user.PhoneNumber)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, errs.NewInternalServerError("Failed to generate tokens", err)
	}

	if err := s.refreshTokenRepo.Create(ctx, tx, &entities.UserRefreshToken{
		UserID:                 user.ID,
		RefreshToken:           &tokenPair.RefreshToken,
		RefreshTokenExpiryTime: &tokenPair.RefreshTokenExpiry,
		IsUsed:                 false,
		IsActive:               true,
		IsDeleted:              false,
		CreatedBy:              user.ID,
		CreatedOn:              time.Now(),
	}); err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	tokenResponse := &dtos.WATokenResponse{
		ID:           user.ID,
		MobileNumber: user.PhoneNumber,
		Name:         derefString(user.Name),
		Email:        derefString(user.Email),
		Roles:        []string{"Basic"}, // This is hardcoded for now
		IsVerified:   user.IsActive,
		JwtToken:     tokenPair.AccessToken,
		IssuedOn:     time.Now(),
		ExpiresOn:    time.Now().Add(time.Duration(tokenPair.AccessTokenExpiry) * time.Second),
		MainRole:     "Basic", // This is hardcoded for now
		WalletPoints: 0,       // This needs to be fetched
		NewUser:      false,   // This needs to be determined
		RefreshToken: tokenPair.RefreshToken,
	}

	s.txnManager.CommitTxn(tx)

	return tokenResponse, nil
}

// SSOInitiate creates an intermediary SSO token for a partner-initiated login
func (s *authService) SSOInitiate(ctx context.Context, req dtos.SSOInitiateRequest) (*dtos.SSOInitiateResponse, error) {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return nil, err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	token, err := utils.GenerateSecureToken(constants.SSO_TOKEN_LENGTH)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, errs.NewInternalServerError("Failed to generate SSO token", err)
	}

	expiresAt := time.Now().Add(constants.SSO_TOKEN_EXPIRY_MINUTES * time.Minute)

	ssoToken := &entities.SSOToken{
		PhoneNumber: req.PhoneNumber,
		Name:        req.Name,
		Token:       token,
		IsUsed:      false,
		ExpiresAt:   expiresAt,
		CreatedBy:   constants.SSO_CREATED_BY,
	}

	if err := s.ssoTokenRepo.Create(ctx, tx, ssoToken); err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, errs.NewInternalServerError("Failed to create SSO token", err)
	}

	cfg := config.GetConfig()
	frontendBaseURL := cfg.FrontendBaseURL
	if frontendBaseURL == "" {
		logrus.Warn("FRONTEND_BASE_URL is not configured; falling back to default production URL for SSO redirect")
		frontendBaseURL = constants.DEFAULT_FRONTEND_BASE_URL
	}
	redirectURL := frontendBaseURL + "?sso_token=" + token

	s.txnManager.CommitTxn(tx)

	logrus.Infof("SSO token generated for phone number %s, expires at %v", utils.SanitizeForLog(req.PhoneNumber), expiresAt)

	return &dtos.SSOInitiateResponse{
		SSOToken:    token,
		RedirectURL: redirectURL,
		ExpiresIn:   int64(constants.SSO_TOKEN_EXPIRY_MINUTES * 60),
		ExpiresAt:   expiresAt,
	}, nil
}

// SSOAuthenticate validates an SSO token and exchanges it for JWT access/refresh tokens
func (s *authService) SSOAuthenticate(ctx context.Context, ssoToken string) (*dtos.SSOAuthenticateResponse, error) {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return nil, err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	token, err := s.ssoTokenRepo.FindByToken(ctx, tx, ssoToken)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, errs.NewInternalServerError("Failed to validate SSO token", err)
	}
	if token == nil {
		s.txnManager.AbortTxn(tx)
		return nil, errs.NewUnauthorizedError("Invalid SSO token", nil)
	}

	if !token.IsValid() {
		s.txnManager.AbortTxn(tx)
		if token.IsUsed {
			return nil, errs.NewUnauthorizedError("SSO token has already been used", nil)
		}
		return nil, errs.NewUnauthorizedError("SSO token has expired", nil)
	}

	// Mark token as used immediately to prevent replay attacks
	if err := s.ssoTokenRepo.MarkAsUsed(ctx, tx, ssoToken); err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, errs.NewInternalServerError("Failed to mark SSO token as used", err)
	}

	isNewUser := false
	user, err := s.userRepo.FindByField(ctx, tx, "phone_number", token.PhoneNumber)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	if user == nil {
		isNewUser = true
		user = &entities.User{
			PhoneNumber:           token.PhoneNumber,
			Name:                  &token.Name,
			RegisteredOn:          time.Now(),
			IsReferral:            false,
			ProfilePercentage:     45,
			LastLanguageSelection: 1,
			PhoneNumberConfirmed:  true,
		}
		if err := s.userRepo.Create(ctx, tx, user); err != nil {
			s.txnManager.AbortTxn(tx)
			return nil, err
		}

		if _, err := s.comicCoinService.CreateUserWallet(ctx, tx, user.ID); err != nil {
			s.txnManager.AbortTxn(tx)
			return nil, err
		}

		logrus.Infof("New user created via SSO: user_id=%s phone_number=%s", user.ID, user.PhoneNumber)
	}

	tokenPair, err := utils.GenerateTokenPair(user.ID, user.PhoneNumber)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, errs.NewInternalServerError("Failed to generate tokens", err)
	}

	if err := s.refreshTokenRepo.Create(ctx, tx, &entities.UserRefreshToken{
		UserID:                 user.ID,
		RefreshToken:           &tokenPair.RefreshToken,
		RefreshTokenExpiryTime: &tokenPair.RefreshTokenExpiry,
		IsUsed:                 false,
		IsActive:               true,
		IsDeleted:              false,
		CreatedBy:              user.ID,
		CreatedOn:              time.Now(),
	}); err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	response := &dtos.SSOAuthenticateResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.AccessTokenExpiry,
		TokenType:    tokenPair.TokenType,
		UserID:       user.ID,
		PhoneNumber:  user.PhoneNumber,
		Name:         derefString(user.Name),
		IsNewUser:    isNewUser,
	}

	s.txnManager.CommitTxn(tx)

	logrus.Infof("SSO authentication successful: user_id=%s is_new_user=%v", user.ID, isNewUser)

	return response, nil
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
