package tinkabot

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lagz0ne/tinkabot/substrate/go/core"
	"github.com/lagz0ne/tinkabot/substrate/go/embednats"
)

func TestTinkaletSchedules(t *testing.T) {
	t.Parallel()
	store := t.TempDir()
	app, err := boot(t, cfgFor(store))
	if err != nil {
		t.Fatal(err)
	}
	env := tinkaletEnv(t)
	importScheduleProfile(t, app, env, "sched")
	mustTinkalet(t, env, "profile", "use", "sched")
	code, out, errOut := runTinkalet(env, "item", "create", "deploy/789/tick", "--value", `{"client":"cannot-write"}`)
	if code != 1 || out != "" {
		t.Fatalf("schedule profile could write item directly: exit/stdout/stderr = %d/%q/%q", code, out, errOut)
	}
	assertSchedulePrivate(t, errOut, app)

	code, out, errOut = runTinkalet(env, "schedule", "set", "deploytick", "--every", "200ms", "--write", "deploy/789/tick", "--value", `{"kind":"tick"}`, "--json")
	if code != 0 || errOut != "" {
		t.Fatalf("schedule set exit/stderr = %d/%q", code, errOut)
	}
	assertSchedulePrivate(t, out, app)
	var sched scheduleDoc
	if err := json.Unmarshal([]byte(out), &sched); err != nil {
		t.Fatalf("schedule json: %v\n%s", err, out)
	}
	if sched.Name != "deploytick" || sched.Status != "active" || sched.EveryMs != 200 || sched.WriteItem != "deploy/789/tick" || string(sched.Value) != `{"kind":"tick"}` {
		t.Fatalf("schedule drift: %#v", sched)
	}

	first := waitScheduleItem(t, env, 0, 5*time.Second)
	if first.Value.Schedule != "deploytick" || first.Value.Sequence == 0 || string(first.Value.Value) != `{"kind":"tick"}` {
		t.Fatalf("scheduled item drift: %#v", first)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := app.Stop(ctx); err != nil {
		cancel()
		t.Fatal(err)
	}
	cancel()
	time.Sleep(350 * time.Millisecond)

	app2, err := boot(t, cfgFor(store))
	if err != nil {
		t.Fatal(err)
	}
	importScheduleProfile(t, app2, env, "sched")
	next := waitScheduleItem(t, env, first.Value.Sequence, 5*time.Second)
	if next.Value.Sequence <= first.Value.Sequence || next.Revision <= first.Revision {
		t.Fatalf("restart did not catch up: before %#v after %#v", first, next)
	}

	code, out, errOut = runTinkalet(env, "schedule", "off", "deploytick", "--json")
	if code != 0 || errOut != "" {
		t.Fatalf("schedule off exit/stderr = %d/%q", code, errOut)
	}
	assertSchedulePrivate(t, out, app2)
	off := readScheduleItem(t, env)
	time.Sleep(700 * time.Millisecond)
	after := readScheduleItem(t, env)
	if after.Value.Sequence != off.Value.Sequence || after.Revision != off.Revision {
		t.Fatalf("schedule off did not stop ticks: off %#v after %#v", off, after)
	}

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	if err := app2.Stop(ctx); err != nil {
		cancel()
		t.Fatal(err)
	}
	cancel()
	app3, err := boot(t, cfgFor(store))
	if err != nil {
		t.Fatal(err)
	}
	importScheduleProfile(t, app3, env, "sched")
	time.Sleep(700 * time.Millisecond)
	afterRestartOff := readScheduleItem(t, env)
	if afterRestartOff.Value.Sequence != off.Value.Sequence || afterRestartOff.Revision != off.Revision {
		t.Fatalf("off schedule restarted ticking: off %#v after restart %#v", off, afterRestartOff)
	}
}

type scheduleDoc struct {
	Name      string          `json:"name"`
	Status    string          `json:"status"`
	EveryMs   int64           `json:"everyMs"`
	WriteItem string          `json:"writeItem"`
	Value     json.RawMessage `json:"value"`
}

type scheduleValue struct {
	Schedule string          `json:"schedule"`
	Sequence int             `json:"sequence"`
	Value    json.RawMessage `json:"value"`
}

type scheduleItem struct {
	Value      scheduleValue
	Revision   uint64
	Provenance itemProv
}

func waitScheduleItem(t *testing.T, env []string, past int, timeout time.Duration) scheduleItem {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		item, ok := tryScheduleItem(t, env)
		if ok && item.Value.Sequence > past {
			return item
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for scheduled item past %d", past)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func readScheduleItem(t *testing.T, env []string) scheduleItem {
	t.Helper()
	item, ok := tryScheduleItem(t, env)
	if !ok {
		t.Fatal("scheduled item is not readable")
	}
	return item
}

func tryScheduleItem(t *testing.T, env []string) (scheduleItem, bool) {
	t.Helper()
	code, out, errOut := runTinkalet(env, "item", "get", "deploy/789/tick", "--json")
	if code != 0 || errOut != "" {
		return scheduleItem{}, false
	}
	item := decodeItem(t, out)
	if item.Key != "deploy/789/tick" || item.Status != "resolved" {
		t.Fatalf("scheduled item drift: %#v", item)
	}
	if item.Provenance.Writer != "tinkabot-schedule" || item.Provenance.Source != "server-schedule:deploytick" {
		t.Fatalf("scheduled item provenance drift: %#v", item.Provenance)
	}
	var val scheduleValue
	if err := json.Unmarshal(item.Value, &val); err != nil {
		t.Fatalf("scheduled value: %v\n%s", err, item.Value)
	}
	return scheduleItem{Value: val, Revision: item.Revision, Provenance: item.Provenance}, true
}

func assertSchedulePrivate(t *testing.T, out string, app *App) {
	t.Helper()
	for _, leak := range []string{"tb_schedules", "tb_items", "$KV", string(mustReadFile(t, app.CredsFile(RoleCaller)))} {
		if strings.Contains(out, leak) {
			t.Fatalf("schedule output leaked substrate detail %q: %s", leak, out)
		}
	}
}

func importScheduleProfile(t *testing.T, app *App, env []string, name string) {
	t.Helper()
	w := wiring()
	pub := []string{
		"$JS.API.INFO",
		"$KV." + w.ScheduleBucket + ".>",
	}
	pub = append(pub, readKV(w.ScheduleBucket)...)
	pub = append(pub, readKV(w.ItemBucket)...)
	creds, err := app.Runtime().MintUser(embednats.AppAccount, principal("principal.test."+name, "lease-test-"+name, core.Permissions{
		Publish:   core.PermList{Allow: pub},
		Subscribe: core.PermList{Allow: []string{"_INBOX.>"}},
	}), servingTTL)
	if err != nil {
		t.Fatal(err)
	}
	store := t.TempDir()
	if err := os.WriteFile(filepath.Join(store, "caller.creds"), creds.File, 0o600); err != nil {
		t.Fatal(err)
	}
	desc := struct {
		Kind       string `json:"kind"`
		Server     string `json:"server"`
		Shell      string `json:"shell"`
		Credential string `json:"credential"`
		Role       string `json:"role"`
		Trust      string `json:"trust"`
		Source     string `json:"source"`
	}{
		Kind:       "tinkabot.localProfile.v1",
		Server:     app.Posture().NATS.ClientURL,
		Shell:      app.Posture().Shell.URL,
		Credential: "caller.creds",
		Role:       "caller",
		Trust:      "local-owner",
		Source:     "local-store:" + store,
	}
	body, err := json.Marshal(desc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(store, "local-profile.json"), body, 0o600); err != nil {
		t.Fatal(err)
	}
	mustTinkalet(t, env, "profile", "import", "local", "--store", store, "--name", name)
}
