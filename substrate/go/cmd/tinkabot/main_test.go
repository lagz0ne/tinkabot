package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRunStartsPrintsPostureAndStopsOnSignal boots the real assembly (real
// embedded NATS, isolated store dir) through the CLI entry and stops it on a
// delivered signal.
func TestRunStartsPrintsPostureAndStopsOnSignal(t *testing.T) {
	t.Parallel()
	sig := make(chan os.Signal, 1)
	sig <- os.Interrupt
	var out bytes.Buffer
	if err := run([]string{"--store", t.TempDir(), "--shell", "127.0.0.1:0"}, &out, sig); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"nats   nats://127.0.0.1", "shell  http://127.0.0.1", "caller.creds", "observer.creds", "author.creds"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("posture print missing %q:\n%s", want, out.String())
		}
	}
}

func TestRunAdmitsParticipantsFromStartupFlag(t *testing.T) {
	t.Parallel()
	store := t.TempDir()
	sig := make(chan os.Signal, 1)
	sig <- os.Interrupt
	var out bytes.Buffer
	if err := run([]string{
		"--store", store,
		"--shell", "127.0.0.1:0",
		"--participant", "demo:alice",
		"--participant", "demo:bob",
	}, &out, sig); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"participant demo alice ",
		"participant demo bob ",
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("participant print missing %q:\n%s", want, out.String())
		}
	}
	for _, path := range []string{
		filepath.Join(store, "participants", "demo", "alice", "local-profile.json"),
		filepath.Join(store, "participants", "demo", "bob", "local-profile.json"),
		filepath.Join(store, "participants", "demo", "alice", "participant.creds"),
		filepath.Join(store, "participants", "demo", "bob", "participant.creds"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("participant startup file missing %s: %v", path, err)
		}
	}
}

func TestRunAdmitsWatchersFromStartupFlag(t *testing.T) {
	t.Parallel()
	store := t.TempDir()
	sig := make(chan os.Signal, 1)
	sig <- os.Interrupt
	var out bytes.Buffer
	if err := run([]string{
		"--store", store,
		"--shell", "127.0.0.1:0",
		"--watcher", "llm:item:artifacts.artifact-browser.results.choice",
	}, &out, sig); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "watcher llm item artifacts.artifact-browser.results.choice ") {
		t.Fatalf("watcher print missing:\n%s", out.String())
	}
	for _, path := range []string{
		filepath.Join(store, "watchers", "llm", "local-profile.json"),
		filepath.Join(store, "watchers", "llm", "watcher.creds"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("watcher startup file missing %s: %v", path, err)
		}
	}
}

func TestRunRequiresStore(t *testing.T) {
	t.Parallel()
	if err := run(nil, io.Discard, nil); err == nil || !strings.Contains(err.Error(), "--store") {
		t.Fatalf("missing store dir not denied: %v", err)
	}
}

func TestRunPrintsVersion(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer
	if err := run([]string{"--version"}, &out, nil); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(out.String()); got != "tinkabot dev" {
		t.Fatalf("version = %q", got)
	}
}
