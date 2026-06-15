package embednats

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/lagz0ne/tinkabot/substrate/go/core"
	"github.com/nats-io/nats.go"
)

const (
	maxScriptOutput = 1 << 20
	maxFrameBody    = 256 << 10
	// maxArtifactFile caps a path-artifact body read host-side, so a jailed
	// script cannot OOM the host by referencing an arbitrarily large file.
	maxArtifactFile = 32 << 20
)

type KVScriptStore struct {
	nc *nats.Conn
	kv nats.KeyValue
}

type KVMaterialStore struct {
	nc             *nats.Conn
	kv             nats.KeyValue
	obj            nats.ObjectStore
	bucket         string
	artifactBucket string
}

// LocalScriptRunner spawns the script directly. Sandbox is opt-in: nil (the
// zero value) runs unjailed (the wired app slot); non-nil wraps every run in
// bubblewrap (bundles), binding the run's outDir as the only writable path.
type LocalScriptRunner struct {
	Sandbox *SandboxConfig
}

type ScriptLoop struct {
	store interface {
		LoadScript(string) (core.ScriptRecord, bool, error)
	}
	runtime      *core.ScriptRuntime
	materializer *core.Materializer
	status       core.StatusSink
	claims       runClaimer
}

type runClaimer interface {
	ClaimRun(core.AcceptedActivation, core.ScriptRecord) (bool, error)
}

type ScriptRunResult struct {
	Activation core.Activation
	Record     core.LedgerRecord
	Run        core.ScriptRun
	Err        error
}

func NewKVScriptStore(ctx context.Context, rt *Runtime, bucket string) (*KVScriptStore, error) {
	return NewKVScriptStoreFor(ctx, rt, core.Auth{}, bucket)
}

func NewKVScriptStoreFor(ctx context.Context, rt *Runtime, auth core.Auth, bucket string) (*KVScriptStore, error) {
	nc, err := connectStore(ctx, rt, auth)
	if err != nil {
		return nil, err
	}
	return OpenKVScriptStore(nc, bucket)
}

// OpenKVScriptStore opens the script bucket over a caller-supplied connection
// (operator-mode assemblies connect with minted creds, where the static
// rt.Connect path does not exist). The store owns nc from here on.
func OpenKVScriptStore(nc *nats.Conn, bucket string) (*KVScriptStore, error) {
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, core.ScriptRecordErr(core.JetStreamUnavailable, "OpenScriptStore", "JetStream context is unavailable", nil, err)
	}
	kv, err := js.KeyValue(bucket)
	if errors.Is(err, nats.ErrBucketNotFound) {
		kv, err = js.CreateKeyValue(&nats.KeyValueConfig{Bucket: bucket, Storage: nats.FileStorage})
	}
	if err != nil {
		nc.Close()
		return nil, core.ScriptRecordErr(core.BucketMissing, "OpenScriptStore", "script KV bucket is unavailable", nil, err)
	}
	return &KVScriptStore{nc: nc, kv: kv}, nil
}

func (s *KVScriptStore) Close() {
	if s != nil && s.nc != nil {
		s.nc.Close()
	}
}

func (s *KVScriptStore) Put(rec core.ScriptRecord) error {
	body, err := json.Marshal(rec)
	if err != nil {
		return core.ScriptRecordErr(core.ProtocolFrameInvalid, "SaveScript", "script record could not be encoded", nil, err)
	}
	if _, err := s.kv.Put("s."+keyEnc(rec.Key), body); err != nil {
		return core.ScriptRecordErr(core.WriteConflict, "SaveScript", "script record could not be written", nil, err)
	}
	return nil
}

func (s *KVScriptStore) LoadScript(key string) (core.ScriptRecord, bool, error) {
	entry, err := s.kv.Get("s." + keyEnc(key))
	if errors.Is(err, nats.ErrKeyNotFound) {
		return core.ScriptRecord{}, false, nil
	}
	if err != nil {
		return core.ScriptRecord{}, false, core.ScriptRecordErr(core.ScriptRecordMissing, "LoadScript", "script record could not be read", nil, err)
	}
	var rec core.ScriptRecord
	if err := decodeStrict(entry.Value(), &rec); err != nil {
		return core.ScriptRecord{}, false, core.ScriptRecordErr(core.ProtocolFrameInvalid, "LoadScript", "script record could not be decoded", nil, err)
	}
	return rec, true, nil
}

func NewKVMaterialStore(ctx context.Context, rt *Runtime, bucket, artifactBucket string) (*KVMaterialStore, error) {
	return NewKVMaterialStoreFor(ctx, rt, core.Auth{}, bucket, artifactBucket)
}

func NewKVMaterialStoreFor(ctx context.Context, rt *Runtime, auth core.Auth, bucket, artifactBucket string) (*KVMaterialStore, error) {
	nc, err := connectStore(ctx, rt, auth)
	if err != nil {
		return nil, err
	}
	return OpenKVMaterialStore(nc, bucket, artifactBucket)
}

// OpenKVMaterialStore opens the material and artifact buckets over a
// caller-supplied connection (operator-mode assemblies connect with minted
// creds, where the static rt.Connect path does not exist). The store owns nc.
func OpenKVMaterialStore(nc *nats.Conn, bucket, artifactBucket string) (*KVMaterialStore, error) {
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, core.MaterialErr(core.JetStreamUnavailable, "OpenMaterialStore", "JetStream context is unavailable", nil, err)
	}
	kv, err := js.KeyValue(bucket)
	if errors.Is(err, nats.ErrBucketNotFound) {
		kv, err = js.CreateKeyValue(&nats.KeyValueConfig{Bucket: bucket, Storage: nats.FileStorage})
	}
	if err != nil {
		nc.Close()
		return nil, core.MaterialErr(core.BucketMissing, "OpenMaterialStore", "material KV bucket is unavailable", nil, err)
	}
	obj, err := js.ObjectStore(artifactBucket)
	if errors.Is(err, nats.ErrBucketNotFound) || errors.Is(err, nats.ErrStreamNotFound) {
		obj, err = js.CreateObjectStore(&nats.ObjectStoreConfig{Bucket: artifactBucket, Storage: nats.FileStorage})
	}
	if err != nil {
		nc.Close()
		return nil, core.MaterialErr(core.BucketMissing, "OpenMaterialStore", "artifact Object Store bucket is unavailable", nil, err)
	}
	return &KVMaterialStore{nc: nc, kv: kv, obj: obj, bucket: bucket, artifactBucket: artifactBucket}, nil
}

func (s *KVMaterialStore) Close() {
	if s != nil && s.nc != nil {
		s.nc.Close()
	}
}

func (s *KVMaterialStore) Bucket() string {
	return s.bucket
}

func (s *KVMaterialStore) ArtifactBucket() string {
	return s.artifactBucket
}

func (s *KVMaterialStore) SaveProjection(proj core.MaterialProjection) error {
	key := "p." + proj.ProjectionID
	entry, err := s.kv.Get(key)
	if err == nil {
		var cur core.MaterialProjection
		if err := json.Unmarshal(entry.Value(), &cur); err == nil && proj.Sequence <= cur.Sequence {
			return core.MaterialErr(core.ProjectionConflict, "SaveProjection", "projection sequence is stale", map[string]string{"projectionId": proj.ProjectionID}, nil)
		}
	} else if !errors.Is(err, nats.ErrKeyNotFound) {
		return core.MaterialErr(core.MaterialWriteFailed, "SaveProjection", "projection could not be read", nil, err)
	}
	body, err := json.Marshal(proj)
	if err != nil {
		return core.MaterialErr(core.MaterialWriteFailed, "SaveProjection", "projection could not be encoded", nil, err)
	}
	if _, err := s.kv.Put(key, body); err != nil {
		return core.MaterialErr(core.MaterialWriteFailed, "SaveProjection", "projection could not be written", nil, err)
	}
	return nil
}

func (s *KVMaterialStore) SaveArtifact(art core.MaterialArtifact) error {
	if art.Name == "" {
		return core.MaterialErr(core.ArtifactWriteFailed, "SaveArtifact", "artifact name is required", nil, nil)
	}
	sum := sha256.Sum256(art.Body)
	art.Digest = "sha256:" + fmt.Sprintf("%x", sum[:])
	if _, err := s.obj.PutBytes(art.Name, art.Body); err != nil {
		return core.MaterialErr(core.ArtifactWriteFailed, "SaveArtifact", "artifact could not be written", map[string]string{"artifactName": art.Name}, err)
	}
	body, err := json.Marshal(art)
	if err != nil {
		return core.MaterialErr(core.ArtifactWriteFailed, "SaveArtifact", "artifact manifest could not be encoded", map[string]string{"artifactName": art.Name}, err)
	}
	if _, err := s.kv.Put("a."+keyEnc(art.Name), body); err != nil {
		return core.MaterialErr(core.ArtifactWriteFailed, "SaveArtifact", "artifact manifest could not be written", map[string]string{"artifactName": art.Name}, err)
	}
	return nil
}

// LoadArtifact returns the stored artifact manifest and body by artifact
// name; ok is false when no such artifact has materialized.
func (s *KVMaterialStore) LoadArtifact(name string) (core.MaterialArtifact, []byte, bool, error) {
	entry, err := s.kv.Get("a." + keyEnc(name))
	if errors.Is(err, nats.ErrKeyNotFound) {
		return core.MaterialArtifact{}, nil, false, nil
	}
	if err != nil {
		return core.MaterialArtifact{}, nil, false, core.MaterialErr(core.MaterialWriteFailed, "LoadArtifact", "artifact manifest could not be read", map[string]string{"artifactName": name}, err)
	}
	var art core.MaterialArtifact
	if err := json.Unmarshal(entry.Value(), &art); err != nil {
		return core.MaterialArtifact{}, nil, false, core.MaterialErr(core.MaterialWriteFailed, "LoadArtifact", "artifact manifest could not be decoded", map[string]string{"artifactName": name}, err)
	}
	body, err := s.obj.GetBytes(name)
	if err != nil {
		return core.MaterialArtifact{}, nil, false, core.MaterialErr(core.ArtifactWriteFailed, "LoadArtifact", "artifact body could not be read", map[string]string{"artifactName": name}, err)
	}
	art.Name = name
	return art, body, true, nil
}

// LoadProjection returns the stored projection record JSON by id; ok is
// false when no such projection has materialized.
func (s *KVMaterialStore) LoadProjection(id string) ([]byte, bool, error) {
	entry, err := s.kv.Get("p." + id)
	if errors.Is(err, nats.ErrKeyNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, core.MaterialErr(core.MaterialWriteFailed, "LoadProjection", "projection could not be read", map[string]string{"projectionId": id}, err)
	}
	return entry.Value(), true, nil
}

func (s *KVMaterialStore) SaveEvent(ev core.EventEnvelope) error {
	body, err := json.Marshal(ev)
	if err != nil {
		return core.MaterialErr(core.MaterialWriteFailed, "SaveEvent", "event could not be encoded", nil, err)
	}
	if _, err := s.kv.Put("e."+ev.EventID, body); err != nil {
		return core.MaterialErr(core.MaterialWriteFailed, "SaveEvent", "event could not be written", nil, err)
	}
	return nil
}

func (s *KVMaterialStore) ClaimRun(acc core.AcceptedActivation, rec core.ScriptRecord) (bool, error) {
	body, err := json.Marshal(map[string]any{
		"kind":           "script.run.claim",
		"activationId":   acc.Activation.ActivationID,
		"scriptKey":      rec.Key,
		"scriptRevision": rec.Revision,
		"observedAt":     acc.Activation.Provenance.CreatedAt,
	})
	if err != nil {
		return false, core.MaterialErr(core.MaterialWriteFailed, "ClaimRun", "run claim could not be encoded", nil, err)
	}
	if _, err := s.kv.Create("r."+runID(acc.Activation, rec), body); err != nil {
		if errors.Is(err, nats.ErrKeyExists) {
			return false, nil
		}
		return false, core.MaterialErr(core.MaterialWriteFailed, "ClaimRun", "run claim could not be written", nil, err)
	}
	return true, nil
}

func (r LocalScriptRunner) Run(inv core.ScriptInvocation) (core.ScriptRun, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(inv.Record.Process.TimeoutMs)*time.Millisecond)
	defer cancel()

	// Per-run output dir for path-artifacts; removed after its files are read.
	outDir, err := os.MkdirTemp("", "tb-artifact-out-")
	if err != nil {
		return core.ScriptRun{}, core.ProtocolErr("Run", "artifact output dir could not be created", err)
	}
	defer os.RemoveAll(outDir)

	command, args := inv.Record.Process.Command, inv.Record.Process.Args
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = inv.Record.Process.Cwd
	if r.Sandbox != nil {
		// bwrap --chdir owns the working dir, so the outer cmd.Dir stays empty.
		command, args = sandboxCommand(*r.Sandbox, command, args, inv.Record.Process.Cwd, outDir)
		cmd = exec.CommandContext(ctx, command, args...)
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	for k, v := range inv.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	cmd.Env = append(cmd.Env, "TB_ARTIFACT_OUT="+outDir)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return core.ScriptRun{}, err
	}
	if err := cmd.Start(); err != nil {
		return core.ScriptRun{}, err
	}
	out, readErr := io.ReadAll(io.LimitReader(stdout, maxScriptOutput+1))
	waitErr := cmd.Wait()
	if ctx.Err() != nil {
		killGroup(cmd)
		_ = cleanup(inv.Record.Process)
		return core.ScriptRun{}, ctx.Err()
	}
	if err := cleanup(inv.Record.Process); err != nil {
		return core.ScriptRun{CleanupErr: err}, nil
	}
	if readErr != nil {
		return core.ScriptRun{}, readErr
	}
	if len(out) > maxScriptOutput {
		return core.ScriptRun{}, core.ProtocolErr("ReadFrame", "script stdout exceeded limit", nil)
	}
	if waitErr != nil {
		return core.ScriptRun{}, waitErr
	}
	effects, err := frames(out)
	if err != nil {
		return core.ScriptRun{}, err
	}
	for i := range effects {
		if err := resolveArtifactPath(outDir, &effects[i]); err != nil {
			return core.ScriptRun{}, err
		}
	}
	return core.ScriptRun{Status: "applied", Effects: effects}, nil
}

// resolveArtifactPath reads a path-artifact's file into its Body. Resolution
// lives in the runner, not the materializer, because only the process boundary
// knows the per-run output dir it handed the child via TB_ARTIFACT_OUT.
//
// This read runs host-side, where the whole FS is reachable, so a jailed script
// can plant `ln -s <secret> $TB_ARTIFACT_OUT/x` and exfiltrate the target. The
// string-confinement below bounds the path; os.OpenInRoot then refuses to
// FOLLOW any symlink out of outDir (it resolves every component within the
// root), closing the no-follow hole. The size cap stops an OOM via a huge file.
func resolveArtifactPath(outDir string, eff *core.ScriptEffect) error {
	if eff.Type != core.ArtifactEffect || eff.Path == "" {
		return nil
	}
	// Confine the script-supplied path under the run's output dir: a cleaned
	// relative join must still live inside outDir, so "../" cannot escape.
	target := filepath.Join(outDir, filepath.Clean("/"+eff.Path))
	prefix := outDir + string(os.PathSeparator)
	if target != outDir && !strings.HasPrefix(target, prefix) {
		return core.ProtocolErr("ReadFrame", "artifact path escapes output dir", nil)
	}
	// Resolve relative to outDir: OpenInRoot blocks symlink traversal that would
	// leave the root, so a symlink planted in outDir cannot read a host file.
	rel := strings.TrimPrefix(target, prefix)
	f, err := os.OpenInRoot(outDir, rel)
	if err != nil {
		return core.ProtocolErr("ReadFrame", "artifact path could not be read", err)
	}
	defer f.Close()
	if info, err := f.Stat(); err == nil && info.Size() > maxArtifactFile {
		return core.ProtocolErr("ReadFrame", "artifact path body exceeded limit", nil)
	}
	body, err := io.ReadAll(io.LimitReader(f, maxArtifactFile+1))
	if err != nil {
		return core.ProtocolErr("ReadFrame", "artifact path could not be read", err)
	}
	if len(body) > maxArtifactFile {
		return core.ProtocolErr("ReadFrame", "artifact path body exceeded limit", nil)
	}
	eff.Body = body
	eff.Path = ""
	return nil
}

func NewScriptLoop(store interface {
	LoadScript(string) (core.ScriptRecord, bool, error)
}, runtime *core.ScriptRuntime, materializer *core.Materializer, status core.StatusSink, claims runClaimer) *ScriptLoop {
	return &ScriptLoop{store: store, runtime: runtime, materializer: materializer, status: status, claims: claims}
}

func (l *ScriptLoop) Watch(in <-chan RouterResult) (<-chan ScriptRunResult, func()) {
	out := make(chan ScriptRunResult, 16)
	stop := make(chan struct{})
	go func() {
		defer close(out)
		for {
			select {
			case <-stop:
				return
			case res, ok := <-in:
				if !ok {
					return
				}
				run := ScriptRunResult{Activation: res.Activation, Record: res.Record, Err: res.Err}
				if res.Err == nil && res.Record.Status == core.Accepted {
					rec, ok, err := l.store.LoadScript(res.Activation.ScriptKey)
					if err != nil {
						run.Err = err
					} else if !ok {
						run.Err = core.ScriptRecordErr(core.ScriptRecordMissing, "LoadScript", "script record is missing", map[string]string{"scriptKey": res.Activation.ScriptKey}, nil)
					} else {
						acc := core.AcceptedActivation{Activation: res.Activation, Record: res.Record}
						if l.claims == nil {
							run.Err = core.MaterialErr(core.ExecutionStateInvalid, "ClaimRun", "run claimer is required", nil, nil)
						} else if claimed, err := l.claims.ClaimRun(acc, rec); err != nil {
							run.Err = err
						} else if !claimed {
							run.Run = core.ScriptRun{ActivationID: res.Activation.ActivationID, Status: "duplicate"}
						}
						if run.Err == nil && run.Run.Status != "duplicate" {
							run.Run, run.Err = l.runtime.Run(acc, rec)
							if run.Err == nil {
								run.Err = l.materializer.Apply(core.MaterialContext{Accepted: acc, Record: rec}, run.Run.Effects)
							}
						}
					}
				}
				if l.status != nil && run.Run.Status != "duplicate" && (res.Record.Status == core.Accepted || run.Err != nil) {
					if err := l.status.SaveEvent(eventFor(run)); err != nil && run.Err == nil {
						run.Err = err
					}
				}
				select {
				case out <- run:
				case <-stop:
					return
				}
			}
		}
	}()
	return out, func() { close(stop) }
}

type frame struct {
	Kind             string          `json:"kind"`
	Type             core.EffectType `json:"effectType"`
	ProjectionID     string          `json:"projectionId"`
	SnapshotRevision string          `json:"snapshotRevision"`
	ArtifactRevision string          `json:"artifactRevision"`
	Sequence         int             `json:"sequence"`
	Value            json.RawMessage `json:"value"`
	ArtifactName     string          `json:"artifactName"`
	MediaType        string          `json:"mediaType"`
	Body             string          `json:"body"`
	Path             string          `json:"path"`
	Subject          string          `json:"subject"`
}

func frames(out []byte) ([]core.ScriptEffect, error) {
	rest := bytes.TrimSpace(out)
	effects := []core.ScriptEffect{}
	for len(rest) > 0 {
		body, next, err := nextFrame(rest)
		if err != nil {
			return nil, err
		}
		rest = bytes.TrimSpace(next)

		if err := checkFrameShape(body); err != nil {
			return nil, err
		}
		var f frame
		if err := decodeStrict(body, &f); err != nil {
			return nil, core.ProtocolErr("ReadFrame", "script frame is malformed", err)
		}
		if f.Kind != "script.effect" {
			return nil, core.ProtocolErr("ReadFrame", "script frame kind is invalid", nil)
		}
		effects = append(effects, core.ScriptEffect{
			Type:             f.Type,
			ProjectionID:     f.ProjectionID,
			SnapshotRevision: f.SnapshotRevision,
			ArtifactRevision: f.ArtifactRevision,
			Sequence:         f.Sequence,
			Value:            f.Value,
			ArtifactName:     f.ArtifactName,
			MediaType:        f.MediaType,
			Body:             []byte(f.Body),
			Path:             f.Path,
			Subject:          f.Subject,
		})
	}
	return effects, nil
}

func checkFrameShape(body []byte) error {
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(body, &doc); err != nil {
		return core.ProtocolErr("ReadFrame", "script frame is malformed", err)
	}
	var typ core.EffectType
	if raw, ok := doc["effectType"]; ok {
		_ = json.Unmarshal(raw, &typ)
	}
	allowed := map[string]bool{"kind": true, "effectType": true}
	switch typ {
	case core.ProjectionEffect:
		for _, k := range []string{"projectionId", "snapshotRevision", "artifactRevision", "sequence", "value"} {
			allowed[k] = true
		}
	case core.ArtifactEffect:
		for _, k := range []string{"artifactName", "artifactRevision", "mediaType", "body", "path"} {
			allowed[k] = true
		}
	case core.PublishEffect:
		for _, k := range []string{"subject", "body"} {
			allowed[k] = true
		}
	default:
		return core.ProtocolErr("ReadFrame", "script frame effect type is invalid", nil)
	}
	for k := range doc {
		if !allowed[k] {
			return core.ProtocolErr("ReadFrame", "script frame field does not belong to effect type", nil)
		}
	}
	return nil
}

func nextFrame(in []byte) ([]byte, []byte, error) {
	i := bytes.Index(in, []byte("\r\n\r\n"))
	step := len("\r\n\r\n")
	if i < 0 {
		i = bytes.Index(in, []byte("\n\n"))
		step = len("\n\n")
	}
	if i < 0 {
		return nil, nil, core.ProtocolErr("ReadFrame", "script frame header is missing", nil)
	}
	n, err := contentLength(in[:i])
	if err != nil {
		return nil, nil, err
	}
	start := i + step
	end := start + n
	if n < 0 || len(in) < end {
		return nil, nil, core.ProtocolErr("ReadFrame", "script frame body is incomplete", nil)
	}
	if n > maxFrameBody {
		return nil, nil, core.ProtocolErr("ReadFrame", "script frame body exceeded limit", nil)
	}
	return in[start:end], in[end:], nil
}

func contentLength(header []byte) (int, error) {
	for _, line := range strings.Split(string(header), "\n") {
		name, val, ok := strings.Cut(strings.TrimSpace(strings.TrimSuffix(line, "\r")), ":")
		if !ok || !strings.EqualFold(name, "Content-Length") {
			continue
		}
		n, err := strconv.Atoi(strings.TrimSpace(val))
		if err != nil || n < 0 {
			return 0, core.ProtocolErr("ReadFrame", "script frame content length is invalid", err)
		}
		return n, nil
	}
	return 0, core.ProtocolErr("ReadFrame", "script frame content length is missing", nil)
}

func decodeStrict(raw []byte, out any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return err
	}
	if dec.Decode(&struct{}{}) != io.EOF {
		return errors.New("extra JSON after document")
	}
	return nil
}

func killGroup(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}

func cleanup(proc core.Process) error {
	if proc.Cleanup != "workdir.delete" {
		return nil
	}
	abs, err := filepath.Abs(proc.Cwd)
	if err != nil {
		return err
	}
	tmp, err := filepath.Abs(os.TempDir())
	if err != nil {
		return err
	}
	if abs == tmp || !strings.HasPrefix(abs, tmp+string(os.PathSeparator)) {
		return fmt.Errorf("cleanup path outside temp dir: %s", abs)
	}
	return os.RemoveAll(abs)
}

func connectStore(ctx context.Context, rt *Runtime, auth core.Auth) (*nats.Conn, error) {
	if auth.User == "" {
		return rt.Connect(ctx)
	}
	return rt.ConnectAs(ctx, auth)
}

func eventFor(run ScriptRunResult) core.EventEnvelope {
	status := "success"
	var errDoc *core.EventError
	if run.Err != nil {
		status = "failed"
		errDoc = errorDoc(run.Err)
	}
	return core.EventEnvelope{
		Kind:         "event.envelope",
		EventID:      "script_run_" + runID(run.Activation, core.ScriptRecord{Revision: run.Activation.ScriptRevision}),
		EventType:    "script.run",
		Status:       status,
		PrincipalID:  run.Activation.Capability.PrincipalID,
		CapabilityID: run.Activation.Capability.CapabilityID,
		Chain:        run.Activation.Chain,
		ObservedAt:   run.Activation.Provenance.CreatedAt,
		Provenance:   run.Activation.Provenance,
		Store:        "material",
		Revisions: map[string]string{
			"script": strconv.Itoa(run.Activation.ScriptRevision),
		},
		Error: errDoc,
	}
}

func runID(act core.Activation, rec core.ScriptRecord) string {
	return keyEnc(act.ActivationID) + "_" + strconv.Itoa(rec.Revision)
}

func errorDoc(err error) *core.EventError {
	var got *core.Error
	if errors.As(err, &got) {
		return &core.EventError{
			Kind:    string(got.Kind),
			Message: got.Message,
			Origin:  core.ErrorOrigin{Layer: got.Layer, Operation: got.Operation},
		}
	}
	return &core.EventError{
		Kind:    string(core.ScriptProcessFailed),
		Message: err.Error(),
		Origin:  core.ErrorOrigin{Layer: "ScriptRuntime", Operation: "Run"},
	}
}

var _ core.MaterialStore = (*KVMaterialStore)(nil)
var _ core.StatusSink = (*KVMaterialStore)(nil)
