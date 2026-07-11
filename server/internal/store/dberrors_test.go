package store

import (
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestIsPostgresCodeRecognizesWrappedErrors(t *testing.T) {
	err := fmt.Errorf("persist resource: %w", &pgconn.PgError{Code: PostgresUniqueViolation})
	if !IsPostgresCode(err, PostgresUniqueViolation) {
		t.Fatal("wrapped unique violation was not recognized")
	}
	if IsPostgresCode(err, PostgresForeignKeyViolation) {
		t.Fatal("unique violation matched foreign-key code")
	}
}
