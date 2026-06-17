package tinkabot

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lagz0ne/tinkabot/substrate/go/core"
	"github.com/lagz0ne/tinkabot/substrate/go/embednats"
)

func TestTinkaletLocalReaction(t *testing.T) {
	t.Parallel()
	store := t.TempDir()
	app, err := boot(t, cfgFor(store))
	if err != nil {
		t.Fatal(err)
	}
	writer := tinkaletEnv(t)
	reactor := append(tinkaletEnv(t), "SECRET_CREDS=do-not-inherit")
	mustTinkalet(t, writer, "profile", "import", "local", "--store", store, "--name", "local")
	mustTinkalet(t, writer, "profile", "use", "local")
	importReactorProfile(t, app, reactor, "reactor", "deploy/789/result", true)
	mustTinkalet(t, reactor, "profile", "use", "reactor")

	counter := filepath.Join(t.TempDir(), "runs")
	cmd := reactionScript(t)
	mustTinkalet(t, writer, "item", "create", "deploy/789", "--value", `{"env":"qa"}`)
	code, out, errOut := runTinkalet(reactor, "item", "get", "deploy/789", "--json")
	if code == 0 || out != "" {
		t.Fatalf("reactor profile could poll item get: exit/stdout/stderr = %d/%q/%q", code, out, errOut)
	}
	mustTinkalet(t, reactor, "reaction", "add", "approve", "--watch", "item", "deploy/789", "--for", "resolved", "--cmd", cmd, "--arg", counter, "--arg", "literal spaces ; $HOME", "--write", "deploy/789/result")
	dataDir := envValue(t, reactor, "TINKALET_DATA_DIR")
	reactionPath := filepath.Join(dataDir, "reactions", "approve.json")
	assertFileMode(t, reactionPath, 0o600)
	assertNoReactionLeak(t, string(mustReadFile(t, reactionPath)), app)

	done := make(chan reactionRun, 1)
	runErr := make(chan string, 1)
	go func() {
		code, out, errOut := runTinkalet(reactor, "daemon", "react", "approve", "--once", "--timeout", "5s", "--json")
		if code != 0 || errOut != "" {
			runErr <- "react exit/stderr = " + errOut
			return
		}
		var got reactionRun
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			runErr <- "react json: " + err.Error()
			return
		}
		done <- got
	}()
	time.Sleep(150 * time.Millisecond)
	mustTinkalet(t, writer, "item", "resolve", "deploy/789", "--value", `{"approved":true}`)

	select {
	case got := <-done:
		if got.Reaction != "approve" || got.Status != "ran" || got.Item != "deploy/789/result" || got.ExitCode != 0 {
			t.Fatalf("reaction result drift: %#v", got)
		}
	case err := <-runErr:
		t.Fatal(err)
	case <-time.After(6 * time.Second):
		t.Fatal("reaction daemon did not run")
	}
	cursorPath := filepath.Join(dataDir, "cursors", "reaction-approve.json")
	assertFileMode(t, cursorPath, 0o600)
	assertNoReactionLeak(t, string(mustReadFile(t, cursorPath)), app)

	code, out, errOut = runTinkalet(writer, "item", "get", "deploy/789/result", "--json")
	if code != 0 || errOut != "" {
		t.Fatalf("result get exit/stderr = %d/%q", code, errOut)
	}
	item := decodeItem(t, out)
	var val reactionValue
	if err := json.Unmarshal(item.Value, &val); err != nil {
		t.Fatalf("reaction value: %v\n%s", err, item.Value)
	}
	if item.Status != "resolved" || val.ExitCode != 0 || val.Stdout != `{"ran":true}` || val.Stderr != "arg ok\n" {
		t.Fatalf("writeback drift: item %#v value %#v", item, val)
	}
	if got := string(mustReadFile(t, counter)); got != "x" {
		t.Fatalf("command run count = %q", got)
	}

	code, out, errOut = runTinkalet(reactor, "daemon", "react", "approve", "--once", "--timeout", "200ms", "--json")
	if code != 1 || out != "" || errOut != "reaction approve denied run: watch-timeout\n" {
		t.Fatalf("duplicate react exit/stdout/stderr = %d/%q/%q", code, out, errOut)
	}
	if got := string(mustReadFile(t, counter)); got != "x" {
		t.Fatalf("duplicate event reran command: %q", got)
	}
}

func TestTinkaletReactionFailureKeepsCursor(t *testing.T) {
	t.Parallel()
	store := t.TempDir()
	app, err := boot(t, cfgFor(store))
	if err != nil {
		t.Fatal(err)
	}
	writer := tinkaletEnv(t)
	reactor := tinkaletEnv(t)
	mustTinkalet(t, writer, "profile", "import", "local", "--store", store, "--name", "local")
	mustTinkalet(t, writer, "profile", "use", "local")
	importReactorProfile(t, app, reactor, "reactor", "deploy/790/result", true)
	mustTinkalet(t, reactor, "profile", "use", "reactor")

	counter := filepath.Join(t.TempDir(), "runs")
	cmd := failingReactionScript(t)
	mustTinkalet(t, writer, "item", "create", "deploy/790", "--value", `{"env":"qa"}`)
	mustTinkalet(t, reactor, "reaction", "add", "fail", "--watch", "item", "deploy/790", "--for", "resolved", "--cmd", cmd, "--arg", counter, "--write", "deploy/790/result")
	mustTinkalet(t, writer, "item", "resolve", "deploy/790", "--value", `{"approved":true}`)

	for i, want := range []string{"x", "xx"} {
		code, out, errOut := runTinkalet(reactor, "daemon", "react", "fail", "--once", "--timeout", "2s", "--json")
		if code != 1 || out != "" || errOut != "reaction fail denied run: command-failed\n" {
			t.Fatalf("failed react %d exit/stdout/stderr = %d/%q/%q", i, code, out, errOut)
		}
		if got := string(mustReadFile(t, counter)); got != want {
			t.Fatalf("failed command run count %d = %q, want %q", i, got, want)
		}
	}
}

func TestTinkaletReactionDeniedWritebackKeepsCursor(t *testing.T) {
	t.Parallel()
	store := t.TempDir()
	app, err := boot(t, cfgFor(store))
	if err != nil {
		t.Fatal(err)
	}
	writer := tinkaletEnv(t)
	reactor := tinkaletEnv(t)
	mustTinkalet(t, writer, "profile", "import", "local", "--store", store, "--name", "local")
	mustTinkalet(t, writer, "profile", "use", "local")
	importReactorProfile(t, app, reactor, "reactor", "deploy/791/result", false)
	mustTinkalet(t, reactor, "profile", "use", "reactor")

	counter := filepath.Join(t.TempDir(), "runs")
	cmd := reactionScript(t)
	mustTinkalet(t, writer, "item", "create", "deploy/791", "--value", `{"env":"qa"}`)
	mustTinkalet(t, reactor, "reaction", "add", "denywrite", "--watch", "item", "deploy/791", "--for", "resolved", "--cmd", cmd, "--arg", counter, "--arg", "literal spaces ; $HOME", "--write", "deploy/791/result")
	mustTinkalet(t, writer, "item", "resolve", "deploy/791", "--value", `{"approved":true}`)

	for i, want := range []string{"x", "xx"} {
		code, out, errOut := runTinkalet(reactor, "daemon", "react", "denywrite", "--once", "--timeout", "2s", "--json")
		if code != 1 || out != "" || errOut != "reaction denywrite denied run: denied-writeback\n" {
			t.Fatalf("denied writeback %d exit/stdout/stderr = %d/%q/%q", i, code, out, errOut)
		}
		if got := string(mustReadFile(t, counter)); got != want {
			t.Fatalf("denied write command run count %d = %q, want %q", i, got, want)
		}
	}
}

func TestTinkaletReactionRemovedProfileDoesNotRun(t *testing.T) {
	t.Parallel()
	store := t.TempDir()
	app, err := boot(t, cfgFor(store))
	if err != nil {
		t.Fatal(err)
	}
	writer := tinkaletEnv(t)
	reactor := tinkaletEnv(t)
	mustTinkalet(t, writer, "profile", "import", "local", "--store", store, "--name", "local")
	mustTinkalet(t, writer, "profile", "use", "local")
	importReactorProfile(t, app, reactor, "reactor", "deploy/792/result", true)
	mustTinkalet(t, reactor, "profile", "use", "reactor")

	counter := filepath.Join(t.TempDir(), "runs")
	cmd := reactionScript(t)
	mustTinkalet(t, writer, "item", "create", "deploy/792", "--value", `{"env":"qa"}`)
	mustTinkalet(t, reactor, "reaction", "add", "removed", "--watch", "item", "deploy/792", "--for", "resolved", "--cmd", cmd, "--arg", counter, "--arg", "literal spaces ; $HOME", "--write", "deploy/792/result")
	mustTinkalet(t, writer, "item", "resolve", "deploy/792", "--value", `{"approved":true}`)
	if err := os.Remove(filepath.Join(envValue(t, reactor, "TINKALET_CONFIG_DIR"), "profiles.json")); err != nil {
		t.Fatal(err)
	}

	code, out, errOut := runTinkalet(reactor, "daemon", "react", "removed", "--once", "--timeout", "2s", "--json")
	if code != 1 || out != "" || errOut != "reaction removed denied run: profile-not-found\n" {
		t.Fatalf("removed profile exit/stdout/stderr = %d/%q/%q", code, out, errOut)
	}
	if _, err := os.Stat(counter); !os.IsNotExist(err) {
		t.Fatalf("removed profile still ran command, stat err %v", err)
	}
}

type reactionRun struct {
	Reaction string `json:"reaction"`
	Status   string `json:"status"`
	Item     string `json:"item"`
	ExitCode int    `json:"exitCode"`
}

type reactionValue struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exitCode"`
}

func reactionScript(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "reaction.sh")
	body := `#!/bin/sh
if [ -n "$SECRET_CREDS" ] || [ -n "$TINKALET_DATA_DIR" ]; then
  exit 9
fi
if [ "$2" != 'literal spaces ; $HOME' ]; then
  exit 8
fi
printf x >> "$1"
printf 'arg ok\n' >&2
printf '{"ran":true}'
`
	if err := os.WriteFile(path, []byte(strings.ReplaceAll(body, "\t", "")), 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

func importReactorProfile(t *testing.T, app *App, env []string, name, writeKey string, allowWrite bool) {
	t.Helper()
	w := wiring()
	pub := []string{
		"$JS.API.INFO",
		"$JS.API.STREAM.INFO.KV_" + w.ItemBucket,
		"$JS.API.CONSUMER.CREATE.KV_" + w.ItemBucket + ".>",
		"$JS.API.CONSUMER.DELETE.KV_" + w.ItemBucket + ".>",
		"$JS.API.CONSUMER.INFO.KV_" + w.ItemBucket + ".>",
	}
	if allowWrite {
		pub = append(pub, "$KV."+w.ItemBucket+"."+writeKey)
	}
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

func assertNoReactionLeak(t *testing.T, body string, app *App) {
	t.Helper()
	for _, leak := range []string{"tb_items", "$KV", string(mustReadFile(t, app.CredsFile(RoleCaller)))} {
		if strings.Contains(body, leak) {
			t.Fatalf("reaction state leaked substrate detail %q: %s", leak, body)
		}
	}
}

func failingReactionScript(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "reaction-fails.sh")
	body := `#!/bin/sh
printf x >> "$1"
exit 3
`
	if err := os.WriteFile(path, []byte(strings.ReplaceAll(body, "\t", "")), 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}
