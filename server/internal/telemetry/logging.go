package telemetry

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
)

type redactingHandler struct {
	inner slog.Handler
}

func NewRedactingJSONHandler(writer io.Writer, options *slog.HandlerOptions) slog.Handler {
	return redactingHandler{inner: slog.NewJSONHandler(writer, options)}
}

func (h redactingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h redactingHandler) Handle(ctx context.Context, record slog.Record) error {
	sanitized := slog.NewRecord(record.Time, record.Level, record.Message, record.PC)
	record.Attrs(func(attribute slog.Attr) bool {
		sanitized.AddAttrs(sanitizeLogAttribute(attribute))
		return true
	})
	return h.inner.Handle(ctx, sanitized)
}

func (h redactingHandler) WithAttrs(attributes []slog.Attr) slog.Handler {
	sanitized := make([]slog.Attr, 0, len(attributes))
	for _, attribute := range attributes {
		sanitized = append(sanitized, sanitizeLogAttribute(attribute))
	}
	return redactingHandler{inner: h.inner.WithAttrs(sanitized)}
}

func (h redactingHandler) WithGroup(name string) slog.Handler {
	return redactingHandler{inner: h.inner.WithGroup(name)}
}

func sanitizeLogAttribute(attribute slog.Attr) slog.Attr {
	attribute.Value = attribute.Value.Resolve()
	key := strings.ToLower(attribute.Key)
	if attribute.Value.Kind() == slog.KindAny {
		if err, ok := attribute.Value.Any().(error); ok {
			return slog.String("error_type", fmt.Sprintf("%T", err))
		}
	}
	if key == "err" || key == "error" {
		return slog.String("error_type", fmt.Sprintf("%T", attribute.Value.Any()))
	}
	for _, fragment := range []string{"authorization", "cookie", "token", "secret", "password", "body", "payload", "query", "sql", "dsn", "url", "hash", "nonce"} {
		if strings.Contains(key, fragment) {
			return slog.String(attribute.Key, "[REDACTED]")
		}
	}
	if attribute.Value.Kind() == slog.KindGroup {
		group := attribute.Value.Group()
		for index := range group {
			group[index] = sanitizeLogAttribute(group[index])
		}
		return slog.Group(attribute.Key, attrsToAny(group)...)
	}
	return attribute
}

func attrsToAny(attributes []slog.Attr) []any {
	values := make([]any, len(attributes))
	for index, attribute := range attributes {
		values[index] = attribute
	}
	return values
}
