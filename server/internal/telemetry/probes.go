package telemetry

import (
	"context"
	"errors"
	"net/http"
)

func NewPingProbe(name string, critical bool, ping func(context.Context) error) Probe {
	return FuncProbe{Component: name, Required: critical, Run: ping}
}

func NewHTTPProbe(name, endpoint string, critical bool, client *http.Client) Probe {
	if client == nil {
		client = http.DefaultClient
	}
	return FuncProbe{
		Component: name,
		Required:  critical,
		Run: func(ctx context.Context) error {
			request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
			if err != nil {
				return err
			}
			response, err := client.Do(request)
			if err != nil {
				return err
			}
			defer response.Body.Close()
			if response.StatusCode < 200 || response.StatusCode >= 300 {
				return errors.New("dependency returned an unsuccessful status")
			}
			return nil
		},
	}
}
