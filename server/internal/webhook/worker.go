package webhook

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"math"
	"math/rand/v2"
	"net/http"
	"strconv"
	"time"

	appcrypto "github.com/djalmajr/tenancit/server/internal/crypto"
)

const maxAttempts = 8

type Worker struct {
	Store          *DeliveryStore
	Cryptor        *appcrypto.Cryptor
	Resolver       Resolver
	Now            func() time.Time
	Batch          int32
	Lease, Timeout time.Duration
	Jitter         func() float64
}

func NewWorker(store *DeliveryStore, cryptor *appcrypto.Cryptor) *Worker {
	return &Worker{Store: store, Cryptor: cryptor, Now: time.Now, Batch: 20, Lease: 30 * time.Second, Timeout: 10 * time.Second, Jitter: rand.Float64}
}

func (w *Worker) Run(ctx context.Context, interval time.Duration) {
	w.run(ctx)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.run(ctx)
		}
	}
}

func (w *Worker) run(ctx context.Context) {
	deliveries, err := w.Store.Claim(ctx, w.Now(), w.Lease, w.Batch)
	if err != nil {
		slog.Error("claim webhook deliveries", "error_type", fmt.Sprintf("%T", err))
		return
	}
	for _, delivery := range deliveries {
		if err := w.deliver(ctx, delivery); err != nil {
			slog.Error("process webhook delivery", "delivery_id", delivery.ID, "error_type", fmt.Sprintf("%T", err))
		}
	}
}

func (w *Worker) deliver(ctx context.Context, delivery Delivery) error {
	rawURL, err := w.Cryptor.Decrypt(appcrypto.Encrypted{Cipher: delivery.URLCipher, Nonce: delivery.URLNonce, KeyVersion: int(delivery.URLKeyVersion)})
	if err != nil {
		return w.fail(ctx, delivery, nil, "decrypt_failed", false)
	}
	secret, err := w.Cryptor.Decrypt(appcrypto.Encrypted{Cipher: delivery.SecretCipher, Nonce: delivery.SecretNonce, KeyVersion: int(delivery.SecretKeyVersion)})
	if err != nil {
		return w.fail(ctx, delivery, nil, "decrypt_failed", false)
	}
	resolved, err := ResolveEndpoint(ctx, rawURL, delivery.AllowLoopbackHTTP, w.Resolver)
	if err != nil {
		return w.fail(ctx, delivery, nil, "endpoint_rejected", true)
	}
	body, err := Render(delivery.Format, delivery.Event)
	if err != nil {
		return w.fail(ctx, delivery, nil, "render_failed", true)
	}
	timestamp := strconv.FormatInt(w.Now().UTC().Unix(), 10)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, resolved.URL.String(), bytes.NewReader(body))
	if err != nil {
		return w.fail(ctx, delivery, nil, "request_failed", false)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Tenancit-Webhook-Id", delivery.Event.ID)
	request.Header.Set("Tenancit-Webhook-Timestamp", timestamp)
	request.Header.Set("Tenancit-Webhook-Signature", Signature([]byte(secret), timestamp, body))
	response, err := ClientFor(resolved, w.Timeout).Do(request)
	if err != nil {
		return w.fail(ctx, delivery, nil, "network_error", false)
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 64<<10))
	_ = response.Body.Close()
	if response.StatusCode >= 200 && response.StatusCode < 300 {
		return w.Store.MarkDelivered(ctx, delivery, w.Now(), response.StatusCode)
	}
	status := response.StatusCode
	permanent := status >= 400 && status < 500 && status != http.StatusRequestTimeout && status != http.StatusTooManyRequests
	return w.fail(ctx, delivery, &status, "http_"+strconv.Itoa(status), permanent)
}

func (w *Worker) fail(ctx context.Context, d Delivery, status *int, code string, permanent bool) error {
	now := w.Now().UTC()
	dead := permanent || d.AttemptCount >= maxAttempts
	delay := time.Duration(math.Pow(2, math.Min(float64(d.AttemptCount-1), 9))) * time.Second
	delay = time.Duration(float64(delay) * (0.8 + 0.4*w.Jitter()))
	next := now.Add(delay)
	var openUntil *time.Time
	if d.ConsecutiveFailures+1 >= 5 && !dead {
		value := now.Add(time.Minute)
		openUntil = &value
		if next.Before(value) {
			next = value
		}
	}
	return w.Store.MarkFailed(ctx, d, now, next, status, code, dead, openUntil)
}
