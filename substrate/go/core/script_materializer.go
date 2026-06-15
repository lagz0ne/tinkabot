package core

import (
	"encoding/json"
	"errors"
	"slices"
	"strconv"
	"strings"
	"sync"
)

const (
	ScriptRecordMissing    Kind = "ScriptRecordMissing"
	ScriptRevisionMismatch Kind = "ScriptRevisionMismatch"
	ScriptProcessFailed    Kind = "ScriptProcessFailed"
	ProtocolFrameInvalid   Kind = "ProtocolFrameInvalid"
	FacadeEffectDenied     Kind = "FacadeEffectDenied"
	ProjectionConflict     Kind = "ProjectionConflict"
	MaterialWriteFailed    Kind = "MaterialWriteFailed"
	ArtifactWriteFailed    Kind = "ArtifactWriteFailed"
	ExecutionStateInvalid  Kind = "ExecutionStateInvalid"
)

var rawWords = []string{
	"subject",
	"reply",
	"token",
	"cred",
	"permission",
	"publish",
	"subscribe",
	"nats",
	"nkey",
	"jwt",
	"seed",
	"secret",
	"password",
	"bearer",
}

type EffectType string

const (
	ProjectionEffect EffectType = "projection"
	ArtifactEffect   EffectType = "artifact"
	PublishEffect    EffectType = "publish"
)

type ScriptRecord struct {
	Kind     string  `json:"kind"`
	Key      string  `json:"scriptKey"`
	Revision int     `json:"scriptRevision"`
	Desc     string  `json:"desc,omitempty"`
	Process  Process `json:"process"`
}

type ScriptPolicy struct {
	AllowedProjections []string
	ProjectionPrefix   string
	ArtifactPrefix     string
	Env                map[string]string
}

type ScriptInvocation struct {
	Accepted AcceptedActivation
	Record   ScriptRecord
	Env      map[string]string
}

type ScriptEffect struct {
	Type             EffectType
	ProjectionID     string
	SnapshotRevision string
	ArtifactRevision string
	Sequence         int
	Value            json.RawMessage
	ArtifactName     string
	MediaType        string
	Body             []byte
	// Path is a runner-relative filename under TB_ARTIFACT_OUT; the runner
	// reads it into Body before the effect leaves the process boundary, so an
	// artifact carries EITHER an inline Body OR a Path, never a body on stdout.
	Path    string
	Subject string
}

type ScriptRun struct {
	ActivationID string
	Status       string
	Effects      []ScriptEffect
	CleanupErr   error
}

type ScriptRunner interface {
	Run(ScriptInvocation) (ScriptRun, error)
}

type ScriptRunnerFunc func(ScriptInvocation) (ScriptRun, error)

func (f ScriptRunnerFunc) Run(inv ScriptInvocation) (ScriptRun, error) {
	return f(inv)
}

type MaterialProjection struct {
	Kind             string          `json:"kind"`
	ProjectionID     string          `json:"projectionId"`
	SnapshotRevision string          `json:"snapshotRevision"`
	ArtifactRevision string          `json:"artifactRevision"`
	Sequence         int             `json:"sequence"`
	Value            json.RawMessage `json:"value"`
	ObservedAt       string          `json:"observedAt"`
	Provenance       Provenance      `json:"provenance"`
}

type MaterialArtifact struct {
	Kind             string     `json:"kind"`
	ArtifactID       string     `json:"artifactId"`
	ArtifactRevision string     `json:"artifactRevision"`
	Digest           string     `json:"digest"`
	MediaType        string     `json:"mediaType"`
	ObjectRef        string     `json:"objectRef"`
	SandboxPolicy    string     `json:"sandboxPolicy"`
	CreatedAt        string     `json:"createdAt"`
	Provenance       Provenance `json:"provenance"`
	Name             string     `json:"-"`
	Body             []byte     `json:"-"`
}

type MaterialStore interface {
	SaveProjection(MaterialProjection) error
	SaveArtifact(MaterialArtifact) error
}

type MaterialContext struct {
	Accepted AcceptedActivation
	Record   ScriptRecord
}

type EventEnvelope struct {
	Kind         string            `json:"kind"`
	EventID      string            `json:"eventId"`
	EventType    string            `json:"eventType"`
	Status       string            `json:"status"`
	PrincipalID  string            `json:"principalId"`
	CapabilityID string            `json:"capabilityId"`
	Chain        Chain             `json:"chain"`
	ObservedAt   string            `json:"observedAt"`
	Provenance   Provenance        `json:"provenance"`
	Store        string            `json:"store,omitempty"`
	Revisions    map[string]string `json:"revisions"`
	Error        *EventError       `json:"error,omitempty"`
}

type EventError struct {
	Kind    string      `json:"kind"`
	Message string      `json:"message"`
	Origin  ErrorOrigin `json:"origin"`
}

type ErrorOrigin struct {
	Layer     string `json:"layer"`
	Operation string `json:"operation"`
}

type StatusSink interface {
	SaveEvent(EventEnvelope) error
}

type AcceptedActivation struct {
	Activation Activation
	Record     LedgerRecord
}

// MemoryMaterialStore is a test fake (allowlisted in substrate/go/fakes-allowlist.json):
// it exists to seed projection-conflict state synchronously and force the
// ProjectionConflict and ExecutionStateInvalid branches inside the core layer.
// The real path is proven by TestScriptMaterializerLoopFromNATSCLI over the
// embednats KVMaterialStore (JetStream KV + Object Store, read via nats CLI).
type MemoryMaterialStore struct {
	mu          sync.Mutex
	projections map[string]MaterialProjection
	artifacts   map[string]MaterialArtifact
}

type ScriptRuntime struct {
	policy ScriptPolicy
	run    ScriptRunner
}

func NewMemoryMaterialStore() *MemoryMaterialStore {
	return &MemoryMaterialStore{
		projections: map[string]MaterialProjection{},
		artifacts:   map[string]MaterialArtifact{},
	}
}

func NewScriptRuntime(policy ScriptPolicy, runner ScriptRunner) (*ScriptRuntime, error) {
	if runner == nil {
		return nil, fail(ProcessConfigInvalid, "ScriptRuntime", "Configure", "runner is required", nil)
	}
	return &ScriptRuntime{policy: policy, run: runner}, nil
}

type Materializer struct {
	store MaterialStore
}

func NewMaterializer(store MaterialStore) (*Materializer, error) {
	if store == nil {
		return nil, fail(ProcessConfigInvalid, "Materializer", "Configure", "material store is required", nil)
	}
	return &Materializer{store: store}, nil
}

func (m *Materializer) Apply(ctx MaterialContext, effects []ScriptEffect) error {
	if m == nil {
		return fail(ProcessConfigInvalid, "Materializer", "Apply", "materializer is nil", nil)
	}
	if err := checkMaterialContext(ctx); err != nil {
		return err
	}
	act := ctx.Accepted.Activation
	for _, eff := range effects {
		switch eff.Type {
		case ProjectionEffect:
			if err := m.store.SaveProjection(MaterialProjection{
				Kind:             "material.projection",
				ProjectionID:     eff.ProjectionID,
				SnapshotRevision: eff.SnapshotRevision,
				ArtifactRevision: eff.ArtifactRevision,
				Sequence:         eff.Sequence,
				Value:            eff.Value,
				ObservedAt:       observedAt(act),
				Provenance:       materialProvenance(act, "script-materializer"),
			}); err != nil {
				return err
			}
		case ArtifactEffect:
			if err := m.store.SaveArtifact(MaterialArtifact{
				Kind:             "artifact.manifest",
				ArtifactID:       eff.ArtifactName,
				ArtifactRevision: eff.ArtifactRevision,
				MediaType:        eff.MediaType,
				ObjectRef:        "obj://script/" + eff.ArtifactName,
				SandboxPolicy:    "sandbox.script.local.v1",
				CreatedAt:        observedAt(act),
				Provenance:       materialProvenance(act, "script-materializer"),
				Name:             eff.ArtifactName,
				Body:             eff.Body,
			}); err != nil {
				return err
			}
		default:
			return fail(ProtocolFrameInvalid, "Materializer", "Apply", "materializer cannot apply effect type", map[string]string{"type": string(eff.Type)})
		}
	}
	return nil
}

func checkMaterialContext(ctx MaterialContext) error {
	act := ctx.Accepted.Activation
	rec := ctx.Record
	if ctx.Accepted.Record.Status != Accepted || ctx.Accepted.Record.ActivationID != act.ActivationID {
		return fail(ExecutionStateInvalid, "Materializer", "Apply", "accepted activation record is required", map[string]string{"activationId": act.ActivationID})
	}
	if rec.Kind != "script.record" || rec.Key == "" || rec.Key != act.ScriptKey {
		return fail(ScriptRecordMissing, "Materializer", "Apply", "script record does not match activation", map[string]string{"scriptKey": act.ScriptKey})
	}
	if act.ScriptRevision != 0 && rec.Revision != act.ScriptRevision {
		return fail(ScriptRevisionMismatch, "Materializer", "Apply", "script revision does not match activation", map[string]string{"scriptKey": act.ScriptKey, "want": strconv.Itoa(act.ScriptRevision), "got": strconv.Itoa(rec.Revision)})
	}
	return nil
}

func observedAt(act Activation) string {
	return act.Provenance.CreatedAt
}

func materialProvenance(act Activation, producer string) Provenance {
	prov := act.Provenance
	prov.Producer = producer
	return prov
}

func (r *ScriptRuntime) Run(acc AcceptedActivation, rec ScriptRecord) (ScriptRun, error) {
	if r == nil {
		return ScriptRun{}, fail(ProcessConfigInvalid, "ScriptRuntime", "Run", "script runtime is nil", nil)
	}
	act := acc.Activation
	if acc.Record.Status != Accepted || acc.Record.ActivationID != act.ActivationID {
		return ScriptRun{}, fail(ExecutionStateInvalid, "ScriptRuntime", "Run", "accepted activation record is required", map[string]string{"activationId": act.ActivationID})
	}
	if rec.Kind != "script.record" {
		return ScriptRun{}, fail(ProtocolFrameInvalid, "ScriptRuntime", "LoadScript", "script record kind is invalid", map[string]string{"kind": rec.Kind})
	}
	if rec.Key == "" || rec.Key != act.ScriptKey {
		return ScriptRun{}, fail(ScriptRecordMissing, "ScriptRuntime", "LoadScript", "script record is missing", map[string]string{"scriptKey": act.ScriptKey})
	}
	if act.ScriptRevision != 0 && rec.Revision != act.ScriptRevision {
		return ScriptRun{}, fail(ScriptRevisionMismatch, "ScriptRuntime", "LoadScript", "script revision does not match activation", map[string]string{"scriptKey": act.ScriptKey, "want": strconv.Itoa(act.ScriptRevision), "got": strconv.Itoa(rec.Revision)})
	}
	if _, err := CheckProcess(rec.Process); err != nil {
		return ScriptRun{}, err
	}

	run, err := r.run.Run(ScriptInvocation{Accepted: acc, Record: rec, Env: r.env(act, rec)})
	if err != nil {
		var owned *Error
		if errors.As(err, &owned) {
			return ScriptRun{}, err
		}
		return ScriptRun{}, fail(ScriptProcessFailed, "ScriptRuntime", "Run", "script process failed", map[string]string{"cause": err.Error()})
	}
	if run.CleanupErr != nil {
		return ScriptRun{}, fail(CleanupFailed, "ProcessBoundary", "Cleanup", "script cleanup failed", map[string]string{"cause": run.CleanupErr.Error()})
	}
	run.ActivationID = act.ActivationID
	if run.Status == "" {
		run.Status = "applied"
	}
	for i := range run.Effects {
		if err := r.Allow(&run.Effects[i]); err != nil {
			return ScriptRun{}, err
		}
	}
	return run, nil
}

func (r *ScriptRuntime) env(act Activation, rec ScriptRecord) map[string]string {
	env := map[string]string{
		"TB_ACTIVATION_ID": act.ActivationID,
		"TB_SCRIPT_KEY":    rec.Key,
		"TB_SCRIPT_REV":    strconv.Itoa(rec.Revision),
	}
	for k, v := range r.policy.Env {
		if rawEnv(k) {
			continue
		}
		env[k] = v
	}
	return env
}

// Allow resolves an effect's short/relative refs to the bundle's derived
// global names, then checks it against the policy. It mutates eff in place so
// the effect that gets materialized carries the resolved names. Both apply
// paths (ScriptRuntime.Run and the filter loop) call it before materializing.
func (r *ScriptRuntime) Allow(eff *ScriptEffect) error {
	r.resolve(eff)
	return r.allow(*eff)
}

// resolve prefixes a short projection id or relative artifact name to its
// derived form. The already-prefixed guard is backward compat: a script that
// emits the full derived name keeps working untouched.
func (r *ScriptRuntime) resolve(eff *ScriptEffect) {
	switch eff.Type {
	case ProjectionEffect:
		if !strings.HasPrefix(eff.ProjectionID, r.policy.ProjectionPrefix) {
			eff.ProjectionID = r.policy.ProjectionPrefix + eff.ProjectionID
		}
	case ArtifactEffect:
		if r.policy.ArtifactPrefix != "" && !strings.HasPrefix(eff.ArtifactName, r.policy.ArtifactPrefix) {
			eff.ArtifactName = r.policy.ArtifactPrefix + eff.ArtifactName
		}
	}
}

func (r *ScriptRuntime) allow(eff ScriptEffect) error {
	switch eff.Type {
	case ProjectionEffect:
		if eff.ProjectionID == "" || eff.SnapshotRevision == "" || eff.ArtifactRevision == "" || eff.Sequence < 0 || len(eff.Value) == 0 {
			return fail(ProtocolFrameInvalid, "ScriptRuntime", "ApplyEffect", "projection effect is incomplete", map[string]string{"projectionId": eff.ProjectionID})
		}
		if !slices.Contains(r.policy.AllowedProjections, eff.ProjectionID) {
			return fail(FacadeEffectDenied, "ScriptFacade", "ApplyEffect", "projection is not allowed", map[string]string{"projectionId": eff.ProjectionID})
		}
		if err := rawValue(eff.Value); err != nil {
			return err
		}
		return nil
	case ArtifactEffect:
		// An artifact is complete with EITHER an inline Body OR a Path the
		// runner will resolve to bytes; require one, not both.
		if eff.ArtifactName == "" || eff.ArtifactRevision == "" || eff.MediaType == "" || (len(eff.Body) == 0 && eff.Path == "") {
			return fail(ProtocolFrameInvalid, "ScriptRuntime", "ApplyEffect", "artifact effect is incomplete", map[string]string{"artifactName": eff.ArtifactName})
		}
		if r.policy.ArtifactPrefix != "" && !strings.HasPrefix(eff.ArtifactName, r.policy.ArtifactPrefix) {
			return fail(FacadeEffectDenied, "ScriptFacade", "ApplyEffect", "artifact path is not allowed", map[string]string{"artifactName": eff.ArtifactName})
		}
		return nil
	case PublishEffect:
		return fail(FacadeEffectDenied, "ScriptFacade", "ApplyEffect", "raw publish effect is denied by default", map[string]string{"subject": eff.Subject})
	default:
		return fail(ProtocolFrameInvalid, "ScriptRuntime", "ApplyEffect", "script effect type is invalid", map[string]string{"type": string(eff.Type)})
	}
}

func (s *MemoryMaterialStore) SaveProjection(proj MaterialProjection) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if proj.ProjectionID == "" {
		return fail(MaterialWriteFailed, "Materializer", "SaveProjection", "projection id is required", nil)
	}
	if cur, ok := s.projections[proj.ProjectionID]; ok && proj.Sequence <= cur.Sequence {
		return fail(ProjectionConflict, "Materializer", "SaveProjection", "projection sequence is stale", map[string]string{"projectionId": proj.ProjectionID})
	}
	s.projections[proj.ProjectionID] = proj
	return nil
}

func (s *MemoryMaterialStore) SaveArtifact(art MaterialArtifact) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if art.Name == "" {
		return fail(ArtifactWriteFailed, "Materializer", "SaveArtifact", "artifact name is required", nil)
	}
	s.artifacts[art.Name] = art
	return nil
}

func (s *MemoryMaterialStore) Projection(id string) (MaterialProjection, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	proj, ok := s.projections[id]
	return proj, ok
}

func (s *MemoryMaterialStore) Artifact(name string) (MaterialArtifact, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	art, ok := s.artifacts[name]
	return art, ok
}

func rawEnv(k string) bool {
	return rawName(k)
}

func rawValue(raw json.RawMessage) error {
	var v any
	if len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return fail(ProtocolFrameInvalid, "ScriptRuntime", "ApplyEffect", "projection value is malformed", map[string]string{"cause": err.Error()})
	}
	if rawKey(v) {
		return fail(FacadeEffectDenied, "ScriptFacade", "ApplyEffect", "raw NATS vocabulary is denied", nil)
	}
	return nil
}

func rawKey(v any) bool {
	switch x := v.(type) {
	case map[string]any:
		for k, v := range x {
			if rawName(k) {
				return true
			}
			if rawKey(v) {
				return true
			}
		}
	case []any:
		for _, v := range x {
			if rawKey(v) {
				return true
			}
		}
	}
	return false
}

func rawName(s string) bool {
	name := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(s, "_", ""), "-", ""))
	for _, word := range rawWords {
		if strings.Contains(name, word) {
			return true
		}
	}
	return false
}

func MaterialErr(kind Kind, op, msg string, details map[string]string, cause error) *Error {
	if details == nil {
		details = map[string]string{}
	}
	if cause != nil {
		details["cause"] = cause.Error()
	}
	return fail(kind, "Materializer", op, msg, details)
}

func ProtocolErr(op, msg string, cause error) *Error {
	details := map[string]string{}
	if cause != nil {
		details["cause"] = cause.Error()
	}
	return fail(ProtocolFrameInvalid, "ScriptRuntime", op, msg, details)
}

func ScriptRecordErr(kind Kind, op, msg string, details map[string]string, cause error) *Error {
	if details == nil {
		details = map[string]string{}
	}
	if cause != nil {
		details["cause"] = cause.Error()
	}
	return fail(kind, "ScriptRuntime", op, msg, details)
}
