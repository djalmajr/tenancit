package telemetry

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

type pgxTraceContextKey struct{}

type PGXTracer struct {
	now func() time.Time
}

func NewPGXTracer() *PGXTracer { return &PGXTracer{now: time.Now} }

func (t *PGXTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, _ pgx.TraceQueryStartData) context.Context {
	return context.WithValue(ctx, pgxTraceContextKey{}, t.now())
}

func (t *PGXTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	started, ok := ctx.Value(pgxTraceContextKey{}).(time.Time)
	if !ok {
		started = t.now()
	}
	outcome := "success"
	if data.Err != nil {
		outcome = "error"
	}
	RecordDependencyOperation(ctx, "postgres", "query", outcome, t.now().Sub(started))
}
