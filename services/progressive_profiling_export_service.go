package services

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"

	"github.com/Infinite-Locus-Product/coca_cola_backend/config"
	"github.com/Infinite-Locus-Product/coca_cola_backend/constants"
	"github.com/Infinite-Locus-Product/coca_cola_backend/utils"
)

const progressiveProfilingExportURLExpiry = 24 * time.Hour
const progressiveProfilingExportContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
const progressiveProfilingExportSheetName = "Sheet1"

// ProgressiveProfilingExportService backfills historical progressive-profiling
// answers (profiling_question_answer, joined with option_master and users)
// into the CDP event shape, so they can be replayed into the CDP platform for
// events that were never fired at submission time.
type ProgressiveProfilingExportService interface {
	// Export builds one CDP-shaped row per submitted answer (brand_name,
	// client_id, event_type, event_sub_type, phone_e164, created_dt —
	// created_dt is the original submission time as a Unix timestamp),
	// uploads the resulting .xlsx workbook to cloud storage, and returns a
	// time-limited signed URL to download it.
	Export(ctx context.Context) (string, error)
}

type progressiveProfilingExportService struct {
	db         *gorm.DB
	cdpConfig  config.CDPConfig
	gcsService utils.GCSService
}

func NewProgressiveProfilingExportService(db *gorm.DB, cdpConfig config.CDPConfig, gcsService utils.GCSService) ProgressiveProfilingExportService {
	return &progressiveProfilingExportService{db: db, cdpConfig: cdpConfig, gcsService: gcsService}
}

type progressiveProfilingExportRow struct {
	// ID must be present and mapped to pqa.id so gorm.FindInBatches can
	// resolve a primary key to paginate past the first batch — without it,
	// batches beyond the first fail with "primary key required" as soon as
	// the result set exceeds progressiveProfilingExportBatchSize.
	ID               int64
	QuestionMasterID int
	OptionText       string
	PhoneNumber      string
	CreatedOn        time.Time
}

const progressiveProfilingExportBatchSize = 500

func (s *progressiveProfilingExportService) Export(ctx context.Context) (string, error) {
	f := excelize.NewFile()
	defer f.Close()

	sw, err := f.NewStreamWriter(progressiveProfilingExportSheetName)
	if err != nil {
		return "", err
	}

	header := []any{"brand_name", "client_id", "event_type", "event_sub_type", "phone_e164", "created_dt"}
	if err := sw.SetRow("A1", header); err != nil {
		return "", err
	}
	rowNum := 2

	questionIDs := make([]int, 0, len(constants.ProgressiveProfilingCDPEventPrefixes))
	for id := range constants.ProgressiveProfilingCDPEventPrefixes {
		questionIDs = append(questionIDs, id)
	}

	if len(questionIDs) > 0 {
		query := s.db.WithContext(ctx).
			Table("profiling_question_answer AS pqa").
			Select("pqa.id AS id, pqa.question_master_id AS question_master_id, om.option_text AS option_text, u.phone_number AS phone_number, pqa.created_on AS created_on").
			Joins("JOIN users u ON u.id = pqa.user_id::uuid").
			Joins("JOIN option_master om ON om.id = pqa.option_id").
			Where("pqa.is_deleted = ?", false).
			Where("pqa.selected_answer = ?", true).
			Where("pqa.question_master_id IN ?", questionIDs).
			Order("pqa.id")

		var batch []progressiveProfilingExportRow
		result := query.FindInBatches(&batch, progressiveProfilingExportBatchSize, func(tx *gorm.DB, batchNum int) error {
			for _, row := range batch {
				prefix, ok := constants.ProgressiveProfilingCDPEventPrefixes[row.QuestionMasterID]
				if !ok {
					continue
				}

				record := []any{
					constants.CDP_BRAND_NAME_SPRITE,
					s.cdpConfig.ClientID,
					"Click",
					prefix + "_" + deriveEventSubTypeSuffix(row.QuestionMasterID, row.OptionText),
					phoneE164(row.PhoneNumber),
					row.CreatedOn.Unix(),
				}
				cell, err := excelize.CoordinatesToCellName(1, rowNum)
				if err != nil {
					return err
				}
				if err := sw.SetRow(cell, record); err != nil {
					return err
				}
				rowNum++
			}
			return nil
		})
		if result.Error != nil {
			return "", result.Error
		}
	}

	if err := sw.Flush(); err != nil {
		return "", err
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return "", err
	}

	filename := fmt.Sprintf("progressive_profiling_cdp_backfill_%s.xlsx", uuid.New().String())
	_, objectPath, err := s.gcsService.UploadFileFromReaderWithOptions(ctx, io.NopCloser(buf), constants.PROGRESSIVE_PROFILING_EXPORT_FOLDER, filename, progressiveProfilingExportContentType)
	if err != nil {
		return "", fmt.Errorf("failed to upload progressive profiling CDP export to storage: %w", err)
	}

	url, err := s.gcsService.GetFileSignedURL(ctx, objectPath, progressiveProfilingExportURLExpiry)
	if err != nil {
		return "", fmt.Errorf("failed to generate signed URL for progressive profiling CDP export: %w", err)
	}

	return url, nil
}

// phoneE164 formats a raw Indian mobile number as E.164 (+91XXXXXXXXXX).
// users.phone_number is stored either as a bare 10-digit number or already
// prefixed with the "91" country code (but no "+"), depending on how the
// number was captured — production has both. Anything else (already E.164,
// or an unrecognized shape) is passed through unchanged rather than guessed
// at.
func phoneE164(phoneNumber string) string {
	trimmed := strings.TrimSpace(phoneNumber)
	switch {
	case strings.HasPrefix(trimmed, "+"):
		return trimmed
	case len(trimmed) == 10:
		return "+91" + trimmed
	case len(trimmed) == 12 && strings.HasPrefix(trimmed, "91"):
		return "+" + trimmed
	default:
		return trimmed
	}
}

// deriveEventSubTypeSuffix reconstructs the CDP option suffix from a stored
// option_text: "morning" -> "Morning", "6+" -> "6Plus". Bottle-count answers
// are numeric and get bucketed at 6+ to match the CDP vocabulary
// (ConsumptionCount_1..5, ConsumptionCount_6Plus) rather than passed through
// as literal digits above 5.
func deriveEventSubTypeSuffix(questionMasterID int, optionText string) string {
	trimmed := strings.TrimSpace(optionText)

	if questionMasterID == constants.PROGRESSIVE_PROFILING_BOTTLE_COUNT_QUESTION_ID {
		if n, err := strconv.Atoi(trimmed); err == nil {
			if n >= 6 {
				return "6Plus"
			}
			return strconv.Itoa(n)
		}
	}

	withoutPlus := strings.ReplaceAll(trimmed, "+", "Plus")
	words := strings.Fields(withoutPlus)
	for i, word := range words {
		words[i] = strings.ToUpper(word[:1]) + word[1:]
	}
	return strings.Join(words, "")
}
