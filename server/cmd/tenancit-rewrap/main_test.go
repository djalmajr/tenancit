package main

import (
	"errors"
	"testing"
	"time"

	"github.com/djalmajr/tenancit/server/internal/rewrap"
)

func TestParseOptionsRequiresOfflineDSNAndReporterForWrites(t *testing.T) {
	getenv := func(name string) string {
		values := map[string]string{"TENANCIT_REWRAP_DATABASE_URL": "postgres://rewrap@db/tenancit"}
		return values[name]
	}
	if _, err := parseOptions(getenv, []string{"--target-version", "2", "--confirm-write", "--job-id", "00000000-0000-4000-8000-000000000007"}); err == nil {
		t.Fatal("write mode accepted missing reporter")
	}
	dry, err := parseOptions(getenv, []string{"--target-version", "2", "--dry-run", "--batch-size", "17", "--max-duration", "2m"})
	if err != nil {
		t.Fatal(err)
	}
	if !dry.DryRun || dry.TargetVersion != 2 || dry.BatchSize != 17 || dry.MaxDuration != 2*time.Minute {
		t.Fatalf("parsed dry-run = %+v", dry)
	}
}

func TestParseOptionsNeverAcceptsKeyMaterialAsArguments(t *testing.T) {
	getenv := func(name string) string {
		if name == "TENANCIT_REWRAP_DATABASE_URL" {
			return "postgres://rewrap@db/tenancit"
		}
		return ""
	}
	for _, argument := range []string{"--key", "--old-key", "--aes-key"} {
		if _, err := parseOptions(getenv, []string{"--dry-run", "--target-version", "2", argument, "secret"}); err == nil {
			t.Fatalf("accepted key argument %q", argument)
		}
	}
}

func TestPublicReasonIsBounded(t *testing.T) {
	if got := publicReason(rewrap.ErrAuthentication); got != "authentication_failed" {
		t.Fatalf("reason=%q", got)
	}
	if got := publicReason(errors.New("plaintext-canary")); got != "internal_error" {
		t.Fatalf("unbounded reason=%q", got)
	}
}
