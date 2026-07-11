package auditops

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var exportCSVHeader = []string{
	"occurred_at", "id", "request_id", "actor_kind", "actor_issuer",
	"actor_subject", "actor_label", "action", "target_type", "target_id",
	"result", "http_method", "route_template", "http_status", "error_code", "metadata",
}

type exportRecord struct {
	OccurredAt    string          `json:"occurred_at"`
	ID            string          `json:"id"`
	RequestID     string          `json:"request_id"`
	ActorKind     string          `json:"actor_kind"`
	ActorIssuer   string          `json:"actor_issuer,omitempty"`
	ActorSubject  string          `json:"actor_subject"`
	ActorLabel    string          `json:"actor_label,omitempty"`
	Action        string          `json:"action"`
	TargetType    string          `json:"target_type"`
	TargetID      string          `json:"target_id"`
	Result        string          `json:"result"`
	HTTPMethod    string          `json:"http_method"`
	RouteTemplate string          `json:"route_template"`
	HTTPStatus    int16           `json:"http_status"`
	ErrorCode     string          `json:"error_code,omitempty"`
	Metadata      json.RawMessage `json:"metadata"`
}

func (r exportRecord) csv() []string {
	return []string{
		r.OccurredAt, r.ID, r.RequestID, r.ActorKind, r.ActorIssuer,
		r.ActorSubject, r.ActorLabel, r.Action, r.TargetType, r.TargetID,
		r.Result, r.HTTPMethod, r.RouteTemplate, fmt.Sprint(r.HTTPStatus),
		r.ErrorCode, string(r.Metadata),
	}
}

func (r *ExportRepository) generate(ctx context.Context, filter ExportFilter, format string) ([]byte, int64, error) {
	rows, err := r.pool.Query(ctx, exportSelect(false)+` ORDER BY occurred_at,id LIMIT 100001`, exportArgs(filter)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	if format == "csv" {
		if err := writer.Write(exportCSVHeader); err != nil {
			return nil, 0, err
		}
	}
	var count int64
	for rows.Next() {
		record, err := scanExportRecord(rows)
		if err != nil {
			return nil, 0, err
		}
		count++
		if count > MaxExportRows {
			return nil, 0, ErrInvalidExport
		}
		if format == "csv" {
			err = writer.Write(record.csv())
		} else {
			var line []byte
			line, err = json.Marshal(record)
			if err == nil {
				buffer.Write(line)
				buffer.WriteByte('\n')
			}
		}
		if err != nil {
			return nil, 0, err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, 0, err
	}
	return buffer.Bytes(), count, rows.Err()
}

type rowScanner interface{ Scan(...any) error }

func scanExportRecord(row rowScanner) (exportRecord, error) {
	var record exportRecord
	var occurred time.Time
	var id uuid.UUID
	var issuer, label, errorCode *string
	err := row.Scan(
		&occurred, &id, &record.RequestID, &record.ActorKind, &issuer,
		&record.ActorSubject, &label, &record.Action, &record.TargetType,
		&record.TargetID, &record.Result, &record.HTTPMethod, &record.RouteTemplate,
		&record.HTTPStatus, &errorCode, &record.Metadata,
	)
	record.OccurredAt = occurred.UTC().Format(time.RFC3339Nano)
	record.ID = id.String()
	record.ActorIssuer = stringValue(issuer)
	record.ActorLabel = stringValue(label)
	record.ErrorCode = stringValue(errorCode)
	return record, err
}

func exportSelect(count bool) string {
	selectPart := `SELECT
		occurred_at,id,request_id,actor_kind,actor_issuer,actor_subject,actor_label,
		action,target_type,target_id,result,http_method,route_template,http_status,error_code,metadata`
	if count {
		selectPart = `SELECT count(*)`
	}
	return selectPart + ` FROM admin_audit_events
		WHERE occurred_at >= $1 AND occurred_at < $2
		AND ($3='' OR actor_kind=$3) AND ($4='' OR actor_subject=$4)
		AND ($5='' OR action=$5) AND ($6='' OR target_type=$6)
		AND ($7='' OR target_id=$7) AND ($8='' OR request_id=$8)
		AND ($9='' OR result=$9)`
}

func exportArgs(f ExportFilter) []any {
	return []any{f.From, f.To, f.ActorKind, f.ActorSubject, f.Action, f.TargetType, f.TargetID, f.RequestID, f.Result}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
