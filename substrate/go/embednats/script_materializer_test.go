package embednats

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/lagz0ne/tinkabot/substrate/go/core"
	"github.com/nats-io/nats.go"
)

func TestScriptMaterializerLoopFromNATSCLI(t *testing.T) {
	t.Parallel()
	h := scriptLoopHarness(t)
	act := h.act
	script := core.ScriptRecord{
		Kind:     "script.record",
		Key:      act.ScriptKey,
		Revision: act.ScriptRevision,
		Process: core.Process{
			Command:   "/bin/sh",
			Args:      []string{"-c", fmt.Sprintf("printf '%%s' '%s'", scriptFrames())},
			Cwd:       t.TempDir(),
			RPC:       core.FramedStdio,
			TimeoutMs: 2000,
			Resource:  core.Resource{CPUMillis: 100, MemoryMB: 64},
			Kill:      "process.kill",
			Cleanup:   "workdir.delete",
			Identity:  "principal.script.001",
		},
	}
	if err := h.scriptStore.Put(script); err != nil {
		t.Fatal(err)
	}

	rtm, err := core.NewScriptRuntime(core.ScriptPolicy{AllowedProjections: []string{"main"}, ArtifactPrefix: "artifact/"}, LocalScriptRunner{})
	if err != nil {
		t.Fatal(err)
	}
	mat, err := core.NewMaterializer(h.material)
	if err != nil {
		t.Fatal(err)
	}
	loop := NewScriptLoop(h.scriptStore, rtm, mat, h.material, h.material)
	route, out, err := h.router.RequestReply(h.nc, act)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { mustStop(t, route) })
	runs, stop := loop.Watch(out)
	t.Cleanup(stop)
	flush(t, h.nc)

	reply, err := natsCLI(h.rt, h.caller, "request", "--raw", "-H", HeaderRequestID+":req-script-001", act.Source.Subject, "run")
	if err != nil {
		t.Fatalf("CLI request failed: %v\n%s", err, reply)
	}
	if strings.TrimSpace(reply) != string(core.Accepted) {
		t.Fatalf("activation reply drift: %q", reply)
	}
	denyOut, denyErr := natsCLI(h.rt, h.caller, "kv", "put", h.ledger, "escape", "bad")
	wantDenied(t, denyOut, denyErr, "caller wrote ledger KV")
	run := waitScriptRun(t, runs)
	if run.Err != nil || run.Run.Status != "applied" {
		t.Fatalf("script run drift: %#v", run)
	}

	got, err := natsCLI(h.rt, h.observer, "kv", "get", h.material.Bucket(), "p.main", "--raw")
	if err != nil {
		t.Fatalf("CLI material read failed: %v\n%s", err, got)
	}
	var proj core.MaterialProjection
	if err := json.Unmarshal([]byte(got), &proj); err != nil {
		t.Fatal(err)
	}
	if proj.Kind != "material.projection" || proj.ProjectionID != "main" || proj.ObservedAt == "" || proj.Provenance.Producer != "script-materializer" || string(proj.Value) != `{"title":"from-script"}` {
		t.Fatalf("material projection drift: %#v", proj)
	}
	ev, err := natsCLI(h.rt, h.observer, "kv", "get", h.material.Bucket(), "e.script_run_"+runID(run.Activation, script), "--raw")
	if err != nil {
		t.Fatalf("CLI event read failed: %v\n%s", err, ev)
	}
	var event core.EventEnvelope
	if err := json.Unmarshal([]byte(ev), &event); err != nil {
		t.Fatal(err)
	}
	if event.Kind != "event.envelope" || event.EventType != "script.run" || event.Status != "success" {
		t.Fatalf("event drift: %#v", event)
	}
	path := t.TempDir() + "/artifact-main.js"
	got, err = natsCLI(h.rt, h.observer, "object", "get", h.material.ArtifactBucket(), "artifact/main.js", "--output", path, "--force", "--no-progress")
	if err != nil {
		t.Fatalf("CLI object read failed: %v\n%s", err, got)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "export default 1" {
		t.Fatalf("artifact body drift: %q", body)
	}
	manifestDoc, err := natsCLI(h.rt, h.observer, "kv", "get", h.material.Bucket(), "a."+keyEnc("artifact/main.js"), "--raw")
	if err != nil {
		t.Fatalf("CLI artifact manifest read failed: %v\n%s", err, manifestDoc)
	}
	var manifest core.MaterialArtifact
	if err := json.Unmarshal([]byte(manifestDoc), &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Kind != "artifact.manifest" || manifest.ArtifactRevision != "artifact.rev.7" || manifest.Digest == "" {
		t.Fatalf("artifact manifest drift: %#v", manifest)
	}
	denyOut, denyErr = natsCLI(h.rt, h.observer, "kv", "put", h.material.Bucket(), "p.main", "{}")
	wantDenied(t, denyOut, denyErr, "observer wrote material KV")
	denyOut, denyErr = natsCLI(h.rt, h.observer, "publish", "$O."+h.material.ArtifactBucket()+".C.deny", "bad")
	wantDenied(t, denyOut, denyErr, "observer wrote object chunk")

	reply, err = natsCLI(h.rt, h.caller, "request", "--raw", "-H", HeaderRequestID+":req-script-001", act.Source.Subject, "run")
	if err != nil {
		t.Fatalf("duplicate CLI request failed: %v\n%s", err, reply)
	}
	if strings.TrimSpace(reply) != string(core.Duplicate) {
		t.Fatalf("duplicate reply drift: %q", reply)
	}
	dup := waitScriptRun(t, runs)
	if dup.Err != nil || dup.Record.Status != core.Duplicate || dup.Run.Status != "" {
		t.Fatalf("duplicate executed script: %#v", dup)
	}
}

func TestLocalScriptRunnerRejectsMalformedFrame(t *testing.T) {
	t.Parallel()
	rec := core.ScriptRecord{
		Kind:     "script.record",
		Key:      "script.bad.frame",
		Revision: 1,
		Process: core.Process{
			Command:   "/bin/sh",
			Args:      []string{"-c", "printf '%s' 'not-a-frame'"},
			Cwd:       t.TempDir(),
			RPC:       core.FramedStdio,
			TimeoutMs: 1000,
			Resource:  core.Resource{CPUMillis: 100, MemoryMB: 64},
			Kill:      "process.kill",
			Cleanup:   "workdir.delete",
			Identity:  "principal.script.001",
		},
	}

	_, err := LocalScriptRunner{}.Run(core.ScriptInvocation{Record: rec})
	assertCore(t, err, core.ProtocolFrameInvalid)
}

func TestLocalScriptRunnerRejectsUnknownFrameField(t *testing.T) {
	t.Parallel()
	for _, body := range []string{
		`{"kind":"script.effect","effectType":"projection","projectionId":"main","snapshotRevision":"snap-001","artifactRevision":"artifact.rev.7","sequence":1,"value":{"title":"ok"},"extra":true}`,
		`{"kind":"script.effect","effectType":"artifact","artifactName":"artifact/main.js","artifactRevision":"artifact.rev.7","mediaType":"application/javascript","body":"export default 1","projectionId":"main"}`,
		`{"kind":"script.effect","effectType":"projection","projectionId":"main","snapshotRevision":"snap-001","artifactRevision":"artifact.rev.7","sequence":1,"value":{"title":"ok"},"subject":"tb.internal.escape"}`,
	} {
		body := body
		t.Run("", func(t *testing.T) {
			rec := core.ScriptRecord{
				Kind:     "script.record",
				Key:      "script.bad.frame",
				Revision: 1,
				Process: core.Process{
					Command:   "/bin/sh",
					Args:      []string{"-c", fmt.Sprintf("printf '%%s' '%s'", wireFrame(body))},
					Cwd:       t.TempDir(),
					RPC:       core.FramedStdio,
					TimeoutMs: 1000,
					Resource:  core.Resource{CPUMillis: 100, MemoryMB: 64},
					Kill:      "process.kill",
					Cleanup:   "workdir.delete",
					Identity:  "principal.script.001",
				},
			}

			_, err := LocalScriptRunner{}.Run(core.ScriptInvocation{Record: rec})
			assertCore(t, err, core.ProtocolFrameInvalid)
		})
	}
}

func TestKVScriptStoreRejectsUnknownRecordField(t *testing.T) {
	t.Parallel()
	h := scriptLoopHarness(t)
	valid := core.ScriptRecord{
		Kind:     "script.record",
		Key:      h.act.ScriptKey,
		Revision: h.act.ScriptRevision,
		Desc:     "Render the main material projection.",
		Process: core.Process{
			Command:   "/bin/sh",
			Args:      []string{"-c", "true"},
			Cwd:       t.TempDir(),
			RPC:       core.FramedStdio,
			TimeoutMs: 1000,
			Resource:  core.Resource{CPUMillis: 100, MemoryMB: 64},
			Kill:      "process.kill",
			Cleanup:   "workdir.delete",
			Identity:  "principal.script.001",
		},
	}
	if err := h.scriptStore.Put(valid); err != nil {
		t.Fatal(err)
	}
	entry, err := h.scriptStore.kv.Get("s." + keyEnc(h.act.ScriptKey))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(entry.Value()), `"env":null`) {
		t.Fatalf("nil process env leaked into script record JSON: %s", entry.Value())
	}
	got, ok, err := h.scriptStore.LoadScript(h.act.ScriptKey)
	if err != nil || !ok || got.Desc != valid.Desc {
		t.Fatalf("valid script record with desc drift: ok=%v rec=%#v err=%v", ok, got, err)
	}
	raw := []byte(`{"kind":"script.record","scriptKey":"script.proof.render","scriptRevision":7,"process":{"command":"/bin/sh","args":["-c","true"],"cwd":"/tmp","rpc":"framed_stdio","timeoutMs":1000,"resource":{"cpuMillis":100,"memoryMB":64},"kill":"process.kill","cleanup":"workdir.delete","identity":"principal.script.001"},"extra":true}`)
	if _, err := h.scriptStore.kv.Put("s."+keyEnc(h.act.ScriptKey), raw); err != nil {
		t.Fatal(err)
	}
	_, _, err = h.scriptStore.LoadScript(h.act.ScriptKey)
	assertCore(t, err, core.ProtocolFrameInvalid)
}

func TestScriptLoopDurableRunClaimRejectsAcceptedReplay(t *testing.T) {
	t.Parallel()
	h := scriptLoopHarness(t)
	act := h.act
	script := core.ScriptRecord{
		Kind:     "script.record",
		Key:      act.ScriptKey,
		Revision: act.ScriptRevision,
		Process: core.Process{
			Command:   "/bin/sh",
			Args:      []string{"-c", fmt.Sprintf("printf '%%s' '%s'", scriptFrames())},
			Cwd:       t.TempDir(),
			RPC:       core.FramedStdio,
			TimeoutMs: 2000,
			Resource:  core.Resource{CPUMillis: 100, MemoryMB: 64},
			Kill:      "process.kill",
			Cleanup:   "workdir.delete",
			Identity:  "principal.script.001",
		},
	}
	if err := h.scriptStore.Put(script); err != nil {
		t.Fatal(err)
	}
	rt, err := core.NewScriptRuntime(core.ScriptPolicy{AllowedProjections: []string{"main"}, ArtifactPrefix: "artifact/"}, LocalScriptRunner{})
	if err != nil {
		t.Fatal(err)
	}
	mat, err := core.NewMaterializer(h.material)
	if err != nil {
		t.Fatal(err)
	}
	loop := NewScriptLoop(h.scriptStore, rt, mat, h.material, h.material)
	in := make(chan RouterResult, 2)
	runs, stop := loop.Watch(in)
	t.Cleanup(stop)
	rec := acceptedRecord(act)
	in <- RouterResult{Activation: act, Record: rec}
	first := waitScriptRun(t, runs)
	if first.Err != nil || first.Run.Status != "applied" {
		t.Fatalf("first run drift: %#v", first)
	}
	in <- RouterResult{Activation: act, Record: rec}
	second := waitScriptRun(t, runs)
	if second.Err != nil || second.Run.Status != "duplicate" {
		t.Fatalf("accepted replay reran script: %#v", second)
	}
}

func TestScriptLoopAttributesStatusWriteFailure(t *testing.T) {
	t.Parallel()
	act := activation(t, read(t, "fixtures/valid/activation-request-reply.json"))
	rec := core.ScriptRecord{
		Kind:     "script.record",
		Key:      act.ScriptKey,
		Revision: act.ScriptRevision,
		Process: core.Process{
			Command:   "/bin/sh",
			Args:      []string{"-c", "true"},
			Cwd:       ".",
			RPC:       core.FramedStdio,
			TimeoutMs: 1000,
			Resource:  core.Resource{CPUMillis: 100, MemoryMB: 64},
			Kill:      "process.kill",
			Cleanup:   "workdir.delete",
			Identity:  "principal.script.001",
		},
	}
	rt, err := core.NewScriptRuntime(core.ScriptPolicy{AllowedProjections: []string{"main"}}, core.ScriptRunnerFunc(func(core.ScriptInvocation) (core.ScriptRun, error) {
		return core.ScriptRun{}, nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	mat, err := core.NewMaterializer(core.NewMemoryMaterialStore())
	if err != nil {
		t.Fatal(err)
	}
	loop := NewScriptLoop(scriptStoreFunc(func(string) (core.ScriptRecord, bool, error) {
		return rec, true, nil
	}), rt, mat, statusFunc(func(core.EventEnvelope) error {
		return core.MaterialErr(core.MaterialWriteFailed, "SaveEvent", "status write failed", nil, nil)
	}), claimFunc(func(core.AcceptedActivation, core.ScriptRecord) (bool, error) {
		return true, nil
	}))
	in := make(chan RouterResult, 1)
	runs, stop := loop.Watch(in)
	t.Cleanup(stop)
	in <- RouterResult{Activation: act, Record: acceptedRecord(act)}

	run := waitScriptRun(t, runs)
	assertCore(t, run.Err, core.MaterialWriteFailed)
}

type scriptHarness struct {
	act         core.Activation
	router      *SourceRouter
	rt          *Runtime
	nc          *nats.Conn
	scriptStore *KVScriptStore
	material    *KVMaterialStore
	caller      core.Auth
	observer    core.Auth
	ledger      string
}

func scriptLoopHarness(t *testing.T) scriptHarness {
	t.Helper()
	act := activation(t, read(t, "fixtures/valid/activation-request-reply.json"))
	base := "tbsmat_" + strconv.FormatInt(time.Now().UnixNano(), 36)
	ledgerBucket := base + "_ledger"
	scriptBucket := base + "_scripts"
	materialBucket := base + "_material"
	artifactBucket := base + "_artifacts"
	source := sourceAuthority(act)
	caller := callerAuth(act)
	routerSvc := scriptRouterAuth(act, ledgerBucket)
	service := materialAuth("principal.runtime.materializer", "lease-runtime-materializer", scriptBucket, materialBucket, artifactBucket, true)
	observer := materialAuth("principal.material.observer", "lease-material-observer", scriptBucket, materialBucket, artifactBucket, false)
	cfg := valid(t)
	cfg.Auth = routerSvc
	cfg.AuthUsers = []core.Auth{caller, service, observer}
	rt, err := start(t, cfg)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	ledgerStore, err := NewKVLedgerStore(ctx, rt, ledgerBucket)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(ledgerStore.Close)
	router, err := NewSourceRouter(source, core.NewDurableLedger(ledgerStore))
	if err != nil {
		t.Fatal(err)
	}
	nc, err := rt.Connect(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(nc.Close)
	scriptStore, err := NewKVScriptStoreFor(ctx, rt, service, scriptBucket)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(scriptStore.Close)
	material, err := NewKVMaterialStoreFor(ctx, rt, service, materialBucket, artifactBucket)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(material.Close)
	return scriptHarness{act: act, router: router, rt: rt, nc: nc, scriptStore: scriptStore, material: material, caller: caller, observer: observer, ledger: ledgerBucket}
}

func materialAuth(user, lease, scriptBucket, materialBucket, artifactBucket string, write bool) core.Auth {
	pub := []string{"$JS.API.INFO", "_INBOX.>"}
	if write {
		pub = append(pub,
			"$KV."+scriptBucket+".>",
			"$KV."+materialBucket+".>",
			"$O."+artifactBucket+".>",
		)
		pub = append(pub, kvAPI(scriptBucket)...)
		pub = append(pub, kvAPI(materialBucket)...)
		pub = append(pub, objAPI(artifactBucket)...)
	} else {
		pub = append(pub, readKVAPI(materialBucket)...)
		pub = append(pub, readObjAPI(artifactBucket)...)
	}
	return core.Auth{
		User: user,
		Capability: core.Capability{
			PrincipalID:  user,
			LeaseID:      lease,
			LeaseStatus:  "active",
			CapabilityID: "cap-" + user,
			SessionID:    "session-script",
		},
		Permissions: core.Permissions{
			Publish:   core.PermList{Allow: pub},
			Subscribe: core.PermList{Allow: []string{"_INBOX.>"}},
		},
	}
}

func callerAuth(act core.Activation) core.Auth {
	return core.Auth{
		User: act.SourcePrincipal.PrincipalID,
		Capability: core.Capability{
			PrincipalID:  act.SourcePrincipal.PrincipalID,
			LeaseID:      act.SourceLease.LeaseID,
			LeaseStatus:  "active",
			CapabilityID: "cap-source-caller",
			SessionID:    "session-source",
		},
		Permissions: core.Permissions{
			Publish:   core.PermList{Allow: []string{act.Source.Subject}},
			Subscribe: core.PermList{Allow: []string{"_INBOX.>"}},
		},
	}
}

func sourceAuthority(act core.Activation) core.Auth {
	auth := routerAuth(act, "")
	auth.Permissions.Publish = core.PermList{}
	auth.Permissions.Subscribe = core.PermList{Allow: []string{act.Source.Subject}, Deny: []string{"tb.internal.>"}}
	return auth
}

func scriptRouterAuth(act core.Activation, ledgerBucket string) core.Auth {
	api := kvAPI(ledgerBucket)
	return core.Auth{
		User: "principal.router.script",
		Capability: core.Capability{
			PrincipalID:  "principal.router.script",
			LeaseID:      "lease-router-script",
			LeaseStatus:  "active",
			CapabilityID: "cap-router-script",
			SessionID:    "session-router",
		},
		Permissions: core.Permissions{
			Publish:   core.PermList{Allow: append([]string{"$JS.API.INFO", "$KV." + ledgerBucket + ".>", "_INBOX.>"}, api...)},
			Subscribe: core.PermList{Allow: []string{act.Source.Subject, "_INBOX.>"}},
		},
	}
}

func kvAPI(bucket string) []string {
	return append([]string{
		"$JS.API.STREAM.CREATE.KV_" + bucket,
	}, readKVAPI(bucket)...)
}

func readKVAPI(bucket string) []string {
	return []string{
		"$JS.API.STREAM.INFO.KV_" + bucket,
		"$JS.API.DIRECT.GET.KV_" + bucket + ".>",
		"$JS.API.STREAM.MSG.GET.KV_" + bucket,
		"$JS.API.CONSUMER.CREATE.KV_" + bucket + ".>",
		"$JS.API.CONSUMER.MSG.NEXT.KV_" + bucket + ".>",
		"$JS.API.CONSUMER.DELETE.KV_" + bucket + ".>",
	}
}

func objAPI(bucket string) []string {
	return append([]string{
		"$JS.API.STREAM.CREATE.OBJ_" + bucket,
	}, readObjAPI(bucket)...)
}

func readObjAPI(bucket string) []string {
	return []string{
		"$JS.API.STREAM.INFO.OBJ_" + bucket,
		"$JS.API.DIRECT.GET.OBJ_" + bucket + ".>",
		"$JS.API.STREAM.MSG.GET.OBJ_" + bucket,
		"$JS.API.CONSUMER.CREATE.OBJ_" + bucket + ".>",
		"$JS.API.CONSUMER.MSG.NEXT.OBJ_" + bucket + ".>",
		"$JS.API.CONSUMER.DELETE.OBJ_" + bucket + ".>",
	}
}

func scriptFrames() string {
	return wireFrame(`{"kind":"script.effect","effectType":"projection","projectionId":"main","snapshotRevision":"snap-001","artifactRevision":"artifact.rev.7","sequence":1,"value":{"title":"from-script"}}`) +
		wireFrame(`{"kind":"script.effect","effectType":"artifact","artifactName":"artifact/main.js","artifactRevision":"artifact.rev.7","mediaType":"application/javascript","body":"export default 1"}`)
}

func wireFrame(body string) string {
	return fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len([]byte(body)), body)
}

func waitScriptRun(t *testing.T, runs <-chan ScriptRunResult) ScriptRunResult {
	t.Helper()
	select {
	case run := <-runs:
		return run
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for script run")
		return ScriptRunResult{}
	}
}

func wantDenied(t *testing.T, out string, err error, msg string) {
	t.Helper()
	low := strings.ToLower(out)
	if err == nil || !(strings.Contains(low, "permission") || strings.Contains(low, "authorization")) {
		t.Fatalf("%s: err=%v out=%s", msg, err, out)
	}
}

type scriptStoreFunc func(string) (core.ScriptRecord, bool, error)

func (f scriptStoreFunc) LoadScript(key string) (core.ScriptRecord, bool, error) {
	return f(key)
}

type statusFunc func(core.EventEnvelope) error

func (f statusFunc) SaveEvent(ev core.EventEnvelope) error {
	return f(ev)
}

type claimFunc func(core.AcceptedActivation, core.ScriptRecord) (bool, error)

func (f claimFunc) ClaimRun(acc core.AcceptedActivation, rec core.ScriptRecord) (bool, error) {
	return f(acc, rec)
}

func acceptedRecord(act core.Activation) core.LedgerRecord {
	return core.LedgerRecord{
		ActivationID: act.ActivationID,
		SourceID:     act.SourcePrincipal.SourceID,
		SourceKind:   act.Source.Kind,
		DedupeKey:    act.DedupeKey,
		ChainID:      act.Chain.ChainID,
		Status:       core.Accepted,
	}
}
