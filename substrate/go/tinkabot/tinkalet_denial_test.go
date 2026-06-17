package tinkabot

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTinkaletAuthorityDenials(t *testing.T) {
	t.Parallel()

	t.Run("revoked credentials", func(t *testing.T) {
		t.Parallel()
		store := t.TempDir()
		cfg := cfgFor(store)
		cfg.BundleDir = clockBundle
		app, err := boot(t, cfg)
		if err != nil {
			t.Fatal(err)
		}
		env := tinkaletEnv(t)
		mustTinkalet(t, env, "profile", "import", "local", "--store", store, "--name", "local")
		mustTinkalet(t, env, "profile", "use", "local")
		if err := app.Runtime().Revoke("TB_APP", app.Creds(RoleCaller).UserPub); err != nil {
			t.Fatal(err)
		}

		code, out, errOut := runTinkalet(env, "trigger", "bundle.clock.tick")
		assertDeniedPrivate(t, code, out, errOut, "profile local denied bundle.clock.tick: revoked-credentials\n", app.CredsFile(RoleCaller))
	})

	t.Run("denied neighbor", func(t *testing.T) {
		t.Parallel()
		aStore := t.TempDir()
		aCfg := cfgFor(aStore)
		aCfg.BundleDir = clockBundle
		a, err := boot(t, aCfg)
		if err != nil {
			t.Fatal(err)
		}
		bStore := t.TempDir()
		bCfg := cfgFor(bStore)
		bCfg.BundleDir = clockBundle
		b, err := boot(t, bCfg)
		if err != nil {
			t.Fatal(err)
		}
		env := tinkaletEnv(t)
		mustTinkalet(t, env, "profile", "import", "local", "--store", aStore, "--name", "local")
		mustTinkalet(t, env, "profile", "use", "local")
		rewriteProfileServer(t, env, "local", b.Posture().NATS.ClientURL)

		code, out, errOut := runTinkalet(env, "trigger", "bundle.clock.tick")
		assertDeniedPrivate(t, code, out, errOut, "profile local denied bundle.clock.tick: denied-neighbor\n", a.CredsFile(RoleCaller))
	})

	t.Run("stale credentials no stronger fallback", func(t *testing.T) {
		t.Parallel()
		store := t.TempDir()
		cfg := cfgFor(store)
		cfg.BundleDir = clockBundle
		app, err := boot(t, cfg)
		if err != nil {
			t.Fatal(err)
		}
		env := tinkaletEnv(t)
		mustTinkalet(t, env, "profile", "import", "local", "--store", store, "--name", "local")
		mustTinkalet(t, env, "profile", "use", "local")
		data := envVal(env, "TINKALET_DATA_DIR")
		if err := os.Remove(filepath.Join(data, "profiles", "local", "caller.creds")); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(data, "profiles", "local", "author.creds"), mustReadFile(t, app.CredsFile(RoleAuthor)), 0o600); err != nil {
			t.Fatal(err)
		}

		code, out, errOut := runTinkalet(env, "trigger", "bundle.clock.tick")
		assertDeniedPrivate(t, code, out, errOut, "profile local denied bundle.clock.tick: stale-credentials\n", app.CredsFile(RoleAuthor))
	})
}

func assertDeniedPrivate(t *testing.T, code int, out, errOut, wantErr, secretFile string) {
	t.Helper()
	if code != 1 || out != "" || errOut != wantErr {
		t.Fatalf("exit/stdout/stderr = %d/%q/%q, want 1/empty/%q", code, out, errOut, wantErr)
	}
	secret := string(mustReadFile(t, secretFile))
	for _, leak := range []string{secret, "tb.bundle.clock.tick", "_INBOX", "operator.nk", "Permission Violation"} {
		if strings.Contains(out+errOut, leak) {
			t.Fatalf("output leaked %q: stdout=%q stderr=%q", leak, out, errOut)
		}
	}
}

func rewriteProfileServer(t *testing.T, env []string, name, server string) {
	t.Helper()
	file := filepath.Join(envVal(env, "TINKALET_CONFIG_DIR"), "profiles.json")
	var doc struct {
		Profiles []struct {
			Name               string `json:"name"`
			Server             string `json:"server"`
			Shell              string `json:"shell"`
			Role               string `json:"role"`
			Trust              string `json:"trust"`
			Source             string `json:"source"`
			CredentialRef      string `json:"credentialRef"`
			CredentialRedacted bool   `json:"credentialRedacted"`
		} `json:"profiles"`
	}
	body := mustReadFile(t, file)
	if err := json.Unmarshal(body, &doc); err != nil {
		t.Fatal(err)
	}
	for i := range doc.Profiles {
		if doc.Profiles[i].Name == name {
			doc.Profiles[i].Server = server
		}
	}
	out, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, out, 0o600); err != nil {
		t.Fatal(err)
	}
}

func envVal(env []string, key string) string {
	for _, item := range env {
		k, v, ok := strings.Cut(item, "=")
		if ok && k == key {
			return v
		}
	}
	return ""
}
