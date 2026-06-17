package tinkabot

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lagz0ne/tinkabot/substrate/go/tinkalet"
)

func TestLocalProfileDescriptor(t *testing.T) {
	t.Parallel()
	store := t.TempDir()
	app, err := boot(t, cfgFor(store))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(store, "local-profile.json")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("local profile descriptor missing: %v", err)
	}
	assertFileMode(t, path, 0o600)

	var desc struct {
		Kind       string `json:"kind"`
		Server     string `json:"server"`
		Shell      string `json:"shell"`
		Credential string `json:"credential"`
		Role       string `json:"role"`
		Trust      string `json:"trust"`
		Source     string `json:"source"`
	}
	if err := json.Unmarshal(body, &desc); err != nil {
		t.Fatal(err)
	}
	if desc.Kind != "tinkabot.localProfile.v1" ||
		desc.Server != app.Posture().NATS.ClientURL ||
		desc.Shell != app.Posture().Shell.URL ||
		desc.Credential != "caller.creds" ||
		desc.Role != "caller" ||
		desc.Trust != "local-owner" ||
		desc.Source != "local-store:"+store {
		t.Fatalf("descriptor drift: %#v, posture %#v", desc, app.Posture())
	}

	env := tinkaletEnv(t)
	code, out, errOut := runTinkalet(env, "profile", "import", "local", "--store", store, "--name", "local")
	if code != 0 || out != "profile local imported\n" || errOut != "" {
		t.Fatalf("import exit/stdout/stderr = %d/%q/%q", code, out, errOut)
	}
	code, out, errOut = runTinkalet(env, "profile", "list", "--json")
	if code != 0 || errOut != "" {
		t.Fatalf("list exit/stderr = %d/%q", code, errOut)
	}
	if strings.Contains(out, string(mustReadFile(t, filepath.Join(store, "caller.creds")))) {
		t.Fatalf("profile json leaked credential contents: %q", out)
	}
	if !strings.Contains(out, app.Posture().NATS.ClientURL) || !strings.Contains(out, app.Posture().Shell.URL) {
		t.Fatalf("profile json missing imported endpoints: %s", out)
	}
}

func tinkaletEnv(t *testing.T) []string {
	t.Helper()
	root := t.TempDir()
	for _, dir := range []string{"home", "xdg-config", "xdg-state"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	return []string{
		"HOME=" + filepath.Join(root, "home"),
		"XDG_CONFIG_HOME=" + filepath.Join(root, "xdg-config"),
		"XDG_STATE_HOME=" + filepath.Join(root, "xdg-state"),
		"TINKALET_CONFIG_DIR=" + filepath.Join(root, "cfg"),
		"TINKALET_DATA_DIR=" + filepath.Join(root, "data"),
	}
}

func runTinkalet(env []string, args ...string) (int, string, string) {
	var out, errOut bytes.Buffer
	code := tinkalet.Run(tinkalet.Config{Args: args, Stdout: &out, Stderr: &errOut, Env: env})
	return code, out.String(), errOut.String()
}

func assertFileMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("%s mode = %04o, want %04o", path, got, want)
	}
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return body
}
