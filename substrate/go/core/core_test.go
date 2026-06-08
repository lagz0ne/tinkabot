package core

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/lagz0ne/tinkabot/substrate/go/contract"
)

func TestBuildPlanConsumesContracts(t *testing.T) {
	plan, err := BuildPlan(reg(t), read(t, "fixtures/valid/auth-policy.json"), read(t, "fixtures/valid/artifact-manifest.json"), read(t, "fixtures/valid/activation-command-acceptance.json"), validCfg())
	if err != nil {
		t.Fatal(err)
	}

	if plan.SchemaID != contract.ContractSchemaID {
		t.Fatalf("schema drift: %s", plan.SchemaID)
	}
	if plan.Topology.Mode != HAScale || !plan.Topology.JetStream || plan.Topology.Quorum != 2 || len(plan.Topology.Routes) != 2 {
		t.Fatalf("topology drift: %#v", plan.Topology)
	}
	if !plan.Topology.WebSocket.Enabled || plan.Topology.WebSocket.Port != 8081 {
		t.Fatalf("websocket posture drift: %#v", plan.Topology.WebSocket)
	}
	if plan.Auth.User != "principal.browser.001" || plan.Auth.Permissions.AllowResponses.ExpiresMs != 30000 {
		t.Fatalf("auth render drift: %#v", plan.Auth)
	}
	if len(plan.Leases) != 2 || plan.Leases[0].Kind != BrowserLease || plan.Leases[1].Kind != ScriptLease {
		t.Fatalf("lease drift: %#v", plan.Leases)
	}
	if plan.Store.KVBucket != "tb.core.kv" || plan.Store.StreamCursor != "stream:42" {
		t.Fatalf("store drift: %#v", plan.Store)
	}
	if plan.Ledger.ActivationID != "act:scripts.proof.select_artifact:browser_command:cmd-001" || plan.Ledger.Status != Accepted {
		t.Fatalf("ledger drift: %#v", plan.Ledger)
	}
	if plan.Process.RPC != FramedStdio || plan.Process.TimeoutMs != 30000 || plan.Process.Resource.CPUMillis != 250 {
		t.Fatalf("process drift: %#v", plan.Process)
	}
	if plan.Gateway.ObjectRef != "obj://frontend/artifact-001/rev-7/bundle.js" || plan.Gateway.BrowserEdgePolicy != "browser.edge.artifact.v1" {
		t.Fatalf("gateway drift: %#v", plan.Gateway)
	}
	if len(plan.Events) == 0 || plan.Events[0].Kind != "substrate.plan.accepted" {
		t.Fatalf("attribution drift: %#v", plan.Events)
	}
}

func TestBuildPlanDeniesBeforeAuthority(t *testing.T) {
	cases := []struct {
		name string
		auth string
		cfg  Config
		kind Kind
	}{
		{"revoked", "fixtures/valid/auth-policy-revoked-lease.json", validCfg(), LeaseRevoked},
		{"expired", "fixtures/valid/auth-policy-expired-lease.json", validCfg(), LeaseExpired},
		{"stale", "fixtures/valid/auth-policy-provenance-mismatch.json", validCfg(), StaleChain},
		{"wildcard", "fixtures/valid/auth-policy-wildcard-overreach.json", validCfg(), WildcardOverreach},
		{"response", "fixtures/valid/auth-policy-unbounded-response.json", validCfg(), PermissionCompileFailed},
		{"quorum", "fixtures/valid/auth-policy.json", func() Config {
			cfg := validCfg()
			cfg.Topology.Quorum = 4
			return cfg
		}(), QuorumUnavailable},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			plan, err := BuildPlan(reg(t), read(t, c.auth), read(t, "fixtures/valid/artifact-manifest.json"), read(t, "fixtures/valid/activation-command-acceptance.json"), c.cfg)
			if plan != nil {
				t.Fatalf("plan created before denial: %#v", plan)
			}
			assertKind(t, err, c.kind)
		})
	}
}

func TestCredentialLeaseBookRevokesIdempotently(t *testing.T) {
	book := NewLeaseBook()
	lease, err := book.Mint(ScriptLease, "lease-script-001", "principal.script.001")
	if err != nil {
		t.Fatal(err)
	}
	if lease.Status != "active" {
		t.Fatalf("lease status drift: %#v", lease)
	}

	if err := book.Use(lease.ID); err != nil {
		t.Fatal(err)
	}
	if err := book.Revoke(lease.ID); err != nil {
		t.Fatal(err)
	}
	if err := book.Revoke(lease.ID); err != nil {
		t.Fatal(err)
	}
	assertKind(t, book.Use(lease.ID), LeaseRevoked)
}

func TestStoreSubstrateErrors(t *testing.T) {
	cases := []struct {
		name string
		edit func(*Store)
		kind Kind
	}{
		{"bucket", func(s *Store) { s.KVBucket = "" }, BucketMissing},
		{"key", func(s *Store) { s.ObjectKey = "" }, KeyMissing},
		{"rev", func(s *Store) { s.ExpectedRev = 5; s.CurrentRev = 4 }, RevisionMismatch},
		{"conflict", func(s *Store) { s.WriteConflict = true }, WriteConflict},
		{"deleted", func(s *Store) { s.Deleted = true }, DeletedRecord},
		{"cursor", func(s *Store) { s.StreamCursor = "" }, CursorFailure},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			store := validCfg().Store
			c.edit(&store)
			_, err := CheckStore(store)
			assertKind(t, err, c.kind)
		})
	}
}

func TestActivationLedgerDedupeLoopAndCursor(t *testing.T) {
	ledger := NewLedger()
	act := activation(t, read(t, "fixtures/valid/activation-command-acceptance.json"))

	first, err := ledger.Accept(act, activeLease())
	if err != nil {
		t.Fatal(err)
	}
	if first.Status != Accepted {
		t.Fatalf("status drift: %#v", first)
	}

	dup, err := ledger.Accept(act, activeLease())
	if err != nil {
		t.Fatal(err)
	}
	if dup.Status != Duplicate {
		t.Fatalf("duplicate drift: %#v", dup)
	}

	loop := act
	loop.Chain.Hop = 5
	loop.Chain.MaxHops = 5
	_, err = ledger.Accept(loop, activeLease())
	assertKind(t, err, LoopSuppressed)

	_, err = ledger.Accept(act, Lease{ID: "lease-001", Status: "revoked"})
	assertKind(t, err, LeaseAcquireFailed)

	ledger.CursorFailed = true
	_, err = ledger.Accept(activation(t, edit(t, "fixtures/valid/activation-request-reply.json", func(doc map[string]any) {
		doc["dedupeKey"] = "request_reply:cursor"
	})), activeLease())
	assertKind(t, err, ReplayCursorFailed)
}

func TestProcessBoundaryRequiresSandboxReadyConfig(t *testing.T) {
	cases := []struct {
		name string
		edit func(*Process)
		kind Kind
	}{
		{"cmd", func(p *Process) { p.Command = "" }, ProcessConfigInvalid},
		{"cwd", func(p *Process) { p.Cwd = "" }, ProcessConfigInvalid},
		{"rpc", func(p *Process) { p.RPC = "" }, ProtocolUnavailable},
		{"timeout", func(p *Process) { p.TimeoutMs = 0 }, ProcessConfigInvalid},
		{"resource", func(p *Process) { p.Resource.CPUMillis = 0 }, ResourceDenied},
		{"kill", func(p *Process) { p.Kill = "" }, KillFailed},
		{"cleanup", func(p *Process) { p.Cleanup = "" }, CleanupFailed},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			proc := validCfg().Process
			c.edit(&proc)
			_, err := CheckProcess(proc)
			assertKind(t, err, c.kind)
		})
	}
}

func TestGatewaySubstrateRejectsUnsafeAuthority(t *testing.T) {
	cases := []struct {
		name     string
		artifact []byte
		edit     func(*Gateway)
		kind     Kind
	}{
		{"missing", nil, nil, ArtifactMissing},
		{"digest", edit(t, "fixtures/valid/artifact-manifest.json", func(doc map[string]any) { doc["digest"] = "sha256:bad" }), nil, DigestMismatch},
		{"namespace", edit(t, "fixtures/valid/artifact-manifest.json", func(doc map[string]any) { doc["objectRef"] = "obj://control/a.js" }), nil, NamespaceDenied},
		{"object-read", read(t, "fixtures/valid/artifact-manifest.json"), func(g *Gateway) { g.AllowObjectRead = false }, ObjectReadDenied},
		{"mime", edit(t, "fixtures/valid/artifact-manifest.json", func(doc map[string]any) { doc["mediaType"] = "text/html" }), nil, MIMEDenied},
		{"csp", edit(t, "fixtures/valid/artifact-manifest.json", func(doc map[string]any) { delete(doc, "cspPolicy") }), nil, CSPMissing},
		{"cache", read(t, "fixtures/valid/artifact-manifest.json"), func(g *Gateway) { g.Cache = "" }, CachePolicyInvalid},
		{"lease", read(t, "fixtures/valid/artifact-manifest.json"), func(g *Gateway) { g.LeaseID = "lease-other" }, GatewayLeaseDenied},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gw := validCfg().Gateway
			if c.edit != nil {
				c.edit(&gw)
			}
			_, err := CheckGateway(c.artifact, gw, activeLease())
			assertKind(t, err, c.kind)
		})
	}
}

func TestErrorAttribution(t *testing.T) {
	_, err := CheckTopology(Topology{Mode: SingleNode})
	var got *Error
	if !errors.As(err, &got) {
		t.Fatalf("expected core error, got %T: %v", err, err)
	}
	ev := got.Event()
	if ev.Kind != "substrate.error" || ev.Layer != "CoreLifecycle" || ev.Operation == "" || ev.Cause != string(TopologyInvalid) {
		t.Fatalf("event drift: %#v", ev)
	}
	if ev.Provenance["origin"] != "go-substrate-core" {
		t.Fatalf("provenance drift: %#v", ev.Provenance)
	}
}

func validCfg() Config {
	return Config{
		Topology: Topology{
			Mode:      HAScale,
			JetStream: true,
			Replicas:  3,
			Quorum:    2,
			Routes:    []string{"nats://route-a:6222", "nats://route-b:6222"},
			Gateways:  []string{"gw-us"},
			Leafs:     []string{"leaf-edge"},
			WebSocket: WebSocket{Enabled: true, Port: 8081},
			Ready:     true,
		},
		Store: Store{
			KVBucket:     "tb.core.kv",
			ObjectBucket: "tb.core.obj",
			Stream:       "tb.core.stream",
			ObjectKey:    "artifact-001/rev-7/bundle.js",
			ExpectedRev:  7,
			CurrentRev:   7,
			StreamCursor: "stream:42",
		},
		Process: Process{
			Command:   "bun",
			Args:      []string{"run", "script.ts"},
			Cwd:       "/work/app",
			Env:       map[string]string{"TB_RUN_ID": "run-001"},
			RPC:       FramedStdio,
			TimeoutMs: 30000,
			Resource:  Resource{CPUMillis: 250, MemoryMB: 128},
			Kill:      "process.kill",
			Cleanup:   "workdir.delete",
			Identity:  "principal.script.001",
		},
		Gateway: Gateway{
			Namespace:         "obj://frontend/",
			ExpectedDigest:    "sha256:5a1e",
			AllowObjectRead:   true,
			AllowedMIME:       []string{"application/javascript"},
			Cache:             "no-store",
			BrowserEdgePolicy: "browser.edge.artifact.v1",
			LeaseID:           "lease-001",
		},
		ScriptPrincipalID: "principal.script.001",
	}
}

func activeLease() Lease {
	return Lease{ID: "lease-001", Kind: BrowserLease, Status: "active", PrincipalID: "principal.browser.001"}
}

func assertKind(t *testing.T, err error, kind Kind) {
	t.Helper()
	var got *Error
	if !errors.As(err, &got) {
		t.Fatalf("expected core error, got %T: %v", err, err)
	}
	if got.Kind != kind {
		t.Fatalf("kind mismatch: got %s want %s (%v)", got.Kind, kind, got)
	}
}

func reg(t *testing.T) *contract.Registry {
	t.Helper()
	reg, err := contract.Open(schemaDir())
	if err != nil {
		t.Fatal(err)
	}
	return reg
}

func read(t *testing.T, fixture string) []byte {
	t.Helper()
	doc, err := os.ReadFile(filepath.Join(schemaDir(), fixture))
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

func edit(t *testing.T, fixture string, fn func(map[string]any)) []byte {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal(read(t, fixture), &doc); err != nil {
		t.Fatal(err)
	}
	fn(doc)
	out, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func activation(t *testing.T, doc []byte) Activation {
	t.Helper()
	var act Activation
	if err := json.Unmarshal(doc, &act); err != nil {
		t.Fatal(err)
	}
	return act
}

func schemaDir() string {
	return filepath.Join("..", "..", "..", "schemas", "endgame", "v1")
}
