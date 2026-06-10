package embednats

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/lagz0ne/tinkabot/substrate/go/core"
	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
)

type Kind string

const (
	AdapterConfigInvalid     Kind = "AdapterConfigInvalid"
	ExposureUndeclared       Kind = "ExposureUndeclared"
	ExposureDenied           Kind = "ExposureDenied"
	ExposureMismatch         Kind = "ExposureMismatch"
	ServerStartFailed        Kind = "ServerStartFailed"
	ClientConnectFailed      Kind = "ClientConnectFailed"
	InProcessConnFailed      Kind = "InProcessConnFailed"
	JetStreamUnavailable     Kind = "JetStreamUnavailable"
	AuthLoadFailed           Kind = "AuthLoadFailed"
	WebSocketUnavailable     Kind = "WebSocketUnavailable"
	TopologyProbeFailed      Kind = "TopologyProbeFailed"
	DrainTimedOut            Kind = "DrainTimedOut"
	ShutdownFailed           Kind = "ShutdownFailed"
	AdapterCritical          Kind = "AdapterCritical"
	RouterConfigInvalid      Kind = "RouterConfigInvalid"
	RequestReplyListenFailed Kind = "RequestReplyListenFailed"
	SubjectSubscribeFailed   Kind = "SubjectSubscribeFailed"
	KVWatchFailed            Kind = "KVWatchFailed"
	ObjectWatchFailed        Kind = "ObjectWatchFailed"
	StreamConsumeFailed      Kind = "StreamConsumeFailed"
	SourceMalformed          Kind = "SourceMalformed"
	RouterCritical           Kind = "RouterCritical"
)

type Error struct {
	Kind      Kind
	Layer     string
	Operation string
	Message   string
	Details   map[string]string
	Cause     error
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s.%s: %s: %v", e.Layer, e.Kind, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s.%s: %s", e.Layer, e.Kind, e.Message)
}

func (e *Error) Unwrap() error {
	return e.Cause
}

type Config struct {
	Core         core.Config
	Auth         core.Auth
	AuthUsers    []core.Auth
	Operator     bool
	Exposure     Exposure
	ServerName   string
	Host         string
	Port         int
	StoreDir     string
	ReadyTimeout time.Duration
	StopTimeout  time.Duration
	WebSocket    WebSocket
	Probe        func(*Runtime) error

	newServer   func(*natsserver.Options) (*natsserver.Server, error)
	connect     func(string, ...nats.Option) (*nats.Conn, error)
	accountInfo func(nats.JetStreamContext) error
	secret      func() (string, error)
}

type WebSocket struct {
	Enabled bool
	Host    string
	Port    int
	NoTLS   bool
}

type WebSocketPosture struct {
	Enabled bool
	Host    string
	Port    int
	URL     string
	NoTLS   bool
}

type Posture struct {
	ServerName string
	ClientURL  string
	StoreDir   string
	Ready      bool
	JetStream  bool
	Topology   core.Topology
	Exposure   ExposurePosture
	WebSocket  WebSocketPosture
	Operator   OperatorPosture
	AuthUser   string
}

type Runtime struct {
	srv     *natsserver.Server
	nc      *nats.Conn
	js      nats.JetStreamContext
	posture Posture
	op      *operator
	user    string
	pass    string
	probe   string
	probePw string

	drain    func(context.Context) error
	shutdown func()
	wait     func()
}

func Start(cfg Config) (rt *Runtime, err error) {
	defer func() {
		if r := recover(); r != nil {
			if rt != nil {
				_ = rt.Stop(context.Background())
			}
			rt = nil
			err = fail(AdapterCritical, "Start", "adapter panic", nil, fmt.Errorf("%v", r))
		}
	}()

	mode, err := cfg.exposure()
	if err != nil {
		return nil, err
	}
	cfg = cfg.defaults()

	top, err := core.CheckTopology(cfg.Core.Topology)
	if err != nil {
		return nil, err
	}
	if _, err := core.CheckStore(cfg.Core.Store); err != nil {
		return nil, err
	}
	if top.Mode != core.SingleNode {
		return nil, fail(AdapterConfigInvalid, "Start", "only single-node live proof is supported in this task", nil, nil)
	}
	if cfg.StoreDir == "" {
		return nil, fail(AdapterConfigInvalid, "Start", "store dir is required", nil, nil)
	}
	var op *operator
	var probe *natsserver.User
	var authUsers []*natsserver.User
	if cfg.Operator {
		// Operator/JWT mode: the substrate-held master key is the authority;
		// static users are not allowed alongside TrustedOperators.
		if op, err = newOperator(cfg.StoreDir); err != nil {
			return nil, err
		}
	} else {
		if authUsers, err = users(cfg.Auth, cfg.AuthUsers); err != nil {
			return nil, err
		}
		probePass, err := cfg.secret()
		if err != nil {
			return nil, fail(AdapterCritical, "Start", "probe credential generation failed", nil, err)
		}
		probe = probeUser(probePass)
	}
	if cfg.WebSocket.Enabled && !cfg.WebSocket.NoTLS {
		return nil, fail(WebSocketUnavailable, "Start", "websocket TLS config is required unless NoTLS is explicit", nil, nil)
	}

	opts := &natsserver.Options{
		ServerName: cfg.ServerName,
		NoLog:      true,
		NoSigs:     true,
		JetStream:  true,
		StoreDir:   cfg.StoreDir,
	}
	if op != nil {
		opts.TrustedOperators = op.trusted
		opts.AccountResolver = op.resolver
		opts.SystemAccount = op.sysPub
	} else {
		opts.Users = append(authUsers, probe)
	}
	if mode == ExposeInProcess {
		opts.DontListen = true
		// In-process conns ride synchronous net.Pipe: when the server rejects
		// auth mid-handshake both sides can block writing, and only the server
		// WriteDeadline (default 10s) breaks the deadlock. Bound it to the
		// adapter's ready timeout so denial resolves promptly.
		opts.WriteDeadline = cfg.ReadyTimeout
	} else {
		opts.Host = cfg.Host
		opts.Port = cfg.Port
	}
	if cfg.WebSocket.Enabled {
		opts.Websocket = natsserver.WebsocketOpts{
			Host:  cfg.WebSocket.Host,
			Port:  cfg.WebSocket.Port,
			NoTLS: cfg.WebSocket.NoTLS,
		}
	}

	srv, err := cfg.newServer(opts)
	if err != nil {
		if cfg.WebSocket.Enabled {
			return nil, fail(WebSocketUnavailable, "Start", "embedded NATS WebSocket listener could not be created", nil, err)
		}
		return nil, fail(ServerStartFailed, "Start", "embedded NATS server could not be created", nil, err)
	}
	srv.Start()
	if !srv.ReadyForConnections(cfg.ReadyTimeout) {
		srv.Shutdown()
		srv.WaitForShutdown()
		return nil, fail(ServerStartFailed, "Start", "embedded NATS server did not become ready", nil, nil)
	}

	// The declared posture must be the posture the server actually has:
	// no socket when in-process, a bound loopback address when loopback.
	addr := ""
	if a := srv.Addr(); a != nil {
		addr = a.String()
	}
	mismatch := mode == ExposeInProcess && addr != ""
	if mode == ExposeLoopback {
		host, _, err := net.SplitHostPort(addr)
		mismatch = err != nil || host != "127.0.0.1"
	}
	if mismatch {
		srv.Shutdown()
		srv.WaitForShutdown()
		return nil, fail(ExposureMismatch, "Start", "declared exposure posture does not match server listening state",
			map[string]string{"mode": string(mode), "addr": addr}, nil)
	}

	clientURL := ""
	if mode != ExposeInProcess {
		clientURL = srv.ClientURL()
	}

	closed := make(chan struct{})
	var once sync.Once
	var owned nats.Option
	if op != nil {
		owned = nats.UserJWTAndSeed(op.probeJWT, op.probeSeed)
	} else {
		owned = nats.UserInfo(probe.Username, probe.Password)
	}
	dial := []nats.Option{
		nats.Timeout(cfg.ReadyTimeout),
		nats.DrainTimeout(cfg.StopTimeout),
		owned,
		nats.ClosedHandler(func(*nats.Conn) {
			once.Do(func() { close(closed) })
		}),
	}
	if mode == ExposeInProcess {
		dial = append(dial, nats.InProcessServer(srv))
	}
	nc, err := cfg.connect(clientURL, dial...)
	if err != nil {
		srv.Shutdown()
		srv.WaitForShutdown()
		if mode == ExposeInProcess {
			return nil, fail(InProcessConnFailed, "Start", "owned in-process NATS client could not connect", nil, err)
		}
		return nil, fail(ClientConnectFailed, "Start", "owned NATS client could not connect", nil, err)
	}
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		srv.Shutdown()
		srv.WaitForShutdown()
		return nil, fail(JetStreamUnavailable, "Start", "JetStream context could not be created", nil, err)
	}
	if err := cfg.accountInfo(js); err != nil {
		nc.Close()
		srv.Shutdown()
		srv.WaitForShutdown()
		return nil, fail(JetStreamUnavailable, "Start", "JetStream account is unavailable", nil, err)
	}

	rt = &Runtime{
		srv:  srv,
		nc:   nc,
		js:   js,
		op:   op,
		user: cfg.Auth.User,
		pass: cfg.Auth.Capability.LeaseID,
		posture: Posture{
			ServerName: cfg.ServerName,
			ClientURL:  clientURL,
			StoreDir:   cfg.StoreDir,
			Ready:      true,
			JetStream:  true,
			Topology:   top,
			Exposure:   ExposurePosture{Mode: mode, Addr: addr},
			WebSocket: WebSocketPosture{
				Enabled: cfg.WebSocket.Enabled,
				Host:    cfg.WebSocket.Host,
				Port:    cfg.WebSocket.Port,
				URL:     srv.WebsocketURL(),
				NoTLS:   cfg.WebSocket.NoTLS,
			},
			Operator: op.posture(),
			AuthUser: cfg.Auth.User,
		},
	}
	if probe != nil {
		rt.probe, rt.probePw = probe.Username, probe.Password
	}
	rt.drain = func(ctx context.Context) error {
		if rt.nc == nil {
			return nil
		}
		if err := rt.nc.Drain(); err != nil {
			rt.nc.Close()
			return err
		}
		select {
		case <-closed:
			return nil
		case <-ctx.Done():
			rt.nc.Close()
			return ctx.Err()
		}
	}
	rt.shutdown = srv.Shutdown
	rt.wait = srv.WaitForShutdown

	if cfg.Probe != nil {
		if err := cfg.Probe(rt); err != nil {
			ctx, cancel := context.WithTimeout(context.Background(), cfg.StopTimeout)
			_ = rt.Stop(ctx)
			cancel()
			return nil, fail(TopologyProbeFailed, "Start", "topology probe failed", nil, err)
		}
	}
	return rt, nil
}

func (r *Runtime) Posture() Posture {
	if r == nil {
		return Posture{}
	}
	return r.posture
}

func (r *Runtime) Connect(ctx context.Context, opts ...nats.Option) (*nats.Conn, error) {
	if r == nil || r.user == "" || r.pass == "" {
		return nil, fail(AdapterCritical, "Connect", "runtime client boundary is unavailable", nil, nil)
	}
	return r.dial(ctx, append(opts, nats.UserInfo(r.user, r.pass)))
}

func (r *Runtime) ConnectAs(ctx context.Context, auth core.Auth, opts ...nats.Option) (*nats.Conn, error) {
	if r == nil {
		return nil, fail(AdapterCritical, "Connect", "runtime client boundary is unavailable", nil, nil)
	}
	return r.dial(ctx, append(opts, nats.UserInfo(auth.User, auth.Capability.LeaseID)))
}

func (r *Runtime) dial(ctx context.Context, opts []nats.Option) (*nats.Conn, error) {
	inproc := r.posture.Exposure.Mode == ExposeInProcess
	if inproc && r.srv == nil || !inproc && r.posture.ClientURL == "" {
		return nil, fail(AdapterCritical, "Connect", "runtime client boundary is unavailable", nil, nil)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, fail(ClientConnectFailed, "Connect", "runtime client context is closed", nil, err)
	}
	dial := make([]nats.Option, 0, len(opts)+2)
	if deadline, ok := ctx.Deadline(); ok {
		if ttl := time.Until(deadline); ttl > 0 {
			dial = append(dial, nats.Timeout(ttl))
		}
	}
	dial = append(dial, opts...)
	if inproc {
		dial = append(dial, nats.InProcessServer(r.srv))
	}
	nc, err := nats.Connect(r.posture.ClientURL, dial...)
	if err != nil {
		if inproc {
			return nil, fail(InProcessConnFailed, "Connect", "in-process connection failed", nil, err)
		}
		return nil, fail(ClientConnectFailed, "Connect", "runtime client could not connect", nil, err)
	}
	return nc, nil
}

func (r *Runtime) Stop(ctx context.Context) (err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = fail(AdapterCritical, "Stop", "adapter stop panic", nil, fmt.Errorf("%v", rec))
		}
	}()

	if r == nil {
		return fail(AdapterCritical, "Stop", "runtime is nil", nil, nil)
	}
	if r.drain == nil && r.shutdown == nil && r.wait == nil {
		return fail(AdapterCritical, "Stop", "runtime internals are nil", nil, nil)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	var drainErr error
	if r.drain != nil {
		drainErr = r.drain(ctx)
	}

	done := make(chan error, 1)
	go func() {
		var err error
		defer func() {
			if rec := recover(); rec != nil {
				err = fmt.Errorf("%v", rec)
			}
			done <- err
		}()
		if r.shutdown != nil {
			r.shutdown()
		}
		if r.wait != nil {
			r.wait()
		}
	}()

	select {
	case stopErr := <-done:
		if stopErr != nil {
			return fail(AdapterCritical, "Stop", "adapter shutdown panic", nil, stopErr)
		}
		if drainErr != nil {
			if ctx.Err() != nil {
				return fail(DrainTimedOut, "Stop", "owned NATS client drain timed out", nil, drainErr)
			}
			return fail(ShutdownFailed, "Stop", "owned NATS client drain failed", nil, drainErr)
		}
		return nil
	case <-ctx.Done():
		return fail(DrainTimedOut, "Stop", "embedded NATS shutdown timed out", nil, ctx.Err())
	}
}

func (cfg Config) defaults() Config {
	if cfg.ServerName == "" {
		cfg.ServerName = "tb-embedded"
	}
	if cfg.Host == "" {
		cfg.Host = "127.0.0.1"
	}
	if cfg.Port == 0 {
		cfg.Port = -1
	}
	if cfg.ReadyTimeout <= 0 {
		cfg.ReadyTimeout = 2 * time.Second
	}
	if cfg.StopTimeout <= 0 {
		cfg.StopTimeout = 2 * time.Second
	}
	if cfg.Core.Topology.WebSocket.Enabled && !cfg.WebSocket.Enabled {
		cfg.WebSocket.Enabled = true
		cfg.WebSocket.Port = cfg.Core.Topology.WebSocket.Port
	}
	if cfg.WebSocket.Enabled {
		if cfg.WebSocket.Host == "" {
			cfg.WebSocket.Host = cfg.Host
		}
		if cfg.WebSocket.Port == 0 {
			cfg.WebSocket.Port = -1
		}
	}
	if cfg.newServer == nil {
		cfg.newServer = natsserver.NewServer
	}
	if cfg.connect == nil {
		cfg.connect = nats.Connect
	}
	if cfg.accountInfo == nil {
		cfg.accountInfo = func(js nats.JetStreamContext) error {
			_, err := js.AccountInfo()
			return err
		}
	}
	if cfg.secret == nil {
		cfg.secret = secret
	}
	return cfg
}

func users(primary core.Auth, extra []core.Auth) ([]*natsserver.User, error) {
	all := append([]core.Auth{primary}, extra...)
	out := make([]*natsserver.User, 0, len(all))
	seen := map[string]bool{}
	for _, auth := range all {
		if seen[auth.User] {
			continue
		}
		user, err := user(auth)
		if err != nil {
			return nil, err
		}
		out = append(out, user)
		seen[auth.User] = true
	}
	return out, nil
}

func user(auth core.Auth) (*natsserver.User, error) {
	cap := auth.Capability
	switch {
	case strings.TrimSpace(auth.User) == "":
		return nil, fail(AuthLoadFailed, "LoadAuth", "auth user is required", nil, nil)
	case strings.TrimSpace(cap.LeaseID) == "":
		return nil, fail(AuthLoadFailed, "LoadAuth", "auth lease is required", nil, nil)
	case cap.LeaseStatus != "active":
		return nil, fail(AuthLoadFailed, "LoadAuth", "auth lease is not active", cap.Details(), nil)
	case cap.PrincipalID != "" && auth.User != cap.PrincipalID:
		return nil, fail(AuthLoadFailed, "LoadAuth", "auth principal does not match user", cap.Details(), nil)
	case auth.Permissions.AllowResponses.Max > 0 && auth.Permissions.AllowResponses.ExpiresMs <= 0:
		return nil, fail(AuthLoadFailed, "LoadAuth", "bounded response TTL is required", nil, nil)
	}
	return &natsserver.User{
		Username:    auth.User,
		Password:    cap.LeaseID,
		Permissions: perms(auth.Permissions),
	}, nil
}

func probeUser(pass string) *natsserver.User {
	return &natsserver.User{
		Username: "_tb_probe",
		Password: pass,
		Permissions: &natsserver.Permissions{
			Publish:   &natsserver.SubjectPermission{Allow: []string{"$JS.API.INFO"}},
			Subscribe: &natsserver.SubjectPermission{Allow: []string{"_INBOX.>"}},
		},
	}
}

func perms(p core.Permissions) *natsserver.Permissions {
	out := &natsserver.Permissions{
		Publish:   subj(p.Publish),
		Subscribe: subj(p.Subscribe),
	}
	if p.AllowResponses.Max > 0 {
		out.Response = &natsserver.ResponsePermission{
			MaxMsgs: p.AllowResponses.Max,
			Expires: time.Duration(p.AllowResponses.ExpiresMs) *
				time.Millisecond,
		}
	}
	return out
}

func subj(p core.PermList) *natsserver.SubjectPermission {
	return &natsserver.SubjectPermission{
		Allow: copyStrings(p.Allow),
		Deny:  copyStrings(p.Deny),
	}
}

func fail(kind Kind, op, msg string, details map[string]string, cause error) *Error {
	if details == nil {
		details = map[string]string{}
	}
	return &Error{
		Kind:      kind,
		Layer:     "EmbeddedNATSAdapter",
		Operation: op,
		Message:   msg,
		Details:   details,
		Cause:     cause,
	}
}

func secret() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func copyStrings(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	out := make([]string, len(items))
	copy(out, items)
	return out
}
