package tinkalet

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/nats-io/jwt/v2"
	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
)

func TestProfileImportListUse(t *testing.T) {
	t.Parallel()
	env := newEnv(t)
	store := localStore(t, "nats://127.0.0.1:4229", "http://127.0.0.1:8429", "SECRET-CREDS")

	code, out, errOut := runCmd(t, env.vars, "profile", "list")
	assertCmd(t, code, out, errOut, 0, "no profiles\n", "")

	code, out, errOut = runCmd(t, env.vars, "profile", "list", "--json")
	assertJSONCmd(t, code, errOut)
	assertList(t, out, "", nil)

	before := snapshot(t, env.home, env.xdgConfig, env.xdgState)
	code, out, errOut = runCmd(t, env.vars, "profile", "import", "local", "--store", store, "--name", "local")
	assertCmd(t, code, out, errOut, 0, "profile local imported\n", "")
	after := snapshot(t, env.home, env.xdgConfig, env.xdgState)
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("import wrote outside explicit dirs:\nbefore %#v\nafter %#v", before, after)
	}

	assertMode(t, filepath.Join(env.config, "profiles.json"), 0o600)
	creds := filepath.Join(env.data, "profiles", "local", "caller.creds")
	assertMode(t, creds, 0o600)
	if got := string(mustRead(t, creds)); got != "SECRET-CREDS" {
		t.Fatalf("managed credential copy = %q", got)
	}

	code, out, errOut = runCmd(t, env.vars, "profile", "list")
	assertCmd(t, code, out, errOut, 0, "- local caller local-owner\n", "")
	if strings.Contains(out, "SECRET-CREDS") {
		t.Fatalf("profile list leaked credential: %q", out)
	}

	code, out, errOut = runCmd(t, env.vars, "profile", "list", "--json")
	assertJSONCmd(t, code, errOut)
	assertList(t, out, "", []wantProfile{{Name: "local", Default: false}})
	if strings.Contains(out, "SECRET-CREDS") {
		t.Fatalf("profile list --json leaked credential: %q", out)
	}

	code, out, errOut = runCmd(t, env.vars, "profile", "use", "local")
	assertCmd(t, code, out, errOut, 0, "profile local selected\n", "")
	assertMode(t, filepath.Join(env.config, "default-profile"), 0o600)

	code, out, errOut = runCmd(t, env.vars, "profile", "list")
	assertCmd(t, code, out, errOut, 0, "* local caller local-owner\n", "")

	code, out, errOut = runCmd(t, env.vars, "profile", "list", "--json")
	assertJSONCmd(t, code, errOut)
	assertList(t, out, "local", []wantProfile{{Name: "local", Default: true}})
}

func TestProfileImportUse(t *testing.T) {
	t.Parallel()
	env := newEnv(t)
	store := localStore(t, "nats://127.0.0.1:4229", "http://127.0.0.1:8429", "SECRET-CREDS")

	code, out, errOut := runCmd(t, env.vars, "profile", "import", "local", "--store", store, "--name", "local", "--use")
	assertCmd(t, code, out, errOut, 0, "profile local imported and selected\n", "")
	assertMode(t, filepath.Join(env.config, "default-profile"), 0o600)

	code, out, errOut = runCmd(t, env.vars, "profile", "list", "--json")
	assertJSONCmd(t, code, errOut)
	assertList(t, out, "local", []wantProfile{{Name: "local", Default: true}})
}

func TestProfileImportDenials(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		make func(t *testing.T) string
		want string
	}{
		{
			name: "missing descriptor",
			make: func(t *testing.T) string {
				return t.TempDir()
			},
			want: "profile local denied profile import: import-source-missing\n",
		},
		{
			name: "malformed descriptor",
			make: func(t *testing.T) string {
				dir := t.TempDir()
				write(t, filepath.Join(dir, "local-profile.json"), "{", 0o600)
				return dir
			},
			want: "profile local denied profile import: import-source-invalid\n",
		},
		{
			name: "missing credential",
			make: func(t *testing.T) string {
				dir := t.TempDir()
				writeDescriptor(t, dir, "missing.creds")
				return dir
			},
			want: "profile local denied profile import: import-source-invalid\n",
		},
		{
			name: "escaping credential",
			make: func(t *testing.T) string {
				dir := t.TempDir()
				writeDescriptor(t, dir, "../caller.creds")
				return dir
			},
			want: "profile local denied profile import: import-source-invalid\n",
		},
		{
			name: "absolute credential",
			make: func(t *testing.T) string {
				dir := t.TempDir()
				writeDescriptor(t, dir, filepath.Join(dir, "caller.creds"))
				return dir
			},
			want: "profile local denied profile import: import-source-invalid\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			env := newEnv(t)
			code, out, errOut := runCmd(t, env.vars, "profile", "import", "local", "--store", tt.make(t), "--name", "local")
			assertCmd(t, code, out, errOut, 1, "", tt.want)
		})
	}
}

func TestProfileUseAndTriggerDenials(t *testing.T) {
	t.Parallel()
	env := newEnv(t)

	code, out, errOut := runCmd(t, env.vars, "profile", "use", "missing")
	assertCmd(t, code, out, errOut, 1, "", "profile missing denied profile use: profile-not-found\n")

	code, out, errOut = runCmd(t, env.vars, "trigger", "bundle.clock.tick")
	assertCmd(t, code, out, errOut, 1, "", "profile default denied bundle.clock.tick: profile-not-found\n")

	store := participantStore(t, "nats://127.0.0.1:1", "http://127.0.0.1:8429", "demo", "alice", testCreds(t))
	mustRun(t, env.vars, "profile", "import", "local", "--store", store, "--name", "alice")
	mustRun(t, env.vars, "profile", "use", "alice")
	code, out, errOut = runCmd(t, env.vars, "trigger", "bundle.clock.tick", "--request-id", "restricted-trigger")
	assertCmd(t, code, out, errOut, 1, "", "profile alice denied bundle.clock.tick: denied-scope\n")
}

func TestItemCommandDenials(t *testing.T) {
	t.Parallel()
	env := newEnv(t)

	code, out, errOut := runCmd(t, env.vars, "item", "create", "deploy/1", "--value", "{")
	assertCmd(t, code, out, errOut, 1, "", "item deploy/1 denied create: malformed-value\n")

	code, out, errOut = runCmd(t, env.vars, "item", "create", "deploy/1")
	assertCmd(t, code, out, errOut, 1, "", "item deploy/1 denied create: profile-not-found\n")
}

func TestActionCommandDenials(t *testing.T) {
	t.Parallel()
	env := newEnv(t)

	code, out, errOut := runCmd(t, env.vars, "action", "submit", "move-1", "--state", "apps.demo.state.board", "--base-revision", "1", "--value", "{")
	assertCmd(t, code, out, errOut, 1, "", "action move-1 denied submit: malformed-value\n")

	code, out, errOut = runCmd(t, env.vars, "action", "submit", "move-1", "--state", "apps.demo.state.board", "--base-revision", "1", "--value", `{"cell":"a1"}`)
	assertCmd(t, code, out, errOut, 1, "", "action move-1 denied submit: profile-not-found\n")

	code, out, errOut = runCmd(t, env.vars, "action", "apply", "apps.demo.participants.alice.actions.move-1", "--value", "{")
	assertCmd(t, code, out, errOut, 1, "", "action apps.demo.participants.alice.actions.move-1 denied apply: malformed-value\n")

	code, out, errOut = runCmd(t, env.vars, "action", "apply", "apps.demo.participants.alice.actions.move-1", "--value", `{"turn":"bob"}`)
	assertCmd(t, code, out, errOut, 1, "", "action apps.demo.participants.alice.actions.move-1 denied apply: profile-not-found\n")

	code, out, errOut = runCmd(t, env.vars, "action", "reject", "apps.demo.participants.alice.actions.move-1", "--reason", "wrong-turn")
	assertCmd(t, code, out, errOut, 1, "", "action apps.demo.participants.alice.actions.move-1 denied reject: profile-not-found\n")

	code, out, errOut = runCmd(t, env.vars, "action", "reject", "apps.demo.participants.alice.actions.move-1", "--reason", "BadReason")
	assertCmd(t, code, out, errOut, 2, "", "usage: tinkalet <command> [options]\n")
}

func TestAppHandlerRegister(t *testing.T) {
	t.Parallel()
	env := newEnv(t)
	template := t.TempDir()
	write(t, filepath.Join(template, "bundle.json"), `{"name":"template"}`, 0o600)

	code, out, errOut := runCmd(t, env.vars, "app", "handler", "register", "vite", "--from", template)
	assertCmd(t, code, out, errOut, 1, "", "app handler vite denied register: profile-not-found\n")

	store := localStore(t, "nats://127.0.0.1:4229", "http://127.0.0.1:8429", "SECRET-CREDS")
	mustRun(t, env.vars, "profile", "import", "local", "--store", store, "--name", "local")
	mustRun(t, env.vars, "profile", "use", "local")

	code, out, errOut = runCmd(t, env.vars, "app", "handler", "register", "vite", "--from", template, "--json")
	assertJSONCmd(t, code, errOut)
	var rec appHandlerRecord
	if err := json.Unmarshal([]byte(out), &rec); err != nil {
		t.Fatalf("handler json: %v\n%s", err, out)
	}
	if rec.Kind != "tinkalet.frontendHandler.v1" || rec.Name != "vite" || rec.Type != "vite" || rec.Profile != "local" {
		t.Fatalf("handler record = %#v", rec)
	}
	if rec.From != mustAbs(t, template) {
		t.Fatalf("handler from = %q, want %q", rec.From, mustAbs(t, template))
	}
	if strings.Contains(out, "SECRET-CREDS") {
		t.Fatalf("handler json leaked credential: %q", out)
	}
	path := filepath.Join(env.data, "app-handlers", "vite.json")
	assertMode(t, path, 0o600)
	var stored appHandlerRecord
	if err := json.Unmarshal(mustRead(t, path), &stored); err != nil {
		t.Fatalf("stored handler: %v", err)
	}
	if stored.Name != rec.Name || stored.From != rec.From || stored.Profile != rec.Profile {
		t.Fatalf("stored handler = %#v, want %#v", stored, rec)
	}

	code, out, errOut = runCmd(t, env.vars, "app", "handler", "register", "vite", "--from", filepath.Join(template, "missing"), "--json")
	assertCmd(t, code, out, errOut, 1, "", "app handler vite denied register: handler-source-invalid\n")
}

func TestAppHandlerRegisterRestrictedProfileDeniedBeforeNetwork(t *testing.T) {
	t.Parallel()
	env := newEnv(t)
	template := t.TempDir()
	write(t, filepath.Join(template, "bundle.json"), `{"name":"template"}`, 0o600)
	store := participantStore(t, "nats://127.0.0.1:1", "http://127.0.0.1:8429", "demo", "alice", testCreds(t))
	mustRun(t, env.vars, "profile", "import", "local", "--store", store, "--name", "alice")
	mustRun(t, env.vars, "profile", "use", "alice")

	code, out, errOut := runCmd(t, env.vars, "app", "handler", "register", "vite", "--from", template, "--json")
	assertCmd(t, code, out, errOut, 1, "", "app handler vite denied register: denied-scope\n")
}

func TestAppCreateFrontend(t *testing.T) {
	t.Parallel()
	srv := natsserver.New(&natsserver.Options{Host: "127.0.0.1", Port: -1, JetStream: true, StoreDir: t.TempDir(), NoLog: true, NoSigs: true})
	go srv.Start()
	if !srv.ReadyForConnections(5 * time.Second) {
		t.Fatal("nats server did not become ready")
	}
	t.Cleanup(srv.Shutdown)

	nc, err := nats.Connect(srv.ClientURL(), nats.NoReconnect())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(nc.Close)
	js, err := nc.JetStream()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := js.CreateKeyValue(&nats.KeyValueConfig{Bucket: itemBucket}); err != nil {
		t.Fatal(err)
	}

	env := newEnv(t)
	template := t.TempDir()
	write(t, filepath.Join(template, "bundle.json"), `{"name":"template"}`, 0o600)

	code, out, errOut := runCmd(t, env.vars, "app", "create", "frontend", "options-site", "--handler", "vite", "--json")
	assertCmd(t, code, out, errOut, 1, "", "app options-site denied create: profile-not-found\n")

	store := localStore(t, srv.ClientURL(), "http://127.0.0.1:8429", testCreds(t))
	mustRun(t, env.vars, "profile", "import", "local", "--store", store, "--name", "local")
	mustRun(t, env.vars, "profile", "use", "local")

	code, out, errOut = runCmd(t, env.vars, "app", "create", "frontend", "options-site", "--handler", "vite", "--json")
	assertCmd(t, code, out, errOut, 1, "", "app options-site denied create: handler-not-found\n")

	mustRun(t, env.vars, "app", "handler", "register", "vite", "--from", template)
	code, out, errOut = runCmd(t, env.vars, "app", "create", "frontend", "options-site", "--handler", "vite", "--json")
	assertJSONCmd(t, code, errOut)
	var item itemView
	if err := json.Unmarshal([]byte(out), &item); err != nil {
		t.Fatalf("app create json: %v\n%s", err, out)
	}
	if item.Key != "apps.options-site.state.frontend" || item.Status != "resolved" {
		t.Fatalf("created item = %#v", item)
	}
	var val frontendAppValue
	if err := json.Unmarshal(item.Value, &val); err != nil {
		t.Fatalf("frontend value: %v\n%s", err, item.Value)
	}
	if val.Kind != "tinkabot.frontendApp.v1" || val.Name != "options-site" || val.Handler != "vite" || val.StateKey != item.Key || val.ResultKey != "artifacts.options-site.results.plan" || val.GeneratedPath != "/artifacts/bundle/template/index.html" {
		t.Fatalf("frontend value = %#v", val)
	}
	if strings.Contains(out, "SECRET-CREDS") || strings.Contains(out, "tb_items") || strings.Contains(out, "$KV") {
		t.Fatalf("app create leaked authority: %q", out)
	}

	code, out, errOut = runCmd(t, env.vars, "app", "create", "frontend", "options-site", "--handler", "vite")
	assertCmd(t, code, out, errOut, 1, "", "app options-site denied create: duplicate-app\n")
}

func TestAppCreateFrontendRestrictedProfileDeniedBeforeNetwork(t *testing.T) {
	t.Parallel()
	env := newEnv(t)
	template := t.TempDir()
	write(t, filepath.Join(template, "bundle.json"), `{"name":"template"}`, 0o600)
	store := participantStore(t, "nats://127.0.0.1:1", "http://127.0.0.1:8429", "demo", "alice", testCreds(t))
	mustRun(t, env.vars, "profile", "import", "local", "--store", store, "--name", "alice")
	mustRun(t, env.vars, "profile", "use", "alice")

	code, out, errOut := runCmd(t, env.vars, "app", "create", "frontend", "options-site", "--handler", "vite", "--json")
	assertCmd(t, code, out, errOut, 1, "", "app options-site denied create: denied-scope\n")
}

func TestWatchCommandDenials(t *testing.T) {
	t.Parallel()
	env := newEnv(t)

	code, out, errOut := runCmd(t, env.vars, "watch", "item", "deploy/1", "--cursor", "bad/name")
	assertCmd(t, code, out, errOut, 2, "", "usage: tinkalet <command> [options]\n")

	code, out, errOut = runCmd(t, env.vars, "watch", "item", "deploy/1", "--cursor", "deploy1")
	assertCmd(t, code, out, errOut, 1, "", "watch deploy/1 denied item: profile-not-found\n")
}

func TestParticipantWatchScopeDenialPrecedesNetwork(t *testing.T) {
	t.Parallel()
	env := newEnv(t)
	store := participantStore(t, "nats://127.0.0.1:1", "http://127.0.0.1:8429", "demo", "alice", testCreds(t))

	mustRun(t, env.vars, "profile", "import", "local", "--store", store, "--name", "alice")
	mustRun(t, env.vars, "profile", "use", "alice")

	code, out, errOut := runCmd(t, env.vars, "watch", "prefix", "apps.demo.participants.bob.actions", "--timeout", "1ms")
	assertCmd(t, code, out, errOut, 1, "", "watch apps.demo.participants.bob.actions denied prefix: denied-scope\n")

	code, out, errOut = runCmd(t, env.vars, "watch", "item", "apps.demo.state.>", "--timeout", "1ms")
	assertCmd(t, code, out, errOut, 2, "", "usage: tinkalet <command> [options]\n")
}

func TestParticipantWatchFiltersDenyMalformedTargets(t *testing.T) {
	t.Parallel()
	prof := Profile{Name: "alice", Trust: "app-participant", AppID: "demo", ParticipantID: "alice"}

	for _, req := range []watchReq{
		{Scope: "item", Target: "apps.demo.state.>"},
		{Scope: "prefix", Target: "apps.demo.state.>"},
	} {
		filters, reason := participantWatchFilters(req, prof)
		if len(filters) != 0 || reason != "denied-scope" {
			t.Fatalf("participant filters for %#v = %#v/%q, want nil/denied-scope", req, filters, reason)
		}
	}
}

func TestReactionCommandDenials(t *testing.T) {
	t.Parallel()
	env := newEnv(t)

	code, out, errOut := runCmd(t, env.vars, "reaction", "add", "bad/name", "--watch", "item", "deploy/1", "--for", "resolved", "--cmd", "/bin/echo", "--write", "deploy/1/out")
	assertCmd(t, code, out, errOut, 2, "", "usage: tinkalet <command> [options]\n")

	code, out, errOut = runCmd(t, env.vars, "reaction", "add", "approve", "--watch", "item", "deploy/1", "--for", "resolved", "--cmd", "/bin/echo", "--write", "deploy/1/out")
	assertCmd(t, code, out, errOut, 1, "", "reaction approve denied add: profile-not-found\n")
}

func TestScheduleCommandDenials(t *testing.T) {
	t.Parallel()
	env := newEnv(t)

	code, out, errOut := runCmd(t, env.vars, "schedule", "set", "deploytick", "--every", "nope", "--write", "deploy/1/tick")
	assertCmd(t, code, out, errOut, 1, "", "schedule deploytick denied set: malformed-duration\n")

	code, out, errOut = runCmd(t, env.vars, "schedule", "set", "deploytick", "--every", "1s", "--write", "deploy/1/tick")
	assertCmd(t, code, out, errOut, 1, "", "schedule deploytick denied set: profile-not-found\n")
}

func TestTriggerProfileOverrideDoesNotChangeDefault(t *testing.T) {
	t.Parallel()
	env := newEnv(t)
	local := localStore(t, "nats://127.0.0.1:4229", "http://127.0.0.1:8429", "LOCAL-CREDS")
	other := localStore(t, "nats://127.0.0.1:4230", "http://127.0.0.1:8430", "OTHER-CREDS")

	mustRun(t, env.vars, "profile", "import", "local", "--store", local, "--name", "local")
	mustRun(t, env.vars, "profile", "import", "local", "--store", other, "--name", "other")
	mustRun(t, env.vars, "profile", "use", "local")

	code, out, errOut := runCmd(t, env.vars, "trigger", "bundle.clock.nope.extra", "--profile", "other", "--json")
	if code != 1 {
		t.Fatalf("exit = %d, stdout = %q, stderr = %q", code, out, errOut)
	}
	if errOut != "" {
		t.Fatalf("stderr = %q", errOut)
	}
	var got triggerJSON
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("trigger json: %v\n%s", err, out)
	}
	if got.Profile != "other" || got.Intent != "bundle.clock.nope.extra" || got.Status != "denied" || got.Reason != "unknown-trigger" {
		t.Fatalf("trigger json = %#v", got)
	}
	if got.Diagnostics.Server != "nats://127.0.0.1:4230" || got.Diagnostics.Shell != "http://127.0.0.1:8430" {
		t.Fatalf("diagnostics = %#v", got.Diagnostics)
	}
	if strings.Contains(out, "OTHER-CREDS") {
		t.Fatalf("trigger json leaked credential: %q", out)
	}

	code, out, errOut = runCmd(t, env.vars, "profile", "list")
	assertCmd(t, code, out, errOut, 0, "* local caller local-owner\n- other caller local-owner\n", "")
}

func TestTriggerMalformedResponse(t *testing.T) {
	t.Parallel()
	srv := natsserver.New(&natsserver.Options{Host: "127.0.0.1", Port: -1, NoLog: true, NoSigs: true})
	go srv.Start()
	if !srv.ReadyForConnections(5 * time.Second) {
		t.Fatal("nats server did not become ready")
	}
	t.Cleanup(srv.Shutdown)

	nc, err := nats.Connect(srv.ClientURL(), nats.NoReconnect())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(nc.Close)
	if _, err := nc.Subscribe("tb.bundle.clock.tick", func(m *nats.Msg) {
		_ = m.Respond([]byte("accepted-but-not-really"))
	}); err != nil {
		t.Fatal(err)
	}
	if err := nc.Flush(); err != nil {
		t.Fatal(err)
	}

	env := newEnv(t)
	store := localStore(t, srv.ClientURL(), "http://127.0.0.1:8429", testCreds(t))
	mustRun(t, env.vars, "profile", "import", "local", "--store", store, "--name", "local")
	mustRun(t, env.vars, "profile", "use", "local")

	code, out, errOut := runCmd(t, env.vars, "trigger", "bundle.clock.tick", "--request-id", "req-malformed")
	assertCmd(t, code, out, errOut, 1, "", "profile local denied bundle.clock.tick: malformed-response\n")
}

func TestTriggerIntentGrammarDenials(t *testing.T) {
	t.Parallel()
	env := newEnv(t)
	store := localStore(t, "nats://127.0.0.1:4229", "http://127.0.0.1:8429", "SECRET-CREDS")
	mustRun(t, env.vars, "profile", "import", "local", "--store", store, "--name", "local")
	mustRun(t, env.vars, "profile", "use", "local")

	for _, intent := range []string{"bundle.Builder.source", "bundle.builder.src_one", "bundle.builder.*"} {
		code, out, errOut := runCmd(t, env.vars, "trigger", intent)
		assertCmd(t, code, out, errOut, 1, "", "profile local denied "+intent+": unknown-trigger\n")
	}
}

func TestTriggerGenericBundleIntent(t *testing.T) {
	t.Parallel()
	srv := natsserver.New(&natsserver.Options{Host: "127.0.0.1", Port: -1, NoLog: true, NoSigs: true})
	go srv.Start()
	if !srv.ReadyForConnections(5 * time.Second) {
		t.Fatal("nats server did not become ready")
	}
	t.Cleanup(srv.Shutdown)

	nc, err := nats.Connect(srv.ClientURL(), nats.NoReconnect())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(nc.Close)
	if _, err := nc.Subscribe("tb.bundle.builder.source", func(m *nats.Msg) {
		_ = m.Respond([]byte("accepted"))
	}); err != nil {
		t.Fatal(err)
	}
	if err := nc.Flush(); err != nil {
		t.Fatal(err)
	}

	env := newEnv(t)
	store := localStore(t, srv.ClientURL(), "http://127.0.0.1:8429", testCreds(t))
	mustRun(t, env.vars, "profile", "import", "local", "--store", store, "--name", "local")
	mustRun(t, env.vars, "profile", "use", "local")

	code, out, errOut := runCmd(t, env.vars, "trigger", "bundle.builder.source", "--request-id", "req-builder")
	assertCmd(t, code, out, errOut, 0, "profile local accepted bundle.builder.source\n", "")
}

func runCmd(t *testing.T, env []string, args ...string) (int, string, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code := Run(Config{Args: args, Stdout: &out, Stderr: &errOut, Env: env, Version: "tinkalet dev"})
	return code, out.String(), errOut.String()
}

func mustRun(t *testing.T, env []string, args ...string) {
	t.Helper()
	code, out, errOut := runCmd(t, env, args...)
	if code != 0 {
		t.Fatalf("%v exit = %d, stdout = %q, stderr = %q", args, code, out, errOut)
	}
}

func assertCmd(t *testing.T, code int, out, errOut string, wantCode int, wantOut, wantErr string) {
	t.Helper()
	if code != wantCode || out != wantOut || errOut != wantErr {
		t.Fatalf("exit/stdout/stderr = %d/%q/%q, want %d/%q/%q", code, out, errOut, wantCode, wantOut, wantErr)
	}
}

func assertJSONCmd(t *testing.T, code int, errOut string) {
	t.Helper()
	if code != 0 || errOut != "" {
		t.Fatalf("exit/stderr = %d/%q, want 0/empty", code, errOut)
	}
}

type testEnv struct {
	home      string
	xdgConfig string
	xdgState  string
	config    string
	data      string
	vars      []string
}

func newEnv(t *testing.T) testEnv {
	t.Helper()
	root := t.TempDir()
	env := testEnv{
		home:      filepath.Join(root, "home"),
		xdgConfig: filepath.Join(root, "xdg-config"),
		xdgState:  filepath.Join(root, "xdg-state"),
		config:    filepath.Join(root, "cfg"),
		data:      filepath.Join(root, "data"),
	}
	for _, dir := range []string{env.home, env.xdgConfig, env.xdgState} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	env.vars = []string{
		"HOME=" + env.home,
		"XDG_CONFIG_HOME=" + env.xdgConfig,
		"XDG_STATE_HOME=" + env.xdgState,
		"TINKALET_CONFIG_DIR=" + env.config,
		"TINKALET_DATA_DIR=" + env.data,
	}
	return env
}

func localStore(t *testing.T, server, shell, creds string) string {
	t.Helper()
	dir := t.TempDir()
	write(t, filepath.Join(dir, "caller.creds"), creds, 0o600)
	desc := map[string]string{
		"kind":       "tinkabot.localProfile.v1",
		"server":     server,
		"shell":      shell,
		"credential": "caller.creds",
		"role":       "caller",
		"trust":      "local-owner",
		"source":     "local-store:" + mustAbs(t, dir),
	}
	body, err := json.Marshal(desc)
	if err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(dir, "local-profile.json"), string(body), 0o600)
	return dir
}

func participantStore(t *testing.T, server, shell, appID, participantID, creds string) string {
	t.Helper()
	dir := t.TempDir()
	write(t, filepath.Join(dir, "participant.creds"), creds, 0o600)
	desc := map[string]string{
		"kind":          "tinkabot.localProfile.v1",
		"server":        server,
		"shell":         shell,
		"credential":    "participant.creds",
		"role":          "participant",
		"trust":         "app-participant",
		"source":        "local-store:" + mustAbs(t, dir),
		"status":        "active",
		"appId":         appID,
		"participantId": participantID,
	}
	body, err := json.Marshal(desc)
	if err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(dir, "local-profile.json"), string(body), 0o600)
	return dir
}

func testCreds(t *testing.T) string {
	t.Helper()
	user, err := nkeys.CreateUser()
	if err != nil {
		t.Fatal(err)
	}
	pub, err := user.PublicKey()
	if err != nil {
		t.Fatal(err)
	}
	account, err := nkeys.CreateAccount()
	if err != nil {
		t.Fatal(err)
	}
	claims := jwt.NewUserClaims(pub)
	claims.Name = "tinkalet-test"
	token, err := claims.Encode(account)
	if err != nil {
		t.Fatal(err)
	}
	seed, err := user.Seed()
	if err != nil {
		t.Fatal(err)
	}
	creds, err := jwt.FormatUserConfig(token, seed)
	if err != nil {
		t.Fatal(err)
	}
	return string(creds)
}

func writeDescriptor(t *testing.T, dir, cred string) {
	t.Helper()
	desc := map[string]string{
		"kind":       "tinkabot.localProfile.v1",
		"server":     "nats://127.0.0.1:4229",
		"shell":      "http://127.0.0.1:8429",
		"credential": cred,
		"role":       "caller",
		"trust":      "local-owner",
		"source":     "local-store:" + mustAbs(t, dir),
	}
	body, err := json.Marshal(desc)
	if err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(dir, "local-profile.json"), string(body), 0o600)
}

func write(t *testing.T, path, body string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), mode); err != nil {
		t.Fatal(err)
	}
}

func mustAbs(t *testing.T, path string) string {
	t.Helper()
	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	return abs
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func assertMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("%s mode = %04o, want %04o", path, got, want)
	}
}

func snapshot(t *testing.T, dirs ...string) []string {
	t.Helper()
	var out []string
	for _, dir := range dirs {
		err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(dir, path)
			if err != nil {
				return err
			}
			out = append(out, filepath.Base(dir)+"/"+rel)
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	sort.Strings(out)
	return out
}

type wantProfile struct {
	Name    string
	Default bool
}

type listJSON struct {
	Default  string `json:"default"`
	Profiles []struct {
		Name               string `json:"name"`
		Default            bool   `json:"default"`
		Server             string `json:"server"`
		Shell              string `json:"shell"`
		Role               string `json:"role"`
		Trust              string `json:"trust"`
		Source             string `json:"source"`
		CredentialRef      string `json:"credentialRef"`
		CredentialRedacted bool   `json:"credentialRedacted"`
	} `json:"profiles"`
}

func assertList(t *testing.T, body, def string, want []wantProfile) {
	t.Helper()
	var got listJSON
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Fatalf("list json: %v\n%s", err, body)
	}
	if got.Default != def {
		t.Fatalf("default = %q, want %q", got.Default, def)
	}
	if len(got.Profiles) != len(want) {
		t.Fatalf("profiles = %#v, want %#v", got.Profiles, want)
	}
	for i, p := range want {
		got := got.Profiles[i]
		if got.Name != p.Name || got.Default != p.Default || got.Role != "caller" || got.Trust != "local-owner" || !got.CredentialRedacted {
			t.Fatalf("profile[%d] = %#v, want %#v", i, got, p)
		}
		if got.CredentialRef != "profiles/"+p.Name+"/caller.creds" {
			t.Fatalf("credentialRef = %q", got.CredentialRef)
		}
	}
}

type triggerJSON struct {
	Profile     string `json:"profile"`
	Intent      string `json:"intent"`
	Status      string `json:"status"`
	Reason      string `json:"reason"`
	RequestID   string `json:"requestId"`
	Diagnostics struct {
		Server string `json:"server"`
		Shell  string `json:"shell"`
	} `json:"diagnostics"`
}
