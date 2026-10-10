// services/dsr_export.go
package services

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"github.com/Infinite-Locus-Product/coca_cola_backend/repository"
)

// dsrExportUser is an explicit, export-safe projection of entities.User.
//
// The raw entity is deliberately NOT embedded in the export document:
// entities.User carries live session credentials (RefreshToken,
// RefreshTokenExpiryTime) which a plain SELECT * loads and which would then
// be serialized verbatim into the export JSON. That JSON is fetched over a
// 7-day signed URL handed to a third-party privacy portal, and the URL
// itself is persisted in dsr_requests.details - so anyone obtaining either
// could mint a live session as the exported user. A projection type also
// means a future credential-shaped column added to entities.User cannot
// leak into the export by accident; it has to be added here on purpose.
//
// Everything below is subject data the requester is entitled to receive.
// The account-security flags (email/phone confirmed, two-factor, lockout,
// access-failed count) are account state rather than secrets, so they are
// kept; no secret value is exported.
type dsrExportUser struct {
	ID                      string     `json:"id"`
	Name                    *string    `json:"name,omitempty"`
	Gender                  *int       `json:"gender,omitempty"`
	DOB                     *time.Time `json:"dob,omitempty"`
	AlternateMobile         *string    `json:"alternate_mobile,omitempty"`
	ProfilePicture          *string    `json:"profile_picture,omitempty"`
	IsActive                bool       `json:"is_active"`
	AvatarID                *int       `json:"avatar_id,omitempty"`
	ReferralCode            *string    `json:"referral_code,omitempty"`
	IsReferral              bool       `json:"is_referral"`
	ReferBy                 *string    `json:"refer_by,omitempty"`
	IsProfileComplete       bool       `json:"is_profile_complete"`
	IPAddress               *string    `json:"ip_address,omitempty"`
	PlatformMasterID        *int       `json:"platform_master_id,omitempty"`
	ProfilePercentage       int        `json:"profile_percentage"`
	LastLanguageSelection   int        `json:"last_language_selection"`
	RegisteredOn            time.Time  `json:"registered_on"`
	Email                   *string    `json:"email,omitempty"`
	PhoneNumber             string     `json:"phone_number,omitempty"`
	ContestOnboardingStatus bool       `json:"contest_onboarding_status"`
	IsTamil                 bool       `json:"is_tamil"`
	EmailConfirmed          bool       `json:"email_confirmed"`
	PhoneNumberConfirmed    bool       `json:"phone_number_confirmed"`
	TwoFactorEnabled        bool       `json:"two_factor_enabled"`
	LockoutEnabled          bool       `json:"lockout_enabled"`
	AccessFailedCount       int        `json:"access_failed_count"`
	HomeOnboardingStatus    bool       `json:"home_onboarding_status"`
	IsProfilePopUp          bool       `json:"is_profile_popup"`
}

func newDSRExportUser(u *entities.User) *dsrExportUser {
	if u == nil {
		return nil
	}
	return &dsrExportUser{
		ID:                      u.ID,
		Name:                    u.Name,
		Gender:                  u.Gender,
		DOB:                     u.DOB,
		AlternateMobile:         u.AlternateMobile,
		ProfilePicture:          u.ProfilePicture,
		IsActive:                u.IsActive,
		AvatarID:                u.AvatarID,
		ReferralCode:            u.ReferralCode,
		IsReferral:              u.IsReferral,
		ReferBy:                 u.ReferBy,
		IsProfileComplete:       u.IsProfileComplete,
		IPAddress:               u.IPAddress,
		PlatformMasterID:        u.PlatformMasterID,
		ProfilePercentage:       u.ProfilePercentage,
		LastLanguageSelection:   u.LastLanguageSelection,
		RegisteredOn:            u.RegisteredOn,
		Email:                   u.Email,
		PhoneNumber:             u.PhoneNumber,
		ContestOnboardingStatus: u.ContestOnboardingStatus,
		IsTamil:                 u.IsTamil,
		EmailConfirmed:          u.EmailConfirmed,
		PhoneNumberConfirmed:    u.PhoneNumberConfirmed,
		TwoFactorEnabled:        u.TwoFactorEnabled,
		LockoutEnabled:          u.LockoutEnabled,
		AccessFailedCount:       u.AccessFailedCount,
		HomeOnboardingStatus:    u.HomeOnboardingStatus,
		IsProfilePopUp:          u.IsProfilePopUp,
	}
}

// dsrExportDeviceToken is the export-safe projection of
// entities.FCMDeviceToken. It deliberately DROPS the device_token value:
// an FCM registration token is a push credential (whoever holds it can send
// notifications to that device), not subject data the requester needs, and
// the same 7-day-signed-URL exposure argument that applies to the refresh
// token applies to it. The registration's existence and lifecycle metadata
// IS subject data ("you had N devices registered, from these dates"), so
// that is kept.
type dsrExportDeviceToken struct {
	ID             int64      `json:"id"`
	UserID         string     `json:"user_id"`
	IsActive       bool       `json:"is_active"`
	IsDeleted      bool       `json:"is_deleted"`
	CreatedOn      time.Time  `json:"created_on"`
	LastModifiedOn *time.Time `json:"last_modified_on,omitempty"`
}

func newDSRExportDeviceTokens(tokens []entities.FCMDeviceToken) []dsrExportDeviceToken {
	out := make([]dsrExportDeviceToken, 0, len(tokens))
	for _, t := range tokens {
		out = append(out, dsrExportDeviceToken{
			ID:             t.ID,
			UserID:         t.UserID,
			IsActive:       t.IsActive,
			IsDeleted:      t.IsDeleted,
			CreatedOn:      t.CreatedOn,
			LastModifiedOn: t.LastModifiedOn,
		})
	}
	return out
}

type dsrExportDocument struct {
	User                   *dsrExportUser                       `json:"user"`
	Addresses              []entities.Address                   `json:"addresses"`
	UserVideos             []entities.UserVideo                 `json:"user_videos"`
	JokeConsumption        []entities.JokeConsumptionByUser     `json:"joke_consumption"`
	UGCContent             []entities.UGCMaster                 `json:"ugc_content"`
	Notifications          []entities.UserNotification          `json:"notifications"`
	DeviceTokens           []dsrExportDeviceToken               `json:"device_tokens"`
	PlagiarizedInfos       []entities.PlagiarizedInfo           `json:"plagiarized_infos"`
	ProfilingAnswers       []entities.ProfilingQuestionAnswer   `json:"profiling_answers"`
	QuestionAnswers        []entities.UserQuestionAnswer        `json:"question_answers"`
	MixCodeDetails         []entities.UserMixCodeDetails        `json:"mix_code_details"`
	Wallet                 []entities.UserWallet                `json:"wallet"`
	WinnerRecords          []entities.UserWinner                `json:"winner_records"`
	CreatorCollabReactions []entities.UserCreatorCollabReaction `json:"creator_collab_reactions"`
	Ranks                  []entities.UserRank                  `json:"ranks"`
	KOCID                  []entities.UserKOCID                 `json:"kocid"`
	VoteHistory            []entities.VoteHistory               `json:"vote_history"`
	Referrals              []entities.UserReferral              `json:"referrals"`
	Vouchers               []entities.VoucherInfo               `json:"vouchers"`
}

// handleAccessRequest owns the entire access/export flow, including its own
// transaction sequencing - unlike delete/update it is dispatched directly
// from HandleRequest rather than through the handler switch.
//
// The export document is returned inline in the API response (export_data)
// rather than uploaded to GCS and handed back as a signed URL - there is no
// file, and no PII-bearing object to leak, orphan, or expire.
//
// The ordering is the point of this function:
//
//  1. All 18 table reads run on tx, so the export is one internally
//     consistent snapshot.
//  2. tx is then COMMITTED (it wrote nothing, so there is nothing to roll
//     back).
//  3. The audit row is written on its own fresh transaction afterwards.
func (s *dsrService) handleAccessRequest(ctx context.Context, tx *gorm.DB, req dtos.DSRRequestDTO, user *entities.User) (*dtos.DSRResponseDTO, error) {
	doc, err := s.buildExportPayload(ctx, tx, user)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		s.recordAccessFailure(ctx, req, user, err)
		return nil, err
	}

	if commitErr := tx.Commit().Error; commitErr != nil {
		wrapped := errs.NewInternalServerError("failed to close dsr export read transaction", commitErr)
		s.recordAccessFailure(ctx, req, user, wrapped)
		return nil, wrapped
	}

	auditTx, err := s.txnManager.StartTxn()
	if err != nil {
		return nil, err
	}
	defer s.txnManager.RollbackOnPanic(auditTx)

	audit, err := s.recordAudit(ctx, auditTx, req, &user.ID, "completed", doc, nil)
	if err != nil {
		s.txnManager.AbortTxn(auditTx)
		s.recordAccessFailure(ctx, req, user, err)
		return nil, err
	}
	if commitErr := auditTx.Commit().Error; commitErr != nil {
		wrapped := errs.NewInternalServerError("failed to commit dsr export audit", commitErr)
		s.recordAccessFailure(ctx, req, user, wrapped)
		return nil, wrapped
	}

	return &dtos.DSRResponseDTO{
		RequestID:  audit.ID,
		Status:     "completed",
		ExportData: doc,
	}, nil
}

// recordAccessFailure writes the "failed" audit row for an access request on
// an independent transaction. Failing to write the audit is logged, never
// returned: the caller always surfaces the original error.
func (s *dsrService) recordAccessFailure(ctx context.Context, req dtos.DSRRequestDTO, user *entities.User, cause error) {
	if auditErr := s.recordFailureAudit(ctx, req, &user.ID, cause.Error(), nil); auditErr != nil {
		logrus.Errorf("failed to record dsr failure audit: %v", auditErr)
	}
}

// buildExportPayload runs every read that makes up the export and returns
// the assembled document. It performs no writes, so the caller is free to
// end tx as soon as it returns.
func (s *dsrService) buildExportPayload(ctx context.Context, tx *gorm.DB, user *entities.User) (*dsrExportDocument, error) {
	doc := dsrExportDocument{User: newDSRExportUser(user)}
	byUserID := map[string]interface{}{"user_id": user.ID}

	var err error
	if doc.Addresses, err = repository.NewGormRepository[entities.Address]().FindWithConditions(ctx, tx, byUserID); err != nil {
		return nil, fmt.Errorf("failed to query user_address: %w", err)
	}
	if doc.UserVideos, err = repository.NewGormRepository[entities.UserVideo]().FindWithConditions(ctx, tx, byUserID); err != nil {
		return nil, fmt.Errorf("failed to query user_video: %w", err)
	}
	if doc.JokeConsumption, err = repository.NewGormRepository[entities.JokeConsumptionByUser]().FindWithConditions(ctx, tx, byUserID); err != nil {
		return nil, fmt.Errorf("failed to query joke_consumption_by_user: %w", err)
	}
	if doc.UGCContent, err = repository.NewGormRepository[entities.UGCMaster]().FindWithConditions(ctx, tx, byUserID); err != nil {
		return nil, fmt.Errorf("failed to query ugc_master: %w", err)
	}
	if doc.Notifications, err = repository.NewGormRepository[entities.UserNotification]().FindWithConditions(ctx, tx, byUserID); err != nil {
		return nil, fmt.Errorf("failed to query user_notification: %w", err)
	}
	deviceTokens, err := repository.NewGormRepository[entities.FCMDeviceToken]().FindWithConditions(ctx, tx, byUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to query fcm_device_tokens: %w", err)
	}
	doc.DeviceTokens = newDSRExportDeviceTokens(deviceTokens)
	if doc.PlagiarizedInfos, err = repository.NewGormRepository[entities.PlagiarizedInfo]().FindWithConditions(ctx, tx, byUserID); err != nil {
		return nil, fmt.Errorf("failed to query plagiarized_infos: %w", err)
	}
	if doc.ProfilingAnswers, err = repository.NewGormRepository[entities.ProfilingQuestionAnswer]().FindWithConditions(ctx, tx, byUserID); err != nil {
		return nil, fmt.Errorf("failed to query profiling_question_answer: %w", err)
	}
	if doc.QuestionAnswers, err = repository.NewGormRepository[entities.UserQuestionAnswer]().FindWithConditions(ctx, tx, byUserID); err != nil {
		return nil, fmt.Errorf("failed to query user_question_answer: %w", err)
	}
	if doc.MixCodeDetails, err = repository.NewGormRepository[entities.UserMixCodeDetails]().FindWithConditions(ctx, tx, byUserID); err != nil {
		return nil, fmt.Errorf("failed to query user_mixcode_details: %w", err)
	}
	if doc.Wallet, err = repository.NewGormRepository[entities.UserWallet]().FindWithConditions(ctx, tx, byUserID); err != nil {
		return nil, fmt.Errorf("failed to query user_wallet: %w", err)
	}
	if doc.WinnerRecords, err = repository.NewGormRepository[entities.UserWinner]().FindWithConditions(ctx, tx, byUserID); err != nil {
		return nil, fmt.Errorf("failed to query user_winners: %w", err)
	}
	if doc.CreatorCollabReactions, err = repository.NewGormRepository[entities.UserCreatorCollabReaction]().FindWithConditions(ctx, tx, byUserID); err != nil {
		return nil, fmt.Errorf("failed to query ad_user_creator_collab_reaction: %w", err)
	}
	if doc.Ranks, err = repository.NewGormRepository[entities.UserRank]().FindWithConditions(ctx, tx, byUserID); err != nil {
		return nil, fmt.Errorf("failed to query user_ranks: %w", err)
	}
	if doc.KOCID, err = repository.NewGormRepository[entities.UserKOCID]().FindWithConditions(ctx, tx, byUserID); err != nil {
		return nil, fmt.Errorf("failed to query user_kocid: %w", err)
	}
	if doc.VoteHistory, err = repository.NewGormRepository[entities.VoteHistory]().FindWithConditions(ctx, tx, byUserID); err != nil {
		return nil, fmt.Errorf("failed to query user_vote_history: %w", err)
	}
	if doc.Referrals, err = repository.NewGormRepository[entities.UserReferral]().FindWithConditions(ctx, tx, byUserID); err != nil {
		return nil, fmt.Errorf("failed to query user_referral: %w", err)
	}
	if doc.Vouchers, err = repository.NewGormRepository[entities.VoucherInfo]().FindWithConditions(ctx, tx, map[string]interface{}{"assigned_to_user_id": user.ID}); err != nil {
		return nil, fmt.Errorf("failed to query vouchers_infos: %w", err)
	}

	return &doc, nil
}
