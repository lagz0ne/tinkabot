package embednats

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/lagz0ne/tinkabot/substrate/go/core"
	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
)

// exposed declares a posture and carries no raw socket fields: under the typed
// API the posture is the only way to ask for a socket. Permissions are widened
// to the test's own KV bucket so the round trip proves real JetStream traffic.
func exposed(t *testing.T, exp Exposure, bucket string) Config {
	t.Helper()
	cfg := valid(t)
	cfg.Exposure = exp
	cfg.Host = ""
	cfg.Port = 0
	cfg.WebSocket = WebSocket{}
	cfg.Core.Topology.WebSocket = core.WebSocket{}
	cfg.Auth.Permissions.Publish.Allow = []string{"$JS.API.>", "$KV." + bucket + ".>"}
	cfg.Auth.Permissions.Subscribe.Allow = []string{"_INBOX.>", "$KV." + bucket + ".>"}
	return cfg
}

func roundTrip(t *testing.T, nc *nats.Conn, bucket string) {
	t.Helper()
	js, err := nc.JetStream()
	if err != nil {
		t.Fatal(err)
	}
	kv, err := js.CreateKeyValue(&nats.KeyValueConfig{Bucket: bucket})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := kv.PutString("posture", "declared"); err != nil {
		t.Fatal(err)
	}
	got, err := kv.Get("posture")
	if err != nil {
		t.Fatal(err)
	}
	if string(got.Value()) != "declared" {
		t.Fatalf("kv round trip drift: %q", got.Value())
	}
}

func TestExposureInProcessDefaultServesWithoutSocket(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		exp    Exposure
		bucket string
	}{
		{"declared in-process", InProcess(), "tb_exposure_inproc_declared"},
		{"zero value defaults to in-process", Exposure{}, "tb_exposure_inproc_default"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rt, err := start(t, exposed(t, tc.exp, tc.bucket))
			if err != nil {
				t.Fatal(err)
			}

			p := rt.Posture()
			if p.Exposure.Mode != ExposeInProcess {
				t.Fatalf("mode drift: got %v want %v", p.Exposure.Mode, ExposeInProcess)
			}
			// No-socket check: an in-process runtime advertises no TCP endpoint.
			if p.Exposure.Addr != "" || p.ClientURL != "" {
				t.Fatalf("in-process posture leaked a TCP endpoint: addr=%q clientURL=%q", p.Exposure.Addr, p.ClientURL)
			}

			// The in-process connection is real: JetStream KV round trip over
			// nats.InProcessServer against the embedded runtime.
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			nc, err := rt.Connect(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer nc.Close()
			roundTrip(t, nc, tc.bucket)
		})
	}
}

func TestExposureLoopbackDeclaredBindsLoopback(t *testing.T) {
	t.Parallel()

	const bucket = "tb_exposure_loopback"
	rt, err := start(t, exposed(t, Loopback(), bucket))
	if err != nil {
		t.Fatal(err)
	}

	p := rt.Posture()
	if p.Exposure.Mode != ExposeLoopback {
		t.Fatalf("mode drift: got %v want %v", p.Exposure.Mode, ExposeLoopback)
	}
	// Bound-address check: declared loopback is the posture the server has.
	host, _, err := net.SplitHostPort(p.Exposure.Addr)
	if err != nil {
		t.Fatalf("loopback posture has no bound address: addr=%q: %v", p.Exposure.Addr, err)
	}
	if host != "127.0.0.1" {
		t.Fatalf("loopback bound beyond 127.0.0.1: %q", p.Exposure.Addr)
	}
	probe, err := net.DialTimeout("tcp", p.Exposure.Addr, 2*time.Second)
	if err != nil {
		t.Fatalf("declared loopback socket is not actually bound: %v", err)
	}
	_ = probe.Close()

	// Outside-in over the real TCP surface, same path the nats CLI proofs use.
	nc, err := nats.Connect(p.ClientURL,
		nats.Timeout(2*time.Second),
		nats.UserInfo("principal.browser.001", "lease-001"),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer nc.Close()
	roundTrip(t, nc, bucket)
}

func TestExposureLoopbackWebSocketBeyondLoopbackDenied(t *testing.T) {
	t.Parallel()

	// A loopback posture must bound EVERY socket, not just the main NATS port:
	// a websocket host beyond 127.0.0.1 is an external surface and may only be
	// requested through ExposeExternal (which stays denied in this slice).
	cfg := exposed(t, Loopback(), "tb_exposure_ws_widen")
	cfg.WebSocket = WebSocket{Enabled: true, Host: "0.0.0.0", Port: -1, NoTLS: true}

	rt, err := start(t, cfg)
	if rt != nil {
		t.Fatalf("loopback posture with non-loopback websocket host produced a runtime: %#v", rt.Posture())
	}
	assertAdapter(t, err, ExposureDenied)
}

func TestExposureLoopbackHostBeyondLoopbackDeniedBeforeBind(t *testing.T) {
	t.Parallel()

	// Deny-wins ordering: a non-loopback main host under a loopback posture is
	// a typed denial BEFORE any server exists — not a post-start mismatch
	// shutdown, which would briefly serve on all interfaces.
	cfg := exposed(t, Loopback(), "tb_exposure_host_widen")
	cfg.Host = "0.0.0.0"
	built := false
	cfg.newServer = func(o *natsserver.Options) (*natsserver.Server, error) {
		built = true
		return natsserver.NewServer(o)
	}

	rt, err := start(t, cfg)
	if rt != nil {
		t.Fatalf("loopback posture with non-loopback host produced a runtime: %#v", rt.Posture())
	}
	assertAdapter(t, err, ExposureDenied)
	if built {
		t.Fatal("denial came after server construction: the widened listener existed before deny")
	}
}

func TestExposureUndeclaredSocketDenied(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		mut  func(*Config)
	}{
		{"raw port without posture", func(cfg *Config) { cfg.Port = -1 }},
		{"raw websocket without posture", func(cfg *Config) {
			cfg.WebSocket = WebSocket{Enabled: true, Host: "127.0.0.1", Port: -1, NoTLS: true}
		}},
		{"unknown exposure mode", func(cfg *Config) {
			cfg.Exposure = Exposure{Mode: ExposureMode("public")}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := exposed(t, Exposure{}, "tb_exposure_undeclared")
			tc.mut(&cfg)
			rt, err := start(t, cfg)
			if rt != nil {
				t.Fatalf("undeclared socket request produced a runtime: %#v", rt.Posture())
			}
			assertAdapter(t, err, ExposureUndeclared)
		})
	}
}

func TestExposureExternalDeniedByDefault(t *testing.T) {
	t.Parallel()

	// Declaration-level TLS material only: this slice types the external tier
	// and its denial paths; live TLS serving is out of scope (plan/endgame-app.md:177).
	tls := TLSFiles{Cert: "testdata/external-cert.pem", Key: "testdata/external-key.pem"}
	cases := []struct {
		name string
		exp  Exposure
	}{
		{"nats surface without auth tier", External(ExternalSurfaces{NATS: true, TLS: tls})},
		{"nats surface without tls", External(ExternalSurfaces{NATS: true, AuthTier: TierExternal})},
		{"websocket surface without auth tier", External(ExternalSurfaces{WebSocket: true, TLS: tls})},
		{"websocket surface without tls", External(ExternalSurfaces{WebSocket: true, AuthTier: TierExternal})},
		{"gateway surface without auth tier", External(ExternalSurfaces{Gateway: true, TLS: tls})},
		{"gateway surface without tls", External(ExternalSurfaces{Gateway: true, AuthTier: TierExternal})},
		// Denied-by-default proof: even a FULLY declared surface (auth tier +
		// both TLS files) only passes the policy check — live external serving
		// is out of scope for this slice (plan/endgame-app.md:177).
		{"fully declared nats surface still denied", External(ExternalSurfaces{NATS: true, AuthTier: TierExternal, TLS: tls})},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rt, err := start(t, exposed(t, tc.exp, "tb_exposure_external"))
			if rt != nil {
				t.Fatalf("external tier policy violation produced a runtime: %#v", rt.Posture())
			}
			// Typed failure, not a warning: the denial is an adapter error kind.
			assertAdapter(t, err, ExposureDenied)
		})
	}
}

func TestExposureDeclaredPostureMatchesActual(t *testing.T) {
	t.Parallel()

	cfg := exposed(t, InProcess(), "tb_exposure_mismatch")
	// A posture mismatch is impossible to force through the public surface once
	// the declared posture drives server options, so the in-package construction
	// hook mutates listen options on the REAL nats-server (no fake type; the
	// server, transport, and JetStream stay real). Start must observe the bound
	// socket behind the declared in-process posture and refuse it.
	cfg.newServer = func(o *natsserver.Options) (*natsserver.Server, error) {
		o.DontListen = false
		o.Host = "127.0.0.1"
		o.Port = -1
		return natsserver.NewServer(o)
	}

	rt, err := start(t, cfg)
	if rt != nil {
		t.Fatalf("mismatched posture produced a runtime: %#v", rt.Posture())
	}
	assertAdapter(t, err, ExposureMismatch)
}

func TestExposureInProcessConnectFailureTyped(t *testing.T) {
	t.Parallel()

	cfg := exposed(t, InProcess(), "tb_exposure_authfail")
	rt, err := start(t, cfg)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	bad := cfg.Auth
	bad.Capability.LeaseID = "lease-wrong"
	nc, err := rt.ConnectAs(ctx, bad)
	if nc != nil {
		nc.Close()
		t.Fatal("rejected in-process credentials produced a connection")
	}
	assertAdapter(t, err, InProcessConnFailed)
}
