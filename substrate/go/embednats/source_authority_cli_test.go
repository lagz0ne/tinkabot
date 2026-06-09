package embednats

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/lagz0ne/tinkabot/substrate/go/core"
	"github.com/nats-io/nats.go"
)

func TestSourceAuthorityCLIAllowedAndDeniedSubject(t *testing.T) {
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

	rt, err := Start(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stop(t, rt) })

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

	out, err := natsCLI(rt, cfg.Auth, "request", "--raw", "tb.proof.runtime.execute", "ping")
	if err != nil {
		t.Fatalf("allowed request failed: %v\n%s", err, out)
	}
	if strings.TrimSpace(out) != "ok" {
		t.Fatalf("reply drift: %q", out)
	}

	out, err = natsCLI(rt, cfg.Auth, "request", "--raw", "tb.proof.runtime.denied", "ping")
	if !strings.Contains(strings.ToLower(out), "permissions") {
		t.Fatalf("denied neighbor request did not surface permission evidence: err=%v out=%s", err, out)
	}
}

func natsCLI(rt *Runtime, auth core.Auth, args ...string) (string, error) {
	base := []string{
		"--no-context",
		"--server", rt.Posture().ClientURL,
		"--user", auth.User,
		"--password", auth.Capability.LeaseID,
		"--timeout", "2s",
	}
	cmd := exec.Command("nats", append(base, args...)...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}
