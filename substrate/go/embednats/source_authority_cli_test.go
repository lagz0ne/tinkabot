package embednats

import (
	"context"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lagz0ne/tinkabot/substrate/go/core"
	"github.com/nats-io/nats.go"
)

var natsTool struct {
	once sync.Once
	path string
	err  error
	out  string
}

func TestSourceAuthorityCLIAllowedAndDeniedSubject(t *testing.T) {
	t.Parallel()
	cfg := valid(t)
	cfg.Auth.User = "principal.source.request"
	cfg.Auth.Capability.PrincipalID = "principal.source.request"
	cfg.Auth.Capability.LeaseID = "lease-source-request-001"
	cfg.Auth.Permissions.Publish = core.PermList{
		Allow: []string{"tb.proof.runtime.execute"},
		Deny:  []string{"tb.proof.runtime.denied"},
	}
	cfg.Auth.Permissions.Subscribe = core.PermList{
		Allow: []string{"tb.proof.runtime.execute", "_INBOX.>"},
		Deny:  []string{"tb.proof.runtime.denied"},
	}
	cfg.Auth.Permissions.AllowResponses = core.AllowResponses{Max: 1, ExpiresMs: 30000}

	rt, err := start(t, cfg)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	nc, err := rt.Connect(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(nc.Close)
	sub, err := nc.Subscribe("tb.proof.runtime.execute", func(msg *nats.Msg) {
		_ = msg.Respond([]byte("ok"))
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sub.Unsubscribe() })
	if err := nc.FlushTimeout(time.Second); err != nil {
		t.Fatal(err)
	}

	out, err := natsCLI(t, rt, cfg.Auth, "request", "--raw", "tb.proof.runtime.execute", "ping")
	if err != nil {
		t.Fatalf("allowed request failed: %v\n%s", err, out)
	}
	if strings.TrimSpace(out) != "ok" {
		t.Fatalf("reply drift: %q", out)
	}

	out, err = natsCLI(t, rt, cfg.Auth, "request", "--raw", "tb.proof.runtime.denied", "ping")
	if !strings.Contains(strings.ToLower(out), "permissions") {
		t.Fatalf("denied neighbor request did not surface permission evidence: err=%v out=%s", err, out)
	}
}

func natsCLIBin(t *testing.T) string {
	t.Helper()
	natsTool.once.Do(func() {
		cmd := exec.Command("go", "tool", "-n", "nats")
		cmd.Dir = natsToolDir(t)
		out, err := cmd.CombinedOutput()
		natsTool.path = strings.TrimSpace(string(out))
		natsTool.err = err
		natsTool.out = string(out)
	})
	if natsTool.err != nil {
		t.Fatalf("nats CLI Go tool unavailable: %v\n%s", natsTool.err, natsTool.out)
	}
	return natsTool.path
}

func natsToolDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not resolve nats CLI tool module")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "tools", "natscli"))
}

func natsCLI(t *testing.T, rt *Runtime, auth core.Auth, args ...string) (string, error) {
	t.Helper()
	base := []string{
		"--no-context",
		"--server", rt.Posture().ClientURL,
		"--user", auth.User,
		"--password", auth.Capability.LeaseID,
		"--timeout", "2s",
	}
	cmd := exec.Command(natsCLIBin(t), append(base, args...)...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}
