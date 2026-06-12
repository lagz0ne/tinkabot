package embednats

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/lagz0ne/tinkabot/substrate/go/core"
)

const filterProjection = "filter.main"

func TestFilterLoopTransforms(t *testing.T) {
	t.Parallel()
	h := filterHarness(t)
	loop := newFilterLoop(t, h, writeFilterScript(t, h.act, loopFilterScript()))
	in := make(chan RouterResult, 2)
	out, stop := loop.Watch(in)
	t.Cleanup(stop)

	in <- filterResult(h.act, 1, "1:7")
	waitFilterApplied(t, out)
	assertFilterProjection(t, h.material, 1, 7)

	in <- filterResult(h.act, 2, "2:11")
	waitFilterApplied(t, out)
	assertFilterProjection(t, h.material, 2, 11)
}

func TestFilterLoopRespawns(t *testing.T) {
	t.Parallel()
	h := filterHarness(t)
	loop := newFilterLoop(t, h, writeFilterScript(t, h.act, oneShotFilterScript()))
	in := make(chan RouterResult, 2)
	out, stop := loop.Watch(in)
	t.Cleanup(stop)

	in <- filterResult(h.act, 1, "1:3")
	waitFilterApplied(t, out)
	assertFilterProjection(t, h.material, 1, 3)
	waitFilterExit(t, out)

	in <- filterResult(h.act, 2, "2:5")
	waitFilterApplied(t, out)
	assertFilterProjection(t, h.material, 2, 5)
}

func TestFilterLoopStopKills(t *testing.T) {
	t.Parallel()
	h := filterHarness(t)
	dir := t.TempDir()
	pidFile := filepath.Join(dir, "pid")
	loop := newFilterLoop(t, h, writeScript(t, dir, blockingFilterScript()))
	in := make(chan RouterResult, 1)
	out, stop := loop.Watch(in)

	in <- filterResult(h.act, 1, "1:1")
	pid := waitPID(t, pidFile)
	stop()
	stop()
	waitProcessGone(t, pid)
	assertClosed(t, out)
}

type filterSetup struct {
	act      core.Activation
	material *KVMaterialStore
}

func filterHarness(t *testing.T) filterSetup {
	t.Helper()
	act := activation(t, read(t, "fixtures/valid/activation-request-reply.json"))
	base := "tbfilt_" + strconv.FormatInt(time.Now().UnixNano(), 36)
	scriptBucket := base + "_scripts"
	materialBucket := base + "_material"
	artifactBucket := base + "_artifacts"
	service := materialAuth("principal.runtime.filter", "lease-runtime-filter", scriptBucket, materialBucket, artifactBucket, true)
	cfg := valid(t)
	cfg.Exposure = InProcess()
	cfg.Host = ""
	cfg.Port = 0
	cfg.WebSocket = WebSocket{}
	cfg.Core.Topology.WebSocket = core.WebSocket{}
	cfg.Auth = service
	rt, err := start(t, cfg)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	material, err := NewKVMaterialStoreFor(ctx, rt, service, materialBucket, artifactBucket)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(material.Close)
	return filterSetup{act: act, material: material}
}

func newFilterLoop(t *testing.T, h filterSetup, script string) *FilterLoop {
	t.Helper()
	rtm, err := core.NewScriptRuntime(core.ScriptPolicy{AllowedProjections: []string{filterProjection}}, LocalScriptRunner{})
	if err != nil {
		t.Fatal(err)
	}
	mat, err := core.NewMaterializer(h.material)
	if err != nil {
		t.Fatal(err)
	}
	return NewFilterLoop(filterRecord(h.act, script), rtm, mat, h.material)
}

func filterRecord(act core.Activation, script string) core.ScriptRecord {
	return core.ScriptRecord{
		Kind:     "script.record",
		Key:      act.ScriptKey,
		Revision: act.ScriptRevision,
		Process: core.Process{
			Command:   "/bin/sh",
			Args:      []string{script},
			Cwd:       filepath.Dir(script),
			RPC:       core.FramedStdio,
			TimeoutMs: 1000,
			Resource:  core.Resource{CPUMillis: 100, MemoryMB: 64},
			Kill:      "process.kill",
			Cleanup:   "workdir.delete",
			Identity:  "principal.script.001",
		},
	}
}

func filterResult(act core.Activation, seq int, payload string) RouterResult {
	next := act
	next.ActivationID = "act:filter:" + strconv.Itoa(seq)
	next.DedupeKey = "filter:" + strconv.Itoa(seq)
	next.Source.RequestID = "req-filter-" + strconv.Itoa(seq)
	next.Provenance.CreatedAt = "2026-06-08T00:00:0" + strconv.Itoa(seq) + ".000Z"
	return RouterResult{Activation: next, Record: acceptedRecord(next), Payload: []byte(payload)}
}

func waitFilterApplied(t *testing.T, out <-chan ScriptRunResult) ScriptRunResult {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		select {
		case run, ok := <-out:
			if !ok {
				t.Fatal("filter output closed before applied effect")
			}
			if run.Err != nil {
				t.Fatalf("filter run failed: %#v", run)
			}
			if run.Run.Status == "applied" {
				return run
			}
		case <-deadline:
			t.Fatal("timed out waiting for filter effect")
		}
	}
}

func waitFilterExit(t *testing.T, out <-chan ScriptRunResult) {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		select {
		case run, ok := <-out:
			if !ok {
				t.Fatal("filter output closed before process exit")
			}
			if run.Err != nil {
				assertCore(t, run.Err, core.ScriptProcessFailed)
				return
			}
		case <-deadline:
			t.Fatal("timed out waiting for filter process exit")
		}
	}
}

func assertFilterProjection(t *testing.T, store *KVMaterialStore, seq, count int) {
	t.Helper()
	raw, ok, err := store.LoadProjection(filterProjection)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatalf("projection %q was not materialized", filterProjection)
	}
	var proj core.MaterialProjection
	if err := json.Unmarshal(raw, &proj); err != nil {
		t.Fatal(err)
	}
	var val struct {
		Count int `json:"count"`
	}
	if err := json.Unmarshal(proj.Value, &val); err != nil {
		t.Fatal(err)
	}
	if proj.Kind != "material.projection" || proj.ProjectionID != filterProjection || proj.Sequence != seq || proj.SnapshotRevision == "" || proj.ArtifactRevision == "" || len(proj.Value) == 0 || val.Count != count {
		t.Fatalf("projection drift: %#v value=%#v", proj, val)
	}
}

func writeFilterScript(t *testing.T, act core.Activation, body string) string {
	t.Helper()
	return writeScript(t, filepath.Join(t.TempDir(), "filter-"+keyEnc(act.ActivationID)), body)
}

func writeScript(t *testing.T, dir, body string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "filter.sh")
	if err := os.WriteFile(path, []byte(body), 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

func loopFilterScript() string {
	return filterScript(`while IFS= read -r line; do
  emit "$line"
done
`)
}

func oneShotFilterScript() string {
	return filterScript(`if IFS= read -r line; then
  emit "$line"
fi
`)
}

func filterScript(readLoop string) string {
	return `#!/bin/sh
proj='` + filterProjection + `'
emit() {
  line=$1
  seq=${line%%:*}
  val=${line#*:}
  body=$(printf '{"kind":"script.effect","effectType":"projection","projectionId":"%s","snapshotRevision":"snap-%s","artifactRevision":"artifact.rev.%s","sequence":%s,"value":{"count":%s}}' "$proj" "$seq" "$seq" "$seq" "$val")
  len=$(printf '%s' "$body" | wc -c | tr -d ' ')
  printf 'Content-Length: %s\r\n\r\n%s' "$len" "$body"
}
` + readLoop
}

func blockingFilterScript() string {
	return `#!/bin/sh
printf '%s\n' "$$" > pid
while IFS= read -r line; do
  sleep 30
done
`
}

func waitPID(t *testing.T, path string) int {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		raw, err := os.ReadFile(path)
		if err == nil {
			pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
			if err == nil && pid > 0 {
				return pid
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for pid file %s", path)
	return 0
}

func waitProcessGone(t *testing.T, pid int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		err := syscall.Kill(pid, 0)
		if errors.Is(err, syscall.ESRCH) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("process %d still exists after stop", pid)
}

func assertClosed(t *testing.T, out <-chan ScriptRunResult) {
	t.Helper()
	select {
	case _, ok := <-out:
		if ok {
			t.Fatal("filter output remained open after stop")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for filter output close")
	}
}
