package main

import (
	"bytes"
	"io"
	"os"
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

func TestRunRequiresStore(t *testing.T) {
	t.Parallel()
	if err := run(nil, io.Discard, nil); err == nil || !strings.Contains(err.Error(), "--store") {
		t.Fatalf("missing store dir not denied: %v", err)
	}
}
