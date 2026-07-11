package rewrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var reportIdentifier = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)

type HTTPReporter struct {
	baseURL string
	token   string
	source  string
	client  *http.Client
	now     func() time.Time
}

func NewHTTPReporter(baseURL, token, source string, allowInsecure bool, client *http.Client) (*HTTPReporter, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
		return nil, errors.New("operations report base URL must be an origin without credentials")
	}
	if parsed.Scheme != "https" && !(allowInsecure && parsed.Scheme == "http") {
		return nil, errors.New("operations report base URL must use HTTPS")
	}
	if len(strings.TrimSpace(token)) < 32 {
		return nil, errors.New("operations report token must contain at least 32 characters")
	}
	if !reportIdentifier.MatchString(source) {
		return nil, errors.New("operations report source is invalid")
	}
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	configuredClient := *client
	configuredClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	parsed.Path = ""
	return &HTTPReporter{baseURL: strings.TrimSuffix(parsed.String(), "/"), token: strings.TrimSpace(token), source: source, client: &configuredClient, now: time.Now}, nil
}

func (r *HTTPReporter) Report(ctx context.Context, summary Summary, status string) error {
	if status != "healthy" && status != "failed" && status != "degraded" {
		return errors.New("invalid rewrap report status")
	}
	now := r.now().UTC()
	body, err := json.Marshal(map[string]any{
		"kind": "rewrap", "source": r.source, "status": status,
		"occurred_at": now, "fresh_for_seconds": 86400,
	})
	if err != nil {
		return fmt.Errorf("encode rewrap report: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, r.baseURL+"/v1/operations/reports", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build rewrap report: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+r.token)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "rewrap-"+summary.JobID+"-"+status)
	response, err := r.client.Do(request)
	if err != nil {
		return errors.New("send rewrap report")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusCreated && response.StatusCode != http.StatusOK {
		return fmt.Errorf("rewrap report rejected with HTTP %d", response.StatusCode)
	}
	return nil
}
