package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunRejectsInvalidRelease(t *testing.T) {
	var stderr bytes.Buffer
	if code := run([]string{"--release=unknown"}, func(string) (string, bool) { return "", false }, &bytes.Buffer{}, &stderr); code != 2 {
		t.Fatalf("run() code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "keepers-v1") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunRequiresDatabaseURL(t *testing.T) {
	var stderr bytes.Buffer
	if code := run([]string{"--release=keepers-v1"}, func(string) (string, bool) { return "", false }, &bytes.Buffer{}, &stderr); code != 1 {
		t.Fatalf("run() code = %d, want 1", code)
	}
	if stderr.String() != "DATABASE_URL is required\n" {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunDoesNotEchoInvalidDatabaseURL(t *testing.T) {
	const secretURL = "postgres://secret:secret@ bad-host/database"
	var stderr bytes.Buffer
	if code := run([]string{"--release=keepers-v1"}, func(string) (string, bool) { return secretURL, true }, &bytes.Buffer{}, &stderr); code != 1 {
		t.Fatalf("run() code = %d, want 1", code)
	}
	if strings.Contains(stderr.String(), secretURL) {
		t.Fatalf("stderr disclosed DATABASE_URL: %q", stderr.String())
	}
}
