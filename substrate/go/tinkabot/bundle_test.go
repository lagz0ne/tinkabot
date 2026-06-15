package tinkabot

// TestBundle proves the bundle surface: pointing the binary at a bundle dir
// serves a complete ephemeral app — manifest-declared scripts wired to
// triggers, boot effects materialized through the normal gate, and the
// frontend reachable as artifacts — while a bundle claiming durably-claimed
// authority is a typed load failure and nothing durable is mutated.

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lagz0ne/tinkabot/substrate/go/core"
	"github.com/lagz0ne/tinkabot/substrate/go/embednats"
	"github.com/nats-io/nats.go"
)

const clockBundle = "../../../examples/clock"

func TestBundle(t *testing.T) {
	t.Parallel()

	t.Run("AppServes", func(t *testing.T) {
		t.Parallel()
		cfg := cfgFor(t.TempDir())
		cfg.BundleDir = clockBundle
		app, err := boot(t, cfg)
		if err != nil {
			t.Fatal(err)
		}
		shell := app.Posture().Shell.URL

		hdr, body := waitFor200(t, shell+"/artifacts/bundle/clock/index.html", 15*time.Second)
		if ct := hdr.Get("Content-Type"); !strings.Contains(ct, "text/html") {
			t.Fatalf("artifact media type drift: %q", ct)
		}
		if csp := hdr.Get("Content-Security-Policy"); !strings.Contains(csp, "sandbox") {
			t.Fatalf("artifact served without sandbox policy: %q", csp)
		}
		if !strings.Contains(string(body), "tinkabot clock") {
			t.Fatalf("artifact body drift: %s", body)
		}

		_, pj := waitFor200(t, shell+"/projections/bundle.clock.state", 5*time.Second)
		first := unixOf(t, pj)

		// Nothing durable mutated: the bundle record never lands in the
		// durable script bucket.
		anc, err := nats.Connect(app.Posture().NATS.ClientURL, nats.UserCredentials(app.CredsFile(RoleAuthor)))
		if err != nil {
			t.Fatal(err)
		}
		defer anc.Close()
		js, err := anc.JetStream()
		if err != nil {
			t.Fatal(err)
		}
		kv, err := js.KeyValue("tb_scripts")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := kv.Get("s." + base64.RawURLEncoding.EncodeToString([]byte("scripts.bundle.clock.tick"))); err == nil {
			t.Fatal("bundle record leaked into the durable script bucket")
		}

		// Account isolation: the bundle's entire plane is invisible from
		// TB_APP — its script bucket does not exist there, and its
		// projections never land in the app's material bucket.
		probe, err := app.Runtime().MintUser(embednats.AppAccount, principal("principal.test.probe", "lease-probe-bundle", core.Permissions{
			Publish:   core.PermList{Allow: []string{"$JS.API.>"}},
			Subscribe: core.PermList{Allow: []string{"_INBOX.>"}},
		}), time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		pctx, pcancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer pcancel()
		pnc, err := app.Runtime().ConnectCreds(pctx, probe.File)
		if err != nil {
			t.Fatal(err)
		}
		defer pnc.Close()
		pjs, err := pnc.JetStream()
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pjs.KeyValue("tb_bundle"); !errors.Is(err, nats.ErrBucketNotFound) {
			t.Fatalf("bundle bucket visible in the app account: %v", err)
		}
		appMaterial, err := pjs.KeyValue("tb_material")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := appMaterial.Get("p.bundle.clock.state"); !errors.Is(err, nats.ErrKeyNotFound) {
			t.Fatalf("bundle projection leaked into the app material bucket: %v", err)
		}

		// Caller trigger round trip re-renders the projection.
		cnc, err := nats.Connect(app.Posture().NATS.ClientURL, nats.UserCredentials(app.CredsFile(RoleCaller)))
		if err != nil {
			t.Fatal(err)
		}
		defer cnc.Close()
		time.Sleep(1100 * time.Millisecond) // sequence is unix seconds; let it advance
		msg := nats.NewMsg("tb.bundle.clock.tick")
		msg.Header.Set(embednats.HeaderRequestID, "req-bundle-test-1")
		msg.Data = []byte("tick")
		reply, err := cnc.RequestMsg(msg, 5*time.Second)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(reply.Data), "accepted") {
			t.Fatalf("bundle trigger not accepted: %s", reply.Data)
		}
		deadline := time.Now().Add(10 * time.Second)
		for {
			_, pj := waitFor200(t, shell+"/projections/bundle.clock.state", 5*time.Second)
			if unixOf(t, pj) > first {
				break
			}
			if time.Now().After(deadline) {
				t.Fatal("projection did not advance after trigger")
			}
			time.Sleep(200 * time.Millisecond)
		}
	})

	t.Run("EphemeralAcrossRestart", func(t *testing.T) {
		t.Parallel()
		store := t.TempDir()
		cfg := cfgFor(store)
		cfg.BundleDir = clockBundle
		app1, err := boot(t, cfg)
		if err != nil {
			t.Fatal(err)
		}
		waitFor200(t, app1.Posture().Shell.URL+"/artifacts/bundle/clock/index.html", 15*time.Second)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := app1.Stop(ctx); err != nil {
			t.Fatal(err)
		}

		app2, err := boot(t, cfgFor(store))
		if err != nil {
			t.Fatal(err)
		}
		shell := app2.Posture().Shell.URL
		if code, _, _ := httpGet(t, shell+"/artifacts/bundle/clock/index.html"); code != http.StatusNotFound {
			t.Fatalf("bundle artifact survived a bundle-less restart: %d", code)
		}
		cnc, err := nats.Connect(app2.Posture().NATS.ClientURL, nats.UserCredentials(app2.CredsFile(RoleCaller)))
		if err != nil {
			t.Fatal(err)
		}
		defer cnc.Close()
		dead := nats.NewMsg("tb.bundle.clock.tick")
		dead.Header.Set(embednats.HeaderRequestID, "req-bundle-test-2")
		if _, err := cnc.RequestMsg(dead, time.Second); err == nil {
			t.Fatal("bundle trigger survived a bundle-less restart")
		}
		alive := nats.NewMsg(app2.Posture().Wiring.TriggerSubject)
		alive.Header.Set(embednats.HeaderRequestID, "req-bundle-test-3")
		if _, err := cnc.RequestMsg(alive, 5*time.Second); err != nil {
			t.Fatalf("manual trigger route is gone: %v", err)
		}
	})

	// A scheduled entry drives its own updates: the manifest declares the
	// cadence as intent, every tick is an ordinary attributed activation
	// through the caller path, and runtime control rides NATS settings —
	// the app config bucket the caller can already write.
	t.Run("ScheduledTicks", func(t *testing.T) {
		t.Parallel()
		manifest := `{"kind":"bundle.manifest","name":"t","scripts":[{"name":"run","file":"scripts/run.sh","command":"/bin/sh","projections":["state"],"boot":true,"every":"300ms"}]}`
		script := "#!/bin/sh\nns=$(date +%s%N)\n" +
			`b1="{\"kind\":\"script.effect\",\"effectType\":\"projection\",\"projectionId\":\"bundle.t.state\",\"snapshotRevision\":\"snap-$ns\",\"artifactRevision\":\"r1\",\"sequence\":$ns,\"value\":{\"ns\":$ns}}"` + "\n" +
			`printf 'Content-Length: %s\r\n\r\n%s' "${#b1}" "$b1"` + "\n"
		cfg := cfgFor(t.TempDir())
		cfg.BundleDir = writeBundleScript(t, manifest, script)
		app, err := boot(t, cfg)
		if err != nil {
			t.Fatal(err)
		}
		shell := app.Posture().Shell.URL
		url := shell + "/projections/bundle.t.state"

		_, p1 := waitFor200(t, url, 15*time.Second)
		first := nsOf(t, p1)
		waitAdvance(t, url, first, 10*time.Second) // no manual trigger anywhere

		// Pause through the settings surface with plain caller authority.
		cnc, err := nats.Connect(app.Posture().NATS.ClientURL, nats.UserCredentials(app.CredsFile(RoleCaller)))
		if err != nil {
			t.Fatal(err)
		}
		defer cnc.Close()
		cjs, err := cnc.JetStream()
		if err != nil {
			t.Fatal(err)
		}
		settings, err := cjs.KeyValue("config_bucket")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := settings.Put("bundle.t.run.every", []byte("off")); err != nil {
			t.Fatal(err)
		}
		paused := false
		deadline := time.Now().Add(10 * time.Second)
		for time.Now().Before(deadline) {
			_, pa := waitFor200(t, url, 5*time.Second)
			a := nsOf(t, pa)
			time.Sleep(1200 * time.Millisecond)
			_, pb := waitFor200(t, url, 5*time.Second)
			if nsOf(t, pb) == a {
				paused = true
				break
			}
		}
		if !paused {
			t.Fatal("ticks did not pause on settings off")
		}

		// Resume at a new cadence through the same settings key.
		if _, err := settings.Put("bundle.t.run.every", []byte("200ms")); err != nil {
			t.Fatal(err)
		}
		_, pc := waitFor200(t, url, 5*time.Second)
		waitAdvance(t, url, nsOf(t, pc), 10*time.Second)
	})

	// Bundle processes run jailed (bwrap): the bundle dir is read-only inside,
	// so a script cannot write back into its own directory. Proves isolation
	// without needing the network.
	t.Run("SandboxBlocksHostWrite", func(t *testing.T) {
		t.Parallel()
		manifest := `{"kind":"bundle.manifest","name":"t","scripts":[{"name":"gen","file":"scripts/run.sh","command":"/bin/sh","projections":["probe"],"boot":true}]}`
		script := "#!/bin/sh\n" +
			"if echo x > ./escape-probe 2>/dev/null; then p=WROTE; else p=BLOCKED; fi\n" +
			`b="{\"kind\":\"script.effect\",\"effectType\":\"projection\",\"projectionId\":\"bundle.t.probe\",\"snapshotRevision\":\"s1\",\"artifactRevision\":\"r1\",\"sequence\":1,\"value\":{\"probe\":\"$p\"}}"` + "\n" +
			`printf 'Content-Length: %s\r\n\r\n%s' "${#b}" "$b"` + "\n"
		cfg := cfgFor(t.TempDir())
		cfg.BundleDir = writeBundleScript(t, manifest, script)
		app, err := boot(t, cfg)
		if err != nil {
			t.Fatal(err)
		}
		_, body := waitFor200(t, app.Posture().Shell.URL+"/projections/bundle.t.probe", 15*time.Second)
		if !strings.Contains(string(body), "BLOCKED") {
			t.Fatalf("bundle script wrote to its host bundle dir — not sandboxed: %s", body)
		}
	})

	// Served artifacts carry an ETag (the Object Store sha256 digest) and a
	// matching If-None-Match revalidates to 304 — caching via the digest the
	// store already computes, no content-hashed URLs needed.
	t.Run("ArtifactETag", func(t *testing.T) {
		t.Parallel()
		manifest := `{"kind":"bundle.manifest","name":"t","scripts":[{"name":"gen","file":"scripts/run.sh","command":"/bin/sh","boot":true}]}`
		script := "#!/bin/sh\n" +
			`b="{\"kind\":\"script.effect\",\"effectType\":\"artifact\",\"artifactName\":\"bundle/t/x.txt\",\"artifactRevision\":\"r1\",\"mediaType\":\"text/plain\",\"body\":\"hello\"}"` + "\n" +
			`printf 'Content-Length: %s\r\n\r\n%s' "${#b}" "$b"` + "\n"
		cfg := cfgFor(t.TempDir())
		cfg.BundleDir = writeBundleScript(t, manifest, script)
		app, err := boot(t, cfg)
		if err != nil {
			t.Fatal(err)
		}
		url := app.Posture().Shell.URL + "/artifacts/bundle/t/x.txt"
		hdr, _ := waitFor200(t, url, 15*time.Second)
		etag := hdr.Get("ETag")
		if etag == "" {
			t.Fatal("artifact response carried no ETag")
		}
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("If-None-Match", etag)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusNotModified {
			t.Fatalf("conditional GET got %d, want 304", resp.StatusCode)
		}
	})

	// A script emits a large artifact by writing it to the run's output dir
	// ($TB_ARTIFACT_OUT) and referencing it by path — the body never rides a
	// stdout frame, so the 256 KiB frame ceiling does not bound artifact size.
	t.Run("LargeArtifactByPath", func(t *testing.T) {
		t.Parallel()
		manifest := `{"kind":"bundle.manifest","name":"t","scripts":[{"name":"gen","file":"scripts/run.sh","command":"/bin/sh","boot":true}]}`
		script := "#!/bin/sh\n" +
			"head -c 300000 /dev/zero | tr '\\0' x > \"$TB_ARTIFACT_OUT/big.js\"\n" +
			`b="{\"kind\":\"script.effect\",\"effectType\":\"artifact\",\"artifactName\":\"bundle/t/big.js\",\"artifactRevision\":\"r1\",\"mediaType\":\"application/javascript\",\"path\":\"big.js\"}"` + "\n" +
			`printf 'Content-Length: %s\r\n\r\n%s' "${#b}" "$b"` + "\n"
		cfg := cfgFor(t.TempDir())
		cfg.BundleDir = writeBundleScript(t, manifest, script)
		app, err := boot(t, cfg)
		if err != nil {
			t.Fatal(err)
		}
		hdr, body := waitFor200(t, app.Posture().Shell.URL+"/artifacts/bundle/t/big.js", 15*time.Second)
		if len(body) != 300000 {
			t.Fatalf("served %d bytes, want 300000 (large artifact by path)", len(body))
		}
		if !strings.Contains(hdr.Get("Content-Type"), "javascript") {
			t.Fatalf("media type drift: %q", hdr.Get("Content-Type"))
		}
	})

	// Chain-reaction: an entry with `watches` is a long-lived filter fed each
	// watched-projection change on stdin (one JSON value per line), emitting
	// framed effects through the same materializer gate into its own granted
	// projections. The frontend consumes only the derived view.
	t.Run("TransformPipe", func(t *testing.T) {
		t.Parallel()
		manifest := `{"kind":"bundle.manifest","name":"t","scripts":[` +
			`{"name":"state","file":"scripts/run.sh","command":"/bin/sh","projections":["state"],"boot":true,"every":"300ms"},` +
			`{"name":"present","file":"scripts/present.sh","command":"/bin/sh","watches":"state","projections":["view"]}]}`
		state := "#!/bin/sh\nns=$(date +%s%N)\n" +
			`b1="{\"kind\":\"script.effect\",\"effectType\":\"projection\",\"projectionId\":\"bundle.t.state\",\"snapshotRevision\":\"snap-$ns\",\"artifactRevision\":\"r1\",\"sequence\":$ns,\"value\":{\"ns\":$ns}}"` + "\n" +
			`printf 'Content-Length: %s\r\n\r\n%s' "${#b1}" "$b1"` + "\n"
		present := "#!/bin/sh\n" +
			"while IFS= read -r line; do\n" +
			`  ns=$(printf '%s' "$line" | sed -n 's/.*"ns":\([0-9]*\).*/\1/p')` + "\n" +
			"  [ -z \"$ns\" ] && continue\n" +
			`  v="{\"kind\":\"script.effect\",\"effectType\":\"projection\",\"projectionId\":\"bundle.t.view\",\"snapshotRevision\":\"snap-v-$ns\",\"artifactRevision\":\"r1\",\"sequence\":$ns,\"value\":{\"sourceNs\":$ns,\"doubled\":$((ns*2))}}"` + "\n" +
			`  printf 'Content-Length: %s\r\n\r\n%s' "${#v}" "$v"` + "\n" +
			"done\n"
		dir := writeBundleScript(t, manifest, state)
		if err := os.WriteFile(filepath.Join(dir, "scripts", "present.sh"), []byte(present), 0o755); err != nil {
			t.Fatal(err)
		}
		cfg := cfgFor(t.TempDir())
		cfg.BundleDir = dir
		app, err := boot(t, cfg)
		if err != nil {
			t.Fatal(err)
		}
		shell := app.Posture().Shell.URL

		// The derived view appears and advances with NO trigger anywhere —
		// state ticks feed the filter, the filter feeds the view.
		_, pv := waitFor200(t, shell+"/projections/bundle.t.view", 20*time.Second)
		var view struct {
			Value struct {
				SourceNs int64 `json:"sourceNs"`
				Doubled  int64 `json:"doubled"`
			} `json:"value"`
		}
		if err := json.Unmarshal(pv, &view); err != nil {
			t.Fatalf("view is not the derived record: %v: %s", err, pv)
		}
		if view.Value.Doubled != 2*view.Value.SourceNs {
			t.Fatalf("transform did not transform: %+v", view.Value)
		}
		first := view.Value.SourceNs
		deadline := time.Now().Add(15 * time.Second)
		for {
			_, pv := waitFor200(t, shell+"/projections/bundle.t.view", 5*time.Second)
			var again struct {
				Value struct {
					SourceNs int64 `json:"sourceNs"`
				} `json:"value"`
			}
			if err := json.Unmarshal(pv, &again); err != nil {
				t.Fatal(err)
			}
			if again.Value.SourceNs > first {
				break
			}
			if time.Now().After(deadline) {
				t.Fatal("derived view did not follow the watched projection")
			}
			time.Sleep(200 * time.Millisecond)
		}
	})

	// A manifest cannot spell a reaction loop: the watches graph must be a
	// DAG, watched projections must exist, and a filter is only a filter.
	t.Run("TransformRejected", func(t *testing.T) {
		t.Parallel()
		filter := func(name, watches, projection string) string {
			return fmt.Sprintf(`{"name":%q,"file":"scripts/noop.sh","command":"/bin/sh","watches":%q,"projections":[%q]}`, name, watches, projection)
		}
		cases := []struct {
			name    string
			scripts string
		}{
			{"SelfWatch", filter("a", "out", "out")},
			{"Cycle", filter("a", "bout", "aout") + "," + filter("b", "aout", "bout")},
			{"UnknownWatch", filter("a", "ghost", "aout")},
			{"WatchWithEvery", `{"name":"a","file":"scripts/noop.sh","command":"/bin/sh","watches":"x","projections":["aout"],"every":"1s"}` + "," +
				bundleEntry("b", `"x"`)},
			{"WatchWithBoot", `{"name":"a","file":"scripts/noop.sh","command":"/bin/sh","watches":"x","projections":["aout"],"boot":true}` + "," +
				bundleEntry("b", `"x"`)},
			{"WatchWithoutOutput", `{"name":"a","file":"scripts/noop.sh","command":"/bin/sh","watches":"x"}` + "," +
				bundleEntry("b", `"x"`)},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				t.Parallel()
				cfg := cfgFor(t.TempDir())
				cfg.BundleDir = writeBundle(t, `{"kind":"bundle.manifest","name":"t","scripts":[`+c.scripts+`]}`)
				_, err := boot(t, cfg)
				assertKind(t, err, BundleRejected)
			})
		}
	})

	// Authority is derived, never declared: a manifest cannot even spell a
	// collision with durable claims — free-form naming fields are unknown
	// fields, and the only name a bundle controls must parse as a name.
	t.Run("InvalidNames", func(t *testing.T) {
		t.Parallel()
		cases := []struct {
			name  string
			entry string
		}{
			{"BadEntryName", bundleEntry("T.tick", `"state"`)},
			{"DuplicateEntryName", bundleEntry("t", `"state"`) + "," + bundleEntry("t", `"other"`)},
			{"DuplicateProjection", bundleEntry("t", `"state"`) + "," + bundleEntry("u", `"state"`)},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				t.Parallel()
				cfg := cfgFor(t.TempDir())
				cfg.BundleDir = writeBundle(t, `{"kind":"bundle.manifest","name":"t","scripts":[`+c.entry+`]}`)
				_, err := boot(t, cfg)
				assertKind(t, err, BundleRejected)
			})
		}
	})

	t.Run("MalformedManifest", func(t *testing.T) {
		t.Parallel()
		cases := []struct {
			name     string
			manifest string
		}{
			{"UnknownField", `{"kind":"bundle.manifest","name":"t","zip":true,"scripts":[` +
				bundleEntry("t", `"state"`) + `]}`},
			{"FreeFormTrigger", `{"kind":"bundle.manifest","name":"t","scripts":[` +
				`{"name":"t","file":"scripts/noop.sh","command":"/bin/sh","trigger":"tb.proof.runtime.execute"}]}`},
			{"MissingCommand", `{"kind":"bundle.manifest","name":"t","scripts":[` +
				`{"name":"t","file":"scripts/noop.sh"}]}`},
			{"MalformedEvery", `{"kind":"bundle.manifest","name":"t","scripts":[` +
				`{"name":"t","file":"scripts/noop.sh","command":"/bin/sh","every":"soon"}]}`},
			{"OverEagerEvery", `{"kind":"bundle.manifest","name":"t","scripts":[` +
				`{"name":"t","file":"scripts/noop.sh","command":"/bin/sh","every":"5ms"}]}`},
			{"WrongKind", `{"kind":"script.record","name":"t","scripts":[]}`},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				t.Parallel()
				cfg := cfgFor(t.TempDir())
				cfg.BundleDir = writeBundle(t, c.manifest)
				_, err := boot(t, cfg)
				assertKind(t, err, BundleRejected)
			})
		}
		t.Run("MissingManifest", func(t *testing.T) {
			t.Parallel()
			cfg := cfgFor(t.TempDir())
			cfg.BundleDir = t.TempDir()
			_, err := boot(t, cfg)
			assertKind(t, err, BundleRejected)
		})
	})
}

func bundleEntry(name, projections string) string {
	return fmt.Sprintf(`{"name":%q,"file":"scripts/noop.sh","command":"/bin/sh","projections":[%s]}`,
		name, projections)
}

func writeBundle(t *testing.T, manifest string) string {
	t.Helper()
	return writeBundleScript(t, manifest, "#!/bin/sh\n")
}

func writeBundleScript(t *testing.T, manifest, script string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"noop.sh", "run.sh"} {
		if err := os.WriteFile(filepath.Join(dir, "scripts", name), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "bundle.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func nsOf(t *testing.T, projection []byte) int64 {
	t.Helper()
	var p struct {
		Value struct {
			NS int64 `json:"ns"`
		} `json:"value"`
	}
	if err := json.Unmarshal(projection, &p); err != nil {
		t.Fatalf("projection is not the stored record: %v: %s", err, projection)
	}
	return p.Value.NS
}

func waitAdvance(t *testing.T, url string, past int64, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		_, p := waitFor200(t, url, timeout)
		if nsOf(t, p) > past {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("projection did not advance past %d within %s", past, timeout)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func httpGet(t *testing.T, url string) (int, http.Header, []byte) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, resp.Header, body
}

func waitFor200(t *testing.T, url string, timeout time.Duration) (http.Header, []byte) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		code, hdr, body := httpGet(t, url)
		if code == http.StatusOK {
			return hdr, body
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s stayed %d past %s", url, code, timeout)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func unixOf(t *testing.T, projection []byte) int64 {
	t.Helper()
	var p struct {
		Value struct {
			Unix int64 `json:"unix"`
		} `json:"value"`
	}
	if err := json.Unmarshal(projection, &p); err != nil {
		t.Fatalf("projection is not the stored record: %v: %s", err, projection)
	}
	return p.Value.Unix
}

// TestBundleSandboxFailClosed: when the host cannot sandbox (bwrap missing),
// a bundle refuses to start rather than running unjailed. TB_BWRAP overrides
// the bwrap binary path; pointing it at a nonexistent file forces the
// preflight to fail.
// gate:serial — sets TB_BWRAP via t.Setenv, which forbids t.Parallel.
func TestBundleSandboxFailClosed(t *testing.T) {
	t.Setenv("TB_BWRAP", "/nonexistent/bwrap")
	manifest := `{"kind":"bundle.manifest","name":"t","scripts":[{"name":"gen","file":"scripts/run.sh","command":"/bin/sh","boot":true}]}`
	cfg := cfgFor(t.TempDir())
	cfg.BundleDir = writeBundleScript(t, manifest, "#!/bin/sh\n")
	_, err := boot(t, cfg)
	assertKind(t, err, BundleRejected)
}
