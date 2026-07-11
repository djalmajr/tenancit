package main

import (
	"net/http"
	"testing"
	"time"
)

func TestNewHTTPServerSetsDefensiveTimeouts(t *testing.T) {
	handler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	srv := newHTTPServer(":0", handler)

	if srv.ReadHeaderTimeout != 5*time.Second {
		t.Fatalf("ReadHeaderTimeout = %s", srv.ReadHeaderTimeout)
	}
	if srv.ReadTimeout != 15*time.Second {
		t.Fatalf("ReadTimeout = %s", srv.ReadTimeout)
	}
	if srv.WriteTimeout != 30*time.Second {
		t.Fatalf("WriteTimeout = %s", srv.WriteTimeout)
	}
	if srv.IdleTimeout != 60*time.Second {
		t.Fatalf("IdleTimeout = %s", srv.IdleTimeout)
	}
}

func TestLoadOperationsCredentialRequiresCompleteStrongPair(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		values  map[string]string
		wantErr bool
	}{
		{name: "disabled", values: map[string]string{}},
		{name: "complete", values: map[string]string{
			"TENANCIT_OPERATIONS_REPORT_TOKEN":              "operations-token-with-at-least-32-characters",
			"TENANCIT_OPERATIONS_REPORT_CREDENTIAL_VERSION": "agent-v1",
		}},
		{name: "token only", values: map[string]string{"TENANCIT_OPERATIONS_REPORT_TOKEN": "operations-token-with-at-least-32-characters"}, wantErr: true},
		{name: "weak token", values: map[string]string{
			"TENANCIT_OPERATIONS_REPORT_TOKEN":              "too-short",
			"TENANCIT_OPERATIONS_REPORT_CREDENTIAL_VERSION": "agent-v1",
		}, wantErr: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			token, version, err := loadOperationsCredential(func(key string) string { return testCase.values[key] })
			if (err != nil) != testCase.wantErr {
				t.Fatalf("token=%q version=%q err=%v", token, version, err)
			}
		})
	}
}
