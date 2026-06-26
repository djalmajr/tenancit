// Package testsupport spins up an ephemeral Postgres via testcontainers for
// integration tests, applies migrations, and returns a ready pgx pool.
//
// Usage in a test:
//
//	pool := testsupport.NewDB(t)
//	q := db.New(pool)
//
// Skips automatically (t.Skip) if Docker is unavailable, so unit-only runs on
// machines without Docker still pass.
package testsupport

import (
	"context"
	"testing"
	"time"

	"github.com/djalmajr/konvario/server/internal/store"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpg "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// NewDB starts a throwaway Postgres, runs migrations, and returns a pool.
// The container and pool are torn down via t.Cleanup.
func NewDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	container, err := tcpg.Run(ctx, "postgres:16-alpine",
		tcpg.WithDatabase("konvario_test"),
		tcpg.WithUsername("postgres"),
		tcpg.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForListeningPort("5432/tcp").WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Skipf("testcontainers/docker unavailable: %v", err)
	}
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}

	if err := store.Migrate(dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Close)

	return pool
}
