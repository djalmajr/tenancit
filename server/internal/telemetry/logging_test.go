package telemetry

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
)

func TestRedactingHandlerRemovesSecretsAndErrorMessages(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(NewRedactingJSONHandler(&output, nil))
	logger.ErrorContext(context.Background(), "request failed",
		"request_id", "request-safe",
		"token", "raw-token",
		"url", "https://user:secret@internal/path",
		"err", errors.New("postgres://admin:password@db/private"),
	)

	encoded := output.String()
	for _, expected := range []string{"request-safe", "error_type"} {
		if !strings.Contains(encoded, expected) {
			t.Fatalf("log missing %q: %s", expected, encoded)
		}
	}
	for _, forbidden := range []string{"raw-token", "user:secret", "admin:password", "postgres://"} {
		if strings.Contains(encoded, forbidden) {
			t.Fatalf("log leaked %q: %s", forbidden, encoded)
		}
	}
}
