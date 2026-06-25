package embednats

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lagz0ne/tinkabot/substrate/go/core"
	"github.com/nats-io/nats.go"
)

func operatorCfg(t *testing.T, mode Exposure) Config {
	t.Helper()
	cfg := valid(t)
	cfg.Operator = true
	cfg.Exposure = mode
	if mode.Mode == ExposeInProcess {
		// The posture seam refuses socket fields without a declared loopback
		// posture (exposure_test.go exposed() does the same normalization).
		cfg.Host = ""
		cfg.Port = 0
		cfg.WebSocket = WebSocket{}
		cfg.Core.Topology.WebSocket = core.WebSocket{}
	}
	return cfg
}

// principal carries the full lease vocabulary that must survive into the minted user JWT.
func principal(id string, perms core.Permissions) core.Auth {
	return core.Auth{
		User: id,
		Capability: core.Capability{
			PrincipalID:   id,
			SessionID:     "session-" + id,
			CapabilityID:  "cap-" + id,
			LeaseID:       "lease-" + id,
			LeaseStatus:   "active",
			AppRevision:   "app.rev.7",
			SchemaVersion: "v1",
		},
		Permissions: perms,
	}
}

func appPerms(allow ...string) core.Permissions {
	return core.Permissions{
		Publish: core.PermList{Allow: allow},
		// NATS-shaped permissions stay authoritative: request/reply needs an
		// explicit inbox subscribe, exactly like the static-auth corpus
		// (browser_gateway_test.go:20, source_authority_cli_test.go:25).
		Subscribe: core.PermList{Allow: append([]string{"_INBOX.>"}, allow...)},
	}
}

// assertLeaseDenial proves a lease-bearing denial is attributed with the
// JWT-carried lease fields, mirroring source_router_test.go's denial oracle.
func assertLeaseDenial(t *testing.T, err error, cap core.Capability) {
	t.Helper()
	var got *Error
	if !errors.As(err, &got) {
		t.Fatalf("expected adapter error, got %T: %v", err, err)
	}
	if got.Details["leaseId"] != cap.LeaseID || got.Details["principalId"] != cap.PrincipalID {
		t.Fatalf("denial attribution lost: got %#v want lease %q principal %q", got.Details, cap.LeaseID, cap.PrincipalID)
	}
}

func mint(t *testing.T, rt *Runtime, account string, auth core.Auth) UserCreds {
	t.Helper()
	creds, err := rt.MintUser(account, auth, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	return creds
}

func connect(t *testing.T, rt *Runtime, creds UserCreds) *nats.Conn {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	nc, err := rt.ConnectCreds(ctx, creds.File)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(nc.Close)
	return nc
}

func TestOperatorKeyFirstStartAndReload(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	var pub, keyFile string
	var keyBytes []byte
	t.Run("first-start-generates", func(t *testing.T) {
		cfg := operatorCfg(t, Loopback())
		cfg.StoreDir = dir
		rt, err := start(t, cfg)
		if err != nil {
			t.Fatal(err)
		}
		op := rt.Posture().Operator
		if !op.Enabled || op.PublicKey == "" || op.KeyFile == "" {
			t.Fatalf("operator posture not live: %#v", op)
		}
		if !strings.HasPrefix(op.KeyFile, dir) {
			t.Fatalf("operator key %q not held under store dir %q", op.KeyFile, dir)
		}
		pub, keyFile = op.PublicKey, op.KeyFile
		keyBytes, err = os.ReadFile(keyFile)
		if err != nil || len(keyBytes) == 0 {
			t.Fatalf("operator key material unreadable: %v", err)
		}
	})

	t.Run("restart-reloads", func(t *testing.T) {
		cfg := operatorCfg(t, Loopback())
		cfg.StoreDir = dir
		rt, err := start(t, cfg)
		if err != nil {
			t.Fatal(err)
		}
		op := rt.Posture().Operator
		if op.PublicKey != pub {
			t.Fatalf("operator identity regenerated: first %q reload %q", pub, op.PublicKey)
		}
		again, err := os.ReadFile(keyFile)
		if err != nil || !bytes.Equal(again, keyBytes) {
			t.Fatalf("operator key material rewritten on reload: %v", err)
		}
	})
}

func TestOperatorMintedUserMatrix(t *testing.T) {
	t.Parallel()
	for _, mode := range []Exposure{InProcess(), Loopback()} {
		t.Run(string(mode.Mode), func(t *testing.T) {
			t.Parallel()
			rt, err := start(t, operatorCfg(t, mode))
			if err != nil {
				t.Fatal(err)
			}

			auth := principal("principal.app.alpha", appPerms("tb.app.>"))
			creds := mint(t, rt, AppAccount, auth)
			if creds.UserPub == "" || len(creds.File) == 0 {
				t.Fatalf("mint returned empty creds: %#v", creds)
			}
			if creds.Lease != auth.Capability {
				t.Fatalf("lease provenance lost in JWT: minted %#v want %#v", creds.Lease, auth.Capability)
			}

			nc := connect(t, rt, creds)
			sub, err := nc.SubscribeSync("tb.app.echo")
			if err != nil {
				t.Fatal(err)
			}
			if err := nc.Publish("tb.app.echo", []byte("ok")); err != nil {
				t.Fatal(err)
			}
			if _, err := sub.NextMsg(2 * time.Second); err != nil {
				t.Fatalf("allowed publish/subscribe did not deliver: %v", err)
			}

			responder := principal("principal.app.svc", core.Permissions{
				Subscribe:      core.PermList{Allow: []string{"tb.app.svc"}},
				AllowResponses: core.AllowResponses{Max: 1, ExpiresMs: 30000},
			})
			rc := connect(t, rt, mint(t, rt, AppAccount, responder))
			if _, err := rc.Subscribe("tb.app.svc", func(m *nats.Msg) { _ = m.Respond([]byte("pong")) }); err != nil {
				t.Fatal(err)
			}
			if err := rc.Flush(); err != nil {
				t.Fatal(err)
			}
			resp, err := nc.Request("tb.app.svc", []byte("ping"), 2*time.Second)
			if err != nil || string(resp.Data) != "pong" {
				t.Fatalf("allowed request/reply failed: %v", err)
			}
		})
	}
}

func TestOperatorAccountSplitDeniesNeighbor(t *testing.T) {
	t.Parallel()
	rt, err := start(t, operatorCfg(t, Loopback()))
	if err != nil {
		t.Fatal(err)
	}

	ctl := connect(t, rt, mint(t, rt, ControlAccount, principal("principal.ctl.watch", appPerms("tb.ctl.>"))))
	app := connect(t, rt, mint(t, rt, AppAccount, principal("principal.app.noisy", appPerms("tb.ctl.>"))))
	peer := connect(t, rt, mint(t, rt, AppAccount, principal("principal.app.peer", appPerms("tb.ctl.>"))))

	ctlSub, err := ctl.SubscribeSync("tb.ctl.ping")
	if err != nil {
		t.Fatal(err)
	}
	peerSub, err := peer.SubscribeSync("tb.ctl.ping")
	if err != nil {
		t.Fatal(err)
	}
	if err := ctl.Flush(); err != nil {
		t.Fatal(err)
	}
	if err := peer.Flush(); err != nil {
		t.Fatal(err)
	}
	if err := app.Publish("tb.ctl.ping", []byte("neighbor")); err != nil {
		t.Fatal(err)
	}
	if err := app.Flush(); err != nil {
		t.Fatal(err)
	}
	if _, err := peerSub.NextMsg(2 * time.Second); err != nil {
		t.Fatalf("same-account delivery failed, split oracle is meaningless: %v", err)
	}
	if msg, err := ctlSub.NextMsg(300 * time.Millisecond); err == nil {
		t.Fatalf("app-plane publish crossed the account split into control plane: %q", msg.Data)
	}
}

func TestOperatorConnDeniedJWTs(t *testing.T) {
	t.Parallel()
	rt, err := start(t, operatorCfg(t, Loopback()))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	t.Run("malformed", func(t *testing.T) {
		garbage := []byte("-----BEGIN NATS USER JWT-----\nnot.a.jwt\n------END NATS USER JWT------\n")
		nc, err := rt.ConnectCreds(ctx, garbage)
		if err == nil {
			nc.Close()
			t.Fatal("malformed JWT was accepted at the connection")
		}
		assertAdapter(t, err, ClientConnectFailed)
	})

	t.Run("expired", func(t *testing.T) {
		auth := principal("principal.app.brief", appPerms("tb.app.>"))
		creds, err := rt.MintUser(AppAccount, auth, 50*time.Millisecond)
		if err != nil {
			t.Fatal(err)
		}
		time.Sleep(300 * time.Millisecond)
		nc, err := rt.ConnectCreds(ctx, creds.File)
		if err == nil {
			nc.Close()
			t.Fatal("expired JWT was accepted at the connection")
		}
		assertAdapter(t, err, ClientConnectFailed)
		assertLeaseDenial(t, err, auth.Capability)
	})
}

// Duplicate principal handling is credential renewal, not lockout.
func TestOperatorDuplicatePrincipal(t *testing.T) {
	t.Parallel()
	rt, err := start(t, operatorCfg(t, Loopback()))
	if err != nil {
		t.Fatal(err)
	}

	auth := principal("principal.app.twice", appPerms("tb.app.>"))
	first := mint(t, rt, AppAccount, auth)
	second := mint(t, rt, AppAccount, auth)
	if first.UserPub == second.UserPub {
		t.Fatalf("duplicate mint reused user identity %q", first.UserPub)
	}
	if first.Lease != second.Lease {
		t.Fatalf("duplicate mint drifted lease provenance: %#v vs %#v", first.Lease, second.Lease)
	}

	a, b := connect(t, rt, first), connect(t, rt, second)
	sub, err := a.SubscribeSync("tb.app.dup")
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Flush(); err != nil {
		t.Fatal(err)
	}
	if err := b.Publish("tb.app.dup", []byte("both-live")); err != nil {
		t.Fatal(err)
	}
	if _, err := sub.NextMsg(2 * time.Second); err != nil {
		t.Fatalf("duplicate principal connections are not both live: %v", err)
	}
}

func TestOperatorLivePushSupersedesStaleClaims(t *testing.T) {
	t.Parallel()
	rt, err := start(t, operatorCfg(t, Loopback()))
	if err != nil {
		t.Fatal(err)
	}

	// Observer has its own full app perms; publisher inherits account defaults.
	observer := connect(t, rt, mint(t, rt, AppAccount, principal("principal.app.observer", appPerms("tb.app.>"))))
	pub := connect(t, rt, mint(t, rt, AppAccount, principal("principal.app.default", core.Permissions{})))

	subA, err := observer.SubscribeSync("tb.app.a")
	if err != nil {
		t.Fatal(err)
	}
	subB, err := observer.SubscribeSync("tb.app.b")
	if err != nil {
		t.Fatal(err)
	}
	if err := observer.Flush(); err != nil {
		t.Fatal(err)
	}

	send := func(subject string) {
		t.Helper()
		if err := pub.Publish(subject, []byte("x")); err != nil {
			t.Fatal(err)
		}
		if err := pub.Flush(); err != nil {
			t.Fatal(err)
		}
	}

	// Before any push the account default is deny-by-default: a permissionless
	// mint holds no publish authority.
	send("tb.app.a")
	if msg, err := subA.NextMsg(300 * time.Millisecond); err == nil {
		t.Fatalf("permissionless mint published before any account push: %q", msg.Data)
	}

	// Push 1: account defaults allow only tb.app.a — applied to the live conn.
	if err := rt.UpdateAccountPerms(AppAccount, appPerms("tb.app.a")); err != nil {
		t.Fatal(err)
	}
	send("tb.app.a")
	if _, err := subA.NextMsg(2 * time.Second); err != nil {
		t.Fatalf("allowed default publish did not deliver after live push: %v", err)
	}
	send("tb.app.b")
	if msg, err := subB.NextMsg(300 * time.Millisecond); err == nil {
		t.Fatalf("publish outside pushed account claims delivered: %q", msg.Data)
	}

	// Push 2: widen — the stale claims are superseded on the same live conn.
	if err := rt.UpdateAccountPerms(AppAccount, appPerms("tb.app.a", "tb.app.b")); err != nil {
		t.Fatal(err)
	}
	send("tb.app.b")
	if _, err := subB.NextMsg(2 * time.Second); err != nil {
		t.Fatalf("stale account claims were not superseded by live push: %v", err)
	}
}

func TestOperatorRevocationDisconnectsLive(t *testing.T) {
	t.Parallel()
	rt, err := start(t, operatorCfg(t, Loopback()))
	if err != nil {
		t.Fatal(err)
	}

	creds := mint(t, rt, AppAccount, principal("principal.app.doomed", appPerms("tb.app.>")))
	nc := connect(t, rt, creds)

	if err := rt.Revoke(AppAccount, creds.UserPub); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for nc.IsConnected() {
		if time.Now().After(deadline) {
			t.Fatal("revoked principal still connected: live auth reload did not enforce")
		}
		time.Sleep(20 * time.Millisecond)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	again, err := rt.ConnectCreds(ctx, creds.File)
	if err == nil {
		again.Close()
		t.Fatal("revoked creds reconnected")
	}
	assertAdapter(t, err, ClientConnectFailed)
	assertLeaseDenial(t, err, creds.Lease)
}

func TestOperatorRevocationAfterRestart(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	var creds UserCreds

	t.Run("mint", func(t *testing.T) {
		cfg := operatorCfg(t, Loopback())
		cfg.StoreDir = dir
		rt, err := start(t, cfg)
		if err != nil {
			t.Fatal(err)
		}
		creds = mint(t, rt, AppAccount, principal("principal.app.restarted", appPerms("tb.app.restart")))
		nc := connect(t, rt, creds)
		nc.Close()
	})

	t.Run("restart-revoke", func(t *testing.T) {
		cfg := operatorCfg(t, Loopback())
		cfg.StoreDir = dir
		restarted, err := start(t, cfg)
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		again, err := restarted.ConnectCreds(ctx, creds.File)
		if err != nil {
			t.Fatalf("persisted root-signed user did not reconnect after restart: %v", err)
		}
		again.Close()

		if err := restarted.Revoke(AppAccount, creds.UserPub); err != nil {
			t.Fatal(err)
		}
		denied, err := restarted.ConnectCreds(ctx, creds.File)
		if err == nil {
			denied.Close()
			t.Fatal("revoked persisted user reconnected after restart")
		}
		assertAdapter(t, err, ClientConnectFailed)
		assertLeaseDenial(t, err, creds.Lease)
	})
}

func TestOperatorKeyMaterialFailureTyped(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	var keyFile string

	t.Run("seed", func(t *testing.T) {
		cfg := operatorCfg(t, Loopback())
		cfg.StoreDir = dir
		rt, err := start(t, cfg)
		if err != nil {
			t.Fatal(err)
		}
		keyFile = rt.Posture().Operator.KeyFile
	})

	t.Run("corrupt-key-fails-typed", func(t *testing.T) {
		if err := os.WriteFile(keyFile, []byte("not-an-nkey-seed"), 0o600); err != nil {
			t.Fatal(err)
		}
		cfg := operatorCfg(t, Loopback())
		cfg.StoreDir = dir
		rt, err := start(t, cfg)
		if rt != nil {
			t.Fatalf("runtime started over corrupt operator key material: %#v", rt.Posture())
		}
		assertAdapter(t, err, OperatorKeyFailed)
	})
}

func TestOperatorFailureFamiliesTyped(t *testing.T) {
	t.Parallel()
	rt, err := start(t, operatorCfg(t, Loopback()))
	if err != nil {
		t.Fatal(err)
	}

	stale := principal("principal.app.stale", appPerms("tb.app.>"))
	stale.Capability.LeaseStatus = "revoked"
	forever := principal("principal.app.forever", appPerms("tb.app.>"))
	anon := principal("principal.app.anon", appPerms("tb.app.>"))
	anon.Capability.SessionID = ""
	anon.Capability.CapabilityID = ""
	// Degenerate response bound: no grant (Max <= 0) yet a TTL — must deny,
	// never mint a root-key-signed JWT with empty (allow-all) permissions.
	degenerate := principal("principal.app.degenerate", core.Permissions{
		AllowResponses: core.AllowResponses{ExpiresMs: 5000},
	})

	cases := []struct {
		name  string
		call  func() error
		kind  Kind
		lease core.Capability // non-zero: denial must carry lease attribution
	}{
		{
			"jwt-mint-inactive-lease",
			func() error {
				_, err := rt.MintUser(AppAccount, stale, time.Hour)
				return err
			},
			JWTMintFailed,
			stale.Capability,
		},
		{
			"jwt-mint-unbounded-ttl",
			func() error {
				_, err := rt.MintUser(AppAccount, forever, 0)
				return err
			},
			JWTMintFailed,
			forever.Capability,
		},
		{
			"jwt-mint-degenerate-response-bound",
			func() error {
				_, err := rt.MintUser(AppAccount, degenerate, time.Hour)
				return err
			},
			JWTMintFailed,
			degenerate.Capability,
		},
		{
			"account-compile-invalid-perms",
			func() error {
				bad := appPerms("tb.app.>")
				bad.AllowResponses = core.AllowResponses{Max: 1, ExpiresMs: 0}
				return rt.UpdateAccountPerms(AppAccount, bad)
			},
			AccountCompileFailed,
			core.Capability{},
		},
		{
			"account-update-unknown-account",
			func() error { return rt.UpdateAccountPerms("TB_NOPE", appPerms("tb.app.>")) },
			AccountUpdateFailed,
			core.Capability{},
		},
		{
			"revocation-unknown-user",
			func() error { return rt.Revoke(AppAccount, "UNOTAUSERKEY") },
			RevocationFailed,
			core.Capability{},
		},
		{
			"provenance-loss-missing-session",
			func() error {
				_, err := rt.MintUser(AppAccount, anon, time.Hour)
				return err
			},
			ProvenanceLost,
			anon.Capability,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.call()
			assertAdapter(t, err, tc.kind)
			if tc.lease.LeaseID != "" {
				assertLeaseDenial(t, err, tc.lease)
			}
		})
	}
}

// TestOperatorCLIRequestWithCreds proves the manual's connection preamble in
// operator mode: a real `nats` CLI caller authenticates with a minted creds
// file, an allowed request/reply behavior command succeeds, and a denied
// neighbor surfaces permission evidence in output (output-parsed oracle:
// nats CLI v0.3.0 exits 0 on permission errors).
func TestOperatorCLIRequestWithCreds(t *testing.T) {
	t.Parallel()
	rt, err := start(t, operatorCfg(t, Loopback()))
	if err != nil {
		t.Fatal(err)
	}

	responder := principal("principal.app.responder", core.Permissions{
		Subscribe:      core.PermList{Allow: []string{"tb.app.proof.echo"}},
		AllowResponses: core.AllowResponses{Max: 1, ExpiresMs: 30000},
	})
	rc, err := rt.MintUser(AppAccount, responder, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	nc := connect(t, rt, rc)
	sub, err := nc.Subscribe("tb.app.proof.echo", func(msg *nats.Msg) {
		_ = msg.Respond([]byte("ok"))
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sub.Unsubscribe() })
	if err := nc.FlushTimeout(time.Second); err != nil {
		t.Fatal(err)
	}

	caller := principal("principal.app.caller", core.Permissions{
		Publish:   core.PermList{Allow: []string{"tb.app.proof.echo"}, Deny: []string{"tb.app.proof.denied"}},
		Subscribe: core.PermList{Allow: []string{"_INBOX.>"}},
	})
	cc, err := rt.MintUser(AppAccount, caller, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	creds := filepath.Join(t.TempDir(), "caller.creds")
	if err := os.WriteFile(creds, cc.File, 0o600); err != nil {
		t.Fatal(err)
	}

	bin := natsCLIBin(t)
	cli := func(args ...string) string {
		base := []string{"--no-context", "--server", rt.Posture().ClientURL, "--creds", creds, "--timeout", "2s"}
		out, _ := exec.Command(bin, append(base, args...)...).CombinedOutput()
		return string(out)
	}

	out := cli("request", "--raw", "tb.app.proof.echo", "ping")
	if strings.TrimSpace(out) != "ok" {
		t.Fatalf("creds CLI request drift: %q", out)
	}
	out = cli("request", "--raw", "tb.app.proof.denied", "ping")
	if !strings.Contains(strings.ToLower(out), "permissions") {
		t.Fatalf("denied neighbor did not surface permission evidence: %s", out)
	}
}
