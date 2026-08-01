package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/thaodangspace/tidekeepers-server/content"
)

func TestRunRejectsMissingVersions(t *testing.T) {
	cases := [][]string{
		{},
		{"--source-version=1"},
		{"--target-version=3"},
		{"--source-version=0", "--target-version=3"},
	}
	for _, args := range cases {
		var stderr bytes.Buffer
		if code := run(args, func(string) (string, bool) { return "", false }, &bytes.Buffer{}, &stderr); code != 2 {
			t.Fatalf("run(%v) code = %d, want 2", args, code)
		}
		if !strings.Contains(stderr.String(), "source-version") {
			t.Fatalf("run(%v) stderr = %q", args, stderr.String())
		}
	}
}

func TestRunRequiresDatabaseURL(t *testing.T) {
	var stderr bytes.Buffer
	if code := run([]string{"--source-version=1", "--target-version=3"}, func(string) (string, bool) { return "", false }, &bytes.Buffer{}, &stderr); code != 1 {
		t.Fatalf("run() code = %d, want 1", code)
	}
	if stderr.String() != "DATABASE_URL is required\n" {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunDoesNotEchoInvalidDatabaseURL(t *testing.T) {
	const secretURL = "postgres://secret:secret@ bad-host/database"
	var stderr bytes.Buffer
	if code := run([]string{"--source-version=1", "--target-version=3"}, func(string) (string, bool) { return secretURL, true }, &bytes.Buffer{}, &stderr); code != 1 {
		t.Fatalf("run() code = %d, want 1", code)
	}
	if strings.Contains(stderr.String(), secretURL) {
		t.Fatalf("stderr disclosed DATABASE_URL: %q", stderr.String())
	}
}

func TestWriteReportDryRun(t *testing.T) {
	var output bytes.Buffer
	writeReport(&output, content.CutoverResult{
		SourceVersion: 1, TargetVersion: 3,
		FingerprintLabels: []string{"migration-seeded-development"},
		SourceCatalog:     "v1",
		ChangedTideCount:  2, TideCount: 2, StateCount: 3,
	})
	text := output.String()
	for _, want := range []string{
		"source version: 1",
		"target version: 3",
		"fingerprint: migration-seeded-development",
		"source catalog: v1",
		"tides: 2",
		"states: 3",
		"dry-run: no changes applied",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("report missing %q:\n%s", want, text)
		}
	}
}

func TestWriteReportIdempotent(t *testing.T) {
	var output bytes.Buffer
	writeReport(&output, content.CutoverResult{
		SourceVersion: 1, TargetVersion: 3,
		Idempotent: true,
	})
	text := output.String()
	if !strings.Contains(text, "already complete: true") || !strings.Contains(text, "no changes required") {
		t.Fatalf("report = %q", text)
	}
}
