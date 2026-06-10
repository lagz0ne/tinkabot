package tinkabot

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lagz0ne/tinkabot/substrate/go/core"
	"github.com/lagz0ne/tinkabot/substrate/go/embednats"
)

// cli runs the real nats CLI against the running binary with a materialized
// role creds file — the manual's connection preamble in creds mode.
func cli(t *testing.T, app *App, role string, args ...string) string {
	t.Helper()
	base := []string{
		"--no-context",
		"--server", app.Posture().NATS.ClientURL,
		"--creds", app.CredsFile(role),
		"--timeout", "2s",
	}
	out, _ := exec.Command("nats", append(base, args...)...).CombinedOutput()
	return string(out)
}

// wantDenied is the output-parsed denial oracle: nats CLI v0.3.0 exits 0 on
// permission errors, so denial evidence must come from output text
// (docs/matched-abstraction/plan/endgame-app.md:177).
func wantDenied(t *testing.T, out, msg string) {
	t.Helper()
	low := strings.ToLower(out)
	if !strings.Contains(low, "permission") && !strings.Contains(low, "authorization") {
		t.Fatalf("%s: no denial evidence in output: %s", msg, out)
	}
}

func wantAllowed(t *testing.T, out, msg string) {
	t.Helper()
	low := strings.ToLower(out)
	if strings.Contains(low, "permission") || strings.Contains(low, "authorization") || strings.Contains(low, "error") {
		t.Fatalf("%s: unexpected denial or error: %s", msg, out)
	}
}

func await(t *testing.T, what string, probe func() (string, bool)) string {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if out, ok := probe(); ok {
			return out
		}
		if time.Now().After(deadline) {
			out, _ := probe()
			t.Fatalf("timed out waiting for %s: %v", what, out)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func frames() string {
	frame := func(body string) string {
		return fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(body), body)
	}
	return frame(`{"kind":"script.effect","effectType":"projection","projectionId":"main","snapshotRevision":"snap-001","artifactRevision":"artifact.rev.7","sequence":1,"value":{"title":"from-script"}}`) +
		frame(`{"kind":"script.effect","effectType":"artifact","artifactName":"artifact/main.js","artifactRevision":"artifact.rev.7","mediaType":"application/javascript","body":"export default 1"}`)
}

// TestBinaryManual runs the manual's flows through the real nats CLI against
// the running binary in creds mode: script define, trigger, observation, the
// KV/Object/publish behavior-commands creds-mode sweep carried from
// operator-jwt-authority (docs/matched-abstraction/task/operator-jwt-authority.md:130),
// denied caller and observer writes, duplicate no-rerun, and revoked-lease
// denial. All denial oracles are output-parsed, never exit-code.
func TestBinaryManual(t *testing.T) {
	t.Parallel()
	app, err := boot(t, cfgFor(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	w := app.Posture().Wiring

	// Defining a script (manual "Defining a script"): the author role writes
	// the strict script record into the script KV bucket over the CLI.
	rec := core.ScriptRecord{
		Kind:     "script.record",
		Key:      w.ScriptKey,
		Revision: w.ScriptRevision,
		Desc:     "Render the main material projection.",
		Process: core.Process{
			Command:   "/bin/sh",
			Args:      []string{"-c", fmt.Sprintf("printf '%%s' '%s'", frames())},
			Cwd:       t.TempDir(),
			RPC:       core.FramedStdio,
			TimeoutMs: 2000,
			Resource:  core.Resource{CPUMillis: 100, MemoryMB: 64},
			Kill:      "process.kill",
			Cleanup:   "workdir.delete",
			Identity:  "principal.script.001",
		},
	}
	body, err := json.Marshal(rec)
	if err != nil {
		t.Fatal(err)
	}
	scriptKVKey := "s." + base64.RawURLEncoding.EncodeToString([]byte(w.ScriptKey))
	wantAllowed(t, cli(t, app, RoleAuthor, "kv", "put", w.ScriptBucket, scriptKVKey, string(body)), "author defined script")

	// Triggering work (request/reply): the reply is the ledger status.
	out := cli(t, app, RoleCaller, "request", "--raw", "-H", embednats.HeaderRequestID+":req-bin-001", w.TriggerSubject, "run")
	if strings.TrimSpace(out) != string(core.Accepted) {
		t.Fatalf("trigger reply drift: %q", out)
	}

	// Denied caller write: a caller cannot write the ledger KV bucket.
	wantDenied(t, cli(t, app, RoleCaller, "kv", "put", w.LedgerBucket, "escape", "bad"), "caller wrote ledger KV")

	// Observing results: projection, artifact manifest, and artifact body.
	projDoc := await(t, "material projection", func() (string, bool) {
		out := cli(t, app, RoleObserver, "kv", "get", w.MaterialBucket, "p.main", "--raw")
		return out, strings.Contains(out, "material.projection")
	})
	var proj core.MaterialProjection
	if err := json.Unmarshal([]byte(projDoc), &proj); err != nil {
		t.Fatalf("projection decode failed: %v\n%s", err, projDoc)
	}
	if proj.ProjectionID != "main" || proj.Provenance.Producer != "script-materializer" || string(proj.Value) != `{"title":"from-script"}` {
		t.Fatalf("material projection drift: %#v", proj)
	}
	manifestDoc := cli(t, app, RoleObserver, "kv", "get", w.MaterialBucket, "a."+base64.RawURLEncoding.EncodeToString([]byte("artifact/main.js")), "--raw")
	var manifest core.MaterialArtifact
	if err := json.Unmarshal([]byte(manifestDoc), &manifest); err != nil {
		t.Fatalf("artifact manifest decode failed: %v\n%s", err, manifestDoc)
	}
	if manifest.Kind != "artifact.manifest" || manifest.Digest == "" {
		t.Fatalf("artifact manifest drift: %#v", manifest)
	}
	artifactPath := filepath.Join(t.TempDir(), "main.js")
	cli(t, app, RoleObserver, "object", "get", w.ArtifactBucket, "artifact/main.js", "--output", artifactPath, "--force", "--no-progress")
	artifactBody, err := os.ReadFile(artifactPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(artifactBody) != "export default 1" {
		t.Fatalf("artifact body drift: %q", artifactBody)
	}

	// Denied observer writes: read-only roles cannot write material KV or
	// Object Store chunks.
	wantDenied(t, cli(t, app, RoleObserver, "kv", "put", w.MaterialBucket, "p.main", "{}"), "observer wrote material KV")
	wantDenied(t, cli(t, app, RoleObserver, "publish", "$O."+w.ArtifactBucket+".C.deny", "bad"), "observer wrote object chunk")

	// Duplicate no-rerun: the same request id returns Duplicate and the
	// materialized projection stays byte-identical.
	out = cli(t, app, RoleCaller, "request", "--raw", "-H", embednats.HeaderRequestID+":req-bin-001", w.TriggerSubject, "run")
	if strings.TrimSpace(out) != string(core.Duplicate) {
		t.Fatalf("duplicate reply drift: %q", out)
	}
	if again := cli(t, app, RoleObserver, "kv", "get", w.MaterialBucket, "p.main", "--raw"); again != projDoc {
		t.Fatalf("duplicate reran the script: %s != %s", again, projDoc)
	}

	// KV/Object/publish behavior-commands creds-mode sweep: the manual's
	// remaining trigger commands run verbatim under minted creds.
	// nats CLI v0.3.0 accepts -H/--header, not --hdr.
	wantAllowed(t, cli(t, app, RoleCaller, "publish", "-H", embednats.HeaderMessageID+":msg-bin-001", w.TriggerSubject, ""), "subject-message trigger")
	wantAllowed(t, cli(t, app, RoleCaller, "kv", "put", w.ConfigBucket, "app_config", `{"state":"new"}`), "KV change trigger")
	bundle := filepath.Join(t.TempDir(), "app.bundle.js")
	if err := os.WriteFile(bundle, []byte("bundle"), 0o600); err != nil {
		t.Fatal(err)
	}
	wantAllowed(t, cli(t, app, RoleCaller, "object", "put", w.UploadBucket, bundle, "--name", "app.bundle.js", "--no-progress"), "Object change trigger")
	wantAllowed(t, cli(t, app, RoleCaller, "publish", w.EventsSubject, "event"), "stream trigger publish")
	wantDenied(t, cli(t, app, RoleCaller, "publish", "tb.internal.escape", "bad"), "caller published internal subject")

	// Revoked-lease denial: revoking the caller principal denies the CLI.
	if err := app.Runtime().Revoke(embednats.AppAccount, app.Creds(RoleCaller).UserPub); err != nil {
		t.Fatal(err)
	}
	wantDenied(t, cli(t, app, RoleCaller, "request", "--raw", "-H", embednats.HeaderRequestID+":req-bin-002", w.TriggerSubject, "run"), "revoked caller still served")
}
