// Assembly-surface tests over the real embedded runtime: each test boots a
// fresh binary on its own store dir and auto-assigned loopback ports.
package tinkabot

import (
	"bytes"
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/lagz0ne/tinkabot/substrate/go/embednats"
)

// boot is the one harness factory seam for this package (gate:parallel):
// every test obtains its binary assembly here, never via Start directly.
// Each call builds a fresh app on its own store dir and auto-assigned
// loopback ports, so parallel tests stay isolated. Shutdown is owned by the
// test via t.Cleanup; Stop must be idempotent.
func boot(t *testing.T, cfg Config) (*App, error) {
	t.Helper()
	app, err := Start(cfg)
	if app != nil {
		t.Cleanup(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = app.Stop(ctx)
		})
	}
	return app, err
}

func cfgFor(dir string) Config {
	return Config{
		StoreDir:  dir,
		Exposure:  embednats.Loopback(),
		ShellAddr: "127.0.0.1:0",
	}
}

func assertKind(t *testing.T, err error, kind Kind) {
	t.Helper()
	var e *Error
	if !errors.As(err, &e) {
		t.Fatalf("error is not a typed binary failure: %v", err)
	}
	if e.Kind != kind {
		t.Fatalf("failure family drift: got %s want %s (%v)", e.Kind, kind, err)
	}
}

// TestBinaryFirstStartMaterializes proves an empty store directory grows the
// substrate-held operator key and the manual roles' creds files at first
// start, and the posture reports the materialized state.
func TestBinaryFirstStartMaterializes(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	app, err := boot(t, cfgFor(dir))
	if err != nil {
		t.Fatal(err)
	}

	key := filepath.Join(dir, "operator.nk")
	if _, err := os.Stat(key); err != nil {
		t.Fatalf("first start did not materialize the operator key: %v", err)
	}
	p := app.Posture()
	if !p.NATS.Operator.Enabled || p.NATS.Operator.KeyFile != key || p.NATS.Operator.PublicKey == "" {
		t.Fatalf("operator posture drift: %#v", p.NATS.Operator)
	}
	for _, role := range []string{RoleCaller, RoleObserver, RoleAuthor} {
		file := app.CredsFile(role)
		if _, err := os.Stat(file); err != nil {
			t.Fatalf("first start did not materialize %s creds: %v", role, err)
		}
		uc := app.Creds(role)
		if uc.UserPub == "" || uc.Lease.LeaseStatus != "active" || uc.Lease.LeaseID == "" {
			t.Fatalf("%s creds lost lease provenance: %#v", role, uc)
		}
	}
	if p.Shell.URL == "" {
		t.Fatalf("shell posture missing serve address: %#v", p.Shell)
	}
}

// TestBinaryRestartReloadsWithoutRegeneration proves a second start over
// existing state reloads the persistent operator authority byte-identical
// instead of regenerating it, and freshly minted role creds still connect.
func TestBinaryRestartReloadsWithoutRegeneration(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	first, err := boot(t, cfgFor(dir))
	if err != nil {
		t.Fatal(err)
	}
	key := filepath.Join(dir, "operator.nk")
	seed, err := os.ReadFile(key)
	if err != nil {
		t.Fatal(err)
	}
	pub := first.Posture().NATS.Operator.PublicKey
	oldCaller := first.Creds(RoleCaller).File
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := first.Stop(ctx); err != nil {
		t.Fatal(err)
	}

	again, err := boot(t, cfgFor(dir))
	if err != nil {
		t.Fatal(err)
	}
	reloaded, err := os.ReadFile(key)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(seed, reloaded) {
		t.Fatal("restart regenerated the operator key")
	}
	if got := again.Posture().NATS.Operator.PublicKey; got != pub {
		t.Fatalf("operator identity drift across restart: %s != %s", got, pub)
	}
	if nc, err := again.Runtime().ConnectCreds(ctx, oldCaller); err == nil {
		nc.Close()
		t.Fatal("pre-stop caller creds reconnected after restart")
	}
	nc, err := again.Runtime().ConnectCreds(ctx, again.Creds(RoleCaller).File)
	if err != nil {
		t.Fatalf("reloaded binary rejected its own minted caller creds: %v", err)
	}
	nc.Close()
}

var assetRef = regexp.MustCompile(`/(assets/[^"]+)`)

// TestBinaryServesEmbeddedShell proves the embedded frontend shell is served
// over the binary's loopback HTTP surface under the proven scope and policy
// headers (edge.CheckServiceWorkerSetup vocabulary: a narrow session scope,
// Service-Worker-Allowed bound to it, no-store caching, worker revision).
func TestBinaryServesEmbeddedShell(t *testing.T) {
	t.Parallel()
	app, err := boot(t, cfgFor(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	shell := app.Posture().Shell

	res, err := http.Get(shell.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("shell index status drift: %d", res.StatusCode)
	}
	if !strings.HasPrefix(shell.Scope, "/__tinkabot_session/") {
		t.Fatalf("shell scope is not the proven narrow session scope: %q", shell.Scope)
	}
	if got := res.Header.Get("Service-Worker-Allowed"); got != shell.Scope {
		t.Fatalf("Service-Worker-Allowed drift: %q want %q", got, shell.Scope)
	}
	if got := res.Header.Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control drift: %q", got)
	}
	if res.Header.Get("X-Tinkabot-Worker-Rev") == "" {
		t.Fatal("shell served without a worker revision header")
	}

	var body bytes.Buffer
	if _, err := body.ReadFrom(res.Body); err != nil {
		t.Fatal(err)
	}
	html := body.String()
	if !strings.Contains(html, `<div id="app">`) || !strings.Contains(html, "/assets/") {
		t.Fatalf("served shell is not the embedded index: %s", html)
	}
	for _, m := range assetRef.FindAllStringSubmatch(html, -1) {
		asset, err := http.Get(shell.URL + "/" + m[1])
		if err != nil {
			t.Fatal(err)
		}
		asset.Body.Close()
		if asset.StatusCode != http.StatusOK {
			t.Fatalf("embedded asset %q not served: %d", m[1], asset.StatusCode)
		}
	}
}

// TestBinaryPostureMatchesServedSurface proves the declared exposure posture
// is the surface the binary actually serves: loopback declaration yields
// loopback NATS and shell addresses, and the wiring posture names every
// NATS-visible surface the manual operates.
func TestBinaryPostureMatchesServedSurface(t *testing.T) {
	t.Parallel()
	app, err := boot(t, cfgFor(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	p := app.Posture()
	if p.NATS.Exposure.Mode != embednats.ExposeLoopback {
		t.Fatalf("declared loopback, served %q", p.NATS.Exposure.Mode)
	}
	if !strings.Contains(p.NATS.ClientURL, "127.0.0.1") {
		t.Fatalf("loopback posture served a non-loopback client URL: %q", p.NATS.ClientURL)
	}
	host, _, err := net.SplitHostPort(strings.TrimPrefix(p.Shell.URL, "http://"))
	if err != nil || host != "127.0.0.1" {
		t.Fatalf("loopback posture served a non-loopback shell: %q (%v)", p.Shell.URL, err)
	}
	w := p.Wiring
	for name, v := range map[string]string{
		"trigger subject": w.TriggerSubject,
		"events subject":  w.EventsSubject,
		"config bucket":   w.ConfigBucket,
		"upload bucket":   w.UploadBucket,
		"script bucket":   w.ScriptBucket,
		"ledger bucket":   w.LedgerBucket,
		"material bucket": w.MaterialBucket,
		"artifact bucket": w.ArtifactBucket,
		"script key":      w.ScriptKey,
	} {
		if v == "" {
			t.Fatalf("wiring posture missing %s: %#v", name, w)
		}
	}
	if w.ScriptRevision == 0 {
		t.Fatalf("wiring posture missing script revision: %#v", w)
	}
}

// TestBinaryDrainShutdown proves Stop drains the runtime cleanly: live
// connections close, the shell surface stops serving, and a second Stop is
// an idempotent no-op rather than a typed failure.
func TestBinaryDrainShutdown(t *testing.T) {
	t.Parallel()
	app, err := boot(t, cfgFor(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	nc, err := app.Runtime().ConnectCreds(ctx, app.Creds(RoleCaller).File)
	if err != nil {
		t.Fatal(err)
	}
	defer nc.Close()
	shell := app.Posture().Shell.URL

	if err := app.Stop(ctx); err != nil {
		t.Fatalf("clean shutdown failed: %v", err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for !nc.IsClosed() {
		if time.Now().After(deadline) {
			t.Fatal("drain left the caller connection open")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if res, err := http.Get(shell); err == nil {
		res.Body.Close()
		t.Fatal("shell still serving after shutdown")
	}
	if err := app.Stop(ctx); err != nil {
		t.Fatalf("second Stop is not idempotent: %v", err)
	}
}

// TestBinaryFailureFamiliesTyped forces each of the five owned failure
// families and asserts the typed kind, never a bare error.
func TestBinaryFailureFamiliesTyped(t *testing.T) {
	t.Parallel()

	t.Run("startup materialization", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "not-a-dir")
		if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := boot(t, cfgFor(file))
		assertKind(t, err, StartupMaterializationFailed)
	})

	t.Run("frontend serve", func(t *testing.T) {
		taken, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		defer taken.Close()
		cfg := cfgFor(t.TempDir())
		cfg.ShellAddr = taken.Addr().String()
		_, err = boot(t, cfg)
		assertKind(t, err, FrontendServeFailed)
	})

	t.Run("wiring mismatch", func(t *testing.T) {
		cfg := cfgFor(t.TempDir())
		cfg.ShellAddr = "203.0.113.7:0" // beyond the declared loopback posture
		_, err := boot(t, cfg)
		assertKind(t, err, WiringMismatch)
	})

	t.Run("manual divergence", func(t *testing.T) {
		app, err := boot(t, cfgFor(t.TempDir()))
		if err != nil {
			t.Fatal(err)
		}
		err = CheckManual([]byte("# Some Other Doc\n\nNo starting-the-binary section here.\n"), app.Posture())
		assertKind(t, err, ManualDivergence)
	})

	t.Run("shutdown", func(t *testing.T) {
		app, err := boot(t, cfgFor(t.TempDir()))
		if err != nil {
			t.Fatal(err)
		}
		expired, cancel := context.WithCancel(context.Background())
		cancel()
		assertKind(t, app.Stop(expired), ShutdownFailed)
	})
}

// TestBinaryManualStartingSection proves docs/manual/v1.md carries the new
// "starting the binary" section and that it names the live wired surface —
// the manual-divergence family's happy path against the real document.
func TestBinaryManualStartingSection(t *testing.T) {
	t.Parallel()
	app, err := boot(t, cfgFor(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	doc, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "manual", "v1.md"))
	if err != nil {
		t.Fatal(err)
	}
	if err := CheckManual(doc, app.Posture()); err != nil {
		t.Fatalf("manual diverges from the served binary surface: %v", err)
	}
}
