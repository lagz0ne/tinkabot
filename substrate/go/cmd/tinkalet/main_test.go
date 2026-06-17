package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunPrintsHelp(t *testing.T) {
	t.Parallel()
	var out, errOut bytes.Buffer
	code := run([]string{"--help"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "usage: tinkalet ") {
		t.Fatalf("help missing usage:\n%s", out.String())
	}
	if errOut.Len() != 0 {
		t.Fatalf("stderr = %q", errOut.String())
	}
}

func TestRunPrintsVersion(t *testing.T) {
	t.Parallel()
	var out, errOut bytes.Buffer
	code := run([]string{"--version"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %q", code, errOut.String())
	}
	if got := strings.TrimSpace(out.String()); got != "tinkalet dev" {
		t.Fatalf("version = %q", got)
	}
}

func TestRunUsageError(t *testing.T) {
	t.Parallel()
	var out, errOut bytes.Buffer
	code := run([]string{"profile"}, &out, &errOut)
	if code != 2 {
		t.Fatalf("exit = %d", code)
	}
	if out.Len() != 0 {
		t.Fatalf("stdout = %q", out.String())
	}
	if !strings.HasPrefix(errOut.String(), "usage: tinkalet ") {
		t.Fatalf("stderr = %q", errOut.String())
	}
}
