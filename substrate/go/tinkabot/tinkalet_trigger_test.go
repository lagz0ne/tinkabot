package tinkabot

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestTinkaletTriggerClock(t *testing.T) {
	t.Parallel()
	store := t.TempDir()
	cfg := cfgFor(store)
	cfg.BundleDir = clockBundle
	app, err := boot(t, cfg)
	if err != nil {
		t.Fatal(err)
	}
	pauseClockSchedule(t, app)
	env := tinkaletEnv(t)
	mustTinkalet(t, env, "profile", "import", "local", "--store", store, "--name", "local")
	mustTinkalet(t, env, "profile", "use", "local")

	url := app.Posture().Shell.URL + "/projections/bundle.clock.state"
	_, before := waitFor200(t, url, 15*time.Second)
	first := unixOf(t, before)
	time.Sleep(1100 * time.Millisecond)

	code, out, errOut := runTinkalet(env, "trigger", "bundle.clock.tick", "--request-id", "req-tinkalet-live-1")
	if code != 0 || out != "profile local accepted bundle.clock.tick\n" || errOut != "" {
		t.Fatalf("trigger exit/stdout/stderr = %d/%q/%q", code, out, errOut)
	}
	waitClockAdvance(t, url, first, 10*time.Second)

	_, stable := waitFor200(t, url, 5*time.Second)
	code, out, errOut = runTinkalet(env, "trigger", "bundle.clock.tick", "--request-id", "req-tinkalet-live-1")
	if code != 0 || out != "profile local duplicate bundle.clock.tick\n" || errOut != "" {
		t.Fatalf("duplicate exit/stdout/stderr = %d/%q/%q", code, out, errOut)
	}
	_, afterDup := waitFor200(t, url, 5*time.Second)
	if string(afterDup) != string(stable) {
		t.Fatalf("duplicate reran clock:\nbefore %s\nafter %s", stable, afterDup)
	}

	code, out, errOut = runTinkalet(env, "trigger", "bundle.clock.tick", "--request-id", "req-tinkalet-live-json", "--json")
	if code != 0 || errOut != "" {
		t.Fatalf("json trigger exit/stderr = %d/%q", code, errOut)
	}
	var doc struct {
		Profile string `json:"profile"`
		Intent  string `json:"intent"`
		Status  string `json:"status"`
		Reason  string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("trigger json: %v\n%s", err, out)
	}
	if doc.Profile != "local" || doc.Intent != "bundle.clock.tick" || doc.Status != "accepted" || doc.Reason != "" {
		t.Fatalf("trigger json drift: %#v", doc)
	}
	if strings.Contains(out, "tb.bundle.clock.tick") || strings.Contains(out, string(mustReadFile(t, app.CredsFile(RoleCaller)))) {
		t.Fatalf("json trigger leaked substrate or credential: %q", out)
	}
}

func mustTinkalet(t *testing.T, env []string, args ...string) {
	t.Helper()
	code, out, errOut := runTinkalet(env, args...)
	if code != 0 {
		t.Fatalf("%v exit/stdout/stderr = %d/%q/%q", args, code, out, errOut)
	}
}

func pauseClockSchedule(t *testing.T, app *App) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	nc, err := app.Runtime().ConnectCreds(ctx, app.Creds(RoleCaller).File)
	if err != nil {
		t.Fatal(err)
	}
	defer nc.Close()
	js, err := nc.JetStream()
	if err != nil {
		t.Fatal(err)
	}
	kv, err := js.KeyValue("config_bucket")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := kv.Put("bundle.clock.tick.every", []byte("off")); err != nil {
		t.Fatal(err)
	}
}

func waitClockAdvance(t *testing.T, url string, past int64, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		_, p := waitFor200(t, url, 5*time.Second)
		if unixOf(t, p) > past || seqOf(t, p) > past {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("clock projection did not advance past %d within %s", past, timeout)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func seqOf(t *testing.T, projection []byte) int64 {
	t.Helper()
	var p struct {
		Sequence int64 `json:"sequence"`
	}
	if err := json.Unmarshal(projection, &p); err != nil {
		t.Fatalf("projection is not the stored record: %v: %s", err, projection)
	}
	return p.Sequence
}
