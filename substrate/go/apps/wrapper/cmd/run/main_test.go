package main

import (
	"strings"
	"testing"
)

func TestRunArgValidation(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		args []string
	}{
		{"no_args", nil},
		{"missing_session", []string{"--nats", "nats://127.0.0.1:1", "--creds", "/tmp/x.creds", "--", "true"}},
		{"missing_agent_command", []string{"--nats", "nats://127.0.0.1:1", "--creds", "/tmp/x.creds", "--session", "s1"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := run(tc.args)
			if err == nil || !strings.Contains(err.Error(), "usage:") {
				t.Fatalf("want usage error, got %v", err)
			}
		})
	}
}

func TestRunBadCreds(t *testing.T) {
	t.Parallel()
	err := run([]string{"--nats", "nats://127.0.0.1:1", "--creds", "/nonexistent.creds", "--session", "s1", "--", "true"})
	if err == nil {
		t.Fatal("want connect/creds failure, got nil")
	}
}
