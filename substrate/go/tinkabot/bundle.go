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

const bundleBucket = "tb_bundle"

// reservedSubjects are subject prefixes owned by other subsystems; bundle
// triggers may never live under them.
var reservedSubjects = []string{"tb.internal.", "tb.session.", "tb.app.", "$"}

var bundleName = regexp.MustCompile(`^[a-z0-9-]+$`)

type bundleManifest struct {
	Kind    string         `json:"kind"`
	Name    string         `json:"name"`
	Scripts []bundleScript `json:"scripts"`
}

type bundleScript struct {
	ScriptKey      string   `json:"scriptKey"`
	ScriptRevision int      `json:"scriptRevision"`
	Desc           string   `json:"desc,omitempty"`
	File           string   `json:"file"`
	Command        string   `json:"command"`
	TimeoutMs      int      `json:"timeoutMs,omitempty"`
	Trigger        string   `json:"trigger"`
	Projections    []string `json:"projections,omitempty"`
	ArtifactPrefix string   `json:"artifactPrefix,omitempty"`
	Boot           bool     `json:"boot,omitempty"`
}

type bundle struct {
	dir      string
	manifest bundleManifest
}

func rejectBundle(msg string, details map[string]string, cause error) *Error {
	return fail(BundleRejected, "LoadBundle", msg, details, cause)
}

// loadBundle reads and validates the manifest against the durable authority
// claims. Pure file work — it runs before any NATS state exists, so a bad
// bundle fails the start before it costs anything.
func loadBundle(dir string, w Wiring) (*bundle, error) {
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
	seenKeys, seenTriggers, seenProjections := map[string]bool{}, map[string]bool{}, map[string]bool{}
	var prefixes []string
	for i := range m.Scripts {
		e := &m.Scripts[i]
		if e.TimeoutMs == 0 {
			e.TimeoutMs = 2000
		}
		at := map[string]string{"scriptKey": e.ScriptKey}
		switch {
		case e.ScriptKey == "" || e.ScriptRevision <= 0 || e.File == "" || e.Command == "" || e.Trigger == "":
			return nil, rejectBundle("bundle script entry is incomplete", at, nil)
		case e.ScriptKey == w.ScriptKey:
			return nil, rejectBundle("bundle script key collides with durable authority", at, nil)
		case e.Trigger == w.TriggerSubject || e.Trigger == w.EventsSubject:
			return nil, rejectBundle("bundle trigger collides with durable authority", at, nil)
		case e.ArtifactPrefix == "" || prefixOverlap(e.ArtifactPrefix, "artifact/"):
			return nil, rejectBundle("bundle artifact prefix collides with durable authority", at, nil)
		case seenKeys[e.ScriptKey]:
			return nil, rejectBundle("bundle script key is duplicated", at, nil)
		case seenTriggers[e.Trigger]:
			return nil, rejectBundle("bundle trigger is duplicated", at, nil)
		}
		for _, res := range reservedSubjects {
			if strings.HasPrefix(e.Trigger, res) {
				return nil, rejectBundle("bundle trigger is under a reserved subject", at, nil)
			}
		}
		for _, p := range e.Projections {
			if p == "main" {
				return nil, rejectBundle("bundle projection collides with durable authority", at, nil)
			}
			if seenProjections[p] {
				return nil, rejectBundle("bundle projection is duplicated", at, nil)
			}
			seenProjections[p] = true
		}
		for _, prev := range prefixes {
			if prefixOverlap(prev, e.ArtifactPrefix) {
				return nil, rejectBundle("bundle artifact prefixes overlap", at, nil)
			}
		}
		if !filepath.IsLocal(e.File) {
			return nil, rejectBundle("bundle script file escapes the bundle dir", at, nil)
		}
		if _, err := os.Stat(filepath.Join(abs, e.File)); err != nil {
			return nil, rejectBundle("bundle script file is missing", at, err)
		}
		seenKeys[e.ScriptKey], seenTriggers[e.Trigger] = true, true
		prefixes = append(prefixes, e.ArtifactPrefix)
	}
	return b, nil
}

func prefixOverlap(a, b string) bool {
	return strings.HasPrefix(a, b) || strings.HasPrefix(b, a)
}

func (b *bundle) triggers() []string {
	subs := make([]string, len(b.manifest.Scripts))
	for i, e := range b.manifest.Scripts {
		subs[i] = e.Trigger
	}
	return subs
}

// bundleDeps carries the Start-time seams the bundle wiring consumes; every
// behavior comes from the same proven pieces the manual slot uses.
type bundleDeps struct {
	cap       core.Capability
	nonce     string
	dial      func(embednats.UserCreds, string) (*nats.Conn, error)
	svc       embednats.UserCreds
	caller    embednats.UserCreds
	scripts   *embednats.KVScriptStore
	ledger    core.LedgerStore
	materials *embednats.KVMaterialStore
	mat       *core.Materializer
	routerNC  *nats.Conn
}

// startBundle lands the bundle's records in the ephemeral bucket, wires one
// trigger route and script loop per entry under that entry's grants, and
// fires the boot entries through the normal request/reply activation path.
func (a *App) startBundle(b *bundle, deps bundleDeps) error {
	for _, e := range b.manifest.Scripts {
		if _, ok, err := deps.scripts.LoadScript(e.ScriptKey); err != nil {
			return rejectBundle("durable script bucket could not be checked", map[string]string{"scriptKey": e.ScriptKey}, err)
		} else if ok {
			return rejectBundle("bundle script key collides with a durable record", map[string]string{"scriptKey": e.ScriptKey}, nil)
		}
	}

	bnc, err := deps.dial(deps.svc, "bundle store")
	if err != nil {
		return err
	}
	js, err := bnc.JetStream()
	if err != nil {
		return rejectBundle("bundle store JetStream context is unavailable", nil, err)
	}
	// Memory storage is the ephemerality mechanism: the bucket and every
	// record in it die with the process; nothing durable is mutated.
	if _, err := js.CreateKeyValue(&nats.KeyValueConfig{Bucket: bundleBucket, Storage: nats.MemoryStorage}); err != nil {
		return rejectBundle("ephemeral bundle bucket could not be created", map[string]string{"bucket": bundleBucket}, err)
	}
	store, err := embednats.OpenKVScriptStore(bnc, bundleBucket)
	if err != nil {
		return rejectBundle("ephemeral bundle bucket could not be opened", map[string]string{"bucket": bundleBucket}, err)
	}

	ledger := core.NewDurableLedger(deps.ledger)
	for _, e := range b.manifest.Scripts {
		rec := core.ScriptRecord{
			Kind:     "script.record",
			Key:      e.ScriptKey,
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
			return rejectBundle("bundle record could not be landed", map[string]string{"scriptKey": e.ScriptKey}, err)
		}

		router, err := embednats.NewSourceRouter(bundleSourceAuthority(e, deps.cap), ledger)
		if err != nil {
			return rejectBundle("bundle trigger authority was denied", map[string]string{"trigger": e.Trigger}, err)
		}
		route, results, err := router.RequestReply(deps.routerNC, bundleActivation(e, deps.cap))
		if err != nil {
			return rejectBundle("bundle trigger route could not be wired", map[string]string{"trigger": e.Trigger}, err)
		}
		a.routes = append(a.routes, route)
		rtm, err := core.NewScriptRuntime(core.ScriptPolicy{AllowedProjections: e.Projections, ArtifactPrefix: e.ArtifactPrefix}, embednats.LocalScriptRunner{})
		if err != nil {
			return rejectBundle("bundle script policy was denied", map[string]string{"scriptKey": e.ScriptKey}, err)
		}
		runs, stop := embednats.NewScriptLoop(store, rtm, deps.mat, deps.materials, deps.materials).Watch(results)
		a.stopLoops = append(a.stopLoops, stop)
		go func() {
			for range runs {
				// Run outcomes are durable event envelopes, same as the
				// manual slot; nothing to do here.
			}
		}()
	}

	return a.bootBundle(b, deps)
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
		msg := nats.NewMsg(e.Trigger)
		msg.Header.Set(embednats.HeaderRequestID, "boot-"+deps.nonce)
		msg.Data = []byte("boot")
		reply, err := nc.RequestMsg(msg, 10*time.Second)
		if err != nil {
			return rejectBundle("bundle boot trigger got no reply", map[string]string{"trigger": e.Trigger}, err)
		}
		body := string(reply.Data)
		if !strings.Contains(body, "accepted") && !strings.Contains(body, "duplicate") {
			return rejectBundle("bundle boot trigger was denied", map[string]string{"trigger": e.Trigger, "reply": body}, nil)
		}
	}
	return nil
}

func bundleSourceAuthority(e bundleScript, cap core.Capability) core.Auth {
	src := e.Trigger
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
		Exposure: map[string]core.Exposure{bundleAuthorityRef(e): {Kind: "request_reply", Subject: src, Desc: "bundle trigger exposure"}},
	}
}

func bundleActivation(e bundleScript, cap core.Capability) core.Activation {
	id := strings.ReplaceAll(e.ScriptKey, ".", "-")
	return core.Activation{
		ScriptKey:      e.ScriptKey,
		ScriptRevision: e.ScriptRevision,
		SourcePrincipal: core.SourcePrincipal{
			PrincipalID:  cap.PrincipalID,
			SourceID:     "src-" + id,
			SourceKind:   "request_reply",
			AuthorityRef: bundleAuthorityRef(e),
		},
		SourceLease: core.SourceLease{
			LeaseID:        cap.LeaseID,
			LeaseStatus:    "active",
			AppRevision:    appRevision,
			SchemaVersion:  "v1",
			ScriptRevision: e.ScriptRevision,
		},
		Source:     core.Source{Kind: "request_reply", ActivationName: "bundle", Subject: e.Trigger},
		Chain:      core.Chain{ChainID: "chain-" + id, RootID: "root-" + id, Hop: 1, MaxHops: 5},
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

func bundleAuthorityRef(e bundleScript) string {
	return "auth.source.bundle." + strings.ReplaceAll(e.ScriptKey, ".", "-")
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
	art, body, ok, err := a.materials.LoadArtifact(name)
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
	body, ok, err := a.materials.LoadProjection(id)
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
