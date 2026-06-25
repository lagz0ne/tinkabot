// Package tinkabot is the v1 product entry surface: one binary assembling the
// proven pieces — embedded NATS in operator/JWT mode, the embedded frontend
// shell, and the script materializer loop — behind a declared exposure
// posture. Assembly only: every behavior comes from the consumed packages.
package tinkabot

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lagz0ne/tinkabot/substrate/go/core"
	"github.com/lagz0ne/tinkabot/substrate/go/edge"
	"github.com/lagz0ne/tinkabot/substrate/go/embednats"
	"github.com/lagz0ne/tinkabot/substrate/go/frontend"
	"github.com/nats-io/nats.go"
)

// Manual roles: every manual flow runs as one of these minted principals,
// each materialized as a creds file in the store dir at start.
const (
	RoleCaller      = "caller"
	RoleObserver    = "observer"
	RoleAuthor      = "author"
	RoleParticipant = "participant"
	RoleWatcher     = "watcher"
)

// Kind names the five failure families this assembly owns.
type Kind string

const (
	StartupMaterializationFailed Kind = "StartupMaterializationFailed"
	FrontendServeFailed          Kind = "FrontendServeFailed"
	WiringMismatch               Kind = "WiringMismatch"
	ManualDivergence             Kind = "ManualDivergence"
	ShutdownFailed               Kind = "ShutdownFailed"
	BundleRejected               Kind = "BundleRejected"
)

type Error struct {
	Kind      Kind
	Operation string
	Message   string
	Details   map[string]string
	Cause     error
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("TinkabotBinary.%s: %s: %v", e.Kind, e.Message, e.Cause)
	}
	return fmt.Sprintf("TinkabotBinary.%s: %s", e.Kind, e.Message)
}

func (e *Error) Unwrap() error { return e.Cause }

// Config is the binary's declared posture: where durable state lives, how
// NATS is exposed, and where the embedded shell serves.
type Config struct {
	StoreDir  string
	Exposure  embednats.Exposure
	ShellAddr string
	// DemoSession, when non-empty, runs a continuously ticking stand-in
	// session under that id so the shell observe panel has something live to
	// watch. Demo gate only — never a product surface.
	DemoSession string
	// BundleDir, when non-empty, serves that directory as an ephemeral app
	// for this run: manifest-declared scripts wired to triggers, nothing
	// durable mutated (docs/matched-abstraction/approach/bundle-v1.md).
	BundleDir string
	// BundleSandbox selects the bundle sandbox tier. "" (default) is the
	// bwrap jail, fail-closed when bwrap is unavailable. "none" is the trusted
	// (unsandboxed) tier — an explicit opt-in for hosts without user
	// namespaces, which runs bundle processes BARE. Any other value is rejected.
	BundleSandbox string
}

// Wiring names every NATS-visible surface the manual operates against this
// binary: trigger and event subjects, and the script, ledger, material,
// artifact, config, and upload stores.
type Wiring struct {
	TriggerSubject string
	EventsSubject  string
	ConfigBucket   string
	UploadBucket   string
	ScriptBucket   string
	LedgerBucket   string
	MaterialBucket string
	ArtifactBucket string
	ItemBucket     string
	ScheduleBucket string
	ScriptKey      string
	ScriptRevision int
}

// ShellPosture is the served embedded-shell surface: its loopback URL and the
// proven service-worker scope and revision behind the policy headers.
type ShellPosture struct {
	URL       string
	Scope     string
	WorkerRev string
}

// Posture reports the live binary surface: the embedded NATS posture, the
// shell surface, and the manual wiring.
type Posture struct {
	NATS   embednats.Posture
	Shell  ShellPosture
	Wiring Wiring
}

type ParticipantProfile struct {
	AppID         string
	ParticipantID string
	StoreDir      string
	CredsFile     string
	UserPub       string
	LeaseID       string
	RecordKey     string
}

type WatcherProfile struct {
	Name      string
	Scope     string
	Target    string
	StoreDir  string
	CredsFile string
	UserPub   string
	LeaseID   string
	Revoked   bool
}

const (
	shellScope   = "/__tinkabot_session/"
	appRevision  = "app.rev.1"
	schemaID     = "tb.schema.base.contract_authority.v1"
	authorityRef = "auth.source.trigger.main"
	eventsStream = "tb_events"
	// servingTTL bounds the assembly's own minted credentials. JWT expiry is
	// enforced on live connections, so a 1h TTL killed the binary's plane at
	// minute 60 (found live 2026-06-12); the bound must comfortably exceed a
	// serving session — teardown revocation is what ends minted authority
	// with the process.
	servingTTL = 30 * 24 * time.Hour
)

// wiring names the served surfaces. The caller-facing literals are quoted
// from docs/manual/v1.md, which the binary must satisfy unchanged
// (docs/matched-abstraction/plan/quality-v1.md:24); gate:manual holds the
// manual's commands verbatim against this surface.
func wiring() Wiring {
	return Wiring{
		TriggerSubject: "tb.proof.runtime.execute",
		EventsSubject:  "tb.proof.events.main",
		ConfigBucket:   "config_bucket",
		UploadBucket:   "artifacts",
		ScriptBucket:   "tb_scripts",
		LedgerBucket:   "tb_ledger",
		MaterialBucket: "tb_material",
		ArtifactBucket: "tb_artifacts",
		ItemBucket:     "tb_items",
		ScheduleBucket: "tb_schedules",
		ScriptKey:      "scripts.app.main",
		ScriptRevision: 1,
	}
}

type App struct {
	rt            *embednats.Runtime
	posture       Posture
	creds         map[string]embednats.UserCreds
	files         map[string]string
	storeDir      string
	bundleSandbox string

	shell           *http.Server
	route           *embednats.Route
	stopLoop        func()
	closers         []func()
	routes          []*embednats.Route
	stopLoops       []func()
	browserWatches  map[string]func()
	materials       *embednats.KVMaterialStore
	bundleMaterials *embednats.KVMaterialStore

	mu      sync.Mutex
	torn    bool
	stopped bool
}

// Start assembles the binary: embedded NATS in operator/JWT mode over the
// declared exposure posture, first-start materialization of operator key and
// role creds, the script materializer loop wired to the manual trigger, and
// the embedded shell under the proven policy headers. Assembly only — every
// behavior comes from the consumed packages.
func Start(cfg Config) (*App, error) {
	if cfg.StoreDir == "" {
		return nil, fail(StartupMaterializationFailed, "Start", "store dir is required", nil, nil)
	}
	shellAddr := cfg.ShellAddr
	if shellAddr == "" {
		shellAddr = "127.0.0.1:0"
	}
	// Deny before any listener exists: a shell host beyond loopback is a
	// surface the declared posture never granted (external serving stays
	// deferred), so it is a wiring mismatch, not a serve failure.
	host, _, err := net.SplitHostPort(shellAddr)
	if err != nil {
		return nil, fail(WiringMismatch, "Start", "shell address is malformed", map[string]string{"addr": shellAddr}, err)
	}
	if host != "127.0.0.1" && host != "localhost" && host != "::1" {
		return nil, fail(WiringMismatch, "Start", "shell host is beyond the declared exposure posture",
			map[string]string{"host": host, "mode": string(cfg.Exposure.Mode)}, nil)
	}

	w := wiring()
	var bun *bundle
	if cfg.BundleDir != "" {
		b, err := loadBundle(cfg.BundleDir)
		if err != nil {
			return nil, err
		}
		bun = b
		// Reject an unknown sandbox tier up front, before any plane exists.
		switch cfg.BundleSandbox {
		case "", "none":
		default:
			return nil, rejectBundle("bundle sandbox tier is unknown", map[string]string{"tier": cfg.BundleSandbox}, nil)
		}
	}
	rt, err := embednats.Start(embednats.Config{
		Core: core.Config{
			Topology: core.Topology{Mode: core.SingleNode, JetStream: true, Ready: true},
			Store: core.Store{
				KVBucket:     w.ConfigBucket,
				ObjectBucket: w.UploadBucket,
				Stream:       eventsStream,
				ObjectKey:    "app.bundle.js",
				ExpectedRev:  1,
				CurrentRev:   1,
				StreamCursor: "stream:1",
			},
		},
		Operator:   true,
		Exposure:   cfg.Exposure,
		ServerName: "tinkabot",
		StoreDir:   cfg.StoreDir,
		WebSocket:  embednats.WebSocket{Enabled: true, Host: "127.0.0.1", Port: -1, NoTLS: true},
	})
	if err != nil {
		return nil, fail(StartupMaterializationFailed, "Start", "embedded runtime did not start", nil, err)
	}
	app := &App{rt: rt, creds: map[string]embednats.UserCreds{}, files: map[string]string{}, storeDir: cfg.StoreDir, bundleSandbox: cfg.BundleSandbox, browserWatches: map[string]func(){}}
	ok := false
	defer func() {
		if !ok {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = app.Stop(ctx)
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	nonce := strconv.FormatInt(time.Now().UnixNano(), 36)
	perms := rolePerms(w)
	if bun != nil {
		// The bundle's triggers are caller authority for this run, nothing
		// more — observers and authors are untouched.
		cp := perms[RoleCaller]
		cp.Publish.Allow = append(cp.Publish.Allow, bun.subjects()...)
		perms[RoleCaller] = cp
	}
	for role, p := range perms {
		uc, err := rt.MintUser(embednats.AppAccount, principal("principal."+role, "lease-"+role+"-"+nonce, p), servingTTL)
		if err != nil {
			return nil, fail(StartupMaterializationFailed, "Start", "role creds could not be minted", map[string]string{"role": role}, err)
		}
		file := filepath.Join(cfg.StoreDir, role+".creds")
		if err := os.WriteFile(file, uc.File, 0o600); err != nil {
			return nil, fail(StartupMaterializationFailed, "Start", "role creds could not be persisted", map[string]string{"role": role, "file": file}, err)
		}
		app.creds[role], app.files[role] = uc, file
	}

	routerUC, err := rt.MintUser(embednats.AppAccount, principal("principal.runtime.router", "lease-router-"+nonce, routerPerms(w)), servingTTL)
	if err != nil {
		return nil, fail(StartupMaterializationFailed, "Start", "router creds could not be minted", nil, err)
	}
	svcUC, err := rt.MintUser(embednats.AppAccount, principal("principal.runtime.materializer", "lease-materializer-"+nonce, servicePerms(w)), servingTTL)
	if err != nil {
		return nil, fail(StartupMaterializationFailed, "Start", "materializer creds could not be minted", nil, err)
	}
	scheduleUC, err := rt.MintUser(embednats.AppAccount, principal("principal.runtime.scheduler", "lease-scheduler-"+nonce, schedulePerms(w)), servingTTL)
	if err != nil {
		return nil, fail(StartupMaterializationFailed, "Start", "scheduler creds could not be minted", nil, err)
	}
	actionUC, err := rt.MintUser(embednats.AppAccount, principal("principal.runtime.actions", "lease-actions-"+nonce, actionServicePerms(w)), servingTTL)
	if err != nil {
		return nil, fail(StartupMaterializationFailed, "Start", "action service creds could not be minted", nil, err)
	}
	browserUC, err := rt.MintUser(embednats.AppAccount, principal("principal.runtime.browser-command", "lease-browser-command-"+nonce, browserCommandPerms(w)), servingTTL)
	if err != nil {
		return nil, fail(StartupMaterializationFailed, "Start", "browser command service creds could not be minted", nil, err)
	}
	dial := func(uc embednats.UserCreds, use string) (*nats.Conn, error) {
		nc, err := rt.ConnectCreds(ctx, uc.File)
		if err != nil {
			return nil, fail(StartupMaterializationFailed, "Start", use+" connection failed", nil, err)
		}
		app.closers = append(app.closers, nc.Close)
		return nc, nil
	}
	routerNC, err := dial(routerUC, "router")
	if err != nil {
		return nil, err
	}
	ledgerNC, err := dial(routerUC, "ledger store")
	if err != nil {
		return nil, err
	}
	scriptNC, err := dial(svcUC, "script store")
	if err != nil {
		return nil, err
	}
	materialNC, err := dial(svcUC, "material store")
	if err != nil {
		return nil, err
	}
	scheduleNC, err := dial(scheduleUC, "schedule store")
	if err != nil {
		return nil, err
	}
	actionNC, err := dial(actionUC, "action service")
	if err != nil {
		return nil, err
	}
	browserNC, err := dial(browserUC, "browser command service")
	if err != nil {
		return nil, err
	}

	if err := materialize(ctx, rt, svcUC, w); err != nil {
		return nil, err
	}
	ledgerStore, err := embednats.OpenKVLedgerStore(ledgerNC, w.LedgerBucket)
	if err != nil {
		return nil, fail(StartupMaterializationFailed, "Start", "ledger store could not be materialized", nil, err)
	}
	scriptStore, err := embednats.OpenKVScriptStore(scriptNC, w.ScriptBucket)
	if err != nil {
		return nil, fail(StartupMaterializationFailed, "Start", "script store could not be materialized", nil, err)
	}
	materialStore, err := embednats.OpenKVMaterialStore(materialNC, w.MaterialBucket, w.ArtifactBucket)
	if err != nil {
		return nil, fail(StartupMaterializationFailed, "Start", "material store could not be materialized", nil, err)
	}
	app.materials = materialStore
	if err := app.startSchedules(scheduleNC, w); err != nil {
		return nil, err
	}
	if err := app.startActionService(actionNC, w); err != nil {
		return nil, err
	}
	if err := app.startBrowserCommandService(browserNC, w); err != nil {
		return nil, err
	}

	cap := app.creds[RoleCaller].Lease
	router, err := embednats.NewSourceRouter(sourceAuthority(w, cap), core.NewDurableLedger(ledgerStore))
	if err != nil {
		return nil, fail(StartupMaterializationFailed, "Start", "source router could not be configured", nil, err)
	}
	route, results, err := router.RequestReply(routerNC, activationFor(w, cap))
	if err != nil {
		return nil, fail(StartupMaterializationFailed, "Start", "manual trigger route could not be wired", nil, err)
	}
	app.route = route
	rtm, err := core.NewScriptRuntime(core.ScriptPolicy{AllowedProjections: []string{"main"}, ArtifactPrefix: "artifact/"}, embednats.LocalScriptRunner{})
	if err != nil {
		return nil, fail(StartupMaterializationFailed, "Start", "script runtime could not be configured", nil, err)
	}
	mat, err := core.NewMaterializer(materialStore)
	if err != nil {
		return nil, fail(StartupMaterializationFailed, "Start", "materializer could not be configured", nil, err)
	}
	runs, stopLoop := embednats.NewScriptLoop(scriptStore, rtm, mat, materialStore, materialStore).Watch(results)
	app.stopLoop = stopLoop
	go func() {
		for range runs {
			// Run outcomes are durable: the loop writes attributed event
			// envelopes to the material store; nothing to do here.
		}
	}()

	if bun != nil {
		if err := app.startBundle(bun, bundleDeps{
			cap:    cap,
			nonce:  nonce,
			dial:   dial,
			caller: app.creds[RoleCaller],
		}); err != nil {
			return nil, err
		}
	}

	shell, err := app.serveShell(shellAddr)
	if err != nil {
		return nil, err
	}
	if cfg.DemoSession != "" {
		if err := app.startDemoSession(cfg.DemoSession); err != nil {
			return nil, fail(StartupMaterializationFailed, "Start", "demo session could not be started", map[string]string{"session": cfg.DemoSession}, err)
		}
	}
	app.posture = Posture{NATS: rt.Posture(), Shell: shell, Wiring: w}
	if err := writeLocalProfile(cfg.StoreDir, app.posture); err != nil {
		return nil, fail(StartupMaterializationFailed, "Start", "local profile descriptor could not be persisted", nil, err)
	}
	if err := app.refreshParticipantDescriptors(); err != nil {
		return nil, err
	}
	ok = true
	return app, nil
}

func (a *App) Posture() Posture                      { return a.posture }
func (a *App) Runtime() *embednats.Runtime           { return a.rt }
func (a *App) Creds(role string) embednats.UserCreds { return a.creds[role] }
func (a *App) CredsFile(role string) string          { return a.files[role] }

func (a *App) AdmitParticipant(appID, id string) (ParticipantProfile, error) {
	if !validParticipantToken(appID) || !validParticipantToken(id) {
		return ParticipantProfile{}, fail(StartupMaterializationFailed, "AdmitParticipant", "participant id is invalid", map[string]string{"app": appID, "participant": id}, nil)
	}
	nonce := strconv.FormatInt(time.Now().UnixNano(), 36)
	prof := ParticipantProfile{
		AppID:         appID,
		ParticipantID: id,
		StoreDir:      filepath.Join(a.storeDir, "participants", appID, id),
		RecordKey:     participantKey(appID, id),
	}
	auth := participantAuth(appID, id, nonce)
	uc, err := a.rt.MintUser(embednats.AppAccount, auth, servingTTL)
	if err != nil {
		return ParticipantProfile{}, fail(StartupMaterializationFailed, "AdmitParticipant", "participant creds could not be minted", map[string]string{"app": appID, "participant": id}, err)
	}
	prof.UserPub = uc.UserPub
	prof.LeaseID = uc.Lease.LeaseID
	ok := false
	defer func() {
		if !ok {
			_ = a.rt.Revoke(embednats.AppAccount, prof.UserPub)
		}
	}()
	if err := a.revokePriorParticipant(prof.RecordKey, prof.UserPub); err != nil {
		return ParticipantProfile{}, err
	}
	if err := os.MkdirAll(prof.StoreDir, 0o700); err != nil {
		return ParticipantProfile{}, fail(StartupMaterializationFailed, "AdmitParticipant", "participant profile dir could not be created", nil, err)
	}
	prof.CredsFile = filepath.Join(prof.StoreDir, "participant.creds")
	if err := os.WriteFile(prof.CredsFile, uc.File, 0o600); err != nil {
		return ParticipantProfile{}, fail(StartupMaterializationFailed, "AdmitParticipant", "participant creds could not be persisted", nil, err)
	}
	if err := writeParticipantDescriptor(prof, a.posture, "active"); err != nil {
		return ParticipantProfile{}, err
	}
	if err := a.writeParticipant(prof, "active", ""); err != nil {
		return ParticipantProfile{}, err
	}
	ok = true
	return prof, nil
}

func (a *App) AdmitWatcher(name, scope, target string) (WatcherProfile, error) {
	if !validParticipantToken(name) || !validWatcherTarget(scope, target) {
		return WatcherProfile{}, fail(StartupMaterializationFailed, "AdmitWatcher", "watcher scope is invalid", map[string]string{"name": name, "scope": scope, "target": target}, nil)
	}
	nonce := strconv.FormatInt(time.Now().UnixNano(), 36)
	prof := WatcherProfile{
		Name:     name,
		Scope:    scope,
		Target:   target,
		StoreDir: filepath.Join(a.storeDir, "watchers", name),
	}
	auth := watcherAuth(name, scope, target, nonce)
	uc, err := a.rt.MintUser(embednats.AppAccount, auth, servingTTL)
	if err != nil {
		return WatcherProfile{}, fail(StartupMaterializationFailed, "AdmitWatcher", "watcher creds could not be minted", map[string]string{"name": name}, err)
	}
	prof.UserPub = uc.UserPub
	prof.LeaseID = uc.Lease.LeaseID
	ok := false
	defer func() {
		if !ok {
			_ = a.rt.Revoke(embednats.AppAccount, prof.UserPub)
		}
	}()
	if err := os.MkdirAll(prof.StoreDir, 0o700); err != nil {
		return WatcherProfile{}, fail(StartupMaterializationFailed, "AdmitWatcher", "watcher profile dir could not be created", nil, err)
	}
	prof.CredsFile = filepath.Join(prof.StoreDir, "watcher.creds")
	if err := os.WriteFile(prof.CredsFile, uc.File, 0o600); err != nil {
		return WatcherProfile{}, fail(StartupMaterializationFailed, "AdmitWatcher", "watcher creds could not be persisted", nil, err)
	}
	if err := writeWatcherDescriptor(prof, a.posture, "active"); err != nil {
		return WatcherProfile{}, fail(StartupMaterializationFailed, "AdmitWatcher", "watcher descriptor could not be persisted", nil, err)
	}
	ok = true
	return prof, nil
}

func (a *App) RevokeWatcher(prof WatcherProfile) error {
	if prof.UserPub == "" {
		return fail(StartupMaterializationFailed, "RevokeWatcher", "watcher user is missing", map[string]string{"name": prof.Name}, nil)
	}
	if err := a.rt.Revoke(embednats.AppAccount, prof.UserPub); err != nil {
		return fail(StartupMaterializationFailed, "RevokeWatcher", "watcher creds could not be revoked", map[string]string{"name": prof.Name}, err)
	}
	prof.Revoked = true
	if err := writeWatcherDescriptor(prof, a.posture, "revoked"); err != nil {
		return fail(StartupMaterializationFailed, "RevokeWatcher", "watcher descriptor could not be updated", map[string]string{"name": prof.Name}, err)
	}
	return nil
}

func (a *App) RevokeParticipant(prof ParticipantProfile) error {
	if prof.UserPub == "" {
		return fail(StartupMaterializationFailed, "RevokeParticipant", "participant user is missing", map[string]string{"app": prof.AppID, "participant": prof.ParticipantID}, nil)
	}
	if err := a.rt.Revoke(embednats.AppAccount, prof.UserPub); err != nil {
		return fail(StartupMaterializationFailed, "RevokeParticipant", "participant creds could not be revoked", map[string]string{"app": prof.AppID, "participant": prof.ParticipantID}, err)
	}
	current, ok, err := a.readParticipant(prof.RecordKey)
	if err != nil {
		return err
	}
	if ok && current.UserPub != "" && current.UserPub != prof.UserPub {
		return nil
	}
	if err := writeParticipantDescriptor(prof, a.posture, "revoked"); err != nil {
		return fail(StartupMaterializationFailed, "RevokeParticipant", "participant descriptor could not be updated", map[string]string{"app": prof.AppID, "participant": prof.ParticipantID}, err)
	}
	return a.writeParticipant(prof, "revoked", time.Now().UTC().Format(time.RFC3339))
}

// Stop drains and shuts the assembly down. It is idempotent after a clean
// stop; a stop that fails (e.g. context already done) stays retryable.
func (a *App) Stop(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.stopped {
		return nil
	}
	a.teardown()
	if err := a.rt.Stop(ctx); err != nil {
		return fail(ShutdownFailed, "Stop", "embedded runtime did not stop cleanly", nil, err)
	}
	a.stopped = true
	return nil
}

// teardown runs once: it stops the loop and shell and revokes the role creds
// this process minted — minted authority dies with the process. Revocation is
// what drains external credentialed connections (the kicked client observes
// denial on reconnect and aborts), so a short bounded grace keeps the server
// up for that second round trip before shutdown.
func (a *App) teardown() {
	if a.torn {
		return
	}
	a.torn = true
	if a.route != nil {
		_ = a.route.Stop()
	}
	for _, r := range a.routes {
		_ = r.Stop()
	}
	if a.stopLoop != nil {
		a.stopLoop()
	}
	for _, stop := range a.stopLoops {
		stop()
	}
	revoked := false
	for _, uc := range a.creds {
		if a.rt.Revoke(embednats.AppAccount, uc.UserPub) == nil {
			revoked = true
		}
	}
	if revoked {
		time.Sleep(150 * time.Millisecond)
	}
	for _, close := range a.closers {
		close()
	}
	if a.shell != nil {
		_ = a.shell.Close()
	}
}

// serveShell serves the embedded frontend shell on the loopback address under
// the proven browser-isolation policy headers (edge.CheckServiceWorkerSetup is
// the header authority; the binary never invents header vocabulary).
func (a *App) serveShell(addr string) (ShellPosture, error) {
	files, err := frontend.Files()
	if err != nil {
		return ShellPosture{}, fail(FrontendServeFailed, "Start", "embedded shell assets are unavailable", nil, err)
	}
	index, err := frontend.Index()
	if err != nil {
		return ShellPosture{}, fail(FrontendServeFailed, "Start", "embedded shell index is unavailable", nil, err)
	}
	sum := sha256.Sum256(index)
	rev := "rev-" + hex.EncodeToString(sum[:4])
	headers, err := edge.CheckServiceWorkerSetup(
		edge.ServiceWorkerPolicy{SessionID: "shell", ScriptURL: shellScope + "sw.js", Scope: shellScope, AllowedScope: shellScope, WorkerRevision: rev, ArtifactScope: "/artifacts/"},
		edge.ServiceWorkerRequest{SessionID: "shell", ScriptURL: shellScope + "sw.js", Scope: shellScope, AllowedScope: shellScope, WorkerRevision: rev},
	)
	if err != nil {
		return ShellPosture{}, fail(FrontendServeFailed, "Start", "shell policy headers were denied", nil, err)
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return ShellPosture{}, fail(FrontendServeFailed, "Start", "shell listener could not bind", map[string]string{"addr": addr}, err)
	}
	fileSrv := http.FileServer(http.FS(files))
	a.shell = &http.Server{Handler: http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/session/viewer":
			a.mintViewer(rw, r)
		case r.URL.Path == "/session/ws":
			a.sessionWS(rw, r)
		case strings.HasPrefix(r.URL.Path, "/artifacts/"):
			a.serveArtifact(rw, r)
		case strings.HasPrefix(r.URL.Path, "/projections/"):
			a.serveProjection(rw, r)
		default:
			for k, v := range headers {
				rw.Header().Set(k, v)
			}
			a.ensureShellCookie(rw, r)
			fileSrv.ServeHTTP(rw, r)
		}
	})}
	go func() { _ = a.shell.Serve(ln) }()
	return ShellPosture{URL: "http://" + ln.Addr().String(), Scope: shellScope, WorkerRev: rev}, nil
}

// materialize creates the manual's caller-facing stores at first start and
// reopens them on restart (JetStream create is idempotent for an unchanged
// config; the built-in app account is durable, so app-plane KV state survives
// restart under the same operator authority).
func materialize(ctx context.Context, rt *embednats.Runtime, svc embednats.UserCreds, w Wiring) error {
	nc, err := rt.ConnectCreds(ctx, svc.File)
	if err != nil {
		return fail(StartupMaterializationFailed, "Start", "store materialization connection failed", nil, err)
	}
	defer nc.Close()
	js, err := nc.JetStream()
	if err != nil {
		return fail(StartupMaterializationFailed, "Start", "JetStream context is unavailable", nil, err)
	}
	if _, err := js.CreateKeyValue(&nats.KeyValueConfig{Bucket: w.ConfigBucket, Storage: nats.FileStorage}); err != nil {
		return fail(StartupMaterializationFailed, "Start", "config bucket could not be materialized", map[string]string{"bucket": w.ConfigBucket}, err)
	}
	if _, err := js.CreateKeyValue(&nats.KeyValueConfig{Bucket: w.ItemBucket, Storage: nats.FileStorage, History: nats.KeyValueMaxHistory}); err != nil {
		return fail(StartupMaterializationFailed, "Start", "item bucket could not be materialized", map[string]string{"bucket": w.ItemBucket}, err)
	}
	if _, err := js.ObjectStore(w.UploadBucket); err != nil {
		if !errors.Is(err, nats.ErrStreamNotFound) && !errors.Is(err, nats.ErrBucketNotFound) {
			return fail(StartupMaterializationFailed, "Start", "upload bucket could not be opened", map[string]string{"bucket": w.UploadBucket}, err)
		}
		if _, err := js.CreateObjectStore(&nats.ObjectStoreConfig{Bucket: w.UploadBucket, Storage: nats.FileStorage}); err != nil {
			return fail(StartupMaterializationFailed, "Start", "upload bucket could not be materialized", map[string]string{"bucket": w.UploadBucket}, err)
		}
	}
	if _, err := js.AddStream(&nats.StreamConfig{Name: eventsStream, Subjects: []string{w.EventsSubject}, Storage: nats.FileStorage}); err != nil {
		return fail(StartupMaterializationFailed, "Start", "events stream could not be materialized", map[string]string{"stream": eventsStream}, err)
	}
	return nil
}

const (
	scheduleKind = "tinkabot.schedule.v1"
	itemKind     = "tinkabot.item.v1"
)

type scheduleRec struct {
	Kind       string          `json:"kind"`
	Name       string          `json:"name"`
	Status     string          `json:"status"`
	EveryMs    int64           `json:"everyMs"`
	WriteItem  string          `json:"writeItem"`
	Value      json.RawMessage `json:"value"`
	Sequence   int             `json:"sequence"`
	LastTickAt string          `json:"lastTickAt,omitempty"`
	UpdatedAt  string          `json:"updatedAt"`
	Provenance itemProv        `json:"provenance"`
}

type itemRec struct {
	Kind       string          `json:"kind"`
	Key        string          `json:"key"`
	Status     string          `json:"status"`
	Value      json.RawMessage `json:"value"`
	CreatedAt  string          `json:"createdAt"`
	UpdatedAt  string          `json:"updatedAt"`
	Provenance itemProv        `json:"provenance"`
}

type itemProv struct {
	Profile string `json:"profile"`
	Source  string `json:"source"`
	Writer  string `json:"writer"`
}

type tickValue struct {
	Schedule    string          `json:"schedule"`
	Sequence    int             `json:"sequence"`
	ScheduledAt string          `json:"scheduledAt"`
	Value       json.RawMessage `json:"value"`
}

type appActionReq struct {
	ActionID     string          `json:"actionId"`
	StateKey     string          `json:"stateKey"`
	BaseRevision uint64          `json:"baseRevision"`
	Value        json.RawMessage `json:"value"`
}

type appActionValue struct {
	Kind          string          `json:"kind"`
	AppID         string          `json:"appId"`
	ParticipantID string          `json:"participantId"`
	ActionID      string          `json:"actionId"`
	StateKey      string          `json:"stateKey"`
	BaseRevision  uint64          `json:"baseRevision"`
	Payload       json.RawMessage `json:"payload"`
}

type appActionResp struct {
	Status          string         `json:"status"`
	Reason          string         `json:"reason,omitempty"`
	Item            *appActionItem `json:"item,omitempty"`
	DeliverySubject string         `json:"deliverySubject,omitempty"`
}

type appActionItem struct {
	Kind       string          `json:"kind"`
	Key        string          `json:"key"`
	Status     string          `json:"status"`
	Value      json.RawMessage `json:"value"`
	Revision   uint64          `json:"revision"`
	CreatedAt  string          `json:"createdAt"`
	UpdatedAt  string          `json:"updatedAt"`
	Provenance itemProv        `json:"provenance"`
}

type browserCommandReq struct {
	Kind             string                `json:"kind"`
	Type             string                `json:"type"`
	Command          string                `json:"command"`
	CommandID        string                `json:"commandId"`
	ExpectedRevision string                `json:"expectedRevision"`
	Payload          json.RawMessage       `json:"payload"`
	Context          browserCommandContext `json:"context"`
}

type browserCommandContext struct {
	SessionID        string   `json:"sessionId"`
	CapabilityID     string   `json:"capabilityId"`
	ArtifactID       string   `json:"artifactId"`
	ArtifactRevision string   `json:"artifactRevision"`
	FrameID          string   `json:"frameId"`
	AppID            string   `json:"appId"`
	ParticipantID    string   `json:"participantId"`
	Chain            chainCtx `json:"chain"`
}

type chainCtx struct {
	ChainID string `json:"chainId"`
	RootID  string `json:"rootId"`
	Hop     int    `json:"hop"`
	MaxHops int    `json:"maxHops"`
}

type browserParticipantActionPayload struct {
	AppID         string          `json:"appId,omitempty"`
	ParticipantID string          `json:"participantId,omitempty"`
	ActionID      string          `json:"actionId"`
	StateKey      string          `json:"stateKey"`
	BaseRevision  uint64          `json:"baseRevision"`
	Value         json.RawMessage `json:"value"`
}

type browserParticipantReadPayload struct {
	Key string `json:"key"`
}

type browserParticipantWatchPayload struct {
	Key      string `json:"key"`
	Delivery string `json:"delivery,omitempty"`
}

type browserItemSubmitPayload struct {
	Key              string          `json:"key"`
	Status           string          `json:"status,omitempty"`
	ExpectedRevision uint64          `json:"expectedRevision,omitempty"`
	Value            json.RawMessage `json:"value"`
}

type browserStateEvent struct {
	Kind       string          `json:"kind"`
	Source     string          `json:"source"`
	Key        string          `json:"key"`
	Status     string          `json:"status"`
	Value      json.RawMessage `json:"value"`
	Revision   uint64          `json:"revision"`
	ObservedAt string          `json:"observedAt"`
}

func (a *App) startSchedules(nc *nats.Conn, w Wiring) error {
	js, err := nc.JetStream()
	if err != nil {
		return fail(StartupMaterializationFailed, "Start", "schedule JetStream context is unavailable", nil, err)
	}
	schedules, err := js.KeyValue(w.ScheduleBucket)
	if err != nil {
		if !errors.Is(err, nats.ErrStreamNotFound) && !errors.Is(err, nats.ErrBucketNotFound) {
			return fail(StartupMaterializationFailed, "Start", "schedule bucket could not be opened", map[string]string{"bucket": w.ScheduleBucket}, err)
		}
		schedules, err = js.CreateKeyValue(&nats.KeyValueConfig{Bucket: w.ScheduleBucket, Storage: nats.FileStorage})
		if err != nil {
			return fail(StartupMaterializationFailed, "Start", "schedule bucket could not be materialized", map[string]string{"bucket": w.ScheduleBucket}, err)
		}
	}
	items, err := js.KeyValue(w.ItemBucket)
	if err != nil {
		return fail(StartupMaterializationFailed, "Start", "item bucket could not be opened for schedules", map[string]string{"bucket": w.ItemBucket}, err)
	}
	stop := make(chan struct{})
	done := make(chan struct{})
	appStop := func() {
		close(stop)
		<-done
	}
	a.stopLoops = append(a.stopLoops, appStop)
	go func() {
		defer close(done)
		tickSchedules(schedules, items)
		ticker := time.NewTicker(minEvery)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				tickSchedules(schedules, items)
			}
		}
	}()
	return nil
}

func (a *App) startActionService(nc *nats.Conn, w Wiring) error {
	js, err := nc.JetStream()
	if err != nil {
		return fail(StartupMaterializationFailed, "Start", "action service JetStream context is unavailable", nil, err)
	}
	items, err := js.KeyValue(w.ItemBucket)
	if err != nil {
		return fail(StartupMaterializationFailed, "Start", "item bucket could not be opened for action service", map[string]string{"bucket": w.ItemBucket}, err)
	}
	sub, err := nc.Subscribe("tb.app.*.participants.*.action", func(msg *nats.Msg) {
		resp := handleAppAction(items, msg)
		body, err := json.Marshal(resp)
		if err != nil {
			body = []byte(`{"status":"denied","reason":"malformed-response"}`)
		}
		_ = msg.Respond(body)
	})
	if err != nil {
		return fail(StartupMaterializationFailed, "Start", "action service route could not be wired", nil, err)
	}
	if err := nc.Flush(); err != nil {
		_ = sub.Unsubscribe()
		return fail(StartupMaterializationFailed, "Start", "action service route did not flush", nil, err)
	}
	a.closers = append(a.closers, func() { _ = sub.Unsubscribe() })
	return nil
}

func (a *App) startBrowserCommandService(nc *nats.Conn, w Wiring) error {
	js, err := nc.JetStream()
	if err != nil {
		return fail(StartupMaterializationFailed, "Start", "browser command JetStream context is unavailable", nil, err)
	}
	items, err := js.KeyValue(w.ItemBucket)
	if err != nil {
		return fail(StartupMaterializationFailed, "Start", "item bucket could not be opened for browser command service", map[string]string{"bucket": w.ItemBucket}, err)
	}
	sub, err := nc.Subscribe("tb.app.browser.command", func(msg *nats.Msg) {
		resp := a.handleBrowserCommand(nc, items, msg)
		body, err := json.Marshal(resp)
		if err != nil {
			body = []byte(`{"status":"denied","reason":"malformed-response"}`)
		}
		_ = msg.Respond(body)
	})
	if err != nil {
		return fail(StartupMaterializationFailed, "Start", "browser command route could not be wired", nil, err)
	}
	if err := nc.Flush(); err != nil {
		_ = sub.Unsubscribe()
		return fail(StartupMaterializationFailed, "Start", "browser command route did not flush", nil, err)
	}
	a.closers = append(a.closers, func() { _ = sub.Unsubscribe() })
	return nil
}

func (a *App) handleBrowserCommand(nc *nats.Conn, items nats.KeyValue, msg *nats.Msg) appActionResp {
	if msg.Subject != "tb.app.browser.command" {
		return denyAppAction("malformed-action")
	}
	var req browserCommandReq
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		return denyAppAction("malformed-action")
	}
	if req.Kind != "browser.command_intent" || req.Type != "content.intent" || req.CommandID == "" || req.ExpectedRevision == "" || req.ExpectedRevision != req.Context.ArtifactRevision {
		return denyAppAction("malformed-action")
	}
	if !validBrowserCommandContext(req.Context) {
		return denyAppAction("malformed-action")
	}
	if browserPayloadHasRawAuthority(req.Payload) {
		return denyAppAction("raw-authority")
	}
	switch req.Command {
	case "participant_action":
		return handleBrowserParticipantAction(nc, req)
	case "participant_read":
		return handleBrowserParticipantRead(items, req)
	case "participant_watch":
		return a.handleBrowserParticipantWatch(nc, items, req)
	case "item_submit":
		return handleBrowserItemSubmit(items, req)
	default:
		return denyAppAction("unknown-command")
	}
}

func validBrowserCommandContext(ctx browserCommandContext) bool {
	if ctx.SessionID == "" || ctx.CapabilityID == "" || ctx.ArtifactID == "" || ctx.ArtifactRevision == "" || ctx.FrameID == "" {
		return false
	}
	if ctx.Chain.ChainID == "" || ctx.Chain.RootID == "" || ctx.Chain.MaxHops < 1 || ctx.Chain.Hop < 0 {
		return false
	}
	if !validParticipantToken(ctx.ArtifactID) {
		return false
	}
	if ctx.AppID == "" && ctx.ParticipantID == "" {
		return true
	}
	return validParticipantToken(ctx.AppID) && validParticipantToken(ctx.ParticipantID)
}

func handleBrowserParticipantAction(nc *nats.Conn, req browserCommandReq) appActionResp {
	if !validParticipantToken(req.Context.AppID) || !validParticipantToken(req.Context.ParticipantID) {
		return denyAppAction("malformed-action")
	}
	var payload browserParticipantActionPayload
	if err := json.Unmarshal(req.Payload, &payload); err != nil {
		return denyAppAction("malformed-action")
	}
	if (payload.AppID != "" && payload.AppID != req.Context.AppID) || (payload.ParticipantID != "" && payload.ParticipantID != req.Context.ParticipantID) {
		return denyAppAction("denied-scope")
	}
	body, err := json.Marshal(appActionReq{
		ActionID:     payload.ActionID,
		StateKey:     payload.StateKey,
		BaseRevision: payload.BaseRevision,
		Value:        payload.Value,
	})
	if err != nil {
		return denyAppAction("malformed-action")
	}
	reply, err := nc.Request(participantActionSubject(req.Context.AppID, req.Context.ParticipantID), body, 5*time.Second)
	if err != nil {
		return denyAppAction(appActionReason(err))
	}
	var resp appActionResp
	if err := json.Unmarshal(reply.Data, &resp); err != nil {
		return denyAppAction("malformed-response")
	}
	return resp
}

func handleBrowserParticipantRead(items nats.KeyValue, req browserCommandReq) appActionResp {
	if !validParticipantToken(req.Context.AppID) || !validParticipantToken(req.Context.ParticipantID) {
		return denyAppAction("malformed-action")
	}
	var payload browserParticipantReadPayload
	if err := json.Unmarshal(req.Payload, &payload); err != nil {
		return denyAppAction("malformed-action")
	}
	reason := browserReadReason(req.Context, payload.Key)
	if reason != "" {
		return denyAppAction(reason)
	}
	entry, err := items.Get(payload.Key)
	if err != nil {
		return denyAppAction(appActionReason(err))
	}
	var rec itemRec
	if err := json.Unmarshal(entry.Value(), &rec); err != nil || rec.Kind != itemKind || rec.Key != payload.Key {
		return denyAppAction("malformed-item")
	}
	item := viewAppAction(rec, entry.Revision())
	return appActionResp{Status: "accepted", Item: &item}
}

func (a *App) handleBrowserParticipantWatch(nc *nats.Conn, items nats.KeyValue, req browserCommandReq) appActionResp {
	if !validParticipantToken(req.Context.AppID) || !validParticipantToken(req.Context.ParticipantID) {
		return denyAppAction("malformed-action")
	}
	var payload browserParticipantWatchPayload
	if err := json.Unmarshal(req.Payload, &payload); err != nil {
		return denyAppAction("malformed-action")
	}
	if reason := browserReadReason(req.Context, payload.Key); reason != "" {
		return denyAppAction(reason)
	}
	subject, ok := browserStateSubject(payload.Delivery, req.Context, payload.Key)
	if !ok {
		return denyAppAction("malformed-action")
	}
	if err := a.startBrowserStateWatch(nc, items, payload.Key, subject); err != nil {
		return denyAppAction(appActionReason(err))
	}
	return appActionResp{Status: "accepted", DeliverySubject: subject}
}

func handleBrowserItemSubmit(items nats.KeyValue, req browserCommandReq) appActionResp {
	var payload browserItemSubmitPayload
	if err := json.Unmarshal(req.Payload, &payload); err != nil {
		return denyAppAction("malformed-action")
	}
	if reason := browserItemSubmitReason(req.Context, payload); reason != "" {
		return denyAppAction(reason)
	}
	status := payload.Status
	if status == "" {
		status = "resolved"
	}
	now := time.Now().UTC().Format(time.RFC3339)
	createdAt := now
	if payload.ExpectedRevision > 0 {
		entry, err := items.Get(payload.Key)
		if err != nil {
			return denyAppAction(itemSubmitReason(err))
		}
		if entry.Revision() != payload.ExpectedRevision {
			return denyAppAction("stale-revision")
		}
		var prev itemRec
		if err := json.Unmarshal(entry.Value(), &prev); err != nil || prev.Kind != itemKind || prev.Key != payload.Key {
			return denyAppAction("malformed-item")
		}
		createdAt = prev.CreatedAt
	}
	rec := itemRec{
		Kind:      itemKind,
		Key:       payload.Key,
		Status:    status,
		Value:     payload.Value,
		CreatedAt: createdAt,
		UpdatedAt: now,
		Provenance: itemProv{
			Profile: "browser:" + req.Context.SessionID,
			Source:  "browser-command:" + req.CommandID,
			Writer:  "tinkabot-browser-command",
		},
	}
	body, err := json.Marshal(rec)
	if err != nil {
		return denyAppAction("malformed-action")
	}
	var rev uint64
	if payload.ExpectedRevision == 0 {
		rev, err = items.Create(payload.Key, body)
	} else {
		rev, err = items.Update(payload.Key, body, payload.ExpectedRevision)
	}
	if err != nil {
		return denyAppAction(itemSubmitReason(err))
	}
	item := viewAppAction(rec, rev)
	return appActionResp{Status: "accepted", Item: &item}
}

func browserItemSubmitReason(ctx browserCommandContext, payload browserItemSubmitPayload) string {
	if payload.Status != "" && payload.Status != "pending" && payload.Status != "resolved" {
		return "malformed-action"
	}
	if len(payload.Value) == 0 || !json.Valid(payload.Value) {
		return "malformed-action"
	}
	if !validProductItemKey(payload.Key) {
		return "malformed-action"
	}
	prefix := "artifacts." + ctx.ArtifactID + ".results."
	if !strings.HasPrefix(payload.Key, prefix) {
		return "denied-scope"
	}
	return ""
}

func itemSubmitReason(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.ToLower(err.Error())
	switch {
	case errors.Is(err, nats.ErrKeyNotFound), strings.Contains(msg, "not found"):
		return "item-not-found"
	case errors.Is(err, nats.ErrKeyExists), strings.Contains(msg, "key exists"):
		return "duplicate-item"
	case strings.Contains(msg, "wrong last sequence"):
		return "stale-revision"
	case strings.Contains(msg, "authorization") || strings.Contains(msg, "authentication") || strings.Contains(msg, "permission"):
		return "denied-scope"
	default:
		return "connection-failed"
	}
}

func browserReadReason(ctx browserCommandContext, key string) string {
	if !validProductItemKey(key) {
		return "malformed-action"
	}
	if strings.HasPrefix(key, appStatePrefix(ctx.AppID)) {
		return ""
	}
	if strings.HasPrefix(key, participantActionPrefix(ctx.AppID, ctx.ParticipantID)+".") {
		return ""
	}
	return "denied-scope"
}

func (a *App) startBrowserStateWatch(nc *nats.Conn, items nats.KeyValue, key, subject string) error {
	watcher, err := items.WatchFiltered([]string{key}, nats.IncludeHistory(), nats.IgnoreDeletes())
	if err != nil {
		return err
	}

	var once sync.Once
	stop := func() {
		once.Do(func() { _ = watcher.Stop() })
	}

	a.mu.Lock()
	if a.browserWatches == nil {
		a.browserWatches = map[string]func(){}
	}
	if _, exists := a.browserWatches[subject]; exists {
		a.mu.Unlock()
		stop()
		return publishBrowserState(nc, items, key, subject)
	}
	a.browserWatches[subject] = stop
	a.stopLoops = append(a.stopLoops, stop)
	a.mu.Unlock()

	go func() {
		defer func() {
			a.mu.Lock()
			if a.browserWatches[subject] != nil {
				delete(a.browserWatches, subject)
			}
			a.mu.Unlock()
		}()
		for {
			select {
			case err, ok := <-watcher.Error():
				if !ok || err != nil {
					return
				}
			case entry, ok := <-watcher.Updates():
				if !ok {
					return
				}
				if entry == nil || entry.Key() != key {
					continue
				}
				ev, ok := browserStateEventFromEntry(entry)
				if !ok {
					continue
				}
				body, err := json.Marshal(ev)
				if err != nil {
					continue
				}
				_ = nc.Publish(subject, body)
			}
		}
	}()
	return nil
}

func publishBrowserState(nc *nats.Conn, items nats.KeyValue, key, subject string) error {
	entry, err := items.Get(key)
	if err != nil {
		return err
	}
	ev, ok := browserStateEventFromEntry(entry)
	if !ok {
		return errors.New("malformed-item")
	}
	body, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	return nc.Publish(subject, body)
}

func browserStateEventFromEntry(entry nats.KeyValueEntry) (browserStateEvent, bool) {
	var rec itemRec
	if err := json.Unmarshal(entry.Value(), &rec); err != nil || rec.Kind != itemKind || rec.Key != entry.Key() {
		return browserStateEvent{}, false
	}
	return browserStateEvent{
		Kind:       "tinkabot.browserState.v1",
		Source:     "trusted-shell.nats-watch.push",
		Key:        rec.Key,
		Status:     rec.Status,
		Value:      rec.Value,
		Revision:   entry.Revision(),
		ObservedAt: time.Now().UTC().Format(time.RFC3339),
	}, true
}

func browserStateSubject(prefix string, ctx browserCommandContext, key string) (string, bool) {
	if !validBrowserStatePrefix(prefix) {
		return "", false
	}
	sum := sha256.Sum256([]byte(ctx.AppID + "\x00" + ctx.ParticipantID + "\x00" + key))
	return prefix + "." + ctx.AppID + "." + ctx.ParticipantID + "." + hex.EncodeToString(sum[:8]), true
}

func validBrowserStatePrefix(prefix string) bool {
	parts := strings.Split(prefix, ".")
	if len(parts) != 5 || parts[0] != "tb" || parts[1] != "app" || parts[2] != "browser" || parts[3] != "state" {
		return false
	}
	return validParticipantToken(parts[4])
}

func browserPayloadHasRawAuthority(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return false
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return true
	}
	return hasRawKey(value)
}

func hasRawKey(value any) bool {
	switch v := value.(type) {
	case []any:
		for _, item := range v {
			if hasRawKey(item) {
				return true
			}
		}
	case map[string]any:
		for key, item := range v {
			if browserRawKey(key) || hasRawKey(item) {
				return true
			}
		}
	}
	return false
}

func browserRawKey(key string) bool {
	name := strings.NewReplacer("_", "", "-", "").Replace(strings.ToLower(key))
	for _, raw := range []string{"allow", "allowresponses", "bearer", "cred", "credential", "credentials", "deny", "headers", "jwt", "nats", "nkey", "password", "permission", "permissions", "publish", "reply", "replysubject", "secret", "seed", "subject", "subjects", "subscribe", "token", "tokens"} {
		if strings.Contains(name, raw) {
			return true
		}
	}
	return false
}

func handleAppAction(items nats.KeyValue, msg *nats.Msg) appActionResp {
	appID, participantID, ok := parseAppActionSubject(msg.Subject)
	if !ok {
		return denyAppAction("malformed-action")
	}
	var req appActionReq
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		return denyAppAction("malformed-action")
	}
	if !validParticipantToken(req.ActionID) || !validProductItemKey(req.StateKey) || !strings.HasPrefix(req.StateKey, appStatePrefix(appID)) || req.BaseRevision == 0 || len(req.Value) == 0 || !json.Valid(req.Value) {
		return denyAppAction("malformed-action")
	}
	state, err := items.Get(req.StateKey)
	if err != nil {
		return denyAppAction(appActionReason(err))
	}
	if state.Revision() != req.BaseRevision {
		return denyAppAction("stale-revision")
	}
	key := participantActionPrefix(appID, participantID) + "." + req.ActionID
	val, err := json.Marshal(appActionValue{
		Kind:          "tinkabot.appAction.v1",
		AppID:         appID,
		ParticipantID: participantID,
		ActionID:      req.ActionID,
		StateKey:      req.StateKey,
		BaseRevision:  req.BaseRevision,
		Payload:       req.Value,
	})
	if err != nil {
		return denyAppAction("malformed-action")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	rec := itemRec{
		Kind:      itemKind,
		Key:       key,
		Status:    "pending",
		Value:     val,
		CreatedAt: now,
		UpdatedAt: now,
		Provenance: itemProv{
			Profile: "participant:" + participantID,
			Source:  "app-action:" + appID + ":" + participantID,
			Writer:  "tinkabot-action",
		},
	}
	body, err := json.Marshal(rec)
	if err != nil {
		return denyAppAction("malformed-action")
	}
	rev, err := items.Create(key, body)
	if err != nil {
		if errors.Is(err, nats.ErrKeyExists) {
			return denyAppAction("duplicate-action")
		}
		return denyAppAction(appActionReason(err))
	}
	item := viewAppAction(rec, rev)
	return appActionResp{Status: "accepted", Item: &item}
}

func parseAppActionSubject(subj string) (string, string, bool) {
	parts := strings.Split(subj, ".")
	if len(parts) != 6 || parts[0] != "tb" || parts[1] != "app" || parts[3] != "participants" || parts[5] != "action" {
		return "", "", false
	}
	if !validParticipantToken(parts[2]) || !validParticipantToken(parts[4]) {
		return "", "", false
	}
	return parts[2], parts[4], true
}

func denyAppAction(reason string) appActionResp {
	if reason == "" {
		reason = "action-denied"
	}
	return appActionResp{Status: "denied", Reason: reason}
}

func viewAppAction(rec itemRec, rev uint64) appActionItem {
	return appActionItem{
		Kind:       rec.Kind,
		Key:        rec.Key,
		Status:     rec.Status,
		Value:      rec.Value,
		Revision:   rev,
		CreatedAt:  rec.CreatedAt,
		UpdatedAt:  rec.UpdatedAt,
		Provenance: rec.Provenance,
	}
}

func appActionReason(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.ToLower(err.Error())
	switch {
	case errors.Is(err, nats.ErrKeyNotFound), strings.Contains(msg, "not found"):
		return "item-not-found"
	case errors.Is(err, nats.ErrKeyExists), strings.Contains(msg, "key exists"):
		return "duplicate-action"
	case strings.Contains(msg, "wrong last sequence"):
		return "stale-revision"
	case strings.Contains(msg, "authorization") || strings.Contains(msg, "authentication") || strings.Contains(msg, "permission"):
		return "denied-scope"
	default:
		return "connection-failed"
	}
}

func tickSchedules(schedules, items nats.KeyValue) {
	keys, err := schedules.Keys()
	if errors.Is(err, nats.ErrNoKeysFound) {
		return
	}
	if err != nil {
		return
	}
	now := time.Now().UTC()
	for _, key := range keys {
		_ = tickSchedule(schedules, items, key, now)
	}
}

func tickSchedule(schedules, items nats.KeyValue, key string, now time.Time) error {
	entry, err := schedules.Get(key)
	if err != nil {
		return err
	}
	var rec scheduleRec
	if err := json.Unmarshal(entry.Value(), &rec); err != nil || rec.Kind != scheduleKind || rec.Name != key || rec.Status != "active" {
		return nil
	}
	every := time.Duration(rec.EveryMs) * time.Millisecond
	if every < minEvery || rec.WriteItem == "" || !scheduleDue(rec, every, now) {
		return nil
	}
	rec.Sequence++
	at := now.Format(time.RFC3339Nano)
	rec.LastTickAt = at
	rec.UpdatedAt = at
	rec.Provenance.Writer = "tinkabot-schedule"
	body, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	if _, err := schedules.Update(key, body, entry.Revision()); err != nil {
		return err
	}
	return writeScheduledItem(items, rec, at)
}

func scheduleDue(rec scheduleRec, every time.Duration, now time.Time) bool {
	if rec.LastTickAt == "" {
		return true
	}
	last, err := time.Parse(time.RFC3339Nano, rec.LastTickAt)
	if err != nil {
		return false
	}
	return !last.Add(every).After(now)
}

func writeScheduledItem(items nats.KeyValue, rec scheduleRec, at string) error {
	val, err := json.Marshal(tickValue{Schedule: rec.Name, Sequence: rec.Sequence, ScheduledAt: at, Value: rec.Value})
	if err != nil {
		return err
	}
	item := itemRec{
		Kind:      itemKind,
		Key:       rec.WriteItem,
		Status:    "resolved",
		Value:     val,
		CreatedAt: at,
		UpdatedAt: at,
		Provenance: itemProv{
			Profile: "tinkabot",
			Source:  "server-schedule:" + rec.Name,
			Writer:  "tinkabot-schedule",
		},
	}
	entry, err := items.Get(rec.WriteItem)
	if err == nil {
		var prev itemRec
		if json.Unmarshal(entry.Value(), &prev) == nil && prev.Kind == itemKind && prev.CreatedAt != "" {
			item.CreatedAt = prev.CreatedAt
		}
		body, err := json.Marshal(item)
		if err != nil {
			return err
		}
		_, err = items.Update(rec.WriteItem, body, entry.Revision())
		return err
	}
	if !errors.Is(err, nats.ErrKeyNotFound) {
		return err
	}
	body, err := json.Marshal(item)
	if err != nil {
		return err
	}
	_, err = items.Create(rec.WriteItem, body)
	return err
}

func writeLocalProfile(store string, p Posture) error {
	abs, err := filepath.Abs(store)
	if err != nil {
		return err
	}
	doc := struct {
		Kind       string `json:"kind"`
		Server     string `json:"server"`
		Shell      string `json:"shell"`
		Credential string `json:"credential"`
		Role       string `json:"role"`
		Trust      string `json:"trust"`
		Source     string `json:"source"`
	}{
		Kind:       "tinkabot.localProfile.v1",
		Server:     p.NATS.ClientURL,
		Shell:      p.Shell.URL,
		Credential: "caller.creds",
		Role:       RoleCaller,
		Trust:      "local-owner",
		Source:     "local-store:" + abs,
	}
	body, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(store, "local-profile.json")
	if err := os.WriteFile(path, append(body, '\n'), 0o600); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}

type participantRecord struct {
	Kind          string   `json:"kind"`
	AppID         string   `json:"appId"`
	ParticipantID string   `json:"participantId"`
	Role          string   `json:"role"`
	Status        string   `json:"status"`
	UserPub       string   `json:"userPub"`
	LeaseID       string   `json:"leaseId"`
	ProfileSource string   `json:"profileSource"`
	CreatedAt     string   `json:"createdAt"`
	UpdatedAt     string   `json:"updatedAt"`
	RevokedAt     string   `json:"revokedAt,omitempty"`
	Provenance    itemProv `json:"provenance"`
}

func writeParticipantDescriptor(prof ParticipantProfile, p Posture, status string) error {
	abs, err := filepath.Abs(prof.StoreDir)
	if err != nil {
		return err
	}
	doc := struct {
		Kind          string `json:"kind"`
		Server        string `json:"server"`
		Shell         string `json:"shell"`
		Credential    string `json:"credential"`
		Role          string `json:"role"`
		Trust         string `json:"trust"`
		Source        string `json:"source"`
		Status        string `json:"status,omitempty"`
		AppID         string `json:"appId"`
		ParticipantID string `json:"participantId"`
	}{
		Kind:          "tinkabot.localProfile.v1",
		Server:        p.NATS.ClientURL,
		Shell:         p.Shell.URL,
		Credential:    "participant.creds",
		Role:          RoleParticipant,
		Trust:         "app-participant",
		Source:        "local-store:" + abs,
		Status:        status,
		AppID:         prof.AppID,
		ParticipantID: prof.ParticipantID,
	}
	body, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(prof.StoreDir, "local-profile.json")
	if err := os.WriteFile(path, append(body, '\n'), 0o600); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}

func writeWatcherDescriptor(prof WatcherProfile, p Posture, status string) error {
	abs, err := filepath.Abs(prof.StoreDir)
	if err != nil {
		return err
	}
	doc := struct {
		Kind        string `json:"kind"`
		Server      string `json:"server"`
		Shell       string `json:"shell"`
		Credential  string `json:"credential"`
		Role        string `json:"role"`
		Trust       string `json:"trust"`
		Source      string `json:"source"`
		Status      string `json:"status,omitempty"`
		WatchScope  string `json:"watchScope"`
		WatchTarget string `json:"watchTarget"`
	}{
		Kind:        "tinkabot.localProfile.v1",
		Server:      p.NATS.ClientURL,
		Shell:       p.Shell.URL,
		Credential:  "watcher.creds",
		Role:        RoleWatcher,
		Trust:       "item-watcher",
		Source:      "local-store:" + abs,
		Status:      status,
		WatchScope:  prof.Scope,
		WatchTarget: prof.Target,
	}
	body, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(prof.StoreDir, "local-profile.json")
	if err := os.WriteFile(path, append(body, '\n'), 0o600); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}

func (a *App) refreshParticipantDescriptors() error {
	root := filepath.Join(a.storeDir, "participants")
	apps, err := os.ReadDir(root)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fail(StartupMaterializationFailed, "Start", "participant profile dir could not be read", map[string]string{"dir": root}, err)
	}
	for _, appDir := range apps {
		if !appDir.IsDir() {
			continue
		}
		appID := appDir.Name()
		ids, err := os.ReadDir(filepath.Join(root, appID))
		if err != nil {
			return fail(StartupMaterializationFailed, "Start", "participant app profile dir could not be read", map[string]string{"app": appID}, err)
		}
		for _, idDir := range ids {
			if !idDir.IsDir() {
				continue
			}
			id := idDir.Name()
			store := filepath.Join(root, appID, id)
			body, err := os.ReadFile(filepath.Join(store, "local-profile.json"))
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			if err != nil {
				return fail(StartupMaterializationFailed, "Start", "participant descriptor could not be read", map[string]string{"app": appID, "participant": id}, err)
			}
			var doc struct {
				Kind          string `json:"kind"`
				Role          string `json:"role"`
				Trust         string `json:"trust"`
				Status        string `json:"status"`
				AppID         string `json:"appId"`
				ParticipantID string `json:"participantId"`
			}
			if err := json.Unmarshal(body, &doc); err != nil {
				return fail(StartupMaterializationFailed, "Start", "participant descriptor is invalid", map[string]string{"app": appID, "participant": id}, err)
			}
			if doc.Kind != "tinkabot.localProfile.v1" || doc.Role != RoleParticipant || doc.Trust != "app-participant" || doc.Status != "active" {
				continue
			}
			if doc.AppID != appID || doc.ParticipantID != id || !validParticipantToken(appID) || !validParticipantToken(id) {
				return fail(StartupMaterializationFailed, "Start", "participant descriptor does not match its profile dir", map[string]string{"app": appID, "participant": id}, nil)
			}
			creds := filepath.Join(store, "participant.creds")
			if _, err := os.Stat(creds); err != nil {
				return fail(StartupMaterializationFailed, "Start", "active participant creds are missing", map[string]string{"app": appID, "participant": id}, err)
			}
			prof := ParticipantProfile{
				AppID:         appID,
				ParticipantID: id,
				StoreDir:      store,
				CredsFile:     creds,
				RecordKey:     participantKey(appID, id),
			}
			if err := writeParticipantDescriptor(prof, a.posture, "active"); err != nil {
				return fail(StartupMaterializationFailed, "Start", "participant descriptor could not be refreshed", map[string]string{"app": appID, "participant": id}, err)
			}
		}
	}
	return nil
}

func participantKey(appID, id string) string {
	return "participants." + appID + "." + id
}

func participantActionPrefix(appID, id string) string {
	return "apps." + appID + ".participants." + id + ".actions"
}

func participantActionSubject(appID, id string) string {
	return "tb.app." + appID + ".participants." + id + ".action"
}

func appStatePrefix(appID string) string {
	return "apps." + appID + ".state."
}

func validParticipantToken(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
		case r == '-':
		default:
			return false
		}
	}
	return true
}

func validProductItemKey(key string) bool {
	if key == "" || strings.HasPrefix(key, "/") || strings.HasSuffix(key, "/") || strings.Contains(key, "//") || strings.Contains(key, "..") {
		return false
	}
	for _, r := range key {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '.', r == '_', r == '-', r == '/':
		default:
			return false
		}
	}
	return true
}

func validWatcherTarget(scope, target string) bool {
	switch scope {
	case "item":
		return validProductItemKey(target)
	case "prefix":
		if target == "" || strings.HasPrefix(target, "/") || strings.Contains(target, "//") || strings.Contains(target, "..") {
			return false
		}
		for _, r := range target {
			switch {
			case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			case r == '.', r == '_', r == '-', r == '/':
			default:
				return false
			}
		}
		return true
	default:
		return false
	}
}

func participantAuth(appID, id, nonce string) core.Auth {
	w := wiring()
	action := participantActionPrefix(appID, id)
	state := appStatePrefix(appID)
	pub := []string{
		"$JS.API.INFO",
		"$JS.API.STREAM.INFO.KV_" + w.ItemBucket,
		"$JS.API.CONSUMER.CREATE.KV_" + w.ItemBucket + ".*.$KV." + w.ItemBucket + "." + state + ">",
		"$JS.API.CONSUMER.CREATE.KV_" + w.ItemBucket + ".*.$KV." + w.ItemBucket + "." + action + ".>",
		"$JS.API.DIRECT.GET.KV_" + w.ItemBucket + ".$KV." + w.ItemBucket + "." + participantKey(appID, id),
		"$JS.API.DIRECT.GET.KV_" + w.ItemBucket + ".$KV." + w.ItemBucket + "." + action + ".>",
		"$JS.API.DIRECT.GET.KV_" + w.ItemBucket + ".$KV." + w.ItemBucket + "." + state + ">",
		participantActionSubject(appID, id),
	}
	user := "participant." + appID + "." + id
	return core.Auth{
		User: user,
		Capability: core.Capability{
			PrincipalID:   user,
			SessionID:     "app." + appID,
			CapabilityID:  "participant." + appID + "." + id,
			LeaseID:       "lease-participant-" + appID + "-" + id + "-" + nonce,
			LeaseStatus:   "active",
			AppRevision:   appRevision,
			SchemaVersion: "v1",
		},
		Permissions: core.Permissions{
			Publish:   core.PermList{Allow: pub, Deny: []string{"tb.internal.>"}},
			Subscribe: core.PermList{Allow: []string{"_INBOX.>"}, Deny: []string{"tb.internal.>"}},
		},
	}
}

func watcherAuth(name, scope, target, nonce string) core.Auth {
	w := wiring()
	filter := target
	if scope == "prefix" {
		filter = strings.TrimSuffix(target, ".") + ".>"
	}
	user := "watcher." + name
	return core.Auth{
		User: user,
		Capability: core.Capability{
			PrincipalID:   user,
			SessionID:     "watch." + name,
			CapabilityID:  "watcher." + name,
			LeaseID:       "lease-watcher-" + name + "-" + nonce,
			LeaseStatus:   "active",
			AppRevision:   appRevision,
			SchemaVersion: "v1",
		},
		Permissions: core.Permissions{
			Publish: core.PermList{Allow: []string{
				"$JS.API.INFO",
				"$JS.API.STREAM.INFO.KV_" + w.ItemBucket,
				"$JS.API.CONSUMER.CREATE.KV_" + w.ItemBucket + ".*.$KV." + w.ItemBucket + "." + filter,
			}, Deny: []string{"tb.internal.>"}},
			Subscribe: core.PermList{Allow: []string{"_INBOX.>"}, Deny: []string{"tb.internal.>"}},
		},
	}
}

func (a *App) revokePriorParticipant(recordKey, currentUserPub string) error {
	prev, ok, err := a.readParticipant(recordKey)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	if prev.UserPub == "" || prev.UserPub == currentUserPub || a.rt.IsRevoked(embednats.AppAccount, prev.UserPub) {
		return nil
	}
	if err := a.rt.Revoke(embednats.AppAccount, prev.UserPub); err != nil {
		return fail(StartupMaterializationFailed, "ParticipantRecord", "prior participant creds could not be revoked", map[string]string{"participant": recordKey, "user": prev.UserPub}, err)
	}
	return nil
}

func (a *App) readParticipant(recordKey string) (participantRecord, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	nc, err := a.rt.ConnectCreds(ctx, a.creds[RoleCaller].File)
	if err != nil {
		return participantRecord{}, false, fail(StartupMaterializationFailed, "ParticipantRecord", "participant record connection failed", nil, err)
	}
	defer nc.Close()
	js, err := nc.JetStream()
	if err != nil {
		return participantRecord{}, false, fail(StartupMaterializationFailed, "ParticipantRecord", "participant record jetstream failed", nil, err)
	}
	kv, err := js.KeyValue(wiring().ItemBucket)
	if err != nil {
		return participantRecord{}, false, fail(StartupMaterializationFailed, "ParticipantRecord", "participant record bucket missing", nil, err)
	}
	entry, err := kv.Get(recordKey)
	if errors.Is(err, nats.ErrKeyNotFound) {
		return participantRecord{}, false, nil
	}
	if err != nil {
		return participantRecord{}, false, fail(StartupMaterializationFailed, "ParticipantRecord", "participant record read failed", nil, err)
	}
	var rec participantRecord
	if err := json.Unmarshal(entry.Value(), &rec); err != nil || rec.Kind != "tinkabot.participant.v1" {
		return participantRecord{}, false, fail(StartupMaterializationFailed, "ParticipantRecord", "participant record invalid", nil, err)
	}
	return rec, true, nil
}

func (a *App) writeParticipant(prof ParticipantProfile, status, revokedAt string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	nc, err := a.rt.ConnectCreds(ctx, a.creds[RoleCaller].File)
	if err != nil {
		return fail(StartupMaterializationFailed, "ParticipantRecord", "participant record connection failed", nil, err)
	}
	defer nc.Close()
	js, err := nc.JetStream()
	if err != nil {
		return fail(StartupMaterializationFailed, "ParticipantRecord", "participant record jetstream failed", nil, err)
	}
	kv, err := js.KeyValue(wiring().ItemBucket)
	if err != nil {
		return fail(StartupMaterializationFailed, "ParticipantRecord", "participant record bucket missing", nil, err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	rec := participantRecord{
		Kind:          "tinkabot.participant.v1",
		AppID:         prof.AppID,
		ParticipantID: prof.ParticipantID,
		Role:          RoleParticipant,
		Status:        status,
		UserPub:       prof.UserPub,
		ProfileSource: "local-store:" + prof.StoreDir,
		UpdatedAt:     now,
		RevokedAt:     revokedAt,
		Provenance: itemProv{
			Profile: "tinkabot",
			Source:  "participant:" + prof.AppID + ":" + prof.ParticipantID,
			Writer:  "tinkabot-participant",
		},
	}
	rec.LeaseID = prof.LeaseID
	entry, err := kv.Get(prof.RecordKey)
	if err == nil {
		var prev participantRecord
		if json.Unmarshal(entry.Value(), &prev) == nil && prev.CreatedAt != "" {
			rec.CreatedAt = prev.CreatedAt
		}
		body, err := json.Marshal(rec)
		if err != nil {
			return fail(StartupMaterializationFailed, "ParticipantRecord", "participant record invalid", nil, err)
		}
		if _, err := kv.Update(prof.RecordKey, body, entry.Revision()); err != nil {
			return fail(StartupMaterializationFailed, "ParticipantRecord", "participant record update failed", nil, err)
		}
		return nil
	}
	if !errors.Is(err, nats.ErrKeyNotFound) {
		return fail(StartupMaterializationFailed, "ParticipantRecord", "participant record read failed", nil, err)
	}
	rec.CreatedAt = now
	body, err := json.Marshal(rec)
	if err != nil {
		return fail(StartupMaterializationFailed, "ParticipantRecord", "participant record invalid", nil, err)
	}
	if _, err := kv.Create(prof.RecordKey, body); err != nil {
		return fail(StartupMaterializationFailed, "ParticipantRecord", "participant record create failed", nil, err)
	}
	return nil
}

// CheckManual verifies docs/manual/v1.md names the served binary surface: the
// starting-the-binary section, the persistent operator key, every manual
// role's creds file, and the shell's service-worker scope.
func CheckManual(doc []byte, p Posture) error {
	text := string(doc)
	miss := func(what, want string) error {
		return fail(ManualDivergence, "CheckManual", "manual diverges from the served surface: missing "+what, map[string]string{"want": want}, nil)
	}
	if !strings.Contains(text, "## Starting the binary") {
		return miss("starting-the-binary section", "## Starting the binary")
	}
	key := filepath.Base(p.NATS.Operator.KeyFile)
	if key == "." || !strings.Contains(text, key) {
		return miss("operator key file", key)
	}
	for _, role := range []string{RoleCaller, RoleObserver, RoleAuthor} {
		if file := role + ".creds"; !strings.Contains(text, file) {
			return miss("role creds file", file)
		}
	}
	if p.Shell.Scope == "" || !strings.Contains(text, p.Shell.Scope) {
		return miss("shell service-worker scope", p.Shell.Scope)
	}
	return nil
}

func principal(user, lease string, perms core.Permissions) core.Auth {
	return core.Auth{
		User: user,
		Capability: core.Capability{
			PrincipalID:   user,
			SessionID:     "session-" + lease,
			CapabilityID:  "cap-" + user,
			LeaseID:       lease,
			LeaseStatus:   "active",
			AppRevision:   appRevision,
			SchemaVersion: "v1",
		},
		Permissions: perms,
	}
}

// rolePerms compiles the manual roles' NATS authority: a caller may trigger
// and feed the config/upload surfaces, an observer reads materials, an author
// writes script records. Deny wins; tb.internal.> stays out of caller reach.
func rolePerms(w Wiring) map[string]core.Permissions {
	inbox := core.PermList{Allow: []string{"_INBOX.>"}}
	caller := append([]string{"$JS.API.INFO", w.TriggerSubject, w.EventsSubject, "$KV." + w.ConfigBucket + ".>", "$KV." + w.ItemBucket + ".>", "$KV." + w.ScheduleBucket + ".>", "$O." + w.UploadBucket + ".>"}, readKV(w.ConfigBucket)...)
	caller = append(caller, readKV(w.ItemBucket)...)
	caller = append(caller, readKV(w.ScheduleBucket)...)
	caller = append(caller, readObj(w.UploadBucket)...)
	observer := append([]string{"$JS.API.INFO", "_INBOX.>"}, readKV(w.MaterialBucket)...)
	observer = append(observer, readObj(w.ArtifactBucket)...)
	author := append([]string{"$JS.API.INFO", "_INBOX.>", "$KV." + w.ScriptBucket + ".>"}, readKV(w.ScriptBucket)...)
	return map[string]core.Permissions{
		RoleCaller: {
			Publish:   core.PermList{Allow: caller, Deny: []string{"tb.internal.>"}},
			Subscribe: core.PermList{Allow: []string{"_INBOX.>"}, Deny: []string{"tb.internal.>"}},
		},
		RoleObserver: {Publish: core.PermList{Allow: observer}, Subscribe: inbox},
		RoleAuthor:   {Publish: core.PermList{Allow: author}, Subscribe: inbox},
	}
}

func routerPerms(w Wiring) core.Permissions {
	api := append([]string{"$JS.API.INFO", "_INBOX.>", "$KV." + w.LedgerBucket + ".>", "$JS.API.STREAM.CREATE.KV_" + w.LedgerBucket}, readKV(w.LedgerBucket)...)
	return core.Permissions{
		Publish:   core.PermList{Allow: api},
		Subscribe: core.PermList{Allow: []string{w.TriggerSubject, "_INBOX.>"}},
	}
}

// servicePerms is the materializer principal's enumerated authority: create
// the assembly's stores, read scripts, write material/artifact state. No
// account-wide JetStream admin — deletion/purge of the ledger and script
// streams stays beyond this principal's reach. PURGE is granted narrowly on
// the artifact OBJ stream only, because object-store overwrite purges the
// prior object's chunks (nats.go Put); without it, re-emitting an artifact
// under a stable name fails ArtifactWriteFailed.
func servicePerms(w Wiring) core.Permissions {
	pub := []string{
		"$JS.API.INFO", "_INBOX.>",
		"$JS.API.STREAM.CREATE." + eventsStream,
		"$JS.API.STREAM.CREATE.KV_" + w.ConfigBucket,
		"$JS.API.STREAM.CREATE.KV_" + w.ItemBucket,
		"$JS.API.STREAM.CREATE.OBJ_" + w.UploadBucket,
		"$JS.API.STREAM.INFO.OBJ_" + w.UploadBucket,
		"$JS.API.STREAM.CREATE.KV_" + w.ScriptBucket,
		"$JS.API.STREAM.CREATE.KV_" + w.MaterialBucket,
		"$JS.API.STREAM.CREATE.OBJ_" + w.ArtifactBucket,
		"$JS.API.STREAM.PURGE.OBJ_" + w.ArtifactBucket,
		"$KV." + w.MaterialBucket + ".>",
		"$O." + w.ArtifactBucket + ".>",
	}
	pub = append(pub, readKV(w.ScriptBucket)...)
	pub = append(pub, readKV(w.MaterialBucket)...)
	pub = append(pub, readObj(w.ArtifactBucket)...)
	return core.Permissions{
		Publish:   core.PermList{Allow: pub},
		Subscribe: core.PermList{Allow: []string{"_INBOX.>"}},
	}
}

func schedulePerms(w Wiring) core.Permissions {
	pub := []string{
		"$JS.API.INFO", "_INBOX.>",
		"$JS.API.STREAM.CREATE.KV_" + w.ScheduleBucket,
		"$KV." + w.ScheduleBucket + ".>",
		"$JS.API.STREAM.INFO.KV_" + w.ItemBucket,
		"$JS.API.DIRECT.GET.KV_" + w.ItemBucket + ".>",
		"$KV." + w.ItemBucket + ".>",
	}
	pub = append(pub, readKV(w.ScheduleBucket)...)
	return core.Permissions{
		Publish:   core.PermList{Allow: pub},
		Subscribe: core.PermList{Allow: []string{"_INBOX.>"}},
	}
}

func actionServicePerms(w Wiring) core.Permissions {
	pub := []string{
		"$JS.API.INFO",
		"$JS.API.STREAM.INFO.KV_" + w.ItemBucket,
		"$JS.API.DIRECT.GET.KV_" + w.ItemBucket + ".$KV." + w.ItemBucket + ".apps.*.state.>",
		"$JS.API.DIRECT.GET.KV_" + w.ItemBucket + ".$KV." + w.ItemBucket + ".apps.*.participants.*.actions.>",
		"$KV." + w.ItemBucket + ".apps.*.participants.*.actions.>",
	}
	return core.Permissions{
		Publish:        core.PermList{Allow: pub, Deny: []string{"tb.internal.>"}},
		Subscribe:      core.PermList{Allow: []string{"tb.app.*.participants.*.action", "_INBOX.>"}, Deny: []string{"tb.internal.>"}},
		AllowResponses: core.AllowResponses{Max: 1, ExpiresMs: 30000},
	}
}

func browserCommandPerms(w Wiring) core.Permissions {
	pub := []string{
		"$JS.API.INFO",
		"$JS.API.STREAM.INFO.KV_" + w.ItemBucket,
		"$JS.API.CONSUMER.CREATE.KV_" + w.ItemBucket + ".*.$KV." + w.ItemBucket + ".apps.*.state.>",
		"$JS.API.CONSUMER.DELETE.KV_" + w.ItemBucket + ".>",
		"$JS.API.DIRECT.GET.KV_" + w.ItemBucket + ".$KV." + w.ItemBucket + ".apps.*.state.>",
		"$JS.API.DIRECT.GET.KV_" + w.ItemBucket + ".$KV." + w.ItemBucket + ".apps.*.participants.*.actions.>",
		"$JS.API.DIRECT.GET.KV_" + w.ItemBucket + ".$KV." + w.ItemBucket + ".artifacts.*.results.>",
		"$KV." + w.ItemBucket + ".artifacts.*.results.>",
		"tb.app.*.participants.*.action",
		"tb.app.browser.state.>",
	}
	return core.Permissions{
		Publish:        core.PermList{Allow: pub, Deny: []string{"tb.internal.>"}},
		Subscribe:      core.PermList{Allow: []string{"tb.app.browser.command", "_INBOX.>"}, Deny: []string{"tb.internal.>"}},
		AllowResponses: core.AllowResponses{Max: 1, ExpiresMs: 30000},
	}
}

func readKV(bucket string) []string  { return readStore("KV_" + bucket) }
func readObj(bucket string) []string { return readStore("OBJ_" + bucket) }

func readStore(stream string) []string {
	return []string{
		"$JS.API.STREAM.INFO." + stream,
		"$JS.API.DIRECT.GET." + stream + ".>",
		"$JS.API.STREAM.MSG.GET." + stream,
		"$JS.API.CONSUMER.CREATE." + stream + ".>",
		"$JS.API.CONSUMER.MSG.NEXT." + stream + ".>",
		"$JS.API.CONSUMER.DELETE." + stream + ".>",
	}
}

// sourceAuthority is the manual trigger's source-authority policy: the caller
// principal's lease bound to the trigger subject with bounded responses.
func sourceAuthority(w Wiring, cap core.Capability) core.Auth {
	src := w.TriggerSubject
	return core.Auth{
		User:       cap.PrincipalID,
		Capability: cap,
		Permissions: core.Permissions{
			Publish:        core.PermList{Allow: []string{src}, Deny: []string{"tb.internal.>"}},
			Subscribe:      core.PermList{Allow: []string{src, "_INBOX.>"}, Deny: []string{"tb.internal.>"}},
			AllowResponses: core.AllowResponses{Max: 1, ExpiresMs: 30000},
		},
		Imports:  map[string]core.Import{"trigger": {Kind: "subscribe", Subjects: []string{src}, Desc: "manual trigger watch"}},
		Exports:  []string{src},
		Exposure: map[string]core.Exposure{authorityRef: {Kind: "request_reply", Subject: src, Desc: "manual trigger exposure"}},
	}
}

func activationFor(w Wiring, cap core.Capability) core.Activation {
	return core.Activation{
		ScriptKey:      w.ScriptKey,
		ScriptRevision: w.ScriptRevision,
		SourcePrincipal: core.SourcePrincipal{
			PrincipalID:  cap.PrincipalID,
			SourceID:     "src-trigger-main",
			SourceKind:   "request_reply",
			AuthorityRef: authorityRef,
		},
		SourceLease: core.SourceLease{
			LeaseID:        cap.LeaseID,
			LeaseStatus:    "active",
			AppRevision:    appRevision,
			SchemaVersion:  "v1",
			ScriptRevision: w.ScriptRevision,
		},
		Source:     core.Source{Kind: "request_reply", ActivationName: "trigger", Subject: w.TriggerSubject},
		Chain:      core.Chain{ChainID: "chain-trigger-main", RootID: "root-trigger-main", Hop: 1, MaxHops: 5},
		Capability: cap,
		Provenance: core.Provenance{
			SchemaID:      schemaID,
			SchemaVersion: "v1",
			AppRevision:   appRevision,
			CreatedAt:     time.Now().UTC().Format(time.RFC3339),
			Producer:      "activation",
		},
	}
}

func fail(kind Kind, op, msg string, details map[string]string, cause error) *Error {
	if details == nil {
		details = map[string]string{}
	}
	return &Error{Kind: kind, Operation: op, Message: msg, Details: details, Cause: cause}
}
