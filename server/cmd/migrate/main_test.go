package main

import "testing"

func TestRunRequiresMigrationDatabaseURL(t *testing.T) {
	if err := run(func(string) string { return "" }); err == nil {
		t.Fatal("migration command accepted an empty database URL")
	}
}
