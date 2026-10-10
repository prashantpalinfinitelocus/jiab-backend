package services

import (
	"testing"

	"github.com/Infinite-Locus-Product/coca_cola_backend/constants"
)

func TestDeriveEventSubTypeSuffix(t *testing.T) {
	cases := []struct {
		name             string
		questionMasterID int
		optionText       string
		want             string
	}{
		{"bottle count 1", constants.PROGRESSIVE_PROFILING_BOTTLE_COUNT_QUESTION_ID, "1", "1"},
		{"bottle count 5", constants.PROGRESSIVE_PROFILING_BOTTLE_COUNT_QUESTION_ID, "5", "5"},
		{"bottle count 6 buckets to 6Plus", constants.PROGRESSIVE_PROFILING_BOTTLE_COUNT_QUESTION_ID, "6", "6Plus"},
		{"bottle count 7 buckets to 6Plus", constants.PROGRESSIVE_PROFILING_BOTTLE_COUNT_QUESTION_ID, "7", "6Plus"},
		{"daypart morning", constants.PROGRESSIVE_PROFILING_DAYPART_QUESTION_ID, "morning", "Morning"},
		{"daypart noon", constants.PROGRESSIVE_PROFILING_DAYPART_QUESTION_ID, "noon", "Noon"},
		{"daypart evening", constants.PROGRESSIVE_PROFILING_DAYPART_QUESTION_ID, "evening", "Evening"},
		{"genre single word", 999, "cricket", "Cricket"},
		{"literal plus sign", constants.PROGRESSIVE_PROFILING_DAYPART_QUESTION_ID, "6+", "6Plus"},
		{"whitespace trimmed", constants.PROGRESSIVE_PROFILING_DAYPART_QUESTION_ID, "  morning  ", "Morning"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := deriveEventSubTypeSuffix(tc.questionMasterID, tc.optionText)
			if got != tc.want {
				t.Errorf("deriveEventSubTypeSuffix(%d, %q) = %q, want %q", tc.questionMasterID, tc.optionText, got, tc.want)
			}
		})
	}
}

func TestPhoneE164(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"9569091401", "+919569091401"},
		{"919569091401", "+919569091401"},
		{"+919569091401", "+919569091401"},
		{"", ""},
	}

	for _, tc := range cases {
		got := phoneE164(tc.in)
		if got != tc.want {
			t.Errorf("phoneE164(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
