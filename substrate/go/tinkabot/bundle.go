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
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
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
}

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
		}
		if !filepath.IsLocal(e.File) {
			return nil, rejectBundle("bundle script file escapes the bundle dir", at, nil)
		}
		if _, err := os.Stat(filepath.Join(abs, e.File)); err != nil {
			return nil, rejectBundle("bundle script file is missing", at, err)
		}
		seenNames[e.Name] = true
	}
	return b, nil
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
func (a *App) startBundle(b *bundle, deps bundleDeps) error {
	acct := b.account()
	if err := a.rt.MintAccount(acct); err != nil {
		return rejectBundle("bundle account could not be minted", map[string]string{"account": acct}, err)
	}
	svcUC, err := a.rt.MintUser(acct, principal("principal.bundle."+b.manifest.Name+".service", "lease-bundle-svc-"+deps.nonce, bundleServicePerms()), time.Hour)
	if err != nil {
		return rejectBundle("bundle service creds could not be minted", map[string]string{"account": acct}, err)
	}
	routerUC, err := a.rt.MintUser(acct, principal("principal.bundle."+b.manifest.Name+".router", "lease-bundle-router-"+deps.nonce, bundleRouterPerms(b)), time.Hour)
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

		router, err := embednats.NewSourceRouter(bundleSourceAuthority(b, e, deps.cap), ledger)
		if err != nil {
			return rejectBundle("bundle trigger authority was denied", map[string]string{"trigger": b.trigger(e)}, err)
		}
		route, results, err := router.RequestReply(routerNC, bundleActivation(b, e, deps.cap))
		if err != nil {
			return rejectBundle("bundle trigger route could not be wired", map[string]string{"trigger": b.trigger(e)}, err)
		}
		a.routes = append(a.routes, route)
		rtm, err := core.NewScriptRuntime(core.ScriptPolicy{AllowedProjections: b.projections(e), ArtifactPrefix: b.artifactPrefix()}, embednats.LocalScriptRunner{})
		if err != nil {
			return rejectBundle("bundle script policy was denied", map[string]string{"script": e.Name}, err)
		}
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
	}

	return a.bootBundle(b, deps)
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
func bundleRouterPerms(b *bundle) core.Permissions {
	pub := append([]string{"$JS.API.INFO", "_INBOX.>", "$KV." + bundleLedgerBucket + ".>", "$JS.API.STREAM.CREATE.KV_" + bundleLedgerBucket}, readKV(bundleLedgerBucket)...)
	subs := []string{"_INBOX.>"}
	for _, e := range b.manifest.Scripts {
		subs = append(subs, b.trigger(e))
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
func (a *App) bootBundle(b *bundle, deps bundleDeps) error {
	var boots []bundleScript
	for _, e := range b.manifest.Scripts {
		if e.Boot {
			boots = append(boots, e)
		}
	}
	if len(boots) == 0 {
		return nil
	}
	nc, err := deps.dial(deps.caller, "bundle boot")
	if err != nil {
		return err
	}
	for _, e := range boots {
		// The import's claims push propagates asynchronously; retry inside a
		// bounded window before declaring the boot dead.
		deadline := time.Now().Add(10 * time.Second)
		for {
			msg := nats.NewMsg(b.trigger(e))
			msg.Header.Set(embednats.HeaderRequestID, "boot-"+deps.nonce)
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
	rw.Header().Set("Cache-Control", "no-store")
	rw.Header().Set("X-Content-Type-Options", "nosniff")
	rw.Header().Set("Access-Control-Allow-Origin", "*")
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
