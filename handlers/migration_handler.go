package handlers

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"github.com/Infinite-Locus-Product/coca_cola_backend/repository"
	"github.com/Infinite-Locus-Product/coca_cola_backend/utils"
)

// Helper function to double URL encode strings to bypass WAF
func doubleURLEncode(s string) string {
	// First encoding
	encoded := url.QueryEscape(s)
	// Second encoding
	return url.QueryEscape(encoded)
}

// Helper function to double URL decode strings
func doubleURLDecode(s string) (string, error) {
	// First decoding
	decoded, err := url.QueryUnescape(s)
	if err != nil {
		return "", err
	}
	// Second decoding
	return url.QueryUnescape(decoded)
}

// Helper function to recursively encode map values
func encodeMapValues(data map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for key, value := range data {
		switch v := value.(type) {
		case string:
			result[key] = doubleURLEncode(v)
		case map[string]interface{}:
			result[key] = encodeMapValues(v)
		case []interface{}:
			result[key] = encodeSliceValues(v)
		default:
			result[key] = v
		}
	}
	return result
}

// Helper function to recursively encode slice values
func encodeSliceValues(data []interface{}) []interface{} {
	result := make([]interface{}, len(data))
	for i, value := range data {
		switch v := value.(type) {
		case string:
			result[i] = doubleURLEncode(v)
		case map[string]interface{}:
			result[i] = encodeMapValues(v)
		case []interface{}:
			result[i] = encodeSliceValues(v)
		default:
			result[i] = v
		}
	}
	return result
}

// Helper function to recursively decode map values
func decodeMapValues(data map[string]interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	for key, value := range data {
		switch v := value.(type) {
		case string:
			decoded, err := doubleURLDecode(v)
			if err != nil {
				return nil, err
			}
			result[key] = decoded
		case map[string]interface{}:
			decoded, err := decodeMapValues(v)
			if err != nil {
				return nil, err
			}
			result[key] = decoded
		case []interface{}:
			decoded, err := decodeSliceValues(v)
			if err != nil {
				return nil, err
			}
			result[key] = decoded
		default:
			result[key] = v
		}
	}
	return result, nil
}

// Helper function to recursively decode slice values
func decodeSliceValues(data []interface{}) ([]interface{}, error) {
	result := make([]interface{}, len(data))
	for i, value := range data {
		switch v := value.(type) {
		case string:
			decoded, err := doubleURLDecode(v)
			if err != nil {
				return nil, err
			}
			result[i] = decoded
		case map[string]interface{}:
			decoded, err := decodeMapValues(v)
			if err != nil {
				return nil, err
			}
			result[i] = decoded
		case []interface{}:
			decoded, err := decodeSliceValues(v)
			if err != nil {
				return nil, err
			}
			result[i] = decoded
		default:
			result[i] = v
		}
	}
	return result, nil
}

// Alternative: Base64 + Gzip encoding for extreme cases
func encodeDataWithGzip(data interface{}) (string, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(jsonData); err != nil {
		return "", err
	}
	if err := gz.Close(); err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

func decodeDataWithGzip(encoded string) (interface{}, error) {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer(decoded)
	gz, err := gzip.NewReader(buf)
	if err != nil {
		return nil, err
	}
	defer gz.Close()

	jsonData, err := io.ReadAll(gz)
	if err != nil {
		return nil, err
	}

	// nosemgrep: go.lang.security.deserialization.unsafe-deserialization-interface.go-unsafe-deserialization-interface -- migration payload is an arbitrary JSON document relayed as-is; no typed schema exists and no type-driven behaviour is selected from it
	var result interface{}
	if err := json.Unmarshal(jsonData, &result); err != nil {
		return nil, err
	}

	return result, nil
}

type MigrationHandler struct {
	userRepo repository.UserRepository
	db       *gorm.DB
}

func NewMigrationHandler(userRepo repository.UserRepository, db *gorm.DB) *MigrationHandler {
	return &MigrationHandler{userRepo: userRepo, db: db}
}

// BulkImportUsers handles POST /main/users/bulk_import
func (h *MigrationHandler) BulkImportUsers(c *gin.Context) {
	var req dtos.BulkImportUsersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(errs.NewBadRequestError("Invalid request body", err))
		return
	}

	var users []entities.User
	for _, u := range req.Users {
		user := entities.User{
			ID:                      u.ID,
			Name:                    u.Name,
			Gender:                  u.Gender,
			DOB:                     u.DOB,
			AlternateMobile:         u.AlternateMobile,
			ProfilePicture:          u.ProfilePicture,
			IsActive:                u.IsActive,
			RefreshToken:            u.RefreshToken,
			RefreshTokenExpiryTime:  u.RefreshTokenExpiryTime,
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
			ContestOnboardingStatus: true,
			HomeOnboardingStatus:    true,
			IsTamil:                 u.IsTamil,
			EmailConfirmed:          u.EmailConfirmed,
			PhoneNumberConfirmed:    u.PhoneNumberConfirmed,
			TwoFactorEnabled:        u.TwoFactorEnabled,
			LockoutEnabled:          u.LockoutEnabled,
			AccessFailedCount:       u.AccessFailedCount,
			IsProfilePopUp:          u.IsProfilePopUp,
		}
		users = append(users, user)
	}

	err := h.userRepo.CreateMany(context.Background(), h.db, users)
	if err != nil {
		logrus.Errorf("Bulk insert failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"inserted": 0, "errors": []string{err.Error()}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"inserted": len(users), "errors": []string{}})
}

// GetMissingUserIDs handles POST /main/users/missing_ids
func (h *MigrationHandler) GetMissingUserIDs(c *gin.Context) {
	var req dtos.MissingIDsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(errs.NewBadRequestError("Invalid request body", err))
		return
	}
	if len(req.IDs) == 0 {
		c.JSON(http.StatusOK, gin.H{"missing_ids": []string{}})
		return
	}
	var present []entities.User
	err := h.db.Where("id IN ?", req.IDs).Find(&present).Error
	if err != nil {
		c.Error(errs.NewInternalServerError("Failed to query users", err))
		return
	}
	presentIDs := make(map[string]struct{})
	for _, u := range present {
		presentIDs[u.ID] = struct{}{}
	}
	var missing []string
	for _, id := range req.IDs {
		if _, ok := presentIDs[id]; !ok {
			missing = append(missing, id)
		}
	}
	c.JSON(http.StatusOK, gin.H{"missing_ids": missing})
}

// TableModelMap - Cleaned up mapping for the 35 tables being migrated
// Maps MSSQL table names to PostgreSQL table names
var TableModelMap = map[string]string{
	"UserAddress":                  "user_address",
	"Avatar":                       "avatar",
	"JokeGenres":                   "joke_genres",
	"UserVideo":                    "user_video",
	"JokeConsumptionByUser":        "joke_consumption_by_user",
	"JokeDetailsReceived":          "joke_details_received",
	"UGCMaster":                    "ugc_master",
	"UGCStatus_lookup":             "ugc_status_lookup",
	"LanguageKey":                  "language_key",
	"LanguageKeyValues":            "language_key_values",
	"LanguageMaster":               "language_master",
	"Languages":                    "languages",
	"States":                       "states",
	"Cities":                       "cities",
	"PinCode":                      "pin_code",
	"NotificationTypes":            "notification_types",
	"UserNotification":             "user_notification",
	"OptionMaster":                 "option_master",
	"OptionMasterLanguages":        "option_master_languages",
	"OTPLogs":                      "otp_logs",
	"PlagiarizedInfos":             "plagiarized_infos",
	"QuestionMaster":               "question_master",
	"QuestionMasterLanguages":      "question_master_languages",
	"Users":                        "users",
	"UserRefreshTokens":            "user_refresh_tokens",
	"UserMixCodeDetails":           "user_mixcode_details",
	"UserQuestionAnswer":           "user_question_answer",
	"UserRanks":                    "user_ranks",
	"AD_UserCreatorCollabReaction": "ad_user_creator_collab_reaction",
	"UserReferral":                 "user_referral",
	"UserWallet":                   "user_wallet",
	"UserWinners":                  "user_winners",
	"VouchersInfos":                "vouchers_infos",
	"WinnerRewards":                "winner_rewards",
	"WinnerRewardLanguages":        "winner_reward_languages",
}

// GenericBulkImport handles POST /main/generic_bulk_import
func (h *MigrationHandler) GenericBulkImport(c *gin.Context) {
	var req dtos.GenericBulkImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(errs.NewBadRequestError("Invalid request body", err))
		return
	}

	if req.Table == "" || len(req.Data) == 0 {
		c.JSON(http.StatusOK, gin.H{"inserted": 0, "errors": []string{}})
		return
	}

	// Map MSSQL table name to PostgreSQL table name
	safeTable := utils.SanitizeForLog(req.Table)
	actualTable, ok := TableModelMap[req.Table]
	if !ok || actualTable == "" {
		logrus.Errorf("Table not allowed for import: %s", safeTable)
		c.Error(errs.NewBadRequestError("Table not allowed for import: "+req.Table, nil))
		return
	}

	logrus.Infof("Starting bulk import for table: %s -> %s, Records: %d", safeTable, actualTable, len(req.Data))

	// Decode double URL encoded data to handle Unicode characters bypassing WAF
	decodedData := make([]interface{}, len(req.Data))
	for i, record := range req.Data {
		if recordMap, ok := record.(map[string]interface{}); ok {
			decoded, err := decodeMapValues(recordMap)
			if err != nil {
				logrus.Errorf("Failed to decode data for table %s at index %d: %v", safeTable, i, err)
				c.JSON(http.StatusBadRequest, gin.H{"inserted": 0, "errors": []string{fmt.Sprintf("Failed to decode data at index %d: %v", i, err)}})
		return
	}
			decodedData[i] = decoded
		} else {
			decodedData[i] = record
		}
	}

	// Validate and convert data based on table type
	validatedData, err := h.validateAndConvertData(req.Table, decodedData)
	if err != nil {
		logrus.Errorf("Data validation failed for table %s: %v", safeTable, err)
		c.JSON(http.StatusBadRequest, gin.H{"inserted": 0, "errors": []string{fmt.Sprintf("Data validation failed: %v", err)}})
		return
	}

	if len(validatedData) == 0 {
		logrus.Warnf("No data to insert for table %s", safeTable)
		c.JSON(http.StatusOK, gin.H{"inserted": 0, "errors": []string{}})
		return
	}

	// Calculate optimal batch size for this table
	batchSize := h.calculateOptimalBatchSize(req.Table)
	logrus.Infof("About to insert %d records into table %s using batch size: %d", len(validatedData), actualTable, batchSize)

	// Start database transaction
	tx := h.db.Begin()
	if tx.Error != nil {
		logrus.Errorf("Failed to start transaction: %v", tx.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"inserted": 0, "errors": []string{"Failed to start transaction"}})
		return
	}

	// Process data in batches to avoid PostgreSQL parameter limit
	totalInserted, err := h.processBatchInsert(tx, actualTable, validatedData, batchSize)
	if err != nil {
		tx.Rollback()
		logrus.Errorf("Batch insert failed for table %s: %v", actualTable, err)
		c.JSON(http.StatusInternalServerError, gin.H{"inserted": 0, "errors": []string{err.Error()}})
		return
	}

	logrus.Infof("Batch insert completed for table: %s, total rows inserted: %d", actualTable, totalInserted)

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		logrus.Errorf("Failed to commit transaction: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"inserted": 0, "errors": []string{"Failed to commit transaction"}})
		return
	}

	// Verify actual insertion count using primary key
	actualInserted, err := h.verifyInsertionCount(actualTable, validatedData)
	if err != nil {
		logrus.Errorf("Failed to verify insertion count: %v", err)
		// Don't fail the request, just log the error and return the batch processing count
		actualInserted = totalInserted
	}

	logrus.Infof("Bulk import completed for table: %s, Expected: %d, Actual: %d", actualTable, len(validatedData), actualInserted)
	c.JSON(http.StatusOK, gin.H{"inserted": actualInserted, "errors": []string{}})
}

// GenericBulkImportEncoded handles POST /main/generic_bulk_import_encoded
// For extreme cases where double URL encoding doesn't work
func (h *MigrationHandler) GenericBulkImportEncoded(c *gin.Context) {
	var req struct {
		Table string `json:"table"`
		Data  string `json:"data"` // Base64 + Gzip encoded data
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(errs.NewBadRequestError("Invalid request body", err))
		return
	}

	if req.Table == "" || req.Data == "" {
		c.JSON(http.StatusOK, gin.H{"inserted": 0, "errors": []string{}})
		return
	}

	// Map MSSQL table name to PostgreSQL table name
	safeTable := utils.SanitizeForLog(req.Table)
	actualTable, ok := TableModelMap[req.Table]
	if !ok || actualTable == "" {
		logrus.Errorf("Table not allowed for import: %s", safeTable)
		c.Error(errs.NewBadRequestError("Table not allowed for import: "+req.Table, nil))
		return
	}

	// Decode base64 + gzip data
	decodedData, err := decodeDataWithGzip(req.Data)
	if err != nil {
		logrus.Errorf("Failed to decode data for table %s: %v", safeTable, err)
		c.JSON(http.StatusBadRequest, gin.H{"inserted": 0, "errors": []string{fmt.Sprintf("Failed to decode data: %v", err)}})
		return
	}

	// Convert to slice of interfaces
	dataSlice, ok := decodedData.([]interface{})
	if !ok {
		logrus.Errorf("Decoded data is not a slice for table %s", safeTable)
		c.JSON(http.StatusBadRequest, gin.H{"inserted": 0, "errors": []string{"Decoded data is not a valid array"}})
		return
	}

	logrus.Infof("Starting encoded bulk import for table: %s -> %s, Records: %d", safeTable, actualTable, len(dataSlice))

	// Validate and convert data based on table type
	validatedData, err := h.validateAndConvertData(req.Table, dataSlice)
	if err != nil {
		logrus.Errorf("Data validation failed for table %s: %v", safeTable, err)
		c.JSON(http.StatusBadRequest, gin.H{"inserted": 0, "errors": []string{fmt.Sprintf("Data validation failed: %v", err)}})
		return
	}

	if len(validatedData) == 0 {
		logrus.Warnf("No data to insert for table %s", safeTable)
		c.JSON(http.StatusOK, gin.H{"inserted": 0, "errors": []string{}})
		return
	}

	// Calculate optimal batch size for this table
	batchSize := h.calculateOptimalBatchSize(req.Table)
	logrus.Infof("About to insert %d records into table %s using batch size: %d", len(validatedData), actualTable, batchSize)

	// Start database transaction
	tx := h.db.Begin()
	if tx.Error != nil {
		logrus.Errorf("Failed to start transaction: %v", tx.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"inserted": 0, "errors": []string{"Failed to start transaction"}})
		return
	}

	// Process data in batches to avoid PostgreSQL parameter limit
	totalInserted, err := h.processBatchInsert(tx, actualTable, validatedData, batchSize)
	if err != nil {
		tx.Rollback()
		logrus.Errorf("Batch insert failed for table %s: %v", actualTable, err)
		c.JSON(http.StatusInternalServerError, gin.H{"inserted": 0, "errors": []string{err.Error()}})
		return
	}

	logrus.Infof("Batch insert completed for table: %s, total rows inserted: %d", actualTable, totalInserted)

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		logrus.Errorf("Failed to commit transaction: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"inserted": 0, "errors": []string{"Failed to commit transaction"}})
		return
	}

	// Verify actual insertion count using primary key
	actualInserted, err := h.verifyInsertionCount(actualTable, validatedData)
	if err != nil {
		logrus.Errorf("Failed to verify insertion count: %v", err)
		// Don't fail the request, just log the error and return the batch processing count
		actualInserted = totalInserted
	}

	logrus.Infof("Encoded bulk import completed for table: %s, Expected: %d, Actual: %d", actualTable, len(validatedData), actualInserted)
	c.JSON(http.StatusOK, gin.H{"inserted": actualInserted, "errors": []string{}})
}

// validateAndConvertData validates and converts data based on table type
func (h *MigrationHandler) validateAndConvertData(tableName string, data []interface{}) ([]map[string]interface{}, error) {
	var validatedData []map[string]interface{}

	for i, record := range data {
		recordMap, ok := record.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid data format at index %d", i)
		}

		convertedRecord, err := h.convertDataTypes(tableName, recordMap)
		if err != nil {
			return nil, fmt.Errorf("data conversion failed at index %d: %v", i, err)
		}

		validatedData = append(validatedData, convertedRecord)
	}

	return validatedData, nil
}

// convertDataTypes handles data type conversion for different tables
func (h *MigrationHandler) convertDataTypes(tableName string, record map[string]interface{}) (map[string]interface{}, error) {
	convertedRecord := make(map[string]interface{})

	for key, value := range record {
		// Convert MSSQL column name to PostgreSQL column name
		pgColumnName := h.convertColumnName(tableName, key)

		convertedValue, err := h.convertFieldValue(tableName, pgColumnName, value)
		if err != nil {
			return nil, fmt.Errorf("failed to convert field %s: %v", key, err)
		}
		convertedRecord[pgColumnName] = convertedValue
	}

	// Add default values for PostgreSQL-only columns
	if tableName == "Users" {
		convertedRecord["home_onboarding_status"] = true
		convertedRecord["contest_onboarding_status"] = true
	}

	return convertedRecord, nil
}

// convertFieldValue converts individual field values based on data type
func (h *MigrationHandler) convertFieldValue(tableName, fieldName string, value interface{}) (interface{}, error) {
	if value == nil {
		return nil, nil
	}

	switch v := value.(type) {
	case string:
		// Handle string conversions
		if v == "" {
			return nil, nil
		}

		// Convert datetime strings to time.Time for timestamp fields
		if h.isTimestampField(tableName, fieldName) {
			if parsedTime, err := time.Parse(time.RFC3339, v); err == nil {
				return parsedTime, nil
			}
			if parsedTime, err := time.Parse("2006-01-02T15:04:05", v); err == nil {
				return parsedTime, nil
			}
		}

		// Convert boolean strings
		if h.isBooleanField(tableName, fieldName) {
			if strings.ToLower(v) == "true" || v == "1" {
				return true, nil
			}
			if strings.ToLower(v) == "false" || v == "0" {
				return false, nil
			}
		}

		// Convert numeric strings
		if h.isIntegerField(tableName, fieldName) {
			if intVal, err := strconv.Atoi(v); err == nil {
				return intVal, nil
			}
		}

		return v, nil

	case float64:
		// Handle JSON number conversions
		if h.isIntegerField(tableName, fieldName) {
			return int(v), nil
		}
		if h.isBooleanField(tableName, fieldName) {
			return v != 0, nil
		}
		return v, nil

	case bool:
		return v, nil

	default:
		return v, nil
	}
}

// Helper functions to identify field types
func (h *MigrationHandler) isTimestampField(tableName, fieldName string) bool {
	timestampFields := map[string][]string{
		"Users":                 {"dob", "refresh_token_expiry_time", "registered_on"},
		"UserAddress":           {"created_on", "last_modified_on"},
		"Avatar":                {"published_on", "created_on", "last_modified_on"},
		"UserVideo":             {"video_watch_on", "created_on", "last_modified_on"},
		"JokeConsumptionByUser": {"joke_generated_at", "created_on", "last_modified_on"},
		"UserNotification":      {"launch_date", "created_on", "last_modified_on"},
		"OTPLogs":               {"created_on", "last_modified_on"},
		"UserRefreshTokens":     {"refresh_token_expiry_time", "created_on", "last_modified_on"},
		"UserMixCodeDetails":    {"created_on", "last_modified_on"},
		"UserQuestionAnswer":    {"created_on", "last_modified_on"},
		"UserRanks":             {"last_modified", "rowversion", "sys_start_time", "sys_end_time"},
		"UserReferral":          {"refer_on", "accept_on", "re_send_on", "created_on", "last_modified_on"},
		"UserWallet":            {"last_redeemed_on", "wallet_updated_on", "created_on", "last_modified_on"},
		"UserWinners":           {"winner_time", "claim_window_open", "claim_window_close", "activity_time", "claim_date", "created_on", "last_modified_on"},
		"VouchersInfos":         {"assigned_on", "created_on", "last_modified_on", "last_date_to_redeem"},
		"WinnerRewards":         {"start_date", "published_on", "created_on", "last_modified_on"},
		"WinnerRewardLanguages": {"created_on", "last_modified_on"},
	}

	if fields, exists := timestampFields[tableName]; exists {
		for _, field := range fields {
			if strings.EqualFold(field, fieldName) {
				return true
			}
		}
	}
	return false
}

func (h *MigrationHandler) isBooleanField(tableName, fieldName string) bool {
	booleanFields := map[string][]string{
		"Users":                 {"is_active", "is_referral", "is_profile_complete", "is_tamil", "email_confirmed", "phone_number_confirmed", "two_factor_enabled", "lockout_enabled", "is_profile_popup"},
		"UserAddress":           {"is_default", "is_active", "is_deleted"},
		"Avatar":                {"is_published", "is_active", "is_deleted"},
		"UserVideo":             {"is_complete", "is_coin_earned", "is_active", "is_deleted"},
		"JokeConsumptionByUser": {"is_active", "is_deleted"},
		"UserNotification":      {"is_new", "is_read", "is_active", "is_deleted"},
		"OTPLogs":               {"is_active", "is_deleted"},
		"UserRefreshTokens":     {"is_used", "is_active", "is_deleted"},
		"UserMixCodeDetails":    {"is_valid_mix_code", "is_already_redeemed_mix_code", "is_active_mix_code", "is_expired_mix_code", "is_redeemed_successfully", "is_gr_or_eq500", "is_active", "is_deleted"},
		"UserQuestionAnswer":    {"selected_answer", "is_active", "is_deleted"},
		"UserRanks":             {"lock"},
		"UserReferral":          {"is_accept", "is_resend", "is_tamil", "is_active", "is_deleted"},
		"UserWallet":            {"is_mix_code_enter", "winning", "is_active", "is_deleted"},
		"UserWinners":           {"is_claim_window_open", "is_activity_required", "is_activity_performed", "is_reward_claimed", "is_promo_code", "is_active", "is_deleted"},
		"VouchersInfos":         {"secure_code_available", "is_active", "is_deleted"},
		"WinnerRewards":         {"is_complete", "is_published", "is_id_required", "is_passport_required", "is_promo_code", "is_active", "is_deleted"},
		"WinnerRewardLanguages": {"is_active", "is_deleted"},
	}

	if fields, exists := booleanFields[tableName]; exists {
		for _, field := range fields {
			if strings.EqualFold(field, fieldName) {
				return true
			}
		}
	}
	return false
}

func (h *MigrationHandler) isIntegerField(tableName, fieldName string) bool {
	integerFields := map[string][]string{
		"Users":                 {"gender", "platform_master_id", "profile_percentage", "last_language_selection", "access_failed_count", "avatar_id"},
		"UserAddress":           {"id", "pincode", "pin_code_id", "city_id", "state_id"},
		"Avatar":                {"id"},
		"UserVideo":             {"id", "coine_earned"},
		"JokeConsumptionByUser": {"id", "joke_detail_id", "platform", "similar_joke", "selected_genre", "selected_language"},
		"UserNotification":      {"id", "notification_type_id"},
		"OTPLogs":               {"id"},
		"UserRefreshTokens":     {"id"},
		"UserMixCodeDetails":    {"id", "points"},
		"UserQuestionAnswer":    {"id", "question_master_id", "option_id"},
		"UserRanks":             {"id", "user_rank", "current_balance"},
		"UserReferral":          {"id", "re_send_count"},
		"UserWallet":            {"id", "total_coins_earned", "current_balance", "last_redeemed_coins", "coins_type_id", "platform_id"},
		"UserWinners":           {"winner_reward_id", "coins", "quota_reward_details_id", "user_rank"},
		"VouchersInfos":         {"id", "secure_code", "winner_reward_id", "user_winners_id"},
		"WinnerRewards":         {"id", "worth", "tier_master_id", "no_of_winners", "quota_type_id", "coin_type_id", "platform_master_id", "no_of_coins", "max_rewards", "total_wins", "total_claims", "display_order"},
		"WinnerRewardLanguages": {"id", "winner_reward_id", "language_id"},
	}

	if fields, exists := integerFields[tableName]; exists {
		for _, field := range fields {
			if strings.EqualFold(field, fieldName) {
				return true
			}
		}
	}
	return false
}

// verifyInsertionCount verifies how many records were actually inserted
func (h *MigrationHandler) verifyInsertionCount(tableName string, data []map[string]interface{}) (int, error) {
	if len(data) == 0 {
		return 0, nil
	}

	// Get PostgreSQL primary key for the PostgreSQL table name
	pgPrimaryKey := h.getPrimaryKey(tableName)
	if pgPrimaryKey == "" {
		return 0, fmt.Errorf("PostgreSQL primary key not found for PostgreSQL table: %s", tableName)
	}

	// Extract primary key values from the data (data already has PostgreSQL column names)
	var primaryKeyValues []interface{}
	for _, record := range data {
		if pkValue, exists := record[pgPrimaryKey]; exists && pkValue != nil {
			primaryKeyValues = append(primaryKeyValues, pkValue)
		}
	}

	if len(primaryKeyValues) == 0 {
		return 0, fmt.Errorf("no primary key values found in data")
	}

	// Count existing records in PostgreSQL
	var count int64
	err := h.db.Table(tableName).Where(fmt.Sprintf("%s IN ?", pgPrimaryKey), primaryKeyValues).Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("failed to count inserted records: %v", err)
	}

	return int(count), nil
}

// getPrimaryKey returns the PostgreSQL primary key field name for PostgreSQL tables
func (h *MigrationHandler) getPrimaryKey(tableName string) string {
	// Map PostgreSQL table names to PostgreSQL primary key names
	primaryKeys := map[string]string{
		"user_address":                    "id",
		"avatar":                          "id",
		"joke_genres":                     "id",
		"user_video":                      "id",
		"joke_consumption_by_user":        "id",
		"joke_details_received":           "id",
		"ugc_master":                      "id",
		"ugc_status_lookup":               "status_id",
		"language_key":                    "id",
		"language_key_values":             "id",
		"language_master":                 "id",
		"languages":                       "id",
		"states":                          "id",
		"cities":                          "id",
		"pin_code":                        "id",
		"notification_types":              "id",
		"user_notification":               "id",
		"option_master":                   "id",
		"option_master_languages":         "id",
		"otp_logs":                        "id",
		"plagiarized_infos":               "id",
		"question_master":                 "id",
		"question_master_languages":       "id",
		"users":                           "id",
		"user_refresh_tokens":             "id",
		"user_mixcode_details":            "id",
		"user_question_answer":            "id",
		"user_ranks":                      "id",
		"ad_user_creator_collab_reaction": "id",
		"user_referral":                   "id",
		"user_wallet":                     "id",
		"user_winners":                    "id",
		"vouchers_infos":                  "id",
		"winner_rewards":                  "id",
		"winner_reward_languages":         "id",
	}

	return primaryKeys[tableName]
}

// convertColumnName converts MSSQL column names to PostgreSQL column names
func (h *MigrationHandler) convertColumnName(tableName, mssqlColumn string) string {
	// Table-specific mappings (First 20 tables - exhaustive mappings)
	tableMappings := map[string]map[string]string{
		"Avatar": {
			// MSSQL -> PostgreSQL mappings for Avatar table
			"Id":             "id",
			"Name":           "name",
			"ImageURL":       "image_url",
			"IsPublished":    "is_published",
			"PublishedBy":    "published_by",
			"PublishedOn":    "published_on",
			"IsActive":       "is_active",
			"IsDeleted":      "is_deleted",
			"CreatedBy":      "created_by",
			"CreatedOn":      "created_on",
			"LastModifiedBy": "last_modified_by",
			"LastModifiedOn": "last_modified_on",
		},
		"UserAddress": {
			// MSSQL -> PostgreSQL mappings for UserAddress table
			"Id":              "id",
			"UserId":          "user_id",
			"Address1":        "address1",
			"Address2":        "address2",
			"Pincode":         "pincode",
			"PinCodeId":       "pin_code_id",
			"CityId":          "city_id",
			"StateId":         "state_id",
			"NearestLandmark": "nearest_landmark",
			"ShippingMobile":  "shipping_mobile",
			"IsDefault":       "is_default",
			"IsActive":        "is_active",
			"IsDeleted":       "is_deleted",
			"CreatedBy":       "created_by",
			"CreatedOn":       "created_on",
			"LastModifiedBy":  "last_modified_by",
			"LastModifiedOn":  "last_modified_on",
		},
		"JokeGenres": {
			// MSSQL -> PostgreSQL mappings for JokeGenres table
			"Id":             "id",
			"Genre":          "genre",
			"GenreEmoticon":  "genre_emoticon",
			"ImageUrl":       "image_url",
			"EmoIsActive":    "emo_is_active",
			"EmoIndex":       "emo_index",
			"IsActive":       "is_active",
			"IsDeleted":      "is_deleted",
			"CreatedBy":      "created_by",
			"CreatedOn":      "created_on",
			"LastModifiedBy": "last_modified_by",
			"LastModifiedOn": "last_modified_on",
		},
		"UserVideo": {
			// MSSQL -> PostgreSQL mappings for UserVideo table
			"Id":             "id",
			"AssetId":        "asset_id",
			"UserId":         "user_id",
			"IsComplete":     "is_complete",
			"VideoWatchOn":   "video_watch_on",
			"IsCoinEarned":   "is_coin_earned",
			"CoineEarned":    "coine_earned",
			"IsActive":       "is_active",
			"IsDeleted":      "is_deleted",
			"CreatedBy":      "created_by",
			"CreatedOn":      "created_on",
			"LastModifiedBy": "last_modified_by",
			"LastModifiedOn": "last_modified_on",
		},
		"JokeConsumptionByUser": {
			// MSSQL -> PostgreSQL mappings for JokeConsumptionByUser table
			"Id":               "id",
			"UserId":           "user_id",
			"JokeId":           "joke_id",
			"ArtistId":         "artist_id",
			"JokeGeneratedAt":  "joke_generated_at",
			"Platform":         "platform",
			"SimilarJoke":      "similar_joke",
			"SelectedLanguage": "selected_language",
			"SelectedGenre":    "selected_genre",
			"JokeDetailId":     "joke_detail_id",
			"IsActive":         "is_active",
			"IsDeleted":        "is_deleted",
			"CreatedBy":        "created_by",
			"CreatedOn":        "created_on",
			"LastModifiedBy":   "last_modified_by",
			"LastModifiedOn":   "last_modified_on",
		},
		"JokeDetailsReceived": {
			// MSSQL -> PostgreSQL mappings for JokeDetailsReceived table
			"Id":              "id",
			"JokeTitle":       "joke_title",
			"JokeContentType": "joke_content_type",
			"JokeCreator":     "joke_creator",
			"JokeLanguage":    "joke_language",
			"JokeGenre":       "joke_genre",
			"JokeDescription": "joke_description",
			"IsActive":        "is_active",
			"IsDeleted":       "is_deleted",
			"CreatedBy":       "created_by",
			"CreatedOn":       "created_on",
			"LastModifiedBy":  "last_modified_by",
			"LastModifiedOn":  "last_modified_on",
		},
		"UGCMaster": {
			// MSSQL -> PostgreSQL mappings for UGCMaster table
			"Id":             "id",
			"LanguageId":     "language_id",
			"Format":         "format",
			"Title":          "title",
			"JokeGenreId":    "joke_genre_id",
			"UserId":         "user_id",
			"AssetId":        "asset_id",
			"ContentUrl":     "content_url",
			"FileName":       "file_name",
			"GluedInFill":    "gluedin_fill",
			"PlatformId":     "platform_id",
			"Status":         "status",
			"IsActive":       "is_active",
			"IsDeleted":      "is_deleted",
			"CreatedBy":      "created_by",
			"CreatedOn":      "created_on",
			"LastModifiedBy": "last_modified_by",
			"LastModifiedOn": "last_modified_on",
		},
		"UGCStatus_lookup": {
			// MSSQL -> PostgreSQL mappings for UGCStatus_lookup table
			"StatusId": "status_id",
			"Status":   "status",
		},
		"LanguageKey": {
			// MSSQL -> PostgreSQL mappings for LanguageKey table
			"Id":             "id",
			"Name":           "name",
			"KeyTypeId":      "key_type_id",
			"IsActive":       "is_active",
			"IsDeleted":      "is_deleted",
			"CreatedBy":      "created_by",
			"CreatedOn":      "created_on",
			"LastModifiedBy": "last_modified_by",
			"LastModifiedOn": "last_modified_on",
		},
		"LanguageKeyValues": {
			// MSSQL -> PostgreSQL mappings for LanguageKeyValues table
			"Id":               "id",
			"LanguageKeyId":    "language_key_id",
			"LanguageMasterId": "language_master_id",
			"KeyValue":         "key_value",
			"IsActive":         "is_active",
			"IsDeleted":        "is_deleted",
			"CreatedBy":        "created_by",
			"CreatedOn":        "created_on",
			"LastModifiedBy":   "last_modified_by",
			"LastModifiedOn":   "last_modified_on",
		},
		"LanguageMaster": {
			// MSSQL -> PostgreSQL mappings for LanguageMaster table
			"Id":             "id",
			"Name":           "name",
			"LanguageCode":   "language_code",
			"PlatformId":     "platform_id",
			"IsActive":       "is_active",
			"IsDeleted":      "is_deleted",
			"CreatedBy":      "created_by",
			"CreatedOn":      "created_on",
			"LastModifiedBy": "last_modified_by",
			"LastModifiedOn": "last_modified_on",
		},
		"Languages": {
			// MSSQL -> PostgreSQL mappings for Languages table
			"Id":             "id",
			"Name":           "name",
			"IsActive":       "is_active",
			"IsDeleted":      "is_deleted",
			"CreatedBy":      "created_by",
			"CreatedOn":      "created_on",
			"LastModifiedBy": "last_modified_by",
			"LastModifiedOn": "last_modified_on",
			"LanguageKey":    "language_key",
		},
		"States": {
			// MSSQL -> PostgreSQL mappings for States table
			"Id":             "id",
			"CountryId":      "country_id",
			"ShortName":      "short_name",
			"Name":           "name",
			"IsActive":       "is_active",
			"IsDeleted":      "is_deleted",
			"CreatedBy":      "created_by",
			"CreatedOn":      "created_on",
			"LastModifiedBy": "last_modified_by",
			"LastModifiedOn": "last_modified_on",
		},
		"Cities": {
			// MSSQL -> PostgreSQL mappings for Cities table
			"Id":             "id",
			"Name":           "name",
			"StateId":        "state_id",
			"IsActive":       "is_active",
			"IsDeleted":      "is_deleted",
			"CreatedBy":      "created_by",
			"CreatedOn":      "created_on",
			"LastModifiedBy": "last_modified_by",
			"LastModifiedOn": "last_modified_on",
		},
		"PinCode": {
			// MSSQL -> PostgreSQL mappings for PinCode table
			"Id":              "id",
			"Pincode":         "pincode",
			"IsBlackList":     "is_black_list",
			"CityId":          "city_id",
			"IsDeliverable":   "is_deliverable",
			"DeliveryCharges": "delivery_charges",
			"IsActive":        "is_active",
			"IsDeleted":       "is_deleted",
			"CreatedBy":       "created_by",
			"CreatedOn":       "created_on",
			"LastModifiedBy":  "last_modified_by",
			"LastModifiedOn":  "last_modified_on",
		},
		"NotificationTypes": {
			// MSSQL -> PostgreSQL mappings for NotificationTypes table
			"Id":               "id",
			"Name":             "name",
			"TitleValue":       "title_value",
			"IconURL":          "icon_url",
			"LanguageMasterId": "language_master_id",
			"IsBroadcast":      "is_broadcast",
			"IsActive":         "is_active",
			"IsDeleted":        "is_deleted",
			"CreatedBy":        "created_by",
			"CreatedOn":        "created_on",
			"LastModifiedBy":   "last_modified_by",
			"LastModifiedOn":   "last_modified_on",
		},
		"UserNotification": {
			// MSSQL -> PostgreSQL mappings for UserNotification table
			"Id":                 "id",
			"UserId":             "user_id",
			"NotificationTypeId": "notification_type_id",
			"NotificationTitle":  "notification_title",
			"NotificationText":   "notification_text",
			"IsNew":              "is_new",
			"IsRead":             "is_read",
			"LaunchDate":         "launch_date",
			"IsActive":           "is_active",
			"IsDeleted":          "is_deleted",
			"CreatedBy":          "created_by",
			"CreatedOn":          "created_on",
			"LastModifiedBy":     "last_modified_by",
			"LastModifiedOn":     "last_modified_on",
		},
		"OptionMaster": {
			// MSSQL -> PostgreSQL mappings for OptionMaster table
			"Id":               "id",
			"QuestionMasterId": "question_master_id",
			"OptionText":       "option_text",
			"DisplayOrder":     "display_order",
			"IsActive":         "is_active",
			"IsDeleted":        "is_deleted",
			"CreatedBy":        "created_by",
			"CreatedOn":        "created_on",
			"LastModifiedBy":   "last_modified_by",
			"LastModifiedOn":   "last_modified_on",
		},
		"OptionMasterLanguages": {
			// MSSQL -> PostgreSQL mappings for OptionMasterLanguages table
			"Id":             "id",
			"OptionMasterId": "option_master_id",
			"LanguageId":     "language_id",
			"OptionText":     "option_text",
			"IsActive":       "is_active",
			"IsDeleted":      "is_deleted",
			"CreatedBy":      "created_by",
			"CreatedOn":      "created_on",
			"LastModifiedBy": "last_modified_by",
			"LastModifiedOn": "last_modified_on",
		},
		"OTPLogs": {
			// MSSQL -> PostgreSQL mappings for OTPLogs table
			"Id":             "id",
			"MobileNo":       "mobile_no",
			"OTP":            "otp",
			"IsActive":       "is_active",
			"IsDeleted":      "is_deleted",
			"CreatedBy":      "created_by",
			"CreatedOn":      "created_on",
			"LastModifiedBy": "last_modified_by",
			"LastModifiedOn": "last_modified_on",
		},
		"PlagiarizedInfos": {
			// MSSQL -> PostgreSQL mappings for PlagiarizedInfos table
			"Id":             "id",
			"AssetId":        "asset_id",
			"UserId":         "user_id",
			"Information":    "information",
			"IsActive":       "is_active",
			"IsDeleted":      "is_deleted",
			"CreatedBy":      "created_by",
			"CreatedOn":      "created_on",
			"LastModifiedBy": "last_modified_by",
			"LastModifiedOn": "last_modified_on",
		},
		"QuestionMaster": {
			// MSSQL -> PostgreSQL mappings for QuestionMaster table
			"Id":             "id",
			"QuestionText":   "question_text",
			"LanguageId":     "language_id",
			"QuesPoint":      "ques_point",
			"IsActive":       "is_active",
			"IsDeleted":      "is_deleted",
			"CreatedBy":      "created_by",
			"CreatedOn":      "created_on",
			"LastModifiedBy": "last_modified_by",
			"LastModifiedOn": "last_modified_on",
		},
		"QuestionMasterLanguages": {
			// MSSQL -> PostgreSQL mappings for QuestionMasterLanguages table
			"Id":               "id",
			"QuestionMasterId": "question_master_id",
			"LanguageId":       "language_id",
			"QuestionText":     "question_text",
			"IsActive":         "is_active",
			"IsDeleted":        "is_deleted",
			"CreatedBy":        "created_by",
			"CreatedOn":        "created_on",
			"LastModifiedBy":   "last_modified_by",
			"LastModifiedOn":   "last_modified_on",
		},
		"Users": {
			// MSSQL -> PostgreSQL mappings for Users table
			"Id":                     "id",
			"Name":                   "name",
			"Gender":                 "gender",
			"DOB":                    "dob",
			"AlternateMobile":        "alternate_mobile",
			"ProfilePicture":         "profile_picture",
			"IsActive":               "is_active",
			"RefreshToken":           "refresh_token",
			"RefreshTokenExpiryTime": "refresh_token_expiry_time",
			"MainRole":               "main_role",
			"AvatarId":               "avatar_id",
			"ReferralCode":           "referral_code",
			"IsReferral":             "is_referral",
			"ReferBy":                "refer_by",
			"IsProfileComplete":      "is_profile_complete",
			"IPAddress":              "ip_address",
			"PlatformMasterId":       "platform_master_id",
			"IsTamil":                "is_tamil",
			"ProfilePercentage":      "profile_percentage",
			"UserName":               "user_name",
			"NormalizedUserName":     "normalized_user_name",
			"Email":                  "email",
			"NormalizedEmail":        "normalized_email",
			"EmailConfirmed":         "email_confirmed",
			"PasswordHash":           "password_hash",
			"SecurityStamp":          "security_stamp",
			"ConcurrencyStamp":       "concurrency_stamp",
			"PhoneNumber":            "phone_number",
			"PhoneNumberConfirmed":   "phone_number_confirmed",
			"TwoFactorEnabled":       "two_factor_enabled",
			"LockoutEnd":             "lockout_end",
			"LockoutEnabled":         "lockout_enabled",
			"AccessFailedCount":      "access_failed_count",
			"IsConsumer":             "is_consumer",
			"IsCreator":              "is_creator",
			"LastLanguageSelection":  "last_language_selection",
			"RegisteredOn":           "registered_on",
			"IsProfilePopup":         "is_profile_popup",
		},
		"UserRefreshTokens": {
			// MSSQL -> PostgreSQL mappings for UserRefreshTokens table
			"Id":                     "id",
			"UserId":                 "user_id",
			"RefreshToken":           "refresh_token",
			"IsUsed":                 "is_used",
			"RefreshTokenExpiryTime": "refresh_token_expiry_time",
			"IsActive":               "is_active",
			"IsDeleted":              "is_deleted",
			"CreatedBy":              "created_by",
			"CreatedOn":              "created_on",
			"LastModifiedBy":         "last_modified_by",
			"LastModifiedOn":         "last_modified_on",
		},
		"UserMixCodeDetails": {
			// MSSQL -> PostgreSQL mappings for UserMixCodeDetails table
			"Id":                       "id",
			"UserId":                   "user_id",
			"MixCode":                  "mix_code",
			"IsValidMixCode":           "is_valid_mix_code",
			"IsAlreadyRedeemedMixCode": "is_already_redeemed_mix_code",
			"IsActiveMixCode":          "is_active_mix_code",
			"IsExpiredMixCode":         "is_expired_mix_code",
			"IsRedeemedSuccessfully":   "is_redeemed_successfully",
			"IsGrOrEq500":              "is_gr_or_eq500",
			"Points":                   "points",
			"VoucherAssigned":          "voucher_assigned",
			"IsActive":                 "is_active",
			"IsDeleted":                "is_deleted",
			"CreatedBy":                "created_by",
			"CreatedOn":                "created_on",
			"LastModifiedBy":           "last_modified_by",
			"LastModifiedOn":           "last_modified_on",
		},
		"UserQuestionAnswer": {
			// MSSQL -> PostgreSQL mappings for UserQuestionAnswer table
			"Id":               "id",
			"UserId":           "user_id",
			"QuestionMasterId": "question_master_id",
			"OptionId":         "option_id",
			"SelectedAnswer":   "selected_answer",
			"IsActive":         "is_active",
			"IsDeleted":        "is_deleted",
			"CreatedBy":        "created_by",
			"CreatedOn":        "created_on",
			"LastModifiedBy":   "last_modified_by",
			"LastModifiedOn":   "last_modified_on",
		},
		"UserRanks": {
			// MSSQL -> PostgreSQL mappings for UserRanks table
			"Id":             "id",
			"UserRank":       "user_rank",
			"UserId":         "user_id",
			"CurrentBalance": "current_balance",
			"Lock":           "lock",
			"LastModified":   "last_modified",
			"Rowversion":     "rowversion",
			"SysStartTime":   "sys_start_time",
			"SysEndTime":     "sys_end_time",
		},
		"AD_UserCreatorCollabReaction": {
			// MSSQL -> PostgreSQL mappings for AD_UserCreatorCollabReaction table
			"ID":           "id",
			"UserID":       "user_id",
			"JokeID":       "joke_id",
			"Reaction":     "reaction",
			"ReactionDate": "reaction_date",
			"ReactionDttm": "reaction_dttm",
		},
		"UserReferral": {
			// MSSQL -> PostgreSQL mappings for UserReferral table
			"Id":             "id",
			"UserId":         "user_id",
			"ReferTo":        "refer_to",
			"ReferOn":        "refer_on",
			"IsAccept":       "is_accept",
			"AcceptOn":       "accept_on",
			"IsResend":       "is_resend",
			"ReSendCount":    "re_send_count",
			"ReSendOn":       "re_send_on",
			"IsTamil":        "is_tamil",
			"IsActive":       "is_active",
			"IsDeleted":      "is_deleted",
			"CreatedBy":      "created_by",
			"CreatedOn":      "created_on",
			"LastModifiedBy": "last_modified_by",
			"LastModifiedOn": "last_modified_on",
			"ReferralCode":   "referral_code",
		},
		"UserWallet": {
			// MSSQL -> PostgreSQL mappings for UserWallet table
			"Id":                "id",
			"UserId":            "user_id",
			"TotalCoinsEarned":  "total_coins_earned",
			"CurrentBalance":    "current_balance",
			"LastRedeemedCoins": "last_redeemed_coins",
			"LastRedeemedOn":    "last_redeemed_on",
			"CoinsTypeId":       "coins_type_id",
			"PlatformId":        "platform_id",
			"WalletUpdatedOn":   "wallet_updated_on",
			"IsMixCodeEnter":    "is_mix_code_enter",
			"Winning":           "winning",
			"IsActive":          "is_active",
			"IsDeleted":         "is_deleted",
			"CreatedBy":         "created_by",
			"CreatedOn":         "created_on",
			"LastModifiedBy":    "last_modified_by",
			"LastModifiedOn":    "last_modified_on",
		},
		"UserWinners": {
			// MSSQL -> PostgreSQL mappings for UserWinners table
			"Id":                   "id",
			"UserId":               "user_id",
			"WinnerRewardId":       "winner_reward_id",
			"Coins":                "coins",
			"WinnerTime":           "winner_time",
			"IsClaimWindowOpen":    "is_claim_window_open",
			"ClaimWindowOpen":      "claim_window_open",
			"ClaimWindowClose":     "claim_window_close",
			"IsActivityRequired":   "is_activity_required",
			"IsActivityPerfomed":   "is_activity_performed",
			"ActivityTime":         "activity_time",
			"IsRewardClaimed":      "is_reward_claimed",
			"ClaimDate":            "claim_date",
			"QuotaRewardDetailsId": "quota_reward_details_id",
			"IsPromoCode":          "is_promo_code",
			"UserRank":             "user_rank",
			"IsActive":             "is_active",
			"IsDeleted":            "is_deleted",
			"CreatedBy":            "created_by",
			"CreatedOn":            "created_on",
			"LastModifiedBy":       "last_modified_by",
			"LastModifiedOn":       "last_modified_on",
		},
		"VouchersInfos": {
			// MSSQL -> PostgreSQL mappings for VouchersInfos table
			"Id":                  "id",
			"VoucherCode":         "voucher_code",
			"SecureCodeAvailable": "secure_code_available",
			"SecureCode":          "secure_code",
			"WinnerRewardId":      "winner_reward_id",
			"AssignedToUserId":    "assigned_to_user_id",
			"AssignedOn":          "assigned_on",
			"UserWinnersId":       "user_winners_id",
			"IsActive":            "is_active",
			"IsDeleted":           "is_deleted",
			"CreatedBy":           "created_by",
			"CreatedOn":           "created_on",
			"LastModifiedBy":      "last_modified_by",
			"LastModifiedOn":      "last_modified_on",
			"LastDateToRedeem":    "last_date_to_redeem",
		},
		// #nosec G101 -- column name mappings (e.g. "PublishedBy" -> "published_by"), not credentials
		"WinnerRewards": {
			// MSSQL -> PostgreSQL mappings for WinnerRewards table
			"Id":                 "id",
			"Name":               "name",
			"Description":        "description",
			"LongDesc":           "long_desc",
			"Information":        "information",
			"ImageUrl":           "image_url",
			"Worth":              "worth",
			"IconUrl":            "icon_url",
			"RedirectUrl":        "redirect_url",
			"OLName":             "ol_name",
			"OLDescription":      "ol_description",
			"OLLongDesc":         "ol_long_desc",
			"OLImageUrl":         "ol_image_url",
			"OLIconUrl":          "ol_icon_url",
			"TierMasterId":       "tier_master_id",
			"NoOfWinners":        "no_of_winners",
			"QuotaTypeId":        "quota_type_id",
			"CoinTypeId":         "coin_type_id",
			"PlatformMasterId":   "platform_master_id",
			"NoOfCoins":          "no_of_coins",
			"MaxRewards":         "max_rewards",
			"StartDate":          "start_date",
			"IsComplete":         "is_complete",
			"TotalWins":          "total_wins",
			"TotalClaims":        "total_claims",
			"DisplayOrder":       "display_order",
			"IsPublished":        "is_published",
			"PublishedBy":        "published_by",
			"PublishedOn":        "published_on",
			"IsIdRequired":       "is_id_required",
			"IsPassportRequired": "is_passport_required",
			"IsPromocode":        "is_promocode",
			"NGPSID":             "ngps_id",
			"shareUrl":           "share_url",
			"IsActive":           "is_active",
			"IsDeleted":          "is_deleted",
			"CreatedBy":          "created_by",
			"CreatedOn":          "created_on",
			"LastModifiedBy":     "last_modified_by",
			"LastModifiedOn":     "last_modified_on",
		},
		"WinnerRewardLanguages": {
			// MSSQL -> PostgreSQL mappings for WinnerRewardLanguages table
			"Id":             "id",
			"WinnerRewardId": "winner_reward_id",
			"LanguageId":     "language_id",
			"Name":           "name",
			"Description":    "description",
			"Information":    "information",
			"buttonValue":    "button_value",
			"buttonValueEn":  "button_value_en",
			"IsActive":       "is_active",
			"IsDeleted":      "is_deleted",
			"CreatedBy":      "created_by",
			"CreatedOn":      "created_on",
			"LastModifiedBy": "last_modified_by",
			"LastModifiedOn": "last_modified_on",
		},
	}

	// Check table-specific mappings first
	if tableMap, exists := tableMappings[tableName]; exists {
		if pgColumn, exists := tableMap[mssqlColumn]; exists {
			return pgColumn
		}
	}

	// If no mapping found, return the original column name (this should not happen with exhaustive mappings)
	logrus.Warnf("No column mapping found for table %s, column %s", utils.SanitizeForLog(tableName), utils.SanitizeForLog(mssqlColumn))
	return mssqlColumn
}

// Helper function to calculate optimal batch size based on table columns
func (h *MigrationHandler) calculateOptimalBatchSize(tableName string) int {
	// Get estimated column count for the table
	columnCount := h.getEstimatedColumnCount(tableName)

	// Calculate safe batch size to stay well below 65535 parameter limit
	// Use 50000 as safe limit (leaving buffer for other parameters)
	maxBatchSize := 50000 / columnCount

	// Set reasonable bounds: minimum 100, maximum 2000
	if maxBatchSize < 100 {
		return 100
	}
	if maxBatchSize > 2000 {
		return 2000
	}

	return maxBatchSize
}

// Helper function to estimate column count for tables
func (h *MigrationHandler) getEstimatedColumnCount(tableName string) int {
	// Estimated column counts for different tables
	columnCounts := map[string]int{
		"UserAddress":                  17,
		"Avatar":                       12,
		"JokeGenres":                   12,
		"UserVideo":                    12,
		"JokeConsumptionByUser":        16,
		"JokeDetailsReceived":          12,
		"UGCMaster":                    16,
		"UGCStatus_lookup":             2,
		"LanguageKey":                  9,
		"LanguageKeyValues":            10,
		"LanguageMaster":               10,
		"Languages":                    9,
		"States":                       10,
		"Cities":                       9,
		"PinCode":                      12,
		"NotificationTypes":            12,
		"UserNotification":             13,
		"OptionMaster":                 10,
		"OptionMasterLanguages":        10,
		"OTPLogs":                      8,
		"PlagiarizedInfos":             10,
		"QuestionMaster":               10,
		"QuestionMasterLanguages":      10,
		"Users":                        40,
		"UserRefreshTokens":            11,
		"UserMixCodeDetails":           16,
		"UserQuestionAnswer":           10,
		"UserRanks":                    9,
		"AD_UserCreatorCollabReaction": 6,
		"UserReferral":                 16,
		"UserWallet":                   16,
		"UserWinners":                  18,
		"VouchersInfos":                14,
		"WinnerRewards":                27,
		"WinnerRewardLanguages":        12,
	}

	if count, exists := columnCounts[tableName]; exists {
		return count
	}

	// Default to 15 columns if not found
	return 15
}

// Helper function to process data in batches
func (h *MigrationHandler) processBatchInsert(tx *gorm.DB, tableName string, data []map[string]interface{}, batchSize int) (int, error) {
	totalInserted := 0

	for i := 0; i < len(data); i += batchSize {
		end := i + batchSize
		if end > len(data) {
			end = len(data)
		}

		batch := data[i:end]
		logrus.Infof("Processing batch %d-%d of %d records for table %s", i+1, end, len(data), tableName)

		result := tx.Table(tableName).Clauses(clause.OnConflict{DoNothing: true}).Create(&batch)
		if result.Error != nil {
			return totalInserted, result.Error
		}

		batchInserted := int(result.RowsAffected)
		totalInserted += batchInserted

		logrus.Infof("Batch %d-%d completed: %d records inserted", i+1, end, batchInserted)
	}

	return totalInserted, nil
}

// S3ToGCSMigrationRequest represents the request for S3 to GCS migration
type S3ToGCSMigrationRequest struct {
	SourcePath     string `json:"source_path" binding:"required"`
	DestinationPath string `json:"destination_path" binding:"required"`
	FileData       string `json:"file_data" binding:"required"` // Base64 encoded file data
	ContentType    string `json:"content_type"`
	OriginalName   string `json:"original_name"`
}

// S3ToGCSBatchMigrationRequest represents the request for batch S3 to GCS migration
type S3ToGCSBatchMigrationRequest struct {
	Files []S3ToGCSMigrationRequest `json:"files" binding:"required"`
}

// S3ToGCSMigrationResponse represents the response for S3 to GCS migration
type S3ToGCSMigrationResponse struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	GCSURL       string `json:"gcs_url,omitempty"`
	GCSPath      string `json:"gcs_path,omitempty"`
	Error        string `json:"error,omitempty"`
}

// S3ToGCSBatchMigrationResponse represents the response for batch S3 to GCS migration
type S3ToGCSBatchMigrationResponse struct {
	Success       bool                           `json:"success"`
	Message       string                         `json:"message"`
	TotalFiles    int                            `json:"total_files"`
	Successful    int                            `json:"successful"`
	Failed        int                            `json:"failed"`
	Results       []S3ToGCSMigrationResponse    `json:"results"`
	Errors        []string                       `json:"errors"`
}

// MigrateS3ToGCS handles the migration of files from S3 to GCS
func (h *MigrationHandler) MigrateS3ToGCS(c *gin.Context) {
	var req S3ToGCSMigrationRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		logrus.Errorf("Invalid request body for S3 to GCS migration: %v", err)
		c.JSON(http.StatusBadRequest, S3ToGCSMigrationResponse{
			Success: false,
			Error:   "Invalid request body: " + err.Error(),
		})
		return
	}

	// Validate required fields
	if req.SourcePath == "" || req.DestinationPath == "" || req.FileData == "" {
		c.JSON(http.StatusBadRequest, S3ToGCSMigrationResponse{
			Success: false,
			Error:   "Missing required fields: source_path, destination_path, or file_data",
		})
		return
	}

	// Decode base64 file data
	fileData, err := base64.StdEncoding.DecodeString(req.FileData)
	if err != nil {
		logrus.Errorf("Failed to decode base64 file data: %v", err)
		c.JSON(http.StatusBadRequest, S3ToGCSMigrationResponse{
			Success: false,
			Error:   "Invalid base64 encoded file data",
		})
		return
	}

	// Create a reader from the decoded data
	reader := io.NopCloser(bytes.NewReader(fileData))

	// Get GCS service from context or create new one
	gcsService, err := utils.NewGCSService("tccc-sfa-prod-bucket-mount", "ai0015455-sfa-prod")
	if err != nil {
		logrus.Errorf("Failed to create GCS service: %v", err)
		c.JSON(http.StatusInternalServerError, S3ToGCSMigrationResponse{
			Success: false,
			Error:   "Failed to initialize GCS service",
		})
		return
	}

	// Upload file to GCS
	ctx := c.Request.Context()
	gcsURL, gcsPath, err := gcsService.UploadFileFromReader(ctx, reader, req.DestinationPath)
	if err != nil {
		logrus.Errorf("Failed to upload file to GCS: %v", err)
		c.JSON(http.StatusInternalServerError, S3ToGCSMigrationResponse{
			Success: false,
			Error:   "Failed to upload file to GCS: " + err.Error(),
		})
		return
	}

	logrus.Infof("Successfully migrated file from S3 path %s to GCS path %s", utils.SanitizeForLog(req.SourcePath), utils.SanitizeForLog(gcsPath))

	c.JSON(http.StatusOK, S3ToGCSMigrationResponse{
		Success: true,
		Message: "File successfully migrated from S3 to GCS",
		GCSURL:  gcsURL,
		GCSPath: gcsPath,
	})
}

// MigrateS3ToGCSBatch handles batch migration of files from S3 to GCS with parallel processing
func (h *MigrationHandler) MigrateS3ToGCSBatch(c *gin.Context) {
	var req S3ToGCSBatchMigrationRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		logrus.Errorf("Invalid request body for batch S3 to GCS migration: %v", err)
		c.JSON(http.StatusBadRequest, S3ToGCSBatchMigrationResponse{
			Success: false,
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}

	if len(req.Files) == 0 {
		c.JSON(http.StatusBadRequest, S3ToGCSBatchMigrationResponse{
			Success: false,
			Message: "No files provided for migration",
		})
		return
	}

	// Limit batch size to prevent memory issues
	maxBatchSize := 50
	if len(req.Files) > maxBatchSize {
		c.JSON(http.StatusBadRequest, S3ToGCSBatchMigrationResponse{
			Success: false,
			Message: fmt.Sprintf("Batch size too large. Maximum allowed: %d, provided: %d", maxBatchSize, len(req.Files)),
		})
		return
	}

	logrus.Infof("Starting batch migration of %d files", len(req.Files))

	// Get GCS service
	gcsService, err := utils.NewGCSService("tccc-sfa-prod-bucket-mount", "ai0015455-sfa-prod")
	if err != nil {
		logrus.Errorf("Failed to create GCS service: %v", err)
		c.JSON(http.StatusInternalServerError, S3ToGCSBatchMigrationResponse{
			Success: false,
			Message: "Failed to initialize GCS service",
		})
		return
	}

	// Create channels for results
	type uploadResult struct {
		Index  int
		Result S3ToGCSMigrationResponse
	}
	resultChan := make(chan uploadResult, len(req.Files))

	// Process files in parallel using goroutines
	ctx := c.Request.Context()
	totalFiles := len(req.Files)
	for i, fileReq := range req.Files {
		go func(index int, fileReq S3ToGCSMigrationRequest) {
			result := S3ToGCSMigrationResponse{
				Success: false,
				Error:   "Unknown error",
			}

			// Validate required fields
			if fileReq.SourcePath == "" || fileReq.DestinationPath == "" || fileReq.FileData == "" {
				result.Error = "Missing required fields"
				resultChan <- uploadResult{Index: index, Result: result}
				return
			}

			// Decode base64 file data
			fileData, err := base64.StdEncoding.DecodeString(fileReq.FileData)
			if err != nil {
				result.Error = "Invalid base64 encoded file data"
				resultChan <- uploadResult{Index: index, Result: result}
				return
			}

			// Create a reader from the decoded data
			reader := io.NopCloser(bytes.NewReader(fileData))

			// Upload file to GCS
			gcsURL, gcsPath, err := gcsService.UploadFileFromReader(ctx, reader, fileReq.DestinationPath)
			if err != nil {
				result.Error = "Failed to upload file to GCS: " + err.Error()
				resultChan <- uploadResult{Index: index, Result: result}
				return
			}

			// Success
			result.Success = true
			result.Message = "File successfully migrated from S3 to GCS"
			result.GCSURL = gcsURL
			result.GCSPath = gcsPath

			logrus.Infof("Successfully migrated file %d/%d: %s", index+1, totalFiles, utils.SanitizeForLog(fileReq.SourcePath))
			resultChan <- uploadResult{Index: index, Result: result}
		}(i, fileReq)
	}

	// Collect results
	results := make([]S3ToGCSMigrationResponse, len(req.Files))
	successful := 0
	failed := 0
	var errors []string

	for i := 0; i < len(req.Files); i++ {
		result := <-resultChan
		results[result.Index] = result.Result

		if result.Result.Success {
			successful++
		} else {
			failed++
			errors = append(errors, fmt.Sprintf("File %d: %s", result.Index+1, result.Result.Error))
		}
	}

	// Prepare response
	response := S3ToGCSBatchMigrationResponse{
		Success:    failed == 0,
		Message:    fmt.Sprintf("Batch migration completed. Successful: %d, Failed: %d", successful, failed),
		TotalFiles: len(req.Files),
		Successful: successful,
		Failed:     failed,
		Results:    results,
		Errors:     errors,
	}

	logrus.Infof("Batch migration completed: %d successful, %d failed", successful, failed)
	c.JSON(http.StatusOK, response)
}
