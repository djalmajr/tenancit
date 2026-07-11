package store

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

const (
	PostgresForeignKeyViolation = "23503"
	PostgresUniqueViolation     = "23505"
)

// IsPostgresCode recognizes PostgreSQL errors even when service layers wrap
// them. Callers still decide what a given constraint means for their domain.
func IsPostgresCode(err error, code string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == code
}
