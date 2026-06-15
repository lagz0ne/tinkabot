package tinkabot

// Adversarial security contracts for the bundle sandbox + effect handling,
// from the substrate-fit review. Each pins a hole the review found.

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// A symlink written into the run's output dir must not let a path-artifact
// exfiltrate a host file: the materializer reads outDir host-side (where the
// whole FS is reachable), so it must not follow a symlink out of outDir.
func TestBundleNoSymlinkArtifactLeak(t *testing.T) {
	t.Parallel()
	secret := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(secret, []byte("TOP-SECRET-XYZ"), 0o600); err != nil {
		t.Fatal(err)
	}
	manifest := `{"kind":"bundle.manifest","name":"t","scripts":[{"name":"gen","file":"scripts/run.sh","command":"/bin/sh","boot":true}]}`
	script := "#!/bin/sh\n" +
		"ln -s " + secret + " \"$TB_ARTIFACT_OUT/leak.html\"\n" +
		`b="{\"kind\":\"script.effect\",\"effectType\":\"artifact\",\"artifactName\":\"leak.html\",\"artifactRevision\":\"r1\",\"mediaType\":\"text/html\",\"path\":\"leak.html\"}"` + "\n" +
		`printf 'Content-Length: %s\r\n\r\n%s' "${#b}" "$b"` + "\n"
	cfg := cfgFor(t.TempDir())
	cfg.BundleDir = writeBundleScript(t, manifest, script)
	app, err := boot(t, cfg)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Second) // let boot's path-artifact attempt resolve
	code, _, body := httpGet(t, app.Posture().Shell.URL+"/artifacts/bundle/t/leak.html")
	if code == http.StatusOK && strings.Contains(string(body), "TOP-SECRET") {
		t.Fatalf("symlink path-artifact leaked a host secret (code %d): %s", code, body)
	}
}

// A jailed bundle process must not be able to read the operator key / role
// creds in the store dir (the substrate's crown jewels).
func TestBundleSandboxHidesStoreSecrets(t *testing.T) {
	t.Parallel()
	// Deliberately NOT t.TempDir(): that lives under /tmp, which the jail's
	// `--tmpfs /tmp` masks by accident. A real store (here under the package
	// dir, in $HOME) is reachable via `--ro-bind / /` — the actual hole.
	wd, _ := os.Getwd()
	store, err := os.MkdirTemp(wd, "tb-sec-store-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(store) })
	opKey := filepath.Join(store, "operator.nk")
	manifest := `{"kind":"bundle.manifest","name":"t","scripts":[{"name":"gen","file":"scripts/run.sh","command":"/bin/sh","projections":["op"],"boot":true}]}`
	script := "#!/bin/sh\n" +
		"if cat " + opKey + " >/dev/null 2>&1; then r=READ; else r=BLOCKED; fi\n" +
		`b="{\"kind\":\"script.effect\",\"effectType\":\"projection\",\"projectionId\":\"op\",\"snapshotRevision\":\"s1\",\"artifactRevision\":\"r1\",\"sequence\":1,\"value\":{\"op\":\"$r\"}}"` + "\n" +
		`printf 'Content-Length: %s\r\n\r\n%s' "${#b}" "$b"` + "\n"
	cfg := cfgFor(store)
	cfg.BundleDir = writeBundleScript(t, manifest, script)
	app, err := boot(t, cfg)
	if err != nil {
		t.Fatal(err)
	}
	_, body := waitFor200(t, app.Posture().Shell.URL+"/projections/bundle.t.op", 15*time.Second)
	if !strings.Contains(string(body), "BLOCKED") {
		t.Fatalf("jailed bundle read the operator key from the store dir: %s", body)
	}
}

// install-at-load must not run package lifecycle scripts (arbitrary host code
// before the jail). A postinstall hook must not fire.
func TestBundleInstallIgnoresScripts(t *testing.T) {
	t.Parallel()
	marker := filepath.Join(t.TempDir(), "pwned-marker")
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "scripts", "run.sh"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bundle.json"),
		[]byte(`{"kind":"bundle.manifest","name":"t","scripts":[{"name":"gen","file":"scripts/run.sh","command":"/bin/sh","boot":true}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"),
		[]byte(`{"name":"x","private":true,"scripts":{"postinstall":"touch `+marker+`"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := cfgFor(t.TempDir())
	cfg.BundleDir = dir
	// install runs synchronously during Start, before the jail and before boot.
	if _, err := boot(t, cfg); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("install-at-load ran a package postinstall script (arbitrary host code)")
	}
}

// ETag must revalidate both ways: a matching If-None-Match -> 304, a
// non-matching one -> 200 with the body (not a blanket 304).
func TestBundleArtifactETagRevalidates(t *testing.T) {
	t.Parallel()
	manifest := `{"kind":"bundle.manifest","name":"t","scripts":[{"name":"gen","file":"scripts/run.sh","command":"/bin/sh","boot":true}]}`
	script := "#!/bin/sh\n" +
		`b="{\"kind\":\"script.effect\",\"effectType\":\"artifact\",\"artifactName\":\"x.txt\",\"artifactRevision\":\"r1\",\"mediaType\":\"text/plain\",\"body\":\"hello\"}"` + "\n" +
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
		t.Fatal("no ETag")
	}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("If-None-Match", `"sha256:deadbeef-not-the-real-digest"`)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("non-matching If-None-Match got %d, want 200 with body", resp.StatusCode)
	}
}

// A jailed RUNTIME bundle process must not read the operator's $HOME secrets
// (~/.ssh etc.) and surface them. The toolchain may live under $HOME (devbox),
// so the jail masks $HOME but re-exposes only the $PATH dirs under it.
func TestBundleSandboxHidesHomeSecrets(t *testing.T) {
	t.Parallel()
	home := os.Getenv("HOME")
	if home == "" {
		t.Skip("no HOME")
	}
	secdir, err := os.MkdirTemp(home, "tb-home-sec-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(secdir) })
	secret := filepath.Join(secdir, "id_rsa")
	if err := os.WriteFile(secret, []byte("HOME-SSH-KEY-LEAKED"), 0o600); err != nil {
		t.Fatal(err)
	}
	manifest := `{"kind":"bundle.manifest","name":"t","scripts":[{"name":"gen","file":"scripts/run.sh","command":"/bin/sh","projections":["leak"],"boot":true}]}`
	script := "#!/bin/sh\n" +
		"v=$(cat " + secret + " 2>/dev/null || echo SAFE)\n" +
		`b="{\"kind\":\"script.effect\",\"effectType\":\"projection\",\"projectionId\":\"leak\",\"snapshotRevision\":\"s1\",\"artifactRevision\":\"r1\",\"sequence\":1,\"value\":{\"data\":\"$v\"}}"` + "\n" +
		`printf 'Content-Length: %s\r\n\r\n%s' "${#b}" "$b"` + "\n"
	cfg := cfgFor(t.TempDir())
	cfg.BundleDir = writeBundleScript(t, manifest, script)
	app, err := boot(t, cfg)
	if err != nil {
		t.Fatal(err)
	}
	_, body := waitFor200(t, app.Posture().Shell.URL+"/projections/bundle.t.leak", 15*time.Second)
	if strings.Contains(string(body), "HOME-SSH-KEY-LEAKED") {
		t.Fatalf("jailed bundle exfiltrated a $HOME secret: %s", body)
	}
}

// The trusted (unsandboxed) tier is an explicit opt-in for hosts without
// user namespaces: with it selected, a bundle runs even when bwrap is
// unavailable — whereas the default stays fail-closed (TestBundleSandboxFailClosed).
// gate:serial — sets TB_BWRAP via t.Setenv, which forbids t.Parallel.
func TestBundleTrustedSandboxTier(t *testing.T) {
	t.Setenv("TB_BWRAP", "/nonexistent/bwrap") // bwrap unavailable
	manifest := `{"kind":"bundle.manifest","name":"t","scripts":[{"name":"gen","file":"scripts/run.sh","command":"/bin/sh","projections":["ok"],"boot":true}]}`
	script := "#!/bin/sh\n" +
		`b="{\"kind\":\"script.effect\",\"effectType\":\"projection\",\"projectionId\":\"ok\",\"snapshotRevision\":\"s1\",\"artifactRevision\":\"r1\",\"sequence\":1,\"value\":{\"ran\":1}}"` + "\n" +
		`printf 'Content-Length: %s\r\n\r\n%s' "${#b}" "$b"` + "\n"
	cfg := cfgFor(t.TempDir())
	cfg.BundleDir = writeBundleScript(t, manifest, script)
	cfg.BundleSandbox = "none" // explicit opt-in to the trusted tier
	app, err := boot(t, cfg)
	if err != nil {
		t.Fatalf("trusted tier should run without bwrap, got: %v", err)
	}
	_, body := waitFor200(t, app.Posture().Shell.URL+"/projections/bundle.t.ok", 15*time.Second)
	if !strings.Contains(string(body), `"ran":1`) {
		t.Fatalf("bundle did not run under the trusted tier: %s", body)
	}
}
