package ratelimit

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestValkeyLimiterIsGlobalAcrossInstancesAndClientRestart(t *testing.T) {
	ctx := context.Background()
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image: "valkey/valkey:8-alpine", ExposedPorts: []string{"6379/tcp"},
			WaitingFor: wait.ForLog("Ready to accept connections").WithStartupTimeout(60 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		if os.Getenv("REQUIRE_DB_TESTS") == "1" {
			t.Fatalf("start Valkey: %v", err)
		}
		t.Skipf("Docker unavailable: %v", err)
	}
	t.Cleanup(func() { _ = container.Terminate(ctx) })
	host, err := container.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}
	port, err := container.MappedPort(ctx, "6379/tcp")
	if err != nil {
		t.Fatal(err)
	}
	url := fmt.Sprintf("redis://%s:%s/0", host, port.Port())
	a, err := NewValkey(url)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	b, err := NewValkey(url)
	if err != nil {
		t.Fatal(err)
	}
	clientID := uuid.NewString()
	for index, limiter := range []*Valkey{a, b} {
		result, err := limiter.Allow(ctx, clientID, 2)
		if err != nil || !result.Allowed {
			t.Fatalf("request %d = %+v, %v", index+1, result, err)
		}
	}
	if result, err := a.Allow(ctx, clientID, 2); err != nil || result.Allowed {
		t.Fatalf("combined third request = %+v, %v, want limited", result, err)
	}
	_ = b.Close()
	b, err = NewValkey(url)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	if result, err := b.Allow(ctx, clientID, 2); err != nil || result.Allowed {
		t.Fatalf("request after client restart = %+v, %v, want limited", result, err)
	}
}
