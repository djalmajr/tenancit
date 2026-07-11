package ratelimit

import (
	"context"
	"testing"
	"time"
)

func TestMemoryLimiterRefillsContinuously(t *testing.T) {
	now := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	limiter := NewMemory(func() time.Time { return now })
	ctx := context.Background()

	for i := 0; i < 2; i++ {
		result, err := limiter.Allow(ctx, "client-1", 2)
		if err != nil || !result.Allowed {
			t.Fatalf("request %d = %+v, %v", i+1, result, err)
		}
	}
	result, err := limiter.Allow(ctx, "client-1", 2)
	if err != nil || result.Allowed || result.RetryAfter <= 0 {
		t.Fatalf("third request = %+v, %v, want limited", result, err)
	}

	now = now.Add(30 * time.Second)
	result, err = limiter.Allow(ctx, "client-1", 2)
	if err != nil || !result.Allowed {
		t.Fatalf("request after refill = %+v, %v", result, err)
	}
}

func TestValkeyKeyContainsOnlyClientID(t *testing.T) {
	if got := valkeyBucketKey("client-uuid"); got != "tenancit:rate-limit:client-uuid" {
		t.Fatalf("bucket key = %q", got)
	}
}
