package tinkabot

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveBwrap(t *testing.T) {
	t.Run("EnvOverrideWins", func(t *testing.T) {
		t.Setenv("TB_BWRAP", "/custom/bwrap")
		t.Setenv("PATH", "")
		stubExe(t, filepath.Join(t.TempDir(), "tinkabot"))
		got, err := resolveBwrap()
		if err != nil {
			t.Fatal(err)
		}
		if got != "/custom/bwrap" {
			t.Fatalf("TB_BWRAP did not win: %q", got)
		}
	})

	t.Run("BundledBeforePath", func(t *testing.T) {
		dir := t.TempDir()
		sidecar := fakeBwrap(t, filepath.Join(dir, "libexec", "tinkabot", "bwrap"), 0)
		pathDir := t.TempDir()
		fakeBwrap(t, filepath.Join(pathDir, "bwrap"), 0)
		t.Setenv("PATH", pathDir)
		stubExe(t, filepath.Join(dir, "tinkabot"))
		got, err := resolveBwrap()
		if err != nil {
			t.Fatal(err)
		}
		if got != sidecar {
			t.Fatalf("bundled bwrap did not win: %q want %q", got, sidecar)
		}
	})

	t.Run("PathFallback", func(t *testing.T) {
		dir := t.TempDir()
		want := fakeBwrap(t, filepath.Join(dir, "bwrap"), 0)
		t.Setenv("PATH", dir)
		stubExe(t, filepath.Join(t.TempDir(), "tinkabot"))
		got, err := resolveBwrap()
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("PATH fallback drift: %q want %q", got, want)
		}
	})
}

func TestBundledBwrapFailsClosed(t *testing.T) {
	dir := t.TempDir()
	fakeBwrap(t, filepath.Join(dir, "libexec", "tinkabot", "bwrap"), 1)
	t.Setenv("PATH", "")
	stubExe(t, filepath.Join(dir, "tinkabot"))

	manifest := `{"kind":"bundle.manifest","name":"t","scripts":[{"name":"gen","file":"scripts/run.sh","command":"/bin/sh","boot":true}]}`
	cfg := cfgFor(t.TempDir())
	cfg.BundleDir = writeBundleScript(t, manifest, "#!/bin/sh\n")
	_, err := boot(t, cfg)
	assertKind(t, err, BundleRejected)
}

func stubExe(t *testing.T, path string) {
	t.Helper()
	old := executable
	executable = func() (string, error) { return path, nil }
	t.Cleanup(func() { executable = old })
}

func fakeBwrap(t *testing.T, path string, code int) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	body := []byte(fmt.Sprintf("#!/bin/sh\nexit %d\n", code))
	if err := os.WriteFile(path, body, 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}
