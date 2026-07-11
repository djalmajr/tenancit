package usage

import (
	"testing"
	"time"
)

func TestRetentionCutoffKeepsSixCompleteMonths(t *testing.T) {
	now := time.Date(2026, 7, 31, 22, 0, 0, 0, time.FixedZone("offset", -3*60*60))
	got := RetentionCutoff(now)
	want := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("cutoff = %v, want %v", got, want)
	}
}
