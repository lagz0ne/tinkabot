package core

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lagz0ne/tinkabot/substrate/go/contract"
)

type Kind string

const (
	TopologyInvalid         Kind = "TopologyInvalid"
	NATSUnavailable         Kind = "NATSUnavailable"
	JetStreamUnavailable    Kind = "JetStreamUnavailable"
	QuorumUnavailable       Kind = "QuorumUnavailable"
	SubstrateCritical       Kind = "SubstrateCritical"
	AuthRenderInvalid       Kind = "AuthRenderInvalid"
	WildcardOverreach       Kind = "WildcardOverreach"
	PermissionCompileFailed Kind = "PermissionCompileFailed"
	LeaseMintDenied         Kind = "LeaseMintDenied"
	LeaseRevoked            Kind = "LeaseRevoked"
	LeaseExpired            Kind = "LeaseExpired"
	BucketMissing           Kind = "BucketMissing"
	KeyMissing              Kind = "KeyMissing"
	RevisionMismatch        Kind = "RevisionMismatch"
	WriteConflict           Kind = "WriteConflict"
	DeletedRecord           Kind = "DeletedRecord"
	CursorFailure           Kind = "CursorFailure"
	DuplicateActivation     Kind = "DuplicateActivation"
	StaleChain              Kind = "StaleChain"
	LoopSuppressed          Kind = "LoopSuppressed"
	LeaseAcquireFailed      Kind = "LeaseAcquireFailed"
	ReplayCursorFailed      Kind = "ReplayCursorFailed"
	ProcessConfigInvalid    Kind = "ProcessConfigInvalid"
	ProtocolUnavailable     Kind = "ProtocolUnavailable"
	ResourceDenied          Kind = "ResourceDenied"
	KillFailed              Kind = "KillFailed"
	CleanupFailed           Kind = "CleanupFailed"
	ArtifactMissing         Kind = "ArtifactMissing"
	DigestMismatch          Kind = "DigestMismatch"
	NamespaceDenied         Kind = "NamespaceDenied"
	ObjectReadDenied        Kind = "ObjectReadDenied"
	MIMEDenied              Kind = "MIMEDenied"
	CSPMissing              Kind = "CSPMissing"
	CachePolicyInvalid      Kind = "CachePolicyInvalid"
	GatewayLeaseDenied      Kind = "GatewayLeaseDenied"
)

type Error struct {
	Kind      Kind
	Layer     string
	Operation string
	Message   string
	Details   map[string]string
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s.%s: %s", e.Layer, e.Kind, e.Message)
}

func (e *Error) Event() Event {
	return Event{
		Kind:      "substrate.error",
		Layer:     e.Layer,
		Operation: e.Operation,
		Cause:     string(e.Kind),
		Provenance: map[string]string{
			"origin": "go-substrate-core",
		},
	}
}

type Event struct {
	Kind       string
	Layer      string
	Operation  string
	Cause      string
	Provenance map[string]string
}

type Mode string

const (
	SingleNode Mode = "single_node"
	HAScale    Mode = "ha_scale"
)

type Status string

const (
	Accepted  Status = "accepted"
	Duplicate Status = "duplicate"
)

type LeaseKind string

const (
	BrowserLease LeaseKind = "browser"
	ScriptLease  LeaseKind = "script"
)

const FramedStdio = "framed_stdio"

type Config struct {
	Topology          Topology
	Store             Store
	Process           Process
	Gateway           Gateway
	ScriptPrincipalID string
}

type Plan struct {
	SchemaID string
	Topology
	Auth
	Leases  []Lease
	Store   Store
	Ledger  LedgerRecord
	Process Process
	Gateway GatewayPlan
	Events  []Event
}

type Topology struct {
	Mode      Mode
	JetStream bool
	Replicas  int
	Quorum    int
	Routes    []string
	Gateways  []string
	Leafs     []string
	WebSocket WebSocket
	Ready     bool
	Degraded  bool
}

type WebSocket struct {
	Enabled bool
	Port    int
}

type Auth struct {
	User       string
	Capability Capability
	Permissions
	Imports  map[string]Import
	Exports  []string
	Exposure map[string]Exposure
}

type Permissions struct {
	Publish        PermList
	Subscribe      PermList
	AllowResponses AllowResponses
}

type PermList struct {
	Allow []string
	Deny  []string
}

type AllowResponses struct {
	Max       int `json:"max"`
	ExpiresMs int `json:"expiresMs"`
}

type Import struct {
	Kind     string   `json:"kind"`
	Subjects []string `json:"subjects"`
	Desc     string   `json:"desc"`
}

type Exposure struct {
	Kind    string `json:"kind"`
	Subject string `json:"subject"`
	Desc    string `json:"desc"`
}

type Lease struct {
	ID          string
	Kind        LeaseKind
	Status      string
	PrincipalID string
	SessionID   string
	Capability  string
}

type Store struct {
	KVBucket      string
	ObjectBucket  string
	Stream        string
	ObjectKey     string
	ExpectedRev   int
	CurrentRev    int
	StreamCursor  string
	WriteConflict bool
	Deleted       bool
}

type Activation struct {
	ActivationID string `json:"activationId"`
	DedupeKey    string `json:"dedupeKey"`
	ScriptKey    string `json:"scriptKey"`
	Source       struct {
		Kind string `json:"kind"`
	} `json:"source"`
	Chain      Chain      `json:"chain"`
	Capability Capability `json:"capability"`
	Provenance Provenance `json:"provenance"`
}

type Chain struct {
	ChainID  string `json:"chainId"`
	RootID   string `json:"rootId"`
	ParentID string `json:"parentId"`
	Hop      int    `json:"hop"`
	MaxHops  int    `json:"maxHops"`
}

type LedgerRecord struct {
	ActivationID string
	DedupeKey    string
	SourceKind   string
	ChainID      string
	Status       Status
}

type Process struct {
	Command   string
	Args      []string
	Cwd       string
	Env       map[string]string
	RPC       string
	TimeoutMs int
	Resource  Resource
	Kill      string
	Cleanup   string
	Identity  string
}

type Resource struct {
	CPUMillis int
	MemoryMB  int
}

type Gateway struct {
	Namespace         string
	ExpectedDigest    string
	AllowObjectRead   bool
	AllowedMIME       []string
	Cache             string
	BrowserEdgePolicy string
	LeaseID           string
}

type GatewayPlan struct {
	ArtifactID        string
	ArtifactRevision  string
	Digest            string
	MediaType         string
	ObjectRef         string
	Namespace         string
	Cache             string
	BrowserEdgePolicy string
	CSPPolicy         string
	FramePolicy       string
	SandboxPolicy     string
}

func BuildPlan(reg *contract.Registry, authDoc, artifactDoc, activationDoc []byte, cfg Config) (*Plan, error) {
	if reg == nil {
		return nil, fail(SubstrateCritical, "GoSubstrate", "BuildPlan", "contract registry is required", nil)
	}
	top, err := CheckTopology(cfg.Topology)
	if err != nil {
		return nil, err
	}
	auth, err := RenderAuth(reg, authDoc)
	if err != nil {
		return nil, err
	}
	leases, err := MintLeases(auth, cfg.ScriptPrincipalID)
	if err != nil {
		return nil, err
	}
	store, err := CheckStore(cfg.Store)
	if err != nil {
		return nil, err
	}
	var act Activation
	if err := validate(reg, activationDoc, &act, AuthRenderInvalid, "ActivationLedger", "DecodeActivation"); err != nil {
		return nil, err
	}
	ledger, err := NewLedger().Accept(act, leases[0])
	if err != nil {
		return nil, err
	}
	proc, err := CheckProcess(cfg.Process)
	if err != nil {
		return nil, err
	}
	var art artifact
	if err := validate(reg, artifactDoc, &art, ArtifactMissing, "GatewaySubstrate", "DecodeArtifact"); err != nil {
		return nil, err
	}
	gw, err := CheckGateway(artifactDoc, cfg.Gateway, leases[0])
	if err != nil {
		return nil, err
	}
	return &Plan{
		SchemaID: auth.Capability.Provenance.SchemaID,
		Topology: top,
		Auth:     auth,
		Leases:   leases,
		Store:    store,
		Ledger:   ledger,
		Process:  proc,
		Gateway:  gw,
		Events: []Event{{
			Kind:       "substrate.plan.accepted",
			Layer:      "GoSubstrate",
			Operation:  "BuildPlan",
			Provenance: auth.Capability.Provenance.Map(),
		}},
	}, nil
}

func CheckTopology(t Topology) (Topology, error) {
	if t.Mode != SingleNode && t.Mode != HAScale {
		return Topology{}, fail(TopologyInvalid, "CoreLifecycle", "CheckTopology", "topology mode is invalid", nil)
	}
	if !t.JetStream {
		return Topology{}, fail(TopologyInvalid, "CoreLifecycle", "CheckTopology", "JetStream is required", nil)
	}
	if !t.Ready {
		return Topology{}, fail(NATSUnavailable, "CoreLifecycle", "CheckTopology", "topology is not ready", nil)
	}
	if t.Mode == SingleNode {
		if t.Replicas == 0 {
			t.Replicas = 1
		}
		if t.Quorum == 0 {
			t.Quorum = 1
		}
	}
	if t.Replicas < 1 {
		return Topology{}, fail(TopologyInvalid, "CoreLifecycle", "CheckTopology", "replicas are required", nil)
	}
	if t.Quorum < 1 || t.Quorum > t.Replicas {
		return Topology{}, fail(QuorumUnavailable, "CoreLifecycle", "CheckTopology", "quorum cannot be satisfied", nil)
	}
	return t, nil
}

func RenderAuth(reg *contract.Registry, doc []byte) (Auth, error) {
	var policy authPolicy
	if err := validate(reg, doc, &policy, AuthRenderInvalid, "AuthRender", "RenderAuth"); err != nil {
		return Auth{}, err
	}
	if policy.Kind != "auth.policy" {
		return Auth{}, fail(AuthRenderInvalid, "AuthRender", "RenderAuth", "expected auth.policy", nil)
	}
	cap := policy.Capability
	cap.Provenance = policy.Provenance
	switch cap.LeaseStatus {
	case "revoked":
		return Auth{}, fail(LeaseRevoked, "AuthRender", "RenderAuth", "capability lease is revoked", cap.Details())
	case "expired":
		return Auth{}, fail(LeaseExpired, "AuthRender", "RenderAuth", "capability lease is expired", cap.Details())
	}
	if cap.LeaseStatus != "active" {
		return Auth{}, fail(LeaseMintDenied, "AuthRender", "RenderAuth", "capability lease is not active", cap.Details())
	}
	if policy.Provenance.AppRevision != cap.AppRevision || policy.Provenance.SchemaVersion != cap.SchemaVersion {
		return Auth{}, fail(StaleChain, "AuthRender", "RenderAuth", "policy provenance does not match capability revision", cap.Details())
	}
	if policy.Permissions.AllowResponses.Max > 0 && policy.Permissions.AllowResponses.ExpiresMs <= 0 {
		return Auth{}, fail(PermissionCompileFailed, "AuthRender", "RenderAuth", "allow_responses requires expiresMs", nil)
	}
	if s := overbroad(policy.Permissions.Publish.Allow, policy.Permissions.Subscribe.Allow); s != "" {
		return Auth{}, fail(WildcardOverreach, "AuthRender", "RenderAuth", "subject wildcard is too broad", map[string]string{"subject": s})
	}
	if s := controlAllow(policy.Permissions.Publish.Allow, policy.Permissions.Subscribe.Allow); s != "" {
		return Auth{}, fail(PermissionCompileFailed, "AuthRender", "RenderAuth", "control subject cannot be granted directly", map[string]string{"subject": s})
	}
	return Auth{
		User:       cap.PrincipalID,
		Capability: cap,
		Permissions: Permissions{
			Publish:        copyPerm(policy.Permissions.Publish),
			Subscribe:      copyPerm(policy.Permissions.Subscribe),
			AllowResponses: policy.Permissions.AllowResponses,
		},
		Imports:  copyImports(policy.Imports),
		Exports:  copyStrings(policy.Exports),
		Exposure: copyExposure(policy.Exposure),
	}, nil
}

func MintLeases(auth Auth, script string) ([]Lease, error) {
	cap := auth.Capability
	if cap.LeaseID == "" || cap.PrincipalID == "" {
		return nil, fail(LeaseMintDenied, "CredentialLease", "MintLeases", "capability cannot mint a lease", cap.Details())
	}
	if strings.TrimSpace(script) == "" {
		return nil, fail(LeaseMintDenied, "CredentialLease", "MintLeases", "script principal is required", nil)
	}
	return []Lease{
		{ID: cap.LeaseID, Kind: BrowserLease, Status: "active", PrincipalID: cap.PrincipalID, SessionID: cap.SessionID, Capability: cap.CapabilityID},
		{ID: cap.LeaseID + ":script", Kind: ScriptLease, Status: "active", PrincipalID: script, SessionID: cap.SessionID, Capability: cap.CapabilityID},
	}, nil
}

type LeaseBook struct {
	leases map[string]Lease
}

func NewLeaseBook() *LeaseBook {
	return &LeaseBook{leases: map[string]Lease{}}
}

func (b *LeaseBook) Mint(kind LeaseKind, id, principal string) (Lease, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(principal) == "" {
		return Lease{}, fail(LeaseMintDenied, "CredentialLease", "Mint", "lease id and principal are required", nil)
	}
	lease := Lease{ID: id, Kind: kind, Status: "active", PrincipalID: principal}
	b.leases[id] = lease
	return lease, nil
}

func (b *LeaseBook) Revoke(id string) error {
	lease, ok := b.leases[id]
	if !ok {
		return fail(LeaseRevoked, "CredentialLease", "Revoke", "lease is not active", map[string]string{"leaseId": id})
	}
	lease.Status = "revoked"
	b.leases[id] = lease
	return nil
}

func (b *LeaseBook) Use(id string) error {
	lease, ok := b.leases[id]
	if !ok || lease.Status == "revoked" {
		return fail(LeaseRevoked, "CredentialLease", "Use", "lease is revoked", map[string]string{"leaseId": id})
	}
	if lease.Status == "expired" {
		return fail(LeaseExpired, "CredentialLease", "Use", "lease is expired", map[string]string{"leaseId": id})
	}
	return nil
}

func CheckStore(s Store) (Store, error) {
	switch {
	case s.KVBucket == "" || s.ObjectBucket == "" || s.Stream == "":
		return Store{}, fail(BucketMissing, "StoreSubstrate", "CheckStore", "store bucket is missing", nil)
	case s.ObjectKey == "":
		return Store{}, fail(KeyMissing, "StoreSubstrate", "CheckStore", "store key is missing", nil)
	case s.ExpectedRev != 0 && s.CurrentRev != 0 && s.ExpectedRev != s.CurrentRev:
		return Store{}, fail(RevisionMismatch, "StoreSubstrate", "CheckStore", "store revision does not match", nil)
	case s.WriteConflict:
		return Store{}, fail(WriteConflict, "StoreSubstrate", "CheckStore", "store write conflict", nil)
	case s.Deleted:
		return Store{}, fail(DeletedRecord, "StoreSubstrate", "CheckStore", "record is deleted", nil)
	case s.StreamCursor == "":
		return Store{}, fail(CursorFailure, "StoreSubstrate", "CheckStore", "stream cursor is missing", nil)
	}
	return s, nil
}

type Ledger struct {
	seen         map[string]LedgerRecord
	CursorFailed bool
}

func NewLedger() *Ledger {
	return &Ledger{seen: map[string]LedgerRecord{}}
}

func (l *Ledger) Accept(act Activation, lease Lease) (LedgerRecord, error) {
	if lease.Status != "active" {
		return LedgerRecord{}, fail(LeaseAcquireFailed, "ActivationLedger", "Accept", "activation lease is not active", map[string]string{"leaseId": lease.ID})
	}
	if act.Chain.MaxHops > 0 && act.Chain.Hop >= act.Chain.MaxHops {
		return LedgerRecord{}, fail(LoopSuppressed, "ActivationLedger", "Accept", "activation chain hop limit reached", map[string]string{"chainId": act.Chain.ChainID})
	}
	if l.CursorFailed {
		return LedgerRecord{}, fail(ReplayCursorFailed, "ActivationLedger", "Accept", "replay cursor failed", nil)
	}
	if act.DedupeKey == "" {
		return LedgerRecord{}, fail(CursorFailure, "ActivationLedger", "Accept", "dedupe key is required", nil)
	}
	if rec, ok := l.seen[act.DedupeKey]; ok {
		rec.Status = Duplicate
		return rec, nil
	}
	rec := LedgerRecord{
		ActivationID: act.ActivationID,
		DedupeKey:    act.DedupeKey,
		SourceKind:   act.Source.Kind,
		ChainID:      act.Chain.ChainID,
		Status:       Accepted,
	}
	l.seen[act.DedupeKey] = rec
	return rec, nil
}

func CheckProcess(p Process) (Process, error) {
	switch {
	case p.Command == "" || p.Cwd == "" || p.TimeoutMs <= 0 || p.Identity == "":
		return Process{}, fail(ProcessConfigInvalid, "ProcessBoundary", "CheckProcess", "process config is incomplete", nil)
	case p.RPC == "":
		return Process{}, fail(ProtocolUnavailable, "ProcessBoundary", "CheckProcess", "process protocol is unavailable", nil)
	case p.Resource.CPUMillis <= 0 || p.Resource.MemoryMB <= 0:
		return Process{}, fail(ResourceDenied, "ProcessBoundary", "CheckProcess", "process resource envelope is invalid", nil)
	case p.Kill == "":
		return Process{}, fail(KillFailed, "ProcessBoundary", "CheckProcess", "kill behavior is required", nil)
	case p.Cleanup == "":
		return Process{}, fail(CleanupFailed, "ProcessBoundary", "CheckProcess", "cleanup behavior is required", nil)
	}
	return p, nil
}

func CheckGateway(doc []byte, cfg Gateway, lease Lease) (GatewayPlan, error) {
	if len(doc) == 0 {
		return GatewayPlan{}, fail(ArtifactMissing, "GatewaySubstrate", "CheckGateway", "artifact manifest is required", nil)
	}
	var art artifact
	if err := json.Unmarshal(doc, &art); err != nil {
		return GatewayPlan{}, fail(ArtifactMissing, "GatewaySubstrate", "CheckGateway", "artifact manifest is invalid", nil)
	}
	if art.Kind != "artifact.manifest" {
		return GatewayPlan{}, fail(ArtifactMissing, "GatewaySubstrate", "CheckGateway", "expected artifact.manifest", nil)
	}
	if cfg.ExpectedDigest != "" && art.Digest != cfg.ExpectedDigest {
		return GatewayPlan{}, fail(DigestMismatch, "GatewaySubstrate", "CheckGateway", "artifact digest mismatch", map[string]string{"expected": cfg.ExpectedDigest, "actual": art.Digest})
	}
	ns := cfg.Namespace
	if ns == "" {
		ns = "obj://frontend/"
	}
	if !strings.HasPrefix(art.ObjectRef, ns) {
		return GatewayPlan{}, fail(NamespaceDenied, "GatewaySubstrate", "CheckGateway", "artifact namespace denied", map[string]string{"objectRef": art.ObjectRef, "namespace": ns})
	}
	if !cfg.AllowObjectRead {
		return GatewayPlan{}, fail(ObjectReadDenied, "GatewaySubstrate", "CheckGateway", "object read authority denied", nil)
	}
	if !contains(cfg.AllowedMIME, art.MediaType) {
		return GatewayPlan{}, fail(MIMEDenied, "GatewaySubstrate", "CheckGateway", "artifact media type denied", map[string]string{"mediaType": art.MediaType})
	}
	if art.CSPPolicy == "" || art.FramePolicy == "" || !strings.HasPrefix(art.SandboxPolicy, "sandbox.browser.") {
		return GatewayPlan{}, fail(CSPMissing, "GatewaySubstrate", "CheckGateway", "browser security policy is incomplete", nil)
	}
	if cfg.Cache == "" {
		return GatewayPlan{}, fail(CachePolicyInvalid, "GatewaySubstrate", "CheckGateway", "cache policy is required", nil)
	}
	if cfg.LeaseID != "" && cfg.LeaseID != lease.ID {
		return GatewayPlan{}, fail(GatewayLeaseDenied, "GatewaySubstrate", "CheckGateway", "gateway lease does not match", map[string]string{"leaseId": lease.ID, "requiredLeaseId": cfg.LeaseID})
	}
	return GatewayPlan{
		ArtifactID:        art.ArtifactID,
		ArtifactRevision:  art.ArtifactRevision,
		Digest:            art.Digest,
		MediaType:         art.MediaType,
		ObjectRef:         art.ObjectRef,
		Namespace:         ns,
		Cache:             cfg.Cache,
		BrowserEdgePolicy: cfg.BrowserEdgePolicy,
		CSPPolicy:         art.CSPPolicy,
		FramePolicy:       art.FramePolicy,
		SandboxPolicy:     art.SandboxPolicy,
	}, nil
}

func validate(reg *contract.Registry, doc []byte, out any, kind Kind, layer, op string) error {
	if reg == nil {
		return fail(SubstrateCritical, layer, op, "contract registry is required", nil)
	}
	if err := reg.Validate(contract.ContractSchemaID, doc); err != nil {
		return fail(kind, layer, op, "contract document is invalid", map[string]string{"cause": err.Error()})
	}
	if err := json.Unmarshal(doc, out); err != nil {
		return fail(kind, layer, op, "contract document JSON is invalid", map[string]string{"cause": err.Error()})
	}
	return nil
}

func fail(kind Kind, layer, op, msg string, details map[string]string) *Error {
	if details == nil {
		details = map[string]string{}
	}
	return &Error{Kind: kind, Layer: layer, Operation: op, Message: msg, Details: details}
}

func overbroad(groups ...[]string) string {
	for _, group := range groups {
		for _, sub := range group {
			if sub == ">" {
				return sub
			}
			if strings.HasSuffix(sub, ".>") && len(strings.Split(strings.TrimSuffix(sub, ".>"), ".")) < 3 {
				return sub
			}
		}
	}
	return ""
}

func controlAllow(groups ...[]string) string {
	for _, group := range groups {
		for _, sub := range group {
			if strings.HasPrefix(sub, "tb.internal.") {
				return sub
			}
		}
	}
	return ""
}

func copyPerm(p permList) PermList {
	return PermList{Allow: copyStrings(p.Allow), Deny: copyStrings(p.Deny)}
}

func copyImports(items map[string]Import) map[string]Import {
	if len(items) == 0 {
		return nil
	}
	out := make(map[string]Import, len(items))
	for k, v := range items {
		v.Subjects = copyStrings(v.Subjects)
		out[k] = v
	}
	return out
}

func copyExposure(items map[string]Exposure) map[string]Exposure {
	if len(items) == 0 {
		return nil
	}
	out := make(map[string]Exposure, len(items))
	for k, v := range items {
		out[k] = v
	}
	return out
}

func copyStrings(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	out := make([]string, len(items))
	copy(out, items)
	return out
}

func contains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

type authPolicy struct {
	Kind        string     `json:"kind"`
	Provenance  Provenance `json:"provenance"`
	Capability  Capability `json:"capability"`
	Permissions struct {
		Publish        permList       `json:"publish"`
		Subscribe      permList       `json:"subscribe"`
		AllowResponses AllowResponses `json:"allow_responses"`
	} `json:"permissions"`
	Imports  map[string]Import   `json:"imports"`
	Exports  []string            `json:"exports"`
	Exposure map[string]Exposure `json:"exposure"`
}

type permList struct {
	Allow []string `json:"allow"`
	Deny  []string `json:"deny"`
}

type Provenance struct {
	SchemaID      string `json:"schemaId"`
	SchemaVersion string `json:"schemaVersion"`
	AppRevision   string `json:"appRevision"`
	CreatedAt     string `json:"createdAt"`
	Producer      string `json:"producer"`
}

func (p Provenance) Map() map[string]string {
	return map[string]string{
		"schemaId":      p.SchemaID,
		"schemaVersion": p.SchemaVersion,
		"appRevision":   p.AppRevision,
		"producer":      p.Producer,
	}
}

type Capability struct {
	PrincipalID   string     `json:"principalId"`
	SessionID     string     `json:"sessionId"`
	CapabilityID  string     `json:"capabilityId"`
	LeaseID       string     `json:"leaseId"`
	LeaseStatus   string     `json:"leaseStatus"`
	AppRevision   string     `json:"appRevision"`
	SchemaVersion string     `json:"schemaVersion"`
	Provenance    Provenance `json:"-"`
}

func (c Capability) Details() map[string]string {
	return map[string]string{
		"principalId":   c.PrincipalID,
		"sessionId":     c.SessionID,
		"capabilityId":  c.CapabilityID,
		"leaseId":       c.LeaseID,
		"leaseStatus":   c.LeaseStatus,
		"appRevision":   c.AppRevision,
		"schemaVersion": c.SchemaVersion,
	}
}

type artifact struct {
	Kind             string `json:"kind"`
	ArtifactID       string `json:"artifactId"`
	ArtifactRevision string `json:"artifactRevision"`
	Digest           string `json:"digest"`
	MediaType        string `json:"mediaType"`
	ObjectRef        string `json:"objectRef"`
	SandboxPolicy    string `json:"sandboxPolicy"`
	FramePolicy      string `json:"framePolicy"`
	CSPPolicy        string `json:"cspPolicy"`
}
