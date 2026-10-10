package main

import (
	"context"
	"testing"

	"github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
	"github.com/Infinite-Locus-Product/coca_cola_backend/repository"
	"github.com/Infinite-Locus-Product/coca_cola_backend/services"
	"github.com/Infinite-Locus-Product/coca_cola_backend/utils"
	"github.com/Infinite-Locus-Product/coca_cola_backend/vendors"
)

func TestSSOInitiateAndAuthenticateSmoke(t *testing.T) {
	db := vendors.InitDatabase()
	utils.Migrate(db)

	txnManager := utils.NewTransactionManager(db)
	otpLogsRepo := repository.NewOTPLogsRepository()
	refreshTokenRepo := repository.NewRefreshTokenRepository()
	userRepo := repository.NewUserRepository()
	referralRepo := repository.NewUserReferralRepository()
	ssoTokenRepo := repository.NewSSOTokenRepository()

	infobipService := services.NewInfobipService(vendors.InitInfobip())
	// SSOAuthenticate's new-user path only calls comicCoinService.CreateUserWallet,
	// which only touches userWalletRepository - safe to leave the rest of these deps nil for this smoke test.
	comicCoinService := services.NewComicCoinService(txnManager, repository.NewUserWalletRepository(), nil, nil, nil, nil, nil, nil, nil, repository.WAAssetActivityRepository{})

	authService := services.NewAuthService(txnManager, otpLogsRepo, refreshTokenRepo, userRepo, referralRepo, ssoTokenRepo, infobipService, comicCoinService, nil)

	ctx := context.Background()

	initResp, err := authService.SSOInitiate(ctx, dtos.SSOInitiateRequest{
		PhoneNumber: "9998887771",
		Name:        "SSO Smoke Test",
	})
	if err != nil {
		t.Fatalf("SSOInitiate failed: %v", err)
	}
	t.Logf("SSOInitiate response: %+v", initResp)
	if initResp.SSOToken == "" {
		t.Fatalf("expected non-empty sso_token")
	}

	authResp, err := authService.SSOAuthenticate(ctx, initResp.SSOToken)
	if err != nil {
		t.Fatalf("SSOAuthenticate failed: %v", err)
	}
	t.Logf("SSOAuthenticate response: %+v", authResp)
	if authResp.AccessToken == "" {
		t.Fatalf("expected non-empty access_token")
	}
	if !authResp.IsNewUser {
		t.Fatalf("expected IsNewUser=true for first-time SSO login")
	}

	// Replay should fail: token already used
	_, err = authService.SSOAuthenticate(ctx, initResp.SSOToken)
	if err == nil {
		t.Fatalf("expected error on SSO token replay, got nil")
	}
	t.Logf("Replay correctly rejected: %v", err)

	// Second login with same phone number should NOT create a new user
	initResp2, err := authService.SSOInitiate(ctx, dtos.SSOInitiateRequest{
		PhoneNumber: "9998887771",
		Name:        "SSO Smoke Test",
	})
	if err != nil {
		t.Fatalf("second SSOInitiate failed: %v", err)
	}
	authResp2, err := authService.SSOAuthenticate(ctx, initResp2.SSOToken)
	if err != nil {
		t.Fatalf("second SSOAuthenticate failed: %v", err)
	}
	if authResp2.IsNewUser {
		t.Fatalf("expected IsNewUser=false on second login for existing user")
	}
	if authResp2.UserID != authResp.UserID {
		t.Fatalf("expected same user_id across logins, got %s vs %s", authResp.UserID, authResp2.UserID)
	}
	t.Logf("Second login correctly reused existing user_id=%s", authResp2.UserID)
}
