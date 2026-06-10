package core

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

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
	SourceAuthDenied        Kind = "SourceAuthDenied"
	DeniedNeighbor          Kind = "DeniedNeighbor"
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
	StaleCursor             Kind = "StaleCursor"
	StaleChain              Kind = "StaleChain"
	LoopSuppressed          Kind = "LoopSuppressed"
	LeaseAcquireFailed      Kind = "LeaseAcquireFailed"
	ReplayCursorFailed      Kind = "ReplayCursorFailed"
	ScheduleConfigInvalid   Kind = "ScheduleConfigInvalid"
	ScheduleLeaseMissing    Kind = "ScheduleLeaseMissing"
	ScheduleLeaseLost       Kind = "ScheduleLeaseLost"
	ClockInvalid            Kind = "ClockInvalid"
	ScheduleTickDuplicate   Kind = "ScheduleTickDuplicate"
	CatchUpFailed           Kind = "CatchUpFailed"
	RestartRecoveryFailed   Kind = "RestartRecoveryFailed"
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
	prov := map[string]string{"origin": "go-substrate-core"}
	for k, v := range e.Details {
		prov[k] = v
	}
	return Event{
		Kind:       "substrate.error",
		Layer:      e.Layer,
		Operation:  e.Operation,
		Cause:      string(e.Kind),
		Provenance: prov,
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
	Accepted   Status = "accepted"
	Duplicate  Status = "duplicate"
	Suppressed Status = "suppressed"
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
	ActivationID    string          `json:"activationId"`
	DedupeKey       string          `json:"dedupeKey"`
	ScriptKey       string          `json:"scriptKey"`
	ScriptRevision  int             `json:"scriptRevision"`
	SourcePrincipal SourcePrincipal `json:"sourcePrincipal"`
	SourceLease     SourceLease     `json:"sourceLease"`
	Source          Source          `json:"source"`
	Chain           Chain           `json:"chain"`
	Capability      Capability      `json:"capability"`
	Provenance      Provenance      `json:"provenance"`
}

type SourcePrincipal struct {
	PrincipalID  string `json:"principalId"`
	SourceID     string `json:"sourceId"`
	SourceKind   string `json:"sourceKind"`
	AuthorityRef string `json:"authorityRef"`
}

type SourceLease struct {
	LeaseID        string `json:"leaseId"`
	LeaseStatus    string `json:"leaseStatus"`
	AppRevision    string `json:"appRevision"`
	SchemaVersion  string `json:"schemaVersion"`
	ScriptRevision int    `json:"scriptRevision"`
}

type Source struct {
	Kind               string `json:"kind"`
	ActivationName     string `json:"activationName"`
	Subject            string `json:"subject"`
	RequestID          string `json:"requestId"`
	CommandID          string `json:"commandId"`
	Pattern            string `json:"pattern"`
	ObservedSubject    string `json:"observedSubject"`
	MessageID          string `json:"messageId"`
	Bucket             string `json:"bucket"`
	Key                string `json:"key"`
	Operation          string `json:"operation"`
	Revision           int64  `json:"revision"`
	WatchRevision      int64  `json:"watchRevision"`
	Resume             string `json:"resume"`
	Name               string `json:"name"`
	Digest             string `json:"digest"`
	ObjectMetaSequence int64  `json:"objectMetaSequence"`
	WatchPosition      string `json:"watchPosition"`
	Stream             string `json:"stream"`
	Consumer           string `json:"consumer"`
	StreamSequence     int64  `json:"streamSequence"`
	ConsumerSequence   int64  `json:"consumerSequence"`
	DeliveryAttempt    int64  `json:"deliveryAttempt"`
	ScheduleID         string `json:"scheduleId"`
	TickID             string `json:"tickId"`
	DueAt              string `json:"dueAt"`
	OwnerPrincipalID   string `json:"ownerPrincipalId"`
	LeaderEpoch        int64  `json:"leaderEpoch"`
	FencingToken       string `json:"fencingToken"`
	AcquiredAt         string `json:"acquiredAt"`
	ExpiresAt          string `json:"expiresAt"`
	ClockID            string `json:"clockId"`
	Clock              string `json:"clock"`
}

type Chain struct {
	ChainID  string `json:"chainId"`
	RootID   string `json:"rootId"`
	ParentID string `json:"parentId"`
	Hop      int    `json:"hop"`
	MaxHops  int    `json:"maxHops"`
}

type LedgerRecord struct {
	ActivationID      string
	DedupeKey         string
	SourceID          string
	SourceKind        string
	SourcePrincipalID string
	SourceLeaseID     string
	SourcePosition    int64
	SourceCursor      string
	ReplayCursor      string
	ChainID           string
	Status            Status
}

type SourceGrant struct {
	SourceID     string
	SourceKind   string
	PrincipalID  string
	LeaseID      string
	AuthorityRef string
	Subject      string
	AllowResponses
	Imports  map[string]Import
	Exports  []string
	Exposure Exposure
	Event    Event
}

type Process struct {
	Command   string            `json:"command"`
	Args      []string          `json:"args"`
	Cwd       string            `json:"cwd"`
	Env       map[string]string `json:"env,omitempty"`
	RPC       string            `json:"rpc"`
	TimeoutMs int               `json:"timeoutMs"`
	Resource  Resource          `json:"resource"`
	Kill      string            `json:"kill"`
	Cleanup   string            `json:"cleanup"`
	Identity  string            `json:"identity"`
}

type Resource struct {
	CPUMillis int `json:"cpuMillis"`
	MemoryMB  int `json:"memoryMB"`
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

func AuthorizeSource(auth Auth, act Activation) (SourceGrant, error) {
	ctx := sourceCtx(act)
	failAuth := func(kind Kind, msg string, details map[string]string) (SourceGrant, error) {
		return SourceGrant{}, fail(kind, "SourceAuthority", "AuthorizeSource", msg, merge(ctx, details))
	}

	if strings.TrimSpace(auth.User) == "" || strings.TrimSpace(act.SourcePrincipal.PrincipalID) == "" {
		return failAuth(SourceAuthDenied, "source principal is required", nil)
	}
	if auth.User != act.SourcePrincipal.PrincipalID {
		return failAuth(SourceAuthDenied, "source principal does not match auth user", map[string]string{"authUser": auth.User})
	}
	if auth.Capability.PrincipalID != "" && auth.Capability.PrincipalID != auth.User {
		return failAuth(SourceAuthDenied, "capability principal does not match auth user", map[string]string{"capabilityPrincipal": auth.Capability.PrincipalID})
	}
	if act.SourcePrincipal.SourceID == "" || act.SourcePrincipal.SourceKind == "" || act.SourcePrincipal.AuthorityRef == "" {
		return failAuth(SourceAuthDenied, "source authority reference is required", nil)
	}
	if act.SourcePrincipal.SourceKind != act.Source.Kind {
		return failAuth(SourceAuthDenied, "source principal kind does not match source", map[string]string{"principalKind": act.SourcePrincipal.SourceKind, "kind": act.Source.Kind})
	}
	if auth.Capability.LeaseID != "" && auth.Capability.LeaseID != act.SourceLease.LeaseID {
		return failAuth(SourceAuthDenied, "source lease does not match auth lease", map[string]string{"authLeaseId": auth.Capability.LeaseID})
	}
	if err := checkSourceLease(act, auth); err != nil {
		return SourceGrant{}, err
	}

	sub, err := sourceSubject(act)
	if err != nil {
		return SourceGrant{}, err
	}
	if err := checkSourceAperture(act); err != nil {
		return SourceGrant{}, err
	}

	exp, ok := auth.Exposure[act.SourcePrincipal.AuthorityRef]
	if !ok {
		return failAuth(SourceAuthDenied, "source exposure is missing", nil)
	}
	if exp.Subject == "" {
		return failAuth(PermissionCompileFailed, "source exposure subject is required", nil)
	}
	if want := exposureKind(act.Source.Kind); want != "" && exp.Kind != want {
		return failAuth(PermissionCompileFailed, "source exposure kind does not match source", map[string]string{"exposureKind": exp.Kind, "expected": want})
	}
	if !subjectMatches(exp.Subject, sub) {
		return failAuth(DeniedNeighbor, "source exposure does not cover observed subject", map[string]string{"exposure": exp.Subject, "subject": sub})
	}
	if !contains(auth.Exports, exp.Subject) {
		return failAuth(PermissionCompileFailed, "source exposure is not exported", map[string]string{"subject": exp.Subject})
	}
	if err := checkSourceImports(auth, sub, ctx); err != nil {
		return SourceGrant{}, err
	}
	if act.Source.Kind == "request_reply" && (auth.AllowResponses.Max <= 0 || auth.AllowResponses.ExpiresMs <= 0) {
		return failAuth(PermissionCompileFailed, "request/reply source requires bounded responses", nil)
	}

	for _, check := range sourceChecks(act.Source, sub) {
		if !allowed(auth.Subscribe, check) {
			return failAuth(DeniedNeighbor, "source subject is outside subscribe aperture", map[string]string{"subject": check})
		}
	}

	return SourceGrant{
		SourceID:       act.SourcePrincipal.SourceID,
		SourceKind:     act.Source.Kind,
		PrincipalID:    act.SourcePrincipal.PrincipalID,
		LeaseID:        act.SourceLease.LeaseID,
		AuthorityRef:   act.SourcePrincipal.AuthorityRef,
		Subject:        sub,
		AllowResponses: auth.AllowResponses,
		Imports:        copyImports(auth.Imports),
		Exports:        copyStrings(auth.Exports),
		Exposure:       exp,
		Event:          sourceEvent("activation.source.authorized", act, sub),
	}, nil
}

func checkSourceLease(act Activation, auth Auth) error {
	ctx := sourceCtx(act)
	status := act.SourceLease.LeaseStatus
	if status == "" {
		return fail(SourceAuthDenied, "SourceAuthority", "AuthorizeSource", "source lease status is required", ctx)
	}
	if status == "revoked" || auth.Capability.LeaseStatus == "revoked" {
		return fail(LeaseRevoked, "SourceAuthority", "AuthorizeSource", "source lease is revoked", ctx)
	}
	if status == "expired" || auth.Capability.LeaseStatus == "expired" {
		return fail(LeaseExpired, "SourceAuthority", "AuthorizeSource", "source lease is expired", ctx)
	}
	if status != "active" {
		return fail(SourceAuthDenied, "SourceAuthority", "AuthorizeSource", "source lease is not active", ctx)
	}
	if act.SourceLease.LeaseID == "" {
		return fail(SourceAuthDenied, "SourceAuthority", "AuthorizeSource", "source lease id is required", ctx)
	}
	if act.SourceLease.AppRevision != act.Provenance.AppRevision || act.SourceLease.SchemaVersion != act.Provenance.SchemaVersion {
		return fail(StaleChain, "SourceAuthority", "AuthorizeSource", "source lease revision is stale", ctx)
	}
	if act.SourceLease.ScriptRevision != 0 && act.ScriptRevision != 0 && act.SourceLease.ScriptRevision != act.ScriptRevision {
		return fail(StaleChain, "SourceAuthority", "AuthorizeSource", "source lease script revision is stale", ctx)
	}
	return nil
}

func checkSourceImports(auth Auth, sub string, ctx map[string]string) error {
	for name, imp := range auth.Imports {
		if imp.Kind == "raw_nats" || imp.Kind == "cli" {
			return fail(PermissionCompileFailed, "SourceAuthority", "AuthorizeSource", "advanced import is denied", merge(ctx, map[string]string{"import": name, "kind": imp.Kind}))
		}
		if imp.Kind != "publish" && imp.Kind != "subscribe" {
			return fail(PermissionCompileFailed, "SourceAuthority", "AuthorizeSource", "source import kind is unsupported", merge(ctx, map[string]string{"import": name, "kind": imp.Kind}))
		}
		for _, impSub := range imp.Subjects {
			if imp.Kind == "subscribe" && !subjectMatches(impSub, sub) {
				return fail(DeniedNeighbor, "SourceAuthority", "AuthorizeSource", "source import does not cover observed subject", merge(ctx, map[string]string{"import": name, "subject": impSub, "observed": sub}))
			}
			if imp.Kind == "subscribe" && !allowed(auth.Subscribe, impSub) {
				return fail(DeniedNeighbor, "SourceAuthority", "AuthorizeSource", "source import is outside subscribe aperture", merge(ctx, map[string]string{"import": name, "subject": impSub}))
			}
			if imp.Kind == "publish" && !allowed(auth.Publish, impSub) {
				return fail(DeniedNeighbor, "SourceAuthority", "AuthorizeSource", "source import is outside publish aperture", merge(ctx, map[string]string{"import": name, "subject": impSub}))
			}
		}
	}
	return nil
}

func sourceSubject(act Activation) (string, error) {
	src := act.Source
	ctx := sourceCtx(act)
	switch src.Kind {
	case "request_reply", "command_acceptance":
		if src.Subject == "" {
			return "", fail(SourceAuthDenied, "SourceAuthority", "AuthorizeSource", "source subject is required", ctx)
		}
		return src.Subject, nil
	case "subject":
		if src.ObservedSubject == "" {
			return "", fail(SourceAuthDenied, "SourceAuthority", "AuthorizeSource", "observed subject is required", ctx)
		}
		return src.ObservedSubject, nil
	case "kv":
		if src.Bucket == "" || src.Key == "" {
			return "", fail(SourceAuthDenied, "SourceAuthority", "AuthorizeSource", "KV source coordinate is required", ctx)
		}
		return "$KV." + src.Bucket + "." + src.Key, nil
	case "object":
		if src.Bucket == "" || src.Name == "" {
			return "", fail(SourceAuthDenied, "SourceAuthority", "AuthorizeSource", "object source coordinate is required", ctx)
		}
		return "$O." + src.Bucket + "." + src.Name, nil
	case "stream":
		if src.Subject == "" {
			return "", fail(SourceAuthDenied, "SourceAuthority", "AuthorizeSource", "stream subject is required", ctx)
		}
		return src.Subject, nil
	case "schedule":
		if src.ScheduleID == "" || src.TickID == "" {
			return "", fail(SourceAuthDenied, "SourceAuthority", "AuthorizeSource", "schedule source coordinate is required", ctx)
		}
		return "tb.schedule." + src.ScheduleID + "." + src.TickID, nil
	default:
		return "", fail(SourceAuthDenied, "SourceAuthority", "AuthorizeSource", "source kind is unsupported", merge(ctx, map[string]string{"kind": src.Kind}))
	}
}

func checkSourceAperture(act Activation) error {
	src := act.Source
	ctx := sourceCtx(act)
	if src.Kind != "subject" {
		return nil
	}
	if src.Pattern == "" {
		return fail(SourceAuthDenied, "SourceAuthority", "AuthorizeSource", "source pattern is required", ctx)
	}
	if !validSubject(src.Pattern) {
		return fail(WildcardOverreach, "SourceAuthority", "AuthorizeSource", "source pattern is invalid", merge(ctx, map[string]string{"subject": src.Pattern}))
	}
	if s := overbroad([]string{src.Pattern}); s != "" {
		return fail(WildcardOverreach, "SourceAuthority", "AuthorizeSource", "source wildcard is too broad", merge(ctx, map[string]string{"subject": s}))
	}
	if contains(strings.Split(src.Pattern, "."), ">") {
		return fail(WildcardOverreach, "SourceAuthority", "AuthorizeSource", "source wildcard is too broad", merge(ctx, map[string]string{"subject": src.Pattern}))
	}
	if !subjectMatches(src.Pattern, src.ObservedSubject) {
		return fail(DeniedNeighbor, "SourceAuthority", "AuthorizeSource", "observed subject does not match source pattern", merge(ctx, map[string]string{"pattern": src.Pattern, "subject": src.ObservedSubject}))
	}
	return nil
}

func sourceChecks(src Source, sub string) []string {
	if src.Kind == "subject" {
		return []string{src.Pattern, sub}
	}
	return []string{sub}
}

func exposureKind(kind string) string {
	switch kind {
	case "request_reply", "command_acceptance":
		return "request_reply"
	case "subject", "schedule":
		return "subject"
	case "kv":
		return "kv_watch"
	case "object":
		return "object_change"
	case "stream":
		return "stream"
	default:
		return ""
	}
}

func allowed(perms PermList, subject string) bool {
	if matchAny(perms.Deny, subject) {
		return false
	}
	return matchAny(perms.Allow, subject)
}

func matchAny(patterns []string, subject string) bool {
	for _, pattern := range patterns {
		if subjectMatches(pattern, subject) {
			return true
		}
	}
	return false
}

func subjectMatches(pattern, subject string) bool {
	pp := strings.Split(pattern, ".")
	ss := strings.Split(subject, ".")
	for i, j := 0, 0; i < len(pp); i, j = i+1, j+1 {
		token := pp[i]
		if token == ">" {
			return len(ss) > j
		}
		if j >= len(ss) {
			return false
		}
		if token != "*" && token != ss[j] {
			return false
		}
	}
	return len(pp) == len(ss)
}

func validSubject(subject string) bool {
	if subject == "" || strings.Contains(subject, "<") || strings.Contains(subject, "{") {
		return false
	}
	tokens := strings.Split(subject, ".")
	for i, token := range tokens {
		if token == "" {
			return false
		}
		wild := token == "*" || token == ">"
		if (strings.Contains(token, "*") || strings.Contains(token, ">")) && !wild {
			return false
		}
		if token == ">" && i != len(tokens)-1 {
			return false
		}
		if wild && i < 2 {
			return false
		}
	}
	return true
}

func sourceCtx(act Activation) map[string]string {
	return map[string]string{
		"origin":       "activation-source-authority",
		"sourceId":     act.SourcePrincipal.SourceID,
		"sourceKind":   act.Source.Kind,
		"principalId":  act.SourcePrincipal.PrincipalID,
		"leaseId":      act.SourceLease.LeaseID,
		"authorityRef": act.SourcePrincipal.AuthorityRef,
	}
}

func sourceEvent(kind string, act Activation, sub string) Event {
	prov := act.Provenance.Map()
	for k, v := range sourceCtx(act) {
		prov[k] = v
	}
	prov["subject"] = sub
	return Event{
		Kind:       kind,
		Layer:      "SourceAuthority",
		Operation:  "AuthorizeSource",
		Provenance: prov,
	}
}

func merge(base, extra map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range base {
		out[k] = v
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
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

type DurableLedger struct {
	store LedgerStore
}

type LedgerStore interface {
	Dedupe(string) (LedgerRecord, bool, error)
	Source(string) (LedgerRecord, bool, error)
	SaveAccepted(LedgerRecord) error
	SaveSuppressed(LedgerRecord) error
	Replay(string, int) ([]LedgerRecord, error)
}

type MemoryLedgerStore struct {
	mu            sync.Mutex
	byDedupe      map[string]LedgerRecord
	byCursor      map[string]LedgerRecord
	bySource      map[string]LedgerRecord
	order         []string
	suppressed    []LedgerRecord
	WriteConflict bool
	CursorFailed  bool
}

type sourceCursor struct {
	pos int64
	cur string
}

func NewMemoryLedgerStore() *MemoryLedgerStore {
	return &MemoryLedgerStore{
		byDedupe: map[string]LedgerRecord{},
		byCursor: map[string]LedgerRecord{},
		bySource: map[string]LedgerRecord{},
	}
}

func NewDurableLedger(store LedgerStore) *DurableLedger {
	if store == nil {
		store = NewMemoryLedgerStore()
	}
	return &DurableLedger{store: store}
}

func (l *DurableLedger) Accept(act Activation, lease Lease) (LedgerRecord, error) {
	if act.DedupeKey == "" {
		return LedgerRecord{}, fail(CursorFailure, "ActivationLedger", "Accept", "dedupe key is required", nil)
	}
	if act.SourcePrincipal.SourceID == "" || act.SourcePrincipal.PrincipalID == "" {
		return LedgerRecord{}, fail(CursorFailure, "ActivationLedger", "Accept", "source principal is required", nil)
	}
	if act.SourcePrincipal.SourceKind == "" || act.SourcePrincipal.SourceKind != act.Source.Kind {
		return LedgerRecord{}, fail(CursorFailure, "ActivationLedger", "Accept", "source principal kind does not match source", map[string]string{"sourceKind": act.SourcePrincipal.SourceKind, "kind": act.Source.Kind})
	}
	if act.SourceLease.LeaseStatus != "active" || lease.Status != "active" {
		return LedgerRecord{}, fail(LeaseAcquireFailed, "ActivationLedger", "Accept", "activation source lease is not active", map[string]string{"leaseId": act.SourceLease.LeaseID})
	}
	if act.SourceLease.LeaseID == "" || lease.ID == "" || lease.ID != act.SourceLease.LeaseID {
		return LedgerRecord{}, fail(LeaseAcquireFailed, "ActivationLedger", "Accept", "activation source lease does not match", map[string]string{"leaseId": lease.ID, "sourceLeaseId": act.SourceLease.LeaseID})
	}
	if rec, ok, err := l.store.Dedupe(act.DedupeKey); err != nil {
		return LedgerRecord{}, err
	} else if ok {
		rec.Status = Duplicate
		return rec, nil
	}
	pos, cur, err := sourcePosition(act.Source)
	if err != nil {
		return LedgerRecord{}, err
	}
	rec := LedgerRecord{
		ActivationID:      act.ActivationID,
		DedupeKey:         act.DedupeKey,
		SourceID:          act.SourcePrincipal.SourceID,
		SourceKind:        act.Source.Kind,
		SourcePrincipalID: act.SourcePrincipal.PrincipalID,
		SourceLeaseID:     act.SourceLease.LeaseID,
		SourcePosition:    pos,
		SourceCursor:      cur,
		ReplayCursor:      replayCursor(act.SourcePrincipal.SourceID, cur),
		ChainID:           act.Chain.ChainID,
		Status:            Accepted,
	}
	if act.Chain.MaxHops > 0 && act.Chain.Hop >= act.Chain.MaxHops {
		rec.Status = Suppressed
		if err := l.store.SaveSuppressed(rec); err != nil {
			return LedgerRecord{}, err
		}
		return LedgerRecord{}, fail(LoopSuppressed, "ActivationLedger", "Accept", "activation chain hop limit reached", map[string]string{"chainId": act.Chain.ChainID})
	}
	if prev, ok, err := l.store.Source(act.SourcePrincipal.SourceID); err != nil {
		return LedgerRecord{}, err
	} else if ok && stale(recordCursor(prev), recordCursor(rec)) {
		return LedgerRecord{}, fail(StaleCursor, "ActivationLedger", "Accept", "source cursor is stale", map[string]string{"sourceId": act.SourcePrincipal.SourceID, "cursor": cur})
	}
	if err := l.store.SaveAccepted(rec); err != nil {
		return LedgerRecord{}, err
	}
	return rec, nil
}

func (l *DurableLedger) Replay(after string, limit int) ([]LedgerRecord, error) {
	return l.store.Replay(after, limit)
}

func (s *MemoryLedgerStore) Dedupe(key string) (LedgerRecord, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.byDedupe[key]
	return rec, ok, nil
}

func (s *MemoryLedgerStore) Source(id string) (LedgerRecord, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.bySource[id]
	return rec, ok, nil
}

func (s *MemoryLedgerStore) SaveAccepted(rec LedgerRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.WriteConflict {
		return fail(WriteConflict, "ActivationLedger", "Accept", "ledger write conflict", map[string]string{"dedupeKey": rec.DedupeKey})
	}
	if _, ok := s.byDedupe[rec.DedupeKey]; ok {
		return fail(WriteConflict, "ActivationLedger", "Accept", "ledger write conflict", map[string]string{"dedupeKey": rec.DedupeKey})
	}
	s.byDedupe[rec.DedupeKey] = rec
	s.byCursor[rec.ReplayCursor] = rec
	s.bySource[rec.SourceID] = rec
	s.order = append(s.order, rec.ReplayCursor)
	return nil
}

func (s *MemoryLedgerStore) SaveSuppressed(rec LedgerRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.suppressed = append(s.suppressed, rec)
	return nil
}

func (s *MemoryLedgerStore) Replay(after string, limit int) ([]LedgerRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.CursorFailed {
		return nil, fail(ReplayCursorFailed, "ActivationLedger", "Replay", "replay cursor failed", nil)
	}
	if limit <= 0 {
		return nil, fail(ReplayCursorFailed, "ActivationLedger", "Replay", "replay limit is invalid", nil)
	}
	if after != "" {
		if _, ok := s.byCursor[after]; !ok {
			return nil, fail(ReplayCursorFailed, "ActivationLedger", "Replay", "replay cursor is unknown", map[string]string{"cursor": after})
		}
	}
	start := after == ""
	out := []LedgerRecord{}
	for _, cur := range s.order {
		if !start {
			start = cur == after
			continue
		}
		if rec, ok := s.byCursor[cur]; ok {
			out = append(out, rec)
			if len(out) == limit {
				break
			}
		}
	}
	return out, nil
}

func (s *MemoryLedgerStore) AcceptedCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.order)
}

func (s *MemoryLedgerStore) SuppressedCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.suppressed)
}

func stale(prev, next sourceCursor) bool {
	if prev.pos > 0 && next.pos > 0 {
		return next.pos <= prev.pos
	}
	return prev.cur != "" && next.cur == prev.cur
}

func recordCursor(rec LedgerRecord) sourceCursor {
	return sourceCursor{pos: rec.SourcePosition, cur: rec.SourceCursor}
}

func replayCursor(sourceID, cursor string) string {
	return fmt.Sprintf("v1.%s.%s", enc(sourceID), enc(cursor))
}

func enc(s string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(s))
}

func sourcePosition(src Source) (int64, string, error) {
	switch src.Kind {
	case "request_reply":
		if src.Subject == "" || src.RequestID == "" {
			return 0, "", fail(CursorFailure, "ActivationLedger", "SourcePosition", "request id is required", nil)
		}
		return 0, src.RequestID, nil
	case "command_acceptance":
		if src.Subject == "" || src.CommandID == "" {
			return 0, "", fail(CursorFailure, "ActivationLedger", "SourcePosition", "command id is required", nil)
		}
		return 0, src.CommandID, nil
	case "subject":
		if src.Pattern == "" || src.ObservedSubject == "" || src.MessageID == "" {
			return 0, "", fail(CursorFailure, "ActivationLedger", "SourcePosition", "subject position is required", nil)
		}
		return 0, src.MessageID, nil
	case "kv":
		if src.Bucket == "" || src.Key == "" || src.Revision <= 0 || src.Resume == "" {
			return 0, "", fail(CursorFailure, "ActivationLedger", "SourcePosition", "KV revision and resume cursor are required", nil)
		}
		return src.Revision, src.Resume, nil
	case "object":
		if src.Bucket == "" || src.Name == "" || src.ObjectMetaSequence <= 0 || src.WatchPosition == "" {
			return 0, "", fail(CursorFailure, "ActivationLedger", "SourcePosition", "object meta sequence is required", nil)
		}
		return src.ObjectMetaSequence, src.WatchPosition, nil
	case "stream":
		if src.Stream == "" || src.Consumer == "" || src.StreamSequence <= 0 || src.ConsumerSequence <= 0 {
			return 0, "", fail(CursorFailure, "ActivationLedger", "SourcePosition", "stream sequence is required", nil)
		}
		return src.StreamSequence, fmt.Sprintf("%s:%s:%d:%d", src.Stream, src.Consumer, src.StreamSequence, src.ConsumerSequence), nil
	case "schedule":
		pos, err := clockPos(src.ClockID, src.Clock)
		if err != nil || src.ScheduleID == "" || src.LeaderEpoch <= 0 || src.TickID == "" || src.FencingToken == "" {
			return 0, "", fail(CursorFailure, "ActivationLedger", "SourcePosition", "schedule clock and fencing position are required", nil)
		}
		return pos, fmt.Sprintf("%s:%s:%s:%d:%s", src.ScheduleID, src.TickID, src.FencingToken, src.LeaderEpoch, src.Clock), nil
	default:
		return 0, "", fail(CursorFailure, "ActivationLedger", "SourcePosition", "source kind is unsupported", map[string]string{"kind": src.Kind})
	}
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
