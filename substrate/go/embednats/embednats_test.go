package embednats

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/lagz0ne/tinkabot/substrate/go/core"
	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
)

func TestEmbeddedRuntimeLifecycle(t *testing.T) {
	rt, err := Start(valid(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stop(t, rt) })

	p := rt.Posture()
	if !p.Ready || !p.JetStream || p.ClientURL == "" || p.StoreDir == "" {
		t.Fatalf("posture drift: %#v", p)
	}
	if p.Topology.Mode != core.SingleNode || p.Topology.Replicas != 1 || p.Topology.Quorum != 1 {
		t.Fatalf("topology drift: %#v", p.Topology)
	}
	if !p.WebSocket.Enabled || !strings.HasPrefix(p.WebSocket.URL, "ws://") {
		t.Fatalf("websocket posture drift: %#v", p.WebSocket)
	}
	if _, err := rt.js.AccountInfo(); err != nil {
		t.Fatalf("JetStream not live: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	nc, err := rt.Connect(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer nc.Close()
	sub, err := nc.SubscribeSync("tb.app.adapter")
	if err != nil {
		t.Fatal(err)
	}
	if err := nc.Publish("tb.app.adapter", []byte("ok")); err != nil {
		t.Fatal(err)
	}
	if _, err := sub.NextMsg(time.Second); err != nil {
		t.Fatalf("runtime client boundary did not use loaded auth: %v", err)
	}
}

func TestPropagatesCoreContractInvalidity(t *testing.T) {
	cases := []struct {
		name string
		edit func(*Config)
		kind core.Kind
	}{
		{"topology", func(c *Config) { c.Core.Topology.JetStream = false }, core.TopologyInvalid},
		{"store", func(c *Config) { c.Core.Store.KVBucket = "" }, core.BucketMissing},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := valid(t)
			tc.edit(&cfg)

			rt, err := Start(cfg)
			if rt != nil {
				t.Fatalf("runtime started with invalid core contract: %#v", rt)
			}
			assertCore(t, err, tc.kind)
		})
	}
}

func TestAdapterOwnsRuntimeFailureMapping(t *testing.T) {
	cases := []struct {
		name string
		edit func(*Config)
		kind Kind
	}{
		{
			"auth",
			func(c *Config) { c.Auth.User = "" },
			AuthLoadFailed,
		},
		{
			"auth-revoked",
			func(c *Config) { c.Auth.Capability.LeaseStatus = "revoked" },
			AuthLoadFailed,
		},
		{
			"auth-expired",
			func(c *Config) { c.Auth.Capability.LeaseStatus = "expired" },
			AuthLoadFailed,
		},
		{
			"auth-missing-lease",
			func(c *Config) { c.Auth.Capability.LeaseID = "" },
			AuthLoadFailed,
		},
		{
			"auth-principal-mismatch",
			func(c *Config) { c.Auth.User = "principal.other" },
			AuthLoadFailed,
		},
		{
			"auth-malformed-response",
			func(c *Config) { c.Auth.Permissions.AllowResponses.ExpiresMs = 0 },
			AuthLoadFailed,
		},
		{
			"adapter-config",
			func(c *Config) { c.StoreDir = "" },
			AdapterConfigInvalid,
		},
		{
			"server-start",
			func(c *Config) {
				c.Core.Topology.WebSocket.Enabled = false
				c.WebSocket.Enabled = false
				c.newServer = func(*natsserver.Options) (*natsserver.Server, error) {
					return nil, errors.New("bind failed")
				}
			},
			ServerStartFailed,
		},
		{
			"client-connect",
			func(c *Config) {
				c.connect = func(string, ...nats.Option) (*nats.Conn, error) {
					return nil, errors.New("dial failed")
				}
			},
			ClientConnectFailed,
		},
		{
			"jetstream",
			func(c *Config) {
				c.accountInfo = func(nats.JetStreamContext) error {
					return errors.New("js disabled")
				}
			},
			JetStreamUnavailable,
		},
		{
			"websocket",
			func(c *Config) {
				c.WebSocket.Enabled = true
				c.WebSocket.NoTLS = false
			},
			WebSocketUnavailable,
		},
		{
			"websocket-start",
			func(c *Config) {
				c.newServer = func(*natsserver.Options) (*natsserver.Server, error) {
					return nil, errors.New("websocket bind failed")
				}
			},
			WebSocketUnavailable,
		},
		{
			"topology-probe",
			func(c *Config) {
				c.Probe = func(*Runtime) error {
					return errors.New("route missing")
				}
			},
			TopologyProbeFailed,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := valid(t)
			tc.edit(&cfg)

			rt, err := Start(cfg)
			if rt != nil {
				stop(t, rt)
				t.Fatalf("runtime started after %s failure", tc.name)
			}
			assertAdapter(t, err, tc.kind)
		})
	}
}

func TestInternalProbeUserIsLeastAuthority(t *testing.T) {
	rt, err := Start(valid(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stop(t, rt) })

	errs := make(chan error, 1)
	nc, err := nats.Connect(
		rt.Posture().ClientURL,
		nats.UserInfo(rt.probe, rt.probePw),
		nats.ErrorHandler(func(_ *nats.Conn, _ *nats.Subscription, err error) {
			errs <- err
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer nc.Close()

	if err := nc.Publish("tb.app.escape", []byte("deny")); err != nil {
		return
	}
	if err := nc.FlushTimeout(200 * time.Millisecond); err != nil {
		return
	}
	select {
	case err := <-errs:
		if err == nil {
			t.Fatal("expected probe permission error")
		}
	case <-time.After(time.Second):
		t.Fatal("probe user published outside readiness subjects")
	}
}

func TestStopMapsTimeout(t *testing.T) {
	block := make(chan struct{})
	defer close(block)

	rt := &Runtime{
		shutdown: func() {},
		wait:     func() { <-block },
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	assertAdapter(t, rt.Stop(ctx), DrainTimedOut)
}

func TestStopMapsShutdownFailure(t *testing.T) {
	shutdown := false
	rt := &Runtime{
		drain:    func(context.Context) error { return errors.New("drain failed") },
		shutdown: func() { shutdown = true },
	}

	assertAdapter(t, rt.Stop(context.Background()), ShutdownFailed)
	if !shutdown {
		t.Fatal("shutdown was not requested after drain failure")
	}
}

func TestAdapterCriticalWrapsUnknowns(t *testing.T) {
	cfg := valid(t)
	cfg.newServer = func(*natsserver.Options) (*natsserver.Server, error) {
		panic("boom")
	}

	rt, err := Start(cfg)
	if rt != nil {
		t.Fatalf("runtime started after panic: %#v", rt)
	}
	assertAdapter(t, err, AdapterCritical)
	assertAdapter(t, (&Runtime{}).Stop(context.Background()), AdapterCritical)
	assertAdapter(t, (&Runtime{
		drain: func(context.Context) error { panic("drain boom") },
	}).Stop(context.Background()), AdapterCritical)
	assertAdapter(t, (&Runtime{
		shutdown: func() { panic("shutdown boom") },
	}).Stop(context.Background()), AdapterCritical)
	assertAdapter(t, (&Runtime{
		wait: func() { panic("wait boom") },
	}).Stop(context.Background()), AdapterCritical)
}

func valid(t *testing.T) Config {
	t.Helper()

	return Config{
		Core: core.Config{
			Topology: core.Topology{
				Mode:      core.SingleNode,
				JetStream: true,
				Ready:     true,
				WebSocket: core.WebSocket{Enabled: true, Port: -1},
			},
			Store: core.Store{
				KVBucket:     "tb.adapter.kv",
				ObjectBucket: "tb.adapter.obj",
				Stream:       "tb.adapter.stream",
				ObjectKey:    "artifact-001/rev-1/bundle.js",
				ExpectedRev:  1,
				CurrentRev:   1,
				StreamCursor: "stream:1",
			},
		},
		Auth: core.Auth{
			User: "principal.browser.001",
			Capability: core.Capability{
				PrincipalID:  "principal.browser.001",
				SessionID:    "session-001",
				CapabilityID: "cap-001",
				LeaseID:      "lease-001",
				LeaseStatus:  "active",
			},
			Permissions: core.Permissions{
				Publish:   core.PermList{Allow: []string{"tb.app.>"}, Deny: []string{"tb.internal.>"}},
				Subscribe: core.PermList{Allow: []string{"tb.app.>"}, Deny: []string{"tb.internal.>"}},
				AllowResponses: core.AllowResponses{
					Max:       1,
					ExpiresMs: 30000,
				},
			},
		},
		ServerName:   "tb-embedded-test",
		Host:         "127.0.0.1",
		Port:         -1,
		StoreDir:     t.TempDir(),
		ReadyTimeout: 2 * time.Second,
		StopTimeout:  2 * time.Second,
		WebSocket: WebSocket{
			Enabled: true,
			Host:    "127.0.0.1",
			Port:    -1,
			NoTLS:   true,
		},
	}
}

func assertAdapter(t *testing.T, err error, kind Kind) {
	t.Helper()

	var got *Error
	if !errors.As(err, &got) {
		t.Fatalf("expected adapter error, got %T: %v", err, err)
	}
	if got.Kind != kind {
		t.Fatalf("kind mismatch: got %s want %s (%v)", got.Kind, kind, got)
	}
}

func assertCore(t *testing.T, err error, kind core.Kind) {
	t.Helper()

	var got *core.Error
	if !errors.As(err, &got) {
		t.Fatalf("expected core error, got %T: %v", err, err)
	}
	if got.Kind != kind {
		t.Fatalf("kind mismatch: got %s want %s (%v)", got.Kind, kind, got)
	}
	var adapter *Error
	if errors.As(err, &adapter) {
		t.Fatalf("core error transformed into adapter error: %v", err)
	}
}

func stop(t *testing.T, rt *Runtime) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := rt.Stop(ctx); err != nil {
		t.Fatal(err)
	}
}
