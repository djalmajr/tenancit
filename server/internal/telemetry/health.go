package telemetry

import (
	"context"
	"encoding/json"
	"time"
)

type Status string

const (
	StatusHealthy     Status = "healthy"
	StatusDegraded    Status = "degraded"
	StatusUnavailable Status = "unavailable"
)

type Probe interface {
	Name() string
	Critical() bool
	Check(context.Context) error
}

type ComponentStatus struct {
	Name      string `json:"name"`
	Status    Status `json:"status"`
	LatencyMS int64  `json:"latency_ms"`
}

type ReadinessReport struct {
	Status     Status            `json:"status"`
	CheckedAt  time.Time         `json:"checked_at"`
	Components []ComponentStatus `json:"components"`
}

func EvaluateReadiness(ctx context.Context, probes []Probe, now func() time.Time) ReadinessReport {
	report := ReadinessReport{
		Status:     StatusHealthy,
		CheckedAt:  now().UTC(),
		Components: make([]ComponentStatus, 0, len(probes)),
	}
	for _, probe := range probes {
		started := time.Now()
		err := probe.Check(ctx)
		component := ComponentStatus{
			Name:      probe.Name(),
			Status:    StatusHealthy,
			LatencyMS: max(0, time.Since(started).Milliseconds()),
		}
		if err != nil {
			component.Status = StatusDegraded
			if probe.Critical() {
				component.Status = StatusUnavailable
				report.Status = StatusUnavailable
			} else if report.Status == StatusHealthy {
				report.Status = StatusDegraded
			}
		}
		report.Components = append(report.Components, component)
	}
	return report
}

func (r ReadinessReport) MarshalJSON() ([]byte, error) {
	type readinessAlias ReadinessReport
	return json.Marshal(readinessAlias(r))
}

type FuncProbe struct {
	Component string
	Required  bool
	Run       func(context.Context) error
}

func (p FuncProbe) Name() string                    { return p.Component }
func (p FuncProbe) Critical() bool                  { return p.Required }
func (p FuncProbe) Check(ctx context.Context) error { return p.Run(ctx) }
