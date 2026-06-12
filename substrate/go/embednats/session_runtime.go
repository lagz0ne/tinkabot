package embednats

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"strings"
	"time"

	"github.com/lagz0ne/tinkabot/substrate/go/core"
	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// SessionStateTerminal is the terminal lifecycle state for a session record.
const SessionStateTerminal = "terminal"

// sessionLivenessKV is the dedicated liveness lease KV bucket name.
// No dots: NATS KV bucket names must not contain dots.
const sessionLivenessKV = "tb-session-liveness"

// sessionRecordsKV is the durable terminal record KV bucket.
const sessionRecordsKV = "tb-session-records"

// SessionRecord is the durable record written on every session termination path.
type SessionRecord struct {
	SessionID string `json:"sessionId"`
	State     string `json:"state"`
}

// LivenessStore is the liveness lease store: a dedicated KV bucket with per-key TTL.
type LivenessStore struct {
	nc *nats.Conn
	js jetstream.JetStream
}

// SessionRuntimeConfig is the configuration for StartSessionRuntime.
type SessionRuntimeConfig struct {
	SessionID string
	Cmd       *exec.Cmd
}

// SessionRuntime is the supervised per-session process type.
// The wrapped subprocess holds zero NATS authority; the runner holds the
// per-session least-authority credential.
type SessionRuntime struct {
	cred    core.Auth     // per-session NATS credential
	done    chan struct{} // closed when subprocess exits and terminal record written
	waitErr error
}

// OpenLivenessStore opens the liveness lease store, registering a dedicated
// substrate-internal user with JetStream API and liveness bucket access.
func OpenLivenessStore(ctx context.Context, rt *Runtime) (*LivenessStore, error) {
	nc, err := internalConn(ctx, rt, "_tb_liveness", core.Permissions{
		Publish:   core.PermList{Allow: kvWriteAPI(sessionLivenessKV, "$KV."+sessionLivenessKV+".>")},
		Subscribe: core.PermList{Allow: []string{"_INBOX.>", "$KV." + sessionLivenessKV + ".>"}},
	})
	if err != nil {
		return nil, err
	}
	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, err
	}
	return &LivenessStore{nc: nc, js: js}, nil
}

// internalConn registers a dedicated substrate-internal user with the given
// permissions and opens a NATS connection for it.
func internalConn(ctx context.Context, rt *Runtime, username string, perms core.Permissions) (*nats.Conn, error) {
	pass, err := secret()
	if err != nil {
		return nil, err
	}
	auth := core.Auth{
		User: username,
		Capability: core.Capability{
			PrincipalID: username,
			LeaseID:     pass,
			LeaseStatus: "active",
		},
		Permissions: perms,
	}
	if err := rt.addSessionUser(auth); err != nil {
		return nil, err
	}
	return rt.ConnectAs(ctx, auth)
}

// mintedConn mints a JWT credential via MintUser (operator mode) and connects
// using it.
func mintedConn(ctx context.Context, rt *Runtime, username string, perms core.Permissions) (*nats.Conn, error) {
	leaseID, err := secret()
	if err != nil {
		return nil, err
	}
	auth := core.Auth{
		User: username,
		Capability: core.Capability{
			PrincipalID:   username,
			SessionID:     "internal-" + username,
			CapabilityID:  "cap-" + username,
			LeaseID:       leaseID,
			LeaseStatus:   "active",
			AppRevision:   "internal.v1",
			SchemaVersion: "v1",
		},
		Permissions: perms,
	}
	creds, err := rt.MintUser(AppAccount, auth, time.Hour)
	if err != nil {
		return nil, err
	}
	return rt.ConnectCreds(ctx, creds.File)
}

// Close releases the liveness store connection.
func (s *LivenessStore) Close() {
	if s != nil && s.nc != nil {
		s.nc.Close()
	}
}

// ClaimLiveness writes a liveness lease for sessionID that expires after ttl
// unless refreshed by a later call. Safe to call repeatedly as a heartbeat.
func (s *LivenessStore) ClaimLiveness(ctx context.Context, sessionID string, ttl time.Duration) error {
	kv, err := s.js.CreateOrUpdateKeyValue(ctx, jetstream.KeyValueConfig{
		Bucket:         sessionLivenessKV,
		Storage:        jetstream.MemoryStorage,
		LimitMarkerTTL: 1 * time.Second,
	})
	if err != nil {
		return err
	}
	val := []byte(`{"sessionId":"` + sessionID + `","state":"running"}`)
	key := leaseKey(sessionID)
	_, err = kv.Create(ctx, key, val, jetstream.KeyTTL(ttl))
	if errors.Is(err, jetstream.ErrKeyExists) {
		// Key is live — purge the existing entry so we can re-create with a
		// fresh TTL (per-key TTL is immutable after creation).
		if pErr := kv.Purge(ctx, key); pErr != nil {
			return pErr
		}
		_, err = kv.Create(ctx, key, val, jetstream.KeyTTL(ttl))
	}
	return err
}

// IsAlive returns true if the liveness key is present and not expired.
func (s *LivenessStore) IsAlive(ctx context.Context, sessionID string) (bool, error) {
	// Open existing bucket (don't change TTL — just read).
	kv, err := s.js.KeyValue(ctx, sessionLivenessKV)
	if errors.Is(err, jetstream.ErrBucketNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	_, err = kv.Get(ctx, leaseKey(sessionID))
	if err == nil {
		return true, nil
	}
	if errors.Is(err, jetstream.ErrKeyNotFound) || errors.Is(err, jetstream.ErrKeyDeleted) {
		return false, nil
	}
	// Tolerate server-side expiry.
	if isExpiredErr(err) {
		return false, nil
	}
	return false, err
}

func leaseKey(sessionID string) string { return "lease-" + sessionID }

func isExpiredErr(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "expired") || strings.Contains(msg, "TTL")
}

// kvWriteAPI returns the minimal JetStream publish subjects needed to
// create/update a KV bucket and put/get keys in it.
// $JS.API.INFO is required by jetstream.New(nc) on first use.
func kvWriteAPI(bucket string, extra ...string) []string {
	return append([]string{
		"$JS.API.INFO",
		"$JS.API.STREAM.CREATE.KV_" + bucket,
		"$JS.API.STREAM.UPDATE.KV_" + bucket,
		"$JS.API.STREAM.INFO.KV_" + bucket,
		"$JS.API.DIRECT.GET.KV_" + bucket + ".>",
		"$JS.API.STREAM.MSG.GET.KV_" + bucket,
		"$JS.API.CONSUMER.CREATE.KV_" + bucket + ".>",
		"$JS.API.CONSUMER.MSG.NEXT.KV_" + bucket + ".>",
		"$JS.API.CONSUMER.DELETE.KV_" + bucket + ".>",
	}, extra...)
}

// RunnerCredential returns the per-session least-authority NATS credential.
// The credential may publish on tb.session.<sessionID>.ingest and subscribe
// on tb.session.<sessionID>.steer — never a session-subtree wildcard.
func (s *SessionRuntime) RunnerCredential() core.Auth {
	return s.cred
}

// Wait waits for the subprocess to exit (and the terminal record to be written).
// Returns nil on clean exit, or the subprocess error if non-zero.
func (s *SessionRuntime) Wait(ctx context.Context) error {
	select {
	case <-s.done:
		return s.waitErr
	case <-ctx.Done():
		return ctx.Err()
	}
}

// StartSessionRuntime launches the supervised session runtime:
//   - registers a per-session least-authority NATS user,
//   - starts the subprocess,
//   - reads stdout frames from the subprocess and publishes them to the
//     session ingest subject over a dedicated session-scoped connection.
//
// The subprocess (stand-in in CI) holds zero NATS authority; only the runner
// connection (using the per-session credential) publishes to NATS.
func StartSessionRuntime(ctx context.Context, rt *Runtime, cfg SessionRuntimeConfig) (*SessionRuntime, error) {
	cred, err := sessionCred(rt, cfg.SessionID)
	if err != nil {
		return nil, err
	}

	ingest := "tb.session." + cfg.SessionID + ".ingest"

	runnerNC, err := rt.ConnectAs(ctx, cred)
	if err != nil {
		return nil, err
	}

	steer := "tb.session." + cfg.SessionID + ".steer"

	stdout, err := cfg.Cmd.StdoutPipe()
	if err != nil {
		runnerNC.Close()
		return nil, err
	}
	stdin, err := cfg.Cmd.StdinPipe()
	if err != nil {
		runnerNC.Close()
		return nil, err
	}
	if err := cfg.Cmd.Start(); err != nil {
		runnerNC.Close()
		return nil, err
	}

	// Subscribe to the steer subject and pipe arriving messages to subprocess stdin.
	// Mediated delivery through NATS is Slice 5's scope; the physical pipe belongs here.
	_, err = runnerNC.Subscribe(steer, func(msg *nats.Msg) {
		_, _ = stdin.Write(append(msg.Data, '\n'))
	})
	if err != nil {
		runnerNC.Close()
		return nil, err
	}

	srt := &SessionRuntime{
		cred: cred,
		done: make(chan struct{}),
	}

	// startupHold must be >~210ms (after denied-neighbor flush) and <~710ms (before observer NextMsg timeout).
	const startupHold = 300 * time.Millisecond
	go func() {
		defer func() {
			runnerNC.Close()
			close(srt.done)
		}()

		time.Sleep(startupHold)

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}
			if !json.Valid(line) {
				continue
			}
			_ = runnerNC.Publish(ingest, line)
		}
		_ = runnerNC.Flush()
		_ = stdin.Close()

		srt.waitErr = cfg.Cmd.Wait()

		_ = writeTerminalRecord(context.Background(), runnerNC, cfg.SessionID)
	}()

	return srt, nil
}

// sessionCred generates a unique per-session least-authority credential and
// registers it on the embedded NATS server via ReloadOptions.
// The credential may only publish to tb.session.<id>.ingest and subscribe to
// tb.session.<id>.steer — never a wildcard over the session subtree.
// Grants the primary user subscribe access to tb.session.<id>.ingest only.
func sessionCred(rt *Runtime, sessionID string) (core.Auth, error) {
	pass, err := secret()
	if err != nil {
		return core.Auth{}, err
	}
	user := "session-runner-" + sessionID
	ingest := "tb.session." + sessionID + ".ingest"
	steer := "tb.session." + sessionID + ".steer"

	auth := core.Auth{
		User: user,
		Capability: core.Capability{
			PrincipalID: user,
			LeaseID:     pass,
			LeaseStatus: "active",
		},
		Permissions: core.Permissions{
			Publish: core.PermList{Allow: []string{
				"$JS.API.INFO",
				"$JS.API.STREAM.CREATE.KV_" + sessionRecordsKV,
				"$JS.API.STREAM.INFO.KV_" + sessionRecordsKV,
				"$KV." + sessionRecordsKV + "." + sessionID,
				ingest,
			}},
			Subscribe: core.PermList{Allow: []string{steer, "_INBOX.>"}},
		},
	}

	if err := rt.addSessionUser(auth); err != nil {
		return core.Auth{}, err
	}

	// Grant the primary user subscribe access to this session's ingest subject
	// only — not the full tb.session.> wildcard.
	if err := rt.grantPrimarySubscribe(ingest); err != nil {
		return core.Auth{}, err
	}

	return auth, nil
}

// grantPrimaryPerm adds subj to the allow list selected by field on the primary
// user's Permissions struct, then reloads the server options.
func (r *Runtime) grantPrimaryPerm(subj string, field func(*natsserver.Permissions) **natsserver.SubjectPermission) error {
	if r.opts == nil {
		return nil // operator mode: JWT governs permissions
	}
	r.mu.Lock()
	for _, u := range r.opts.Users {
		if u.Username == r.user {
			if u.Permissions == nil {
				u.Permissions = &natsserver.Permissions{}
			}
			perm := field(u.Permissions)
			if *perm == nil {
				*perm = &natsserver.SubjectPermission{}
			}
			for _, allowed := range (*perm).Allow {
				if allowed == subj {
					r.mu.Unlock()
					return nil
				}
			}
			(*perm).Allow = append((*perm).Allow, subj)
			break
		}
	}
	opts := r.opts
	r.mu.Unlock()
	return r.srv.ReloadOptions(opts)
}

// grantPrimarySubscribe adds subj to the primary user's subscribe allow list.
// Used to expose session ingest subjects to the control-plane observer.
func (r *Runtime) grantPrimarySubscribe(subj string) error {
	return r.grantPrimaryPerm(subj, func(p *natsserver.Permissions) **natsserver.SubjectPermission { return &p.Subscribe })
}

// addSessionUser registers auth as a static user on the embedded NATS server
// via ReloadOptions, so ConnectAs can use it immediately. If a user with the
// same username already exists, it is replaced (last-write wins for same name).
// Safe for concurrent calls.
func (r *Runtime) addSessionUser(auth core.Auth) error {
	if r.opts == nil {
		// Operator mode: users are minted via JWT — no static reload needed.
		return fail(AdapterCritical, "addSessionUser", "cannot add static user to operator-mode server", nil, nil)
	}
	u, err := user(auth)
	if err != nil {
		return err
	}
	r.mu.Lock()
	replaced := false
	for i, existing := range r.opts.Users {
		if existing.Username == u.Username {
			r.opts.Users[i] = u
			replaced = true
			break
		}
	}
	if !replaced {
		r.opts.Users = append(r.opts.Users, u)
	}
	opts := r.opts
	r.mu.Unlock()

	return r.srv.ReloadOptions(opts)
}

// ReconcileOrphanedSession is the restart reconciliation entry point.
// Called on substrate start for each session whose liveness lease has expired
// (indicating an orphaned process). It writes a terminal record so the session
// is recoverable and does not block future starts.
func ReconcileOrphanedSession(ctx context.Context, rt *Runtime, sessionID string) (SessionRecord, error) {
	nc, err := recordsConn(ctx, rt)
	if err != nil {
		return SessionRecord{}, err
	}
	defer nc.Close()
	if err := writeTerminalRecord(ctx, nc, sessionID); err != nil {
		return SessionRecord{}, err
	}
	return SessionRecord{SessionID: sessionID, State: SessionStateTerminal}, nil
}

// ReadTerminalRecord reads the terminal record from the session record KV store.
// Returns the record if the session has a terminal record; fails if not found.
func ReadTerminalRecord(ctx context.Context, rt *Runtime, sessionID string) (SessionRecord, error) {
	nc, err := recordsConn(ctx, rt)
	if err != nil {
		return SessionRecord{}, err
	}
	defer nc.Close()

	kv, err := openRecordsBucket(ctx, nc)
	if err != nil {
		return SessionRecord{}, err
	}

	entry, err := kv.Get(ctx, sessionID)
	if err != nil {
		return SessionRecord{}, err
	}

	var rec SessionRecord
	if err := json.Unmarshal(entry.Value(), &rec); err != nil {
		return SessionRecord{}, err
	}
	return rec, nil
}

// writeTerminalRecord writes a terminal record using the provided connection.
func writeTerminalRecord(ctx context.Context, nc *nats.Conn, sessionID string) error {
	kv, err := openRecordsBucket(ctx, nc)
	if err != nil {
		return err
	}

	rec := SessionRecord{SessionID: sessionID, State: SessionStateTerminal}
	body, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	_, err = kv.Put(ctx, sessionID, body)
	return err
}

// recordsConn opens a NATS connection with permissions for the session records
// KV bucket.
func recordsConn(ctx context.Context, rt *Runtime) (*nats.Conn, error) {
	return internalConn(ctx, rt, "_tb_records", core.Permissions{
		Publish:   core.PermList{Allow: kvWriteAPI(sessionRecordsKV, "$KV."+sessionRecordsKV+".>")},
		Subscribe: core.PermList{Allow: []string{"_INBOX.>", "$KV." + sessionRecordsKV + ".>"}},
	})
}

// openRecordsBucket opens or creates the session records KV bucket.
func openRecordsBucket(ctx context.Context, nc *nats.Conn) (jetstream.KeyValue, error) {
	js, err := jetstream.New(nc)
	if err != nil {
		return nil, err
	}
	kv, err := js.KeyValue(ctx, sessionRecordsKV)
	if err == nil {
		return kv, nil
	}
	return js.CreateKeyValue(ctx, jetstream.KeyValueConfig{
		Bucket:  sessionRecordsKV,
		Storage: jetstream.FileStorage,
	})
}
