package tinkabot

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lagz0ne/tinkabot/substrate/go/core"
	"github.com/lagz0ne/tinkabot/substrate/go/embednats"
)

func TestTinkaletItemWatchUsesWatchStream(t *testing.T) {
	t.Parallel()
	store := t.TempDir()
	app, err := boot(t, cfgFor(store))
	if err != nil {
		t.Fatal(err)
	}
	writer, watcher := tinkaletEnv(t), tinkaletEnv(t)
	mustTinkalet(t, writer, "profile", "import", "local", "--store", store, "--name", "local")
	mustTinkalet(t, writer, "profile", "use", "local")
	importWatchOnly(t, app, watcher, "watch")
	mustTinkalet(t, watcher, "profile", "use", "watch")

	mustTinkalet(t, writer, "item", "create", "deploy/123", "--value", `{"env":"staging"}`)
	code, out, errOut := runTinkalet(watcher, "item", "get", "deploy/123", "--json")
	if code == 0 || out != "" {
		t.Fatalf("watch-only profile could poll item get: exit/stdout/stderr = %d/%q/%q", code, out, errOut)
	}

	done := make(chan watchRun, 1)
	watchErr := make(chan string, 1)
	go func() {
		code, out, errOut := runTinkalet(watcher, "watch", "item", "deploy/123", "--cursor", "stream", "--limit", "3", "--timeout", "5s", "--json")
		if code != 0 || errOut != "" {
			watchErr <- "watch exit/stderr = " + errOut
			return
		}
		events, err := parseWatchEvents(out)
		if err != nil {
			watchErr <- err.Error()
			return
		}
		done <- watchRun{out: out, events: events}
	}()
	time.Sleep(150 * time.Millisecond)
	mustTinkalet(t, writer, "item", "resolve", "deploy/123", "--value", `{"approved":true}`)
	mustTinkalet(t, writer, "item", "resolve", "deploy/123", "--value", `{"approved":false}`)

	select {
	case res := <-done:
		assertWatchPrivate(t, res.out, app)
		events := res.events
		if len(events) != 3 {
			t.Fatalf("watch event count = %d: %#v", len(events), events)
		}
		if events[0].Source != "replay" || events[0].Status != "pending" {
			t.Fatalf("initial replay drift: %#v", events[0])
		}
		if events[1].Source != "watch" || events[2].Source != "watch" || events[1].Revision >= events[2].Revision {
			t.Fatalf("live watch did not preserve ordered intermediate revisions: %#v", events)
		}
		for _, ev := range events {
			if ev.Key != "deploy/123" {
				t.Fatalf("wrong key in watch event: %#v", ev)
			}
		}
	case err := <-watchErr:
		t.Fatal(err)
	case <-time.After(6 * time.Second):
		t.Fatal("watch stream did not finish")
	}
}

func TestTinkaletDaemonWatchCursorRestartCatchesRetainedEvents(t *testing.T) {
	t.Parallel()
	store := t.TempDir()
	app, err := boot(t, cfgFor(store))
	if err != nil {
		t.Fatal(err)
	}
	writer, daemon := tinkaletEnv(t), tinkaletEnv(t)
	mustTinkalet(t, writer, "profile", "import", "local", "--store", store, "--name", "local")
	mustTinkalet(t, writer, "profile", "use", "local")
	importWatchOnly(t, app, daemon, "daemon")
	mustTinkalet(t, daemon, "profile", "use", "daemon")

	mustTinkalet(t, writer, "item", "create", "deploy/456", "--value", `{"env":"prod"}`)
	code, out, errOut := runTinkalet(daemon, "daemon", "watch", "item", "deploy/456", "--cursor", "daemon456", "--limit", "1", "--timeout", "2s", "--json")
	if code != 0 || errOut != "" {
		t.Fatalf("daemon seed exit/stderr = %d/%q", code, errOut)
	}
	assertWatchPrivate(t, out, app)
	seed := decodeWatchEvent(t, out)
	if seed.Source != "replay" || seed.Status != "pending" {
		t.Fatalf("daemon seed drift: %#v", seed)
	}

	mustTinkalet(t, writer, "item", "resolve", "deploy/456", "--value", `{"approved":true}`)
	mustTinkalet(t, writer, "item", "resolve", "deploy/456", "--value", `{"approved":false}`)

	code, out, errOut = runTinkalet(daemon, "daemon", "watch", "item", "deploy/456", "--cursor", "daemon456", "--limit", "2", "--timeout", "2s", "--json")
	if code != 0 || errOut != "" {
		t.Fatalf("daemon replay exit/stderr = %d/%q", code, errOut)
	}
	assertWatchPrivate(t, out, app)
	replayed, err := parseWatchEvents(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(replayed) != 2 || replayed[0].Revision <= seed.Revision || replayed[1].Revision <= replayed[0].Revision {
		t.Fatalf("daemon replay missed retained revisions: seed %#v replay %#v", seed, replayed)
	}
	for _, ev := range replayed {
		if ev.Key != "deploy/456" || ev.Source != "replay" || ev.Status != "resolved" {
			t.Fatalf("daemon replay drift: %#v", ev)
		}
	}
	assertFileMode(t, filepath.Join(envValue(t, daemon, "TINKALET_DATA_DIR"), "cursors", "daemon456.json"), 0o600)

	code, out, errOut = runTinkalet(daemon, "daemon", "watch", "item", "deploy/456", "--cursor", "daemon456", "--limit", "1", "--timeout", "200ms", "--json")
	if code != 1 || out != "" || errOut != "watch deploy/456 denied item: watch-timeout\n" {
		t.Fatalf("daemon duplicate-skip exit/stdout/stderr = %d/%q/%q", code, out, errOut)
	}

	writeCursor(t, daemon, "future", "daemon", "local-store:"+watchStore(t, daemon, "daemon"), "item", "deploy/456", replayed[1].Revision+100)
	code, out, errOut = runTinkalet(daemon, "daemon", "watch", "item", "deploy/456", "--cursor", "future", "--limit", "1", "--timeout", "2s", "--json")
	if code != 1 || out != "" || errOut != "watch deploy/456 denied item: stale-cursor\n" {
		t.Fatalf("future cursor exit/stdout/stderr = %d/%q/%q", code, out, errOut)
	}
}

func TestTinkaletScopedWatcherProfile(t *testing.T) {
	t.Parallel()
	store := t.TempDir()
	app, err := boot(t, cfgFor(store))
	if err != nil {
		t.Fatal(err)
	}
	owner, watcher := tinkaletEnv(t), tinkaletEnv(t)
	mustTinkalet(t, owner, "profile", "import", "local", "--store", store, "--name", "owner")
	mustTinkalet(t, owner, "profile", "use", "owner")

	key := "artifacts.artifact-browser.results.choice"
	prof, err := app.AdmitWatcher("llm", "item", key)
	if err != nil {
		t.Fatal(err)
	}
	mustTinkalet(t, watcher, "profile", "import", "local", "--store", prof.StoreDir, "--name", "llm")
	mustTinkalet(t, watcher, "profile", "use", "llm")

	code, out, errOut := runTinkalet(watcher, "item", "get", key, "--json")
	if code != 1 || out != "" || errOut != "item "+key+" denied get: denied-scope\n" {
		t.Fatalf("scoped watcher direct get exit/stdout/stderr = %d/%q/%q", code, out, errOut)
	}
	code, out, errOut = runTinkalet(watcher, "watch", "prefix", "artifacts.artifact-browser.results", "--timeout", "10ms", "--json")
	if code != 1 || out != "" || errOut != "watch artifacts.artifact-browser.results denied prefix: denied-scope\n" {
		t.Fatalf("scoped watcher broad watch exit/stdout/stderr = %d/%q/%q", code, out, errOut)
	}

	done := make(chan string, 1)
	go func() {
		code, out, errOut := runTinkalet(watcher, "watch", "item", key, "--cursor", "llm-result", "--limit", "1", "--timeout", "5s", "--json")
		if code != 0 || errOut != "" {
			done <- fmt.Sprintf("watch failed %d stdout=%q stderr=%q", code, out, errOut)
			return
		}
		done <- out
	}()
	time.Sleep(150 * time.Millisecond)
	mustTinkalet(t, owner, "item", "create", key, "--value", `{"choice":"diagram-a"}`)

	select {
	case got := <-done:
		events, err := parseWatchEvents(got)
		if err != nil {
			t.Fatal(err)
		}
		if len(events) != 1 || events[0].Key != key || string(events[0].Value) != `{"choice":"diagram-a"}` {
			t.Fatalf("scoped watcher event drift: %s", got)
		}
		assertWatchPrivate(t, got, app)
	case <-time.After(6 * time.Second):
		t.Fatal("scoped watcher did not observe result")
	}

	if err := app.RevokeWatcher(prof); err != nil {
		t.Fatal(err)
	}
	code, out, errOut = runTinkalet(watcher, "watch", "item", key, "--limit", "1", "--timeout", "10ms", "--json")
	if code != 1 || out != "" || errOut != "watch "+key+" denied item: revoked-credentials\n" {
		t.Fatalf("revoked watcher exit/stdout/stderr = %d/%q/%q", code, out, errOut)
	}
}

type watchRun struct {
	out    string
	events []watchEvent
}

type watchEvent struct {
	Key      string          `json:"key"`
	Status   string          `json:"status"`
	Value    json.RawMessage `json:"value"`
	Revision uint64          `json:"revision"`
	Source   string          `json:"source"`
}

func importWatchOnly(t *testing.T, app *App, env []string, name string) {
	t.Helper()
	creds, err := app.Runtime().MintUser(embednats.AppAccount, principal("principal.test."+name, "lease-test-"+name, core.Permissions{
		Publish: core.PermList{Allow: []string{
			"$JS.API.INFO",
			"$JS.API.STREAM.INFO.KV_" + wiring().ItemBucket,
			"$JS.API.CONSUMER.CREATE.KV_" + wiring().ItemBucket + ".>",
			"$JS.API.CONSUMER.DELETE.KV_" + wiring().ItemBucket + ".>",
			"$JS.API.CONSUMER.INFO.KV_" + wiring().ItemBucket + ".>",
		}},
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

func assertWatchPrivate(t *testing.T, out string, app *App) {
	t.Helper()
	for _, leak := range []string{"tb_items", "$KV", string(mustReadFile(t, app.CredsFile(RoleCaller)))} {
		if strings.Contains(out, leak) {
			t.Fatalf("watch output leaked substrate detail %q: %s", leak, out)
		}
	}
}

func writeCursor(t *testing.T, env []string, name, profile, source, scope, target string, rev uint64) {
	t.Helper()
	body, err := json.MarshalIndent(struct {
		Kind      string `json:"kind"`
		Name      string `json:"name"`
		Profile   string `json:"profile"`
		Source    string `json:"source"`
		Scope     string `json:"scope"`
		Target    string `json:"target"`
		Revision  uint64 `json:"revision"`
		UpdatedAt string `json:"updatedAt"`
	}{
		Kind:      "tinkalet.cursor.item.v1",
		Name:      name,
		Profile:   profile,
		Source:    source,
		Scope:     scope,
		Target:    target,
		Revision:  rev,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(envValue(t, env, "TINKALET_DATA_DIR"), "cursors"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(envValue(t, env, "TINKALET_DATA_DIR"), "cursors", name+".json"), append(body, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
}

func watchStore(t *testing.T, env []string, profile string) string {
	t.Helper()
	body := mustReadFile(t, filepath.Join(envValue(t, env, "TINKALET_CONFIG_DIR"), "profiles.json"))
	var file struct {
		Profiles []struct {
			Name   string `json:"name"`
			Source string `json:"source"`
		} `json:"profiles"`
	}
	if err := json.Unmarshal(body, &file); err != nil {
		t.Fatal(err)
	}
	for _, prof := range file.Profiles {
		if prof.Name == profile && strings.HasPrefix(prof.Source, "local-store:") {
			return strings.TrimPrefix(prof.Source, "local-store:")
		}
	}
	t.Fatalf("profile %s source not found", profile)
	return ""
}

func decodeWatchEvent(t *testing.T, out string) watchEvent {
	t.Helper()
	events, err := parseWatchEvents(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("watch event count = %d: %s", len(events), out)
	}
	return events[0]
}

func parseWatchEvents(out string) ([]watchEvent, error) {
	lines := strings.Split(strings.TrimSpace(out), "\n")
	events := make([]watchEvent, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var ev watchEvent
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			return nil, err
		}
		events = append(events, ev)
	}
	return events, nil
}

func envValue(t *testing.T, env []string, key string) string {
	t.Helper()
	prefix := key + "="
	for _, kv := range env {
		if strings.HasPrefix(kv, prefix) {
			return strings.TrimPrefix(kv, prefix)
		}
	}
	t.Fatalf("env missing %s", key)
	return ""
}
