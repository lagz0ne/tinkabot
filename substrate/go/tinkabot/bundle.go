package tinkabot

// Bundle: one folder served as one ephemeral app for the lifetime of the run
// (docs/matched-abstraction/approach/bundle-v1.md). The loader is an
// automated author — manifest entries become ordinary script records in a
// memory-storage bucket that dies with the process, wired to per-entry
// trigger routes whose effects pass the normal materializer gate. A bundle
// may not claim any durably-claimed authority; collisions are typed load
// failures and the binary refuses to start.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/lagz0ne/tinkabot/substrate/go/core"
	"github.com/lagz0ne/tinkabot/substrate/go/embednats"
	"github.com/nats-io/nats.go"
)

// The bundle plane reuses the app plane's bucket names on purpose: the
// bundle's NATS account is the namespace, so the same names hold unrelated
// state — nothing to coordinate, nothing to collide.
const (
	bundleBucket         = "tb_bundle"
	bundleMaterialBucket = "tb_material"
	bundleArtifactBucket = "tb_artifacts"
	bundleLedgerBucket   = "tb_ledger"
)

var bundleName = regexp.MustCompile(`^[a-z0-9-]+$`)

type bundleManifest struct {
	Kind    string         `json:"kind"`
	Name    string         `json:"name"`
	Scripts []bundleScript `json:"scripts"`
}

// bundleScript declares no authority: script key, trigger subject,
// projection ids, and artifact prefix are all derived under the bundle's
// namespace, so a manifest cannot even spell a collision with durable
// claims. Projections name short ids prefixed at load.
type bundleScript struct {
	Name           string   `json:"name"`
	Desc           string   `json:"desc,omitempty"`
	File           string   `json:"file"`
	Command        string   `json:"command"`
	TimeoutMs      int      `json:"timeoutMs,omitempty"`
	ScriptRevision int      `json:"scriptRevision,omitempty"`
	Projections    []string `json:"projections,omitempty"`
	Boot           bool     `json:"boot,omitempty"`
	// Every declares the automated cadence as intent; runtime control rides
	// the app config bucket (`bundle.<bundle>.<entry>.every`: a duration
	// retunes, `off` pauses, delete falls back here).
	Every string `json:"every,omitempty"`
	// Watches names the short projection id this entry observes; a watches
	// entry is a long-lived filter fed each change of that projection.
	Watches string `json:"watches,omitempty"`

	everyDur time.Duration
}

// minEvery floors the tick cadence — below this a schedule is a busy loop,
// not automation.
const minEvery = 100 * time.Millisecond

type bundle struct {
	dir      string
	manifest bundleManifest
}

func (b *bundle) scriptKey(e bundleScript) string {
	return "scripts.bundle." + b.manifest.Name + "." + e.Name
}

func (b *bundle) trigger(e bundleScript) string {
	return "tb.bundle." + b.manifest.Name + "." + e.Name
}

func (b *bundle) projections(e bundleScript) []string {
	ids := make([]string, len(e.Projections))
	for i, p := range e.Projections {
		ids[i] = "bundle." + b.manifest.Name + "." + p
	}
	return ids
}

func (b *bundle) artifactPrefix() string {
	return "bundle/" + b.manifest.Name + "/"
}

func rejectBundle(msg string, details map[string]string, cause error) *Error {
	return fail(BundleRejected, "LoadBundle", msg, details, cause)
}

// loadBundle reads and validates the manifest. Pure file work — it runs
// before any NATS state exists, so a bad bundle fails the start before it
// costs anything.
func loadBundle(dir string) (*bundle, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, rejectBundle("bundle dir could not be resolved", map[string]string{"dir": dir}, err)
	}
	raw, err := os.ReadFile(filepath.Join(abs, "bundle.json"))
	if err != nil {
		return nil, rejectBundle("bundle manifest could not be read", map[string]string{"dir": abs}, err)
	}
	var m bundleManifest
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&m); err != nil {
		return nil, rejectBundle("bundle manifest could not be decoded", map[string]string{"dir": abs}, err)
	}
	if dec.Decode(&struct{}{}) != io.EOF {
		return nil, rejectBundle("bundle manifest has trailing content", map[string]string{"dir": abs}, nil)
	}
	if m.Kind != "bundle.manifest" {
		return nil, rejectBundle("bundle manifest kind drift", map[string]string{"kind": m.Kind}, nil)
	}
	if !bundleName.MatchString(m.Name) {
		return nil, rejectBundle("bundle name is invalid", map[string]string{"name": m.Name}, nil)
	}
	if len(m.Scripts) == 0 {
		return nil, rejectBundle("bundle declares no scripts", map[string]string{"name": m.Name}, nil)
	}
	b := &bundle{dir: abs, manifest: m}
	seenNames, seenProjections := map[string]bool{}, map[string]bool{}
	// projectionOwner maps short projection id -> entry name that owns it.
	projectionOwner := map[string]string{}
	for i := range m.Scripts {
		e := &m.Scripts[i]
		if e.TimeoutMs == 0 {
			e.TimeoutMs = 2000
		}
		if e.ScriptRevision == 0 {
			e.ScriptRevision = 1
		}
		at := map[string]string{"script": e.Name}
		switch {
		case !bundleName.MatchString(e.Name):
			return nil, rejectBundle("bundle script name is invalid", at, nil)
		case e.File == "" || e.Command == "" || e.ScriptRevision < 0:
			return nil, rejectBundle("bundle script entry is incomplete", at, nil)
		case seenNames[e.Name]:
			return nil, rejectBundle("bundle script name is duplicated", at, nil)
		}
		for _, p := range e.Projections {
			if !bundleName.MatchString(p) {
				return nil, rejectBundle("bundle projection id is invalid", at, nil)
			}
			if seenProjections[p] {
				return nil, rejectBundle("bundle projection is duplicated", at, nil)
			}
			seenProjections[p] = true
			projectionOwner[p] = e.Name
		}
		if !filepath.IsLocal(e.File) {
			return nil, rejectBundle("bundle script file escapes the bundle dir", at, nil)
		}
		if _, err := os.Stat(filepath.Join(abs, e.File)); err != nil {
			return nil, rejectBundle("bundle script file is missing", at, err)
		}
		if e.Every != "" {
			d, err := time.ParseDuration(e.Every)
			if err != nil || d < minEvery {
				return nil, rejectBundle("bundle schedule cadence is invalid", at, err)
			}
			e.everyDur = d
		}
		seenNames[e.Name] = true
	}
	// Second pass: validate filter (watches) entries.
	for _, e := range m.Scripts {
		if e.Watches == "" {
			continue
		}
		at := map[string]string{"script": e.Name}
		if !bundleName.MatchString(e.Watches) {
			return nil, rejectBundle("bundle watches id is invalid", at, nil)
		}
		if e.Boot {
			return nil, rejectBundle("bundle filter must not declare boot", at, nil)
		}
		if e.everyDur > 0 {
			return nil, rejectBundle("bundle filter must not declare every", at, nil)
		}
		if len(e.Projections) == 0 {
			return nil, rejectBundle("bundle filter must declare at least one projection", at, nil)
		}
		owner, known := projectionOwner[e.Watches]
		if !known {
			return nil, rejectBundle("bundle filter watches unknown projection", at, nil)
		}
		if owner == e.Name {
			return nil, rejectBundle("bundle filter must not watch its own projection", at, nil)
		}
	}
	// DAG check: detect cycles in the watches graph (edge: watcher -> owner of watched projection).
	if err := checkWatchDAG(m.Scripts, projectionOwner); err != nil {
		return nil, rejectBundle("bundle watches graph contains a cycle", nil, err)
	}
	return b, nil
}

// checkWatchDAG returns an error if the watches graph contains a cycle.
func checkWatchDAG(scripts []bundleScript, projectionOwner map[string]string) error {
	// Build adjacency: entry name -> name of entry it watches (via owned projection).
	watchEdge := map[string]string{}
	for _, e := range scripts {
		if e.Watches != "" {
			if owner, ok := projectionOwner[e.Watches]; ok {
				watchEdge[e.Name] = owner
			}
		}
	}
	visited := map[string]bool{}
	inStack := map[string]bool{}
	var visit func(name string) bool
	visit = func(name string) bool {
		if inStack[name] {
			return true // cycle
		}
		if visited[name] {
			return false
		}
		visited[name] = true
		inStack[name] = true
		if next, ok := watchEdge[name]; ok {
			if visit(next) {
				return true
			}
		}
		inStack[name] = false
		return false
	}
	for _, e := range scripts {
		if visit(e.Name) {
			return fmt.Errorf("cycle detected")
		}
	}
	return nil
}

// subjects is the bundle's entire app-account reach: one wildcard under its
// derived namespace, mapped onto the bundle account by service imports.
func (b *bundle) subjects() []string {
	return []string{"tb.bundle." + b.manifest.Name + ".>"}
}

func (b *bundle) account() string {
	return "TB_BUNDLE_" + strings.ToUpper(strings.ReplaceAll(b.manifest.Name, "-", "_"))
}

// bundleDeps carries the Start-time seams the bundle wiring consumes; every
// behavior comes from the same proven pieces the manual slot uses.
type bundleDeps struct {
	cap    core.Capability
	nonce  string
	dial   func(embednats.UserCreds, string) (*nats.Conn, error)
	caller embednats.UserCreds
}

// startBundle serves the bundle inside its own minted NATS account: scripts,
// materials, artifacts, and ledger all live in that account's JetStream
// plane (memory storage, dead with the process), and the only crossing into
// TB_APP is the service export/import per trigger. The account boundary —
// not naming — is what keeps bundle and app state apart.
// preflightSandbox resolves the bwrap binary (TB_BWRAP override, else PATH)
// and proves it actually jails before any bundle entry is wired. Fail-closed:
// a missing binary or a failed smoke run rejects the bundle rather than
// letting it run unjailed.
func preflightSandbox() (string, error) {
	bin := os.Getenv("TB_BWRAP")
	if bin == "" {
		resolved, err := exec.LookPath("bwrap")
		if err != nil {
			return "", err
		}
		bin = resolved
	}
	smoke := exec.Command(bin, "--ro-bind", "/", "/", "--unshare-all", "true")
	if err := smoke.Run(); err != nil {
		return "", err
	}
	return bin, nil
}

// installDeps runs `bun install` at load. It runs JAILED, not on the bare host:
// an install runs package lifecycle scripts, which would be host RCE with full
// env. The install jail differs from the runtime jail in one way only — it
// SHARES the network so deps download — while still masking the substrate store
// and the user's $HOME so a lifecycle script can neither read secrets nor write
// outside the bundle.
//
// --ignore-scripts skips ALL package lifecycle scripts (both the bundle's own
// and dependencies'), which is the arbitrary-code-at-install hole closed. It
// does NOT break the builder: vite/esbuild ship their native binaries as
// per-platform optionalDependency packages (e.g. @esbuild/linux-x64), which
// need no postinstall — verified against bun 1.3 (esbuild --version works
// after --ignore-scripts). The jail is defense-in-depth: even the dependency
// resolution/download bun does runs with secrets masked and writes confined.
//
// Layout: --ro-bind / / for the toolchain, then the bundle dir bound READ-WRITE
// (node_modules lands there), --tmpfs over $HOME and the store dir (secrets
// masked; a fresh HOME is set so bun has a clean writable cache), --tmpfs /tmp,
// share-net kept (only --unshare-pid/--uts), a 120s timeout, and a scrubbed env
// exposing only PATH + HOME so bun finds itself but inherits no host secrets.
//
// Skipped silently when the bundle declares no package.json.
func installDeps(dir, bwrapBin, storeDir string) error {
	if _, err := os.Stat(filepath.Join(dir, "package.json")); err != nil {
		return nil
	}
	bunBin, err := exec.LookPath("bun")
	if err != nil {
		return rejectBundle("bundle dependency install needs bun", map[string]string{"dir": dir}, err)
	}
	home := os.Getenv("HOME")
	// A fresh writable HOME inside the jail: bun's cache/config live under $HOME,
	// but the host $HOME (with the user's secrets) stays masked by --tmpfs.
	jailHome := "/tmp/tb-bun-home"

	argv := []string{
		"--ro-bind", "/", "/",
		"--dev", "/dev",
		"--proc", "/proc",
	}
	if home != "" {
		// Mask the user's HOME (ssh/aws/etc.) BEFORE the binds below: in dev
		// setups the bundle dir AND the bun binary live UNDER $HOME, so the
		// tmpfs must precede their re-binds or it clobbers them (chdir would
		// then fail). Order is load-bearing.
		argv = append(argv, "--tmpfs", home)
	}
	// /tmp tmpfs, then the writable bun HOME inside it (tmpfs before --dir).
	argv = append(argv, "--tmpfs", "/tmp", "--dir", jailHome)
	// Re-expose the bundle dir (rw — node_modules lands here) and the bun
	// binary dir (ro) AFTER the HOME mask above.
	argv = append(argv, "--bind", dir, dir, "--ro-bind", filepath.Dir(bunBin), filepath.Dir(bunBin))
	if storeDir != "" {
		argv = append(argv, "--tmpfs", storeDir) // explicit store mask if not under HOME
	}
	argv = append(argv,
		"--chdir", dir,
		"--unshare-pid", "--unshare-uts",
		"--die-with-parent",
		"--clearenv",
		"--setenv", "PATH", os.Getenv("PATH"),
		"--setenv", "HOME", jailHome,
		"--", bunBin, "install", "--ignore-scripts",
	)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bwrapBin, argv...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return rejectBundle("bundle dependency install failed", map[string]string{"dir": dir, "output": string(out)}, err)
	}
	return nil
}

func (a *App) startBundle(b *bundle, deps bundleDeps) error {
	// Fail-closed: bundle processes only run jailed. If the host cannot
	// sandbox, the bundle refuses to start.
	bwrapBin, err := preflightSandbox()
	if err != nil {
		return rejectBundle("bundle sandbox unavailable", map[string]string{"dir": b.dir}, err)
	}
	// Mask the substrate store dir inside the runtime jail: --ro-bind / / would
	// otherwise let a jailed script read operator.nk / role creds.
	storeDir, err := filepath.Abs(a.storeDir)
	if err != nil {
		return rejectBundle("bundle store dir could not be resolved", map[string]string{"dir": a.storeDir}, err)
	}
	sandbox := &embednats.SandboxConfig{BundleDir: b.dir, Bwrap: bwrapBin, StoreDir: storeDir}
	// Install deps in a jail that shares net (deps download) but masks the
	// store + $HOME so a package lifecycle script cannot read or write secrets.
	if err := installDeps(b.dir, bwrapBin, storeDir); err != nil {
		return err
	}
	acct := b.account()
	if err := a.rt.MintAccount(acct); err != nil {
		return rejectBundle("bundle account could not be minted", map[string]string{"account": acct}, err)
	}
	svcUC, err := a.rt.MintUser(acct, principal("principal.bundle."+b.manifest.Name+".service", "lease-bundle-svc-"+deps.nonce, bundleServicePerms()), servingTTL)
	if err != nil {
		return rejectBundle("bundle service creds could not be minted", map[string]string{"account": acct}, err)
	}
	routerUC, err := a.rt.MintUser(acct, principal("principal.bundle."+b.manifest.Name+".router", "lease-bundle-router-"+deps.nonce, bundleRouterPerms(b)), servingTTL)
	if err != nil {
		return rejectBundle("bundle router creds could not be minted", map[string]string{"account": acct}, err)
	}

	scriptsNC, err := deps.dial(svcUC, "bundle script store")
	if err != nil {
		return err
	}
	js, err := scriptsNC.JetStream()
	if err != nil {
		return rejectBundle("bundle store JetStream context is unavailable", nil, err)
	}
	// Memory storage keeps the store dir clean of per-run files; the account
	// lifecycle is what makes the plane ephemeral.
	for _, bucket := range []string{bundleBucket, bundleMaterialBucket, bundleLedgerBucket} {
		if _, err := js.CreateKeyValue(&nats.KeyValueConfig{Bucket: bucket, Storage: nats.MemoryStorage}); err != nil {
			return rejectBundle("bundle bucket could not be created", map[string]string{"bucket": bucket}, err)
		}
	}
	if _, err := js.CreateObjectStore(&nats.ObjectStoreConfig{Bucket: bundleArtifactBucket, Storage: nats.MemoryStorage}); err != nil {
		return rejectBundle("bundle bucket could not be created", map[string]string{"bucket": bundleArtifactBucket}, err)
	}
	store, err := embednats.OpenKVScriptStore(scriptsNC, bundleBucket)
	if err != nil {
		return rejectBundle("bundle script bucket could not be opened", map[string]string{"bucket": bundleBucket}, err)
	}
	materialsNC, err := deps.dial(svcUC, "bundle material store")
	if err != nil {
		return err
	}
	materials, err := embednats.OpenKVMaterialStore(materialsNC, bundleMaterialBucket, bundleArtifactBucket)
	if err != nil {
		return rejectBundle("bundle material store could not be opened", nil, err)
	}
	a.bundleMaterials = materials
	ledgerNC, err := deps.dial(routerUC, "bundle ledger store")
	if err != nil {
		return err
	}
	ledgerStore, err := embednats.OpenKVLedgerStore(ledgerNC, bundleLedgerBucket)
	if err != nil {
		return rejectBundle("bundle ledger could not be opened", nil, err)
	}
	routerNC, err := deps.dial(routerUC, "bundle router")
	if err != nil {
		return err
	}
	mat, err := core.NewMaterializer(materials)
	if err != nil {
		return rejectBundle("bundle materializer could not be configured", nil, err)
	}

	// Obtain the material KV handle via the router connection for filter entries.
	routerJS, err := routerNC.JetStream()
	if err != nil {
		return rejectBundle("bundle router JetStream context is unavailable", nil, err)
	}

	ledger := core.NewDurableLedger(ledgerStore)
	for _, e := range b.manifest.Scripts {
		rec := core.ScriptRecord{
			Kind:     "script.record",
			Key:      b.scriptKey(e),
			Revision: e.ScriptRevision,
			Desc:     e.Desc,
			Process: core.Process{
				Command:   e.Command,
				Args:      []string{filepath.Join(b.dir, e.File)},
				Cwd:       b.dir,
				RPC:       "framed_stdio",
				TimeoutMs: e.TimeoutMs,
				Resource:  core.Resource{CPUMillis: 100, MemoryMB: 64},
				Kill:      "process.kill",
				Cleanup:   "workdir.keep",
				Identity:  "principal.bundle." + b.manifest.Name,
			},
		}
		if err := store.Put(rec); err != nil {
			return rejectBundle("bundle record could not be landed", map[string]string{"script": e.Name}, err)
		}

		rtm, err := core.NewScriptRuntime(core.ScriptPolicy{AllowedProjections: b.projections(e), ProjectionPrefix: "bundle." + b.manifest.Name + ".", ArtifactPrefix: b.artifactPrefix()}, embednats.LocalScriptRunner{Sandbox: sandbox})
		if err != nil {
			return rejectBundle("bundle script policy was denied", map[string]string{"script": e.Name}, err)
		}

		if e.Watches == "" {
			// Trigger entry: request/reply route + script loop + export/import.
			router, err := embednats.NewSourceRouter(bundleSourceAuthority(b, e, deps.cap), ledger)
			if err != nil {
				return rejectBundle("bundle trigger authority was denied", map[string]string{"trigger": b.trigger(e)}, err)
			}
			route, results, err := router.RequestReply(routerNC, bundleActivation(b, e, deps.cap))
			if err != nil {
				return rejectBundle("bundle trigger route could not be wired", map[string]string{"trigger": b.trigger(e)}, err)
			}
			a.routes = append(a.routes, route)
			runs, stop := embednats.NewScriptLoop(store, rtm, mat, materials, materials).Watch(results)
			a.stopLoops = append(a.stopLoops, stop)
			go func() {
				for range runs {
					// Run outcomes are event envelopes in the bundle's own
					// material bucket; nothing to do here.
				}
			}()
			// The only crossing: export the trigger service, import it into the
			// app account under the same derived name.
			if err := a.rt.ExportService(acct, b.trigger(e)); err != nil {
				return rejectBundle("bundle trigger could not be exported", map[string]string{"trigger": b.trigger(e)}, err)
			}
			if err := a.rt.ImportService(embednats.AppAccount, acct, b.trigger(e), ""); err != nil {
				return rejectBundle("bundle trigger could not be imported", map[string]string{"trigger": b.trigger(e)}, err)
			}
		} else {
			// Filter entry: KV watch on the material bucket key for the watched projection.
			kvh, err := routerJS.KeyValue(bundleMaterialBucket)
			if err != nil {
				return rejectBundle("bundle material KV handle is unavailable", map[string]string{"script": e.Name}, err)
			}
			router, err := embednats.NewSourceRouter(bundleKVSourceAuthority(b, e, deps.cap), ledger)
			if err != nil {
				return rejectBundle("bundle filter authority was denied", map[string]string{"script": e.Name}, err)
			}
			route, results, err := router.KV(kvh, bundleKVActivation(b, e, deps.cap))
			if err != nil {
				return rejectBundle("bundle filter route could not be wired", map[string]string{"script": e.Name}, err)
			}
			a.routes = append(a.routes, route)
			runs, stopLoop := embednats.NewFilterLoop(rec, rtm, mat, materials).WithSandbox(sandbox).Watch(results)
			a.stopLoops = append(a.stopLoops, stopLoop)
			go func() {
				for range runs {
					// Filter run outcomes land in the material bucket; nothing to do here.
				}
			}()
		}
	}

	callerNC, err := deps.dial(deps.caller, "bundle caller")
	if err != nil {
		return err
	}
	if err := a.bootBundle(b, callerNC, deps.nonce); err != nil {
		return err
	}
	return a.scheduleBundle(b, callerNC, deps.nonce)
}

// scheduleBundle starts one ticker per scheduled entry. Each tick fires the
// entry's trigger through the same caller path as boot — an ordinary
// attributed activation — and consults the app config bucket before every
// cycle so the cadence is NATS-controllable settings, not baked-in state.
func (a *App) scheduleBundle(b *bundle, nc *nats.Conn, nonce string) error {
	var scheduled []bundleScript
	for _, e := range b.manifest.Scripts {
		if e.everyDur > 0 {
			scheduled = append(scheduled, e)
		}
	}
	if len(scheduled) == 0 {
		return nil
	}
	js, err := nc.JetStream()
	if err != nil {
		return rejectBundle("bundle schedule JetStream context is unavailable", nil, err)
	}
	settings, err := js.KeyValue(wiring().ConfigBucket)
	if err != nil {
		return rejectBundle("bundle schedule settings bucket is unavailable", nil, err)
	}
	for _, e := range scheduled {
		stop := make(chan struct{})
		a.stopLoops = append(a.stopLoops, func() { close(stop) })
		go a.tickBundle(b, e, nc, settings, nonce, stop)
	}
	return nil
}

func (a *App) tickBundle(b *bundle, e bundleScript, nc *nats.Conn, settings nats.KeyValue, nonce string, stop <-chan struct{}) {
	key := "bundle." + b.manifest.Name + "." + e.Name + ".every"
	for n := 1; ; n++ {
		cadence, paused := e.everyDur, false
		if entry, err := settings.Get(key); err == nil {
			switch v := strings.TrimSpace(string(entry.Value())); {
			case v == "off":
				paused = true
			default:
				if d, err := time.ParseDuration(v); err == nil && d >= minEvery {
					cadence = d
				}
			}
		}
		select {
		case <-stop:
			return
		case <-time.After(cadence):
		}
		if paused {
			continue
		}
		msg := nats.NewMsg(b.trigger(e))
		msg.Header.Set(embednats.HeaderRequestID, "tick-"+nonce+"-"+e.Name+"-"+strconv.Itoa(n))
		msg.Data = []byte("tick")
		// Tick outcomes are recorded by the ledger like any activation; a
		// denied or unanswered tick must not kill the running app.
		_, _ = nc.RequestMsg(msg, time.Second)
	}
}

// bundleServicePerms is the bundle-plane materializer authority, enumerated
// against the bundle account's own buckets — the account boundary already
// bars everything else.
func bundleServicePerms() core.Permissions {
	pub := []string{
		"$JS.API.INFO", "_INBOX.>",
		"$JS.API.STREAM.CREATE.KV_" + bundleBucket,
		"$JS.API.STREAM.CREATE.KV_" + bundleMaterialBucket,
		"$JS.API.STREAM.CREATE.KV_" + bundleLedgerBucket,
		"$JS.API.STREAM.CREATE.OBJ_" + bundleArtifactBucket,
		// Overwriting an artifact under a stable name purges the prior
		// object's chunks (nats.go Put), so the filter chain that rebuilds an
		// app on every change needs PURGE on its own artifact bucket.
		"$JS.API.STREAM.PURGE.OBJ_" + bundleArtifactBucket,
		"$KV." + bundleBucket + ".>",
		"$KV." + bundleMaterialBucket + ".>",
		"$O." + bundleArtifactBucket + ".>",
	}
	pub = append(pub, readKV(bundleBucket)...)
	pub = append(pub, readKV(bundleMaterialBucket)...)
	pub = append(pub, readObj(bundleArtifactBucket)...)
	return core.Permissions{
		Publish:   core.PermList{Allow: pub},
		Subscribe: core.PermList{Allow: []string{"_INBOX.>"}},
	}
}

// bundleRouterPerms subscribes the bundle's trigger subjects and owns its
// ledger; replies cross the account boundary on the bounded response grant.
// readKV(bundleMaterialBucket) is included so filter entries can create a
// KV watch consumer on the material stream.
func bundleRouterPerms(b *bundle) core.Permissions {
	pub := append([]string{"$JS.API.INFO", "_INBOX.>", "$KV." + bundleLedgerBucket + ".>", "$JS.API.STREAM.CREATE.KV_" + bundleLedgerBucket}, readKV(bundleLedgerBucket)...)
	pub = append(pub, readKV(bundleMaterialBucket)...)
	subs := []string{"_INBOX.>", "$KV." + bundleMaterialBucket + ".>"}
	for _, e := range b.manifest.Scripts {
		if e.Watches == "" {
			subs = append(subs, b.trigger(e))
		}
	}
	return core.Permissions{
		Publish:        core.PermList{Allow: pub},
		Subscribe:      core.PermList{Allow: subs},
		AllowResponses: core.AllowResponses{Max: 1, ExpiresMs: 30000},
	}
}

// bootBundle fires each boot entry once through the ordinary caller-creds
// request/reply path — bundle load is just one more trigger source, deduped
// per run by the request id.
func (a *App) bootBundle(b *bundle, nc *nats.Conn, nonce string) error {
	var boots []bundleScript
	for _, e := range b.manifest.Scripts {
		if e.Boot {
			boots = append(boots, e)
		}
	}
	if len(boots) == 0 {
		return nil
	}
	for _, e := range boots {
		// The import's claims push propagates asynchronously; retry inside a
		// bounded window before declaring the boot dead.
		deadline := time.Now().Add(10 * time.Second)
		for {
			msg := nats.NewMsg(b.trigger(e))
			msg.Header.Set(embednats.HeaderRequestID, "boot-"+nonce)
			msg.Data = []byte("boot")
			reply, err := nc.RequestMsg(msg, time.Second)
			if err == nil {
				body := string(reply.Data)
				if !strings.Contains(body, "accepted") && !strings.Contains(body, "duplicate") {
					return rejectBundle("bundle boot trigger was denied", map[string]string{"trigger": b.trigger(e), "reply": body}, nil)
				}
				break
			}
			if time.Now().After(deadline) {
				return rejectBundle("bundle boot trigger got no reply", map[string]string{"trigger": b.trigger(e)}, err)
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
	return nil
}

func bundleSourceAuthority(b *bundle, e bundleScript, cap core.Capability) core.Auth {
	src := b.trigger(e)
	return core.Auth{
		User:       cap.PrincipalID,
		Capability: cap,
		Permissions: core.Permissions{
			Publish:        core.PermList{Allow: []string{src}, Deny: []string{"tb.internal.>"}},
			Subscribe:      core.PermList{Allow: []string{src, "_INBOX.>"}, Deny: []string{"tb.internal.>"}},
			AllowResponses: core.AllowResponses{Max: 1, ExpiresMs: 30000},
		},
		Imports:  map[string]core.Import{"trigger": {Kind: "subscribe", Subjects: []string{src}, Desc: "bundle trigger watch"}},
		Exports:  []string{src},
		Exposure: map[string]core.Exposure{bundleAuthorityRef(b, e): {Kind: "request_reply", Subject: src, Desc: "bundle trigger exposure"}},
	}
}

func bundleActivation(b *bundle, e bundleScript, cap core.Capability) core.Activation {
	id := b.manifest.Name + "-" + e.Name
	return core.Activation{
		ScriptKey:      b.scriptKey(e),
		ScriptRevision: e.ScriptRevision,
		SourcePrincipal: core.SourcePrincipal{
			PrincipalID:  cap.PrincipalID,
			SourceID:     "src-bundle-" + id,
			SourceKind:   "request_reply",
			AuthorityRef: bundleAuthorityRef(b, e),
		},
		SourceLease: core.SourceLease{
			LeaseID:        cap.LeaseID,
			LeaseStatus:    "active",
			AppRevision:    appRevision,
			SchemaVersion:  "v1",
			ScriptRevision: e.ScriptRevision,
		},
		Source:     core.Source{Kind: "request_reply", ActivationName: "bundle", Subject: b.trigger(e)},
		Chain:      core.Chain{ChainID: "chain-bundle-" + id, RootID: "root-bundle-" + id, Hop: 1, MaxHops: 5},
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

func bundleAuthorityRef(b *bundle, e bundleScript) string {
	return "auth.source.bundle." + b.manifest.Name + "-" + e.Name
}

// bundleKVSourceAuthority builds a source-authority policy for a filter entry
// watching a projection key in the bundle material bucket.
func bundleKVSourceAuthority(b *bundle, e bundleScript, cap core.Capability) core.Auth {
	key := "p.bundle." + b.manifest.Name + "." + e.Watches
	src := "$KV." + bundleMaterialBucket + "." + key
	ref := bundleKVAuthorityRef(b, e)
	return core.Auth{
		User:       cap.PrincipalID,
		Capability: cap,
		Permissions: core.Permissions{
			Publish:        core.PermList{Allow: []string{src, "_INBOX.>"}, Deny: []string{"tb.internal.>"}},
			Subscribe:      core.PermList{Allow: []string{src, "_INBOX.>"}, Deny: []string{"tb.internal.>"}},
			AllowResponses: core.AllowResponses{Max: 1, ExpiresMs: 30000},
		},
		Imports:  map[string]core.Import{"source": {Kind: "subscribe", Subjects: []string{src}, Desc: "bundle filter watch"}},
		Exports:  []string{src},
		Exposure: map[string]core.Exposure{ref: {Kind: "kv_watch", Subject: src, Desc: "bundle filter exposure"}},
	}
}

func bundleKVActivation(b *bundle, e bundleScript, cap core.Capability) core.Activation {
	id := b.manifest.Name + "-" + e.Name
	key := "p.bundle." + b.manifest.Name + "." + e.Watches
	return core.Activation{
		ScriptKey:      b.scriptKey(e),
		ScriptRevision: e.ScriptRevision,
		SourcePrincipal: core.SourcePrincipal{
			PrincipalID:  cap.PrincipalID,
			SourceID:     "src-bundle-filter-" + id,
			SourceKind:   "kv",
			AuthorityRef: bundleKVAuthorityRef(b, e),
		},
		SourceLease: core.SourceLease{
			LeaseID:        cap.LeaseID,
			LeaseStatus:    "active",
			AppRevision:    appRevision,
			SchemaVersion:  "v1",
			ScriptRevision: e.ScriptRevision,
		},
		Source:     core.Source{Kind: "kv", ActivationName: "transform", Bucket: bundleMaterialBucket, Key: key},
		Chain:      core.Chain{ChainID: "chain-bundle-filter-" + id, RootID: "root-bundle-filter-" + id, Hop: 1, MaxHops: 5},
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

func bundleKVAuthorityRef(b *bundle, e bundleScript) string {
	return "auth.source.bundle.kv." + b.manifest.Name + "-" + e.Name
}

// serveArtifact serves artifact bodies read-only under sandbox headers:
// bundle frontend content is untrusted generated material, never trusted
// shell code. The opaque sandbox origin is why reads carry a permissive
// CORS header — no cookie travels from a sandboxed page.
func (a *App) serveArtifact(rw http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/artifacts/")
	if name == "" {
		http.NotFound(rw, r)
		return
	}
	// Scoped projection route: a bundle page served under bundle/<bname>/ may
	// fetch its own projection relatively as `_p/<short>`. Resolve
	// bundle/<bname>/_p/<short> to the derived projection id
	// bundle.<bname>.<short> and serve it as JSON like serveProjection — never
	// falling through to an artifact lookup for `_p/` paths.
	if parts := strings.Split(name, "/"); len(parts) == 4 && parts[0] == "bundle" && parts[2] == "_p" && parts[1] != "" && parts[3] != "" {
		if a.bundleMaterials == nil {
			http.NotFound(rw, r)
			return
		}
		proj := "bundle." + parts[1] + "." + parts[3]
		body, ok, err := a.bundleMaterials.LoadProjection(proj)
		if err != nil {
			http.Error(rw, "projection unavailable", http.StatusBadGateway)
			return
		}
		if !ok {
			http.NotFound(rw, r)
			return
		}
		rw.Header().Set("Content-Type", "application/json")
		rw.Header().Set("Cache-Control", "no-cache")
		rw.Header().Set("Access-Control-Allow-Origin", "*")
		_, _ = rw.Write(body)
		return
	}
	store := a.materials
	if a.bundleMaterials != nil && strings.HasPrefix(name, "bundle/") {
		store = a.bundleMaterials
	}
	art, body, ok, err := store.LoadArtifact(name)
	if err != nil {
		http.Error(rw, "artifact unavailable", http.StatusBadGateway)
		return
	}
	if !ok {
		http.NotFound(rw, r)
		return
	}
	ct := art.MediaType
	if ct == "" {
		ct = "application/octet-stream"
	}
	rw.Header().Set("Content-Type", ct)
	rw.Header().Set("Content-Security-Policy", "sandbox allow-scripts")
	// no-cache (not no-store) so the body is cacheable but always revalidated;
	// the digest ETag lets a revalidation short-circuit to 304.
	rw.Header().Set("Cache-Control", "no-cache")
	rw.Header().Set("X-Content-Type-Options", "nosniff")
	rw.Header().Set("Access-Control-Allow-Origin", "*")
	// Content-addressed revalidation: the Object Store already digests every
	// artifact body, so the digest is the strong validator. A revalidating GET
	// that matches short-circuits to 304 with no body.
	if art.Digest != "" {
		etag := "\"" + art.Digest + "\""
		rw.Header().Set("ETag", etag)
		if r.Header.Get("If-None-Match") == etag {
			rw.WriteHeader(http.StatusNotModified)
			return
		}
	}
	_, _ = rw.Write(body)
}

// serveProjection serves the stored projection record read-only as JSON —
// the same truth an observer reads from the material bucket.
func (a *App) serveProjection(rw http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/projections/")
	if id == "" {
		http.NotFound(rw, r)
		return
	}
	store := a.materials
	if a.bundleMaterials != nil && strings.HasPrefix(id, "bundle.") {
		store = a.bundleMaterials
	}
	body, ok, err := store.LoadProjection(id)
	if err != nil {
		http.Error(rw, "projection unavailable", http.StatusBadGateway)
		return
	}
	if !ok {
		http.NotFound(rw, r)
		return
	}
	rw.Header().Set("Content-Type", "application/json")
	rw.Header().Set("Cache-Control", "no-store")
	rw.Header().Set("Access-Control-Allow-Origin", "*")
	_, _ = rw.Write(body)
}
