package usage

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type retentionStoreStub struct{ cutoffs chan pgtype.Date }

func (s retentionStoreStub) DeleteExpiredAPIClientUsage(_ context.Context, cutoff pgtype.Date) (int64, error) {
	s.cutoffs <- cutoff
	return 0, nil
}

type retentionPolicyStub struct {
	months int
	err    error
}

func (p retentionPolicyStub) UsageRetentionMonths(context.Context) (int, error) {
	return p.months, p.err
}

func TestRetentionCutoffKeepsSixCompleteMonths(t *testing.T) {
	now := time.Date(2026, 7, 31, 22, 0, 0, 0, time.FixedZone("offset", -3*60*60))
	got := RetentionCutoff(now)
	want := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("cutoff = %v, want %v", got, want)
	}
}

func TestRunRetentionUsesRuntimePolicyAndFailsClosed(t *testing.T) {
	now := func() time.Time { return time.Date(2026, 7, 31, 22, 0, 0, 0, time.FixedZone("offset", -3*60*60)) }

	for _, testCase := range []struct {
		name       string
		policy     retentionPolicyStub
		wantCutoff *time.Time
	}{
		{name: "configured months", policy: retentionPolicyStub{months: 3}, wantCutoff: timePointer(time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC))},
		{name: "policy unavailable", policy: retentionPolicyStub{err: errors.New("unavailable")}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cutoffs := make(chan pgtype.Date, 1)
			done := make(chan struct{})
			go func() {
				RunRetention(ctx, retentionStoreStub{cutoffs: cutoffs}, testCase.policy, now, time.Hour)
				close(done)
			}()
			if testCase.wantCutoff == nil {
				select {
				case cutoff := <-cutoffs:
					t.Fatalf("retention ran while policy unavailable: %v", cutoff)
				case <-time.After(25 * time.Millisecond):
				}
			} else {
				select {
				case cutoff := <-cutoffs:
					if !cutoff.Valid || !cutoff.Time.Equal(*testCase.wantCutoff) {
						t.Fatalf("cutoff=%v want=%v", cutoff, *testCase.wantCutoff)
					}
				case <-time.After(time.Second):
					t.Fatal("retention did not run")
				}
			}
			cancel()
			<-done
		})
	}
}

func timePointer(value time.Time) *time.Time { return &value }
