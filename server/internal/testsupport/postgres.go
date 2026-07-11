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
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/djalmajr/tenancit/server/internal/migration"
	"github.com/jackc/pgx/v5"
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
	if baseDSN := os.Getenv("TENANCIT_TEST_DATABASE_URL"); baseDSN != "" {
		return newIsolatedDatabase(t, ctx, baseDSN)
	}

	container, err := tcpg.Run(ctx, "postgres:16-alpine",
		tcpg.WithDatabase("tenancit_test"),
		tcpg.WithUsername("postgres"),
		tcpg.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForListeningPort("5432/tcp").WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		if os.Getenv("REQUIRE_DB_TESTS") == "1" {
			t.Fatalf("testcontainers/docker required but unavailable: %v", err)
		}
		t.Skipf("testcontainers/docker unavailable: %v (set REQUIRE_DB_TESTS=1 to fail instead)", err)
	}
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}

	if err := migration.Up(dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Close)

	return pool
}

// newIsolatedDatabase creates one database per test on a shared PostgreSQL
// service. CI uses this path to preserve test isolation without starting one
// Docker container per package/test. The caller-provided DSN must point to an
// administrative database and role allowed to CREATE/DROP DATABASE.
func newIsolatedDatabase(t *testing.T, ctx context.Context, baseDSN string) *pgxpool.Pool {
	t.Helper()

	adminConfig, err := pgx.ParseConfig(baseDSN)
	if err != nil {
		t.Fatalf("parse TENANCIT_TEST_DATABASE_URL: %v", err)
	}
	admin, err := pgx.ConnectConfig(ctx, adminConfig)
	if err != nil {
		t.Fatalf("connect shared test postgres: %v", err)
	}

	databaseName := randomDatabaseName(t)
	identifier := pgx.Identifier{databaseName}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+identifier); err != nil {
		_ = admin.Close(ctx)
		t.Fatalf("create isolated test database: %v", err)
	}

	t.Cleanup(func() {
		// Pools are registered after this cleanup, so testing runs pool.Close
		// first (LIFO). Terminating leftovers makes cleanup deterministic after
		// a failed assertion or leaked connection.
		_, _ = admin.Exec(ctx,
			"SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1 AND pid <> pg_backend_pid()",
			databaseName,
		)
		_, _ = admin.Exec(ctx, "DROP DATABASE IF EXISTS "+identifier)
		_ = admin.Close(ctx)
	})

	testDSN, err := databaseDSN(baseDSN, databaseName)
	if err != nil {
		t.Fatalf("build isolated test database DSN: %v", err)
	}
	testConfig, err := pgxpool.ParseConfig(testDSN)
	if err != nil {
		t.Fatalf("parse shared test postgres pool config: %v", err)
	}
	if err := migration.Up(testDSN); err != nil {
		t.Fatalf("migrate isolated test database: %v", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, testConfig)
	if err != nil {
		t.Fatalf("open isolated test database: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func databaseDSN(baseDSN, databaseName string) (string, error) {
	u, err := url.Parse(baseDSN)
	if err != nil {
		return "", err
	}
	if u.Scheme != "postgres" && u.Scheme != "postgresql" {
		return "", fmt.Errorf("TENANCIT_TEST_DATABASE_URL must use postgres:// or postgresql://")
	}
	u.Path = "/" + databaseName
	return u.String(), nil
}

func randomDatabaseName(t *testing.T) string {
	t.Helper()
	var suffix [8]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		t.Fatalf("generate isolated database name: %v", err)
	}
	return fmt.Sprintf("tenancit_test_%s", hex.EncodeToString(suffix[:]))
}
