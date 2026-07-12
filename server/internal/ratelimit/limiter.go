package ratelimit

import (
	"context"
	"errors"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/djalmajr/tenancit/server/internal/telemetry"
	"github.com/redis/go-redis/v9"
)

var ErrUnavailable = errors.New("rate limiter unavailable")

type Result struct {
	Allowed    bool
	Limit      int32
	Remaining  int64
	RetryAfter time.Duration
	ResetAfter time.Duration
}

type Limiter interface {
	Allow(context.Context, string, int32) (Result, error)
}

type memoryBucket struct {
	tokens float64
	at     time.Time
}

type Memory struct {
	mu      sync.Mutex
	now     func() time.Time
	buckets map[string]memoryBucket
}

func NewMemory(now func() time.Time) *Memory {
	return &Memory{now: now, buckets: make(map[string]memoryBucket)}
}

func (m *Memory) Allow(_ context.Context, clientID string, rpm int32) (Result, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := m.now()
	capacity := float64(rpm)
	rate := capacity / float64(time.Minute)
	bucket, exists := m.buckets[clientID]
	if !exists {
		bucket = memoryBucket{tokens: capacity, at: now}
	}
	bucket.tokens = math.Min(capacity, bucket.tokens+float64(now.Sub(bucket.at))*rate)
	bucket.at = now
	result := Result{Limit: rpm}
	if bucket.tokens >= 1 {
		bucket.tokens--
		result.Allowed = true
	} else {
		result.RetryAfter = time.Duration(math.Ceil((1 - bucket.tokens) / rate))
	}
	result.Remaining = int64(math.Floor(bucket.tokens))
	result.ResetAfter = time.Duration(math.Ceil((capacity - bucket.tokens) / rate))
	m.buckets[clientID] = bucket
	return result, nil
}

type Valkey struct {
	client *redis.Client
}

func NewValkey(rawURL string) (*Valkey, error) {
	options, err := redis.ParseURL(rawURL)
	if err != nil {
		return nil, err
	}
	return &Valkey{client: redis.NewClient(options)}, nil
}

func (v *Valkey) Ping(ctx context.Context) error {
	started := time.Now()
	err := v.client.Ping(ctx).Err()
	outcome := "success"
	if err != nil {
		outcome = "error"
	}
	telemetry.RecordDependencyOperation(ctx, "valkey", "ping", outcome, time.Since(started))
	return err
}

func (v *Valkey) Close() error { return v.client.Close() }

func valkeyBucketKey(clientID string) string {
	return "tenancit:rate-limit:" + clientID
}

var tokenBucketScript = redis.NewScript(`
local tm = redis.call('TIME')
local now = tonumber(tm[1]) * 1000 + math.floor(tonumber(tm[2]) / 1000)
local capacity = tonumber(ARGV[1])
local rate = capacity / 60000
local values = redis.call('HMGET', KEYS[1], 'tokens', 'at')
local tokens = tonumber(values[1]) or capacity
local at = tonumber(values[2]) or now
tokens = math.min(capacity, tokens + math.max(0, now - at) * rate)
local allowed = 0
local retry = 0
if tokens >= 1 then
  tokens = tokens - 1
  allowed = 1
else
  retry = math.ceil((1 - tokens) / rate)
end
local reset = math.ceil((capacity - tokens) / rate)
redis.call('HSET', KEYS[1], 'tokens', tokens, 'at', now)
redis.call('PEXPIRE', KEYS[1], math.ceil((capacity / rate) * 2))
return {allowed, math.floor(tokens), retry, reset}
`)

func (v *Valkey) Allow(ctx context.Context, clientID string, rpm int32) (Result, error) {
	started := time.Now()
	values, err := tokenBucketScript.Run(ctx, v.client, []string{valkeyBucketKey(clientID)}, rpm).Slice()
	if err != nil || len(values) != 4 {
		telemetry.RecordDependencyOperation(ctx, "valkey", "query", "error", time.Since(started))
		return Result{}, ErrUnavailable
	}
	parsed := make([]int64, 4)
	for i, value := range values {
		switch typed := value.(type) {
		case int64:
			parsed[i] = typed
		case string:
			parsed[i], err = strconv.ParseInt(typed, 10, 64)
		default:
			err = errors.New("unexpected limiter response")
		}
		if err != nil {
			telemetry.RecordDependencyOperation(ctx, "valkey", "query", "error", time.Since(started))
			return Result{}, ErrUnavailable
		}
	}
	if parsed[1] < 0 || parsed[1] > int64(rpm) ||
		parsed[2] < 0 || parsed[2] > math.MaxInt64/int64(time.Millisecond) ||
		parsed[3] < 0 || parsed[3] > math.MaxInt64/int64(time.Millisecond) {
		telemetry.RecordDependencyOperation(ctx, "valkey", "query", "error", time.Since(started))
		return Result{}, ErrUnavailable
	}
	telemetry.RecordDependencyOperation(ctx, "valkey", "query", "success", time.Since(started))
	return Result{
		Allowed: parsed[0] == 1, Limit: rpm, Remaining: parsed[1],
		RetryAfter: time.Duration(parsed[2]) * time.Millisecond,
		ResetAfter: time.Duration(parsed[3]) * time.Millisecond,
	}, nil
}
