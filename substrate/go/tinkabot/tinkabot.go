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
	RoleCaller   = "caller"
	RoleObserver = "observer"
	RoleAuthor   = "author"
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
	app := &App{rt: rt, creds: map[string]embednats.UserCreds{}, files: map[string]string{}, storeDir: cfg.StoreDir, bundleSandbox: cfg.BundleSandbox}
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
	ok = true
	return app, nil
}

func (a *App) Posture() Posture                      { return a.posture }
func (a *App) Runtime() *embednats.Runtime           { return a.rt }
func (a *App) Creds(role string) embednats.UserCreds { return a.creds[role] }
func (a *App) CredsFile(role string) string          { return a.files[role] }

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
// config; account identity is ephemeral, so a restart starts a fresh plane
// over the same operator authority).
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
	caller := append([]string{"$JS.API.INFO", w.TriggerSubject, w.EventsSubject, "$KV." + w.ConfigBucket + ".>", "$O." + w.UploadBucket + ".>"}, readKV(w.ConfigBucket)...)
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
