package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/djalmajr/tenancit/server/internal/ratelimit"
	usageevents "github.com/djalmajr/tenancit/server/internal/usage"
)

type fakeRateLimiter struct {
	result ratelimit.Result
	err    error
}

func (f fakeRateLimiter) Allow(context.Context, string, int32) (ratelimit.Result, error) {
	return f.result, f.err
}

type captureUsageRecorder struct{ events []usageevents.Event }

func (c *captureUsageRecorder) Record(event usageevents.Event) { c.events = append(c.events, event) }

func rateLimitedRequest() *http.Request {
	ctx := contextWithAPIClientPrincipal(context.Background(), apiClientPrincipal{
		ID: "018f74d0-1337-7abc-8def-0123456789ab", RPMLimit: pointerTo(int32(60)),
	})
	return httptest.NewRequest(http.MethodGet, "/v1/identify", nil).WithContext(ctx)
}

func TestRateLimitReturnsContractHeadersAndUsage(t *testing.T) {
	recorder := &captureUsageRecorder{}
	middleware := EnforceAPIClientRateLimit(fakeRateLimiter{result: ratelimit.Result{
		Allowed: false, Limit: 60, Remaining: 0,
		RetryAfter: 500 * time.Millisecond, ResetAfter: time.Minute,
	}}, recorder, "identify", time.Now)
	rec := httptest.NewRecorder()
	middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rec, rateLimitedRequest())

	if rec.Code != http.StatusTooManyRequests || rec.Header().Get("Retry-After") != "1" ||
		rec.Header().Get("RateLimit-Limit") != "60" || rec.Header().Get("RateLimit-Remaining") != "0" {
		t.Fatalf("response = %d headers=%v", rec.Code, rec.Header())
	}
	if len(recorder.events) != 1 || !recorder.events[0].RateLimited {
		t.Fatalf("usage events = %+v", recorder.events)
	}
}

func TestRateLimitFailsClosedWhenValkeyIsUnavailable(t *testing.T) {
	middleware := EnforceAPIClientRateLimit(
		fakeRateLimiter{err: ratelimit.ErrUnavailable}, &captureUsageRecorder{}, "resolve", time.Now,
	)
	rec := httptest.NewRecorder()
	middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rec, rateLimitedRequest())
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("code = %d, want 503", rec.Code)
	}
}

func pointerTo[T any](value T) *T { return &value }
