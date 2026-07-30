package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunVersionDoesNotRequireConfiguration(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if code := run([]string{"--version"}, &stdout, &stderr); code != 0 {
		t.Fatalf("run() code = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "tidekeepers-api version=") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunRejectsUnknownFlag(t *testing.T) {
	if code := run([]string{"--unknown"}, &bytes.Buffer{}, &bytes.Buffer{}); code != 2 {
		t.Fatalf("run() code = %d, want 2", code)
	}
}
