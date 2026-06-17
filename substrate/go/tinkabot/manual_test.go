package tinkabot

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lagz0ne/tinkabot/substrate/go/core"
	"github.com/lagz0ne/tinkabot/substrate/go/embednats"
	"github.com/nats-io/nats.go"
)

func conn(t *testing.T, app *App, role string) *nats.Conn {
	t.Helper()
	nc, err := nats.Connect(app.Posture().NATS.ClientURL, nats.UserCredentials(app.CredsFile(role)))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(nc.Close)
	return nc
}

func wantDenied(t *testing.T, nc *nats.Conn, err error, msg string) {
	t.Helper()
	var text string
	if err != nil {
		text = err.Error()
	}
	if nc != nil {
		if nc.IsClosed() {
			text += " closed"
		}
		if err := nc.LastError(); err != nil {
			text += " " + err.Error()
		}
	}
	low := strings.ToLower(text)
	for _, s := range []string{"permission", "authorization", "revoked", "closed"} {
		if strings.Contains(low, s) {
			return
		}
	}
	t.Fatalf("%s: no denial evidence in error: %s", msg, text)
}

func request(t *testing.T, nc *nats.Conn, subject, id, body string) string {
	t.Helper()
	msg := nats.NewMsg(subject)
	msg.Header.Set(embednats.HeaderRequestID, id)
	msg.Data = []byte(body)
	reply, err := nc.RequestMsg(msg, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	return string(reply.Data)
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

// TestBinaryManual runs the manual's flows through nats.go against the running
// binary in creds mode: script define, trigger, observation, the
// KV/Object/publish behavior sweep carried from
// operator-jwt-authority (docs/matched-abstraction/task/operator-jwt-authority.md:130),
// denied caller and observer writes, duplicate no-rerun, and revoked-lease
// denial. The manual-verbatim CLI proof lives in gate:manual; this package test
// stays hermetic to the Go ecosystem.
func TestBinaryManual(t *testing.T) {
	t.Parallel()
	app, err := boot(t, cfgFor(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	w := app.Posture().Wiring
	author := conn(t, app, RoleAuthor)
	caller := conn(t, app, RoleCaller)
	observer := conn(t, app, RoleObserver)
	authorJS, err := author.JetStream()
	if err != nil {
		t.Fatal(err)
	}
	callerJS, err := caller.JetStream()
	if err != nil {
		t.Fatal(err)
	}
	observerJS, err := observer.JetStream()
	if err != nil {
		t.Fatal(err)
	}
	scripts, err := authorJS.KeyValue(w.ScriptBucket)
	if err != nil {
		t.Fatal(err)
	}
	config, err := callerJS.KeyValue(w.ConfigBucket)
	if err != nil {
		t.Fatal(err)
	}
	material, err := observerJS.KeyValue(w.MaterialBucket)
	if err != nil {
		t.Fatal(err)
	}

	// Defining a script (manual "Defining a script"): the author role writes the
	// strict script record into the script KV bucket with minted creds.
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
	if _, err := scripts.Put(scriptKVKey, body); err != nil {
		t.Fatalf("author defined script: %v", err)
	}

	// Triggering work (request/reply): the reply is the ledger status.
	out := request(t, caller, w.TriggerSubject, "req-bin-001", "run")
	if strings.TrimSpace(out) != string(core.Accepted) {
		t.Fatalf("trigger reply drift: %q", out)
	}

	// Denied caller write: a caller cannot write the ledger KV bucket.
	ledger, err := callerJS.KeyValue(w.LedgerBucket)
	if err == nil {
		_, err = ledger.Put("escape", []byte("bad"))
	}
	wantDenied(t, caller, err, "caller wrote ledger KV")

	// Observing results: projection, artifact manifest, and artifact body.
	projDoc := await(t, "material projection", func() (string, bool) {
		entry, err := material.Get("p.main")
		if err != nil {
			return err.Error(), false
		}
		out := string(entry.Value())
		return out, strings.Contains(out, "material.projection")
	})
	var proj core.MaterialProjection
	if err := json.Unmarshal([]byte(projDoc), &proj); err != nil {
		t.Fatalf("projection decode failed: %v\n%s", err, projDoc)
	}
	if proj.ProjectionID != "main" || proj.Provenance.Producer != "script-materializer" || string(proj.Value) != `{"title":"from-script"}` {
		t.Fatalf("material projection drift: %#v", proj)
	}
	entry, err := material.Get("a." + base64.RawURLEncoding.EncodeToString([]byte("artifact/main.js")))
	if err != nil {
		t.Fatal(err)
	}
	manifestDoc := string(entry.Value())
	var manifest core.MaterialArtifact
	if err := json.Unmarshal([]byte(manifestDoc), &manifest); err != nil {
		t.Fatalf("artifact manifest decode failed: %v\n%s", err, manifestDoc)
	}
	if manifest.Kind != "artifact.manifest" || manifest.Digest == "" {
		t.Fatalf("artifact manifest drift: %#v", manifest)
	}
	obj, err := observerJS.ObjectStore(w.ArtifactBucket)
	if err != nil {
		t.Fatal(err)
	}
	artifactPath := filepath.Join(t.TempDir(), "main.js")
	if err := obj.GetFile("artifact/main.js", artifactPath); err != nil {
		t.Fatal(err)
	}
	artifactBody, err := os.ReadFile(artifactPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(artifactBody) != "export default 1" {
		t.Fatalf("artifact body drift: %q", artifactBody)
	}

	// Denied observer writes: read-only roles cannot write material KV or
	// Object Store chunks.
	_, err = material.Put("p.main", []byte("{}"))
	wantDenied(t, observer, err, "observer wrote material KV")
	err = observer.Publish("$O."+w.ArtifactBucket+".C.deny", []byte("bad"))
	if flushErr := observer.FlushTimeout(2 * time.Second); err == nil {
		err = flushErr
	}
	wantDenied(t, observer, err, "observer wrote object chunk")

	// Duplicate no-rerun: the same request id returns Duplicate and the
	// materialized projection stays byte-identical.
	out = request(t, caller, w.TriggerSubject, "req-bin-001", "run")
	if strings.TrimSpace(out) != string(core.Duplicate) {
		t.Fatalf("duplicate reply drift: %q", out)
	}
	again, err := material.Get("p.main")
	if err != nil {
		t.Fatal(err)
	}
	if string(again.Value()) != projDoc {
		t.Fatalf("duplicate reran the script: %s != %s", again, projDoc)
	}

	// KV/Object/publish behavior sweep under minted creds.
	msg := nats.NewMsg(w.TriggerSubject)
	msg.Header.Set(embednats.HeaderMessageID, "msg-bin-001")
	if err := caller.PublishMsg(msg); err != nil {
		t.Fatalf("subject-message trigger: %v", err)
	}
	if _, err := config.Put("app_config", []byte(`{"state":"new"}`)); err != nil {
		t.Fatalf("KV change trigger: %v", err)
	}
	bundle := filepath.Join(t.TempDir(), "app.bundle.js")
	if err := os.WriteFile(bundle, []byte("bundle"), 0o600); err != nil {
		t.Fatal(err)
	}
	uploads, err := callerJS.ObjectStore(w.UploadBucket)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := uploads.PutFile(bundle); err != nil {
		t.Fatalf("Object change trigger: %v", err)
	}
	if err := caller.Publish(w.EventsSubject, []byte("event")); err != nil {
		t.Fatalf("stream trigger publish: %v", err)
	}
	err = caller.Publish("tb.internal.escape", []byte("bad"))
	if flushErr := caller.FlushTimeout(2 * time.Second); err == nil {
		err = flushErr
	}
	wantDenied(t, caller, err, "caller published internal subject")

	// Revoked-lease denial: revoking the caller principal closes the live
	// connection and denies reconnect with the same creds.
	if err := app.Runtime().Revoke(embednats.AppAccount, app.Creds(RoleCaller).UserPub); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for caller.IsConnected() {
		if time.Now().After(deadline) {
			t.Fatal("revoked caller connection stayed open")
		}
		time.Sleep(10 * time.Millisecond)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	nc, err := app.Runtime().ConnectCreds(ctx, app.Creds(RoleCaller).File)
	if err == nil {
		nc.Close()
		t.Fatal("revoked caller reconnected")
	}
	wantDenied(t, nil, err, "revoked caller reconnected")
}
