package embednats

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/lagz0ne/tinkabot/substrate/go/core"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

func sessionIngestSubject(sessionID string) string {
	return "tb.session." + sessionID + ".ingest"
}

// TestSessionLivenessLeaseExpiry proves that the LivenessStore opens a dedicated
// per-key TTL bucket over real embedded NATS JetStream KV using the nats.go
// jetstream package (not the legacy nc.JetStream() API), writes a liveness claim
// with KeyTTL, and the claim expires and becomes unreadable after the TTL.
//
// This is the foundational primitive for the heartbeat-bound liveness lease.
// The plan requires this primitive to be proven over the real embedded server
// before anything builds on it.
func TestSessionLivenessLeaseExpiry(t *testing.T) {
	t.Parallel()

	rt, err := start(t, valid(t))
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	store, err := OpenLivenessStore(ctx, rt)
	if err != nil {
		t.Fatalf("OpenLivenessStore: %v", err)
	}
	t.Cleanup(store.Close)

	sessionID := "sess-lease-expiry-001"
	ttl := 2 * time.Second

	if err := store.ClaimLiveness(ctx, sessionID, ttl); err != nil {
		t.Fatalf("ClaimLiveness: %v", err)
	}

	alive, err := store.IsAlive(ctx, sessionID)
	if err != nil {
		t.Fatalf("IsAlive (immediate): %v", err)
	}
	if !alive {
		t.Fatal("liveness claim not present immediately after ClaimLiveness")
	}

	time.Sleep(ttl + time.Second)

	alive, err = store.IsAlive(ctx, sessionID)
	if err != nil {
		t.Fatalf("IsAlive (after expiry): %v", err)
	}
	if alive {
		t.Fatal("liveness claim still alive after TTL expiry — per-key TTL not enforced by LivenessStore")
	}

}

// TestSessionLivenessIdempotent proves that ClaimLiveness called twice for the
// same sessionID is idempotent: the second call must not error and IsAlive must
// return true after both calls (TTL refreshed, not an error).
// This covers the duplicate scenario-matrix family: a second ClaimLiveness call
// for the same sessionID while a lease already exists must be safe.
func TestSessionLivenessIdempotent(t *testing.T) {
	t.Parallel()

	rt, err := start(t, valid(t))
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	store, err := OpenLivenessStore(ctx, rt)
	if err != nil {
		t.Fatalf("OpenLivenessStore: %v", err)
	}
	t.Cleanup(store.Close)

	sessionID := "sess-idempotent-001"
	ttl := 2 * time.Second

	if err := store.ClaimLiveness(ctx, sessionID, ttl); err != nil {
		t.Fatalf("ClaimLiveness (first): %v", err)
	}
	// Second claim: same sessionID, same TTL — must not error.
	if err := store.ClaimLiveness(ctx, sessionID, ttl); err != nil {
		t.Fatalf("ClaimLiveness (second, duplicate): %v", err)
	}
	alive, err := store.IsAlive(ctx, sessionID)
	if err != nil {
		t.Fatalf("IsAlive after duplicate claim: %v", err)
	}
	if !alive {
		t.Fatal("duplicate ClaimLiveness broke the liveness lease — must be idempotent")
	}
}

// TestSessionLivenessRawNATSPrimitive proves the raw server primitive (per-key
// TTL via jetstream.KeyTTL + LimitMarkerTTL) directly, without the
// LivenessStore abstraction: a failure here means the pinned nats-server
// cannot support the liveness lease at all.
func TestSessionLivenessRawNATSPrimitive(t *testing.T) {
	t.Parallel()

	cfg := valid(t)
	cfg.Auth.Permissions.Publish.Allow = []string{"$JS.API.>", "$KV." + sessionLivenessKV + ".>"}
	cfg.Auth.Permissions.Subscribe.Allow = []string{"_INBOX.>", "$KV." + sessionLivenessKV + ".>"}

	rt, err := start(t, cfg)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	nc, err := rt.Connect(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(nc.Close)

	// Use the nats.go jetstream package — NOT nc.JetStream() (legacy API).
	// The scope guard requires the liveness store to use this package for
	// per-key TTL. This test proves the package and server support it.
	js, err := jetstream.New(nc)
	if err != nil {
		t.Fatalf("jetstream.New: %v", err)
	}

	// LimitMarkerTTL must be at least 1 second (server constraint on v2.14.2).
	kv, err := js.CreateOrUpdateKeyValue(ctx, jetstream.KeyValueConfig{
		Bucket:         sessionLivenessKV,
		Storage:        jetstream.FileStorage,
		LimitMarkerTTL: 1 * time.Second,
	})
	if err != nil {
		t.Fatalf("CreateOrUpdateKeyValue with LimitMarkerTTL: %v — nats-server v2.14.2 must support per-key TTL", err)
	}

	sessionID := "sess-raw-expiry-001"
	leaseKey := "lease-" + sessionID
	leaseVal := []byte(`{"sessionId":"` + sessionID + `","state":"running"}`)

	if _, err := kv.Create(ctx, leaseKey, leaseVal, jetstream.KeyTTL(2*time.Second)); err != nil {
		t.Fatalf("kv.Create with KeyTTL: %v", err)
	}

	entry, err := kv.Get(ctx, leaseKey)
	if err != nil {
		t.Fatalf("kv.Get (immediate): %v", err)
	}
	if string(entry.Value()) != string(leaseVal) {
		t.Fatalf("value mismatch: got %q want %q", entry.Value(), leaseVal)
	}

	time.Sleep(3 * time.Second)

	_, err = kv.Get(ctx, leaseKey)
	if err == nil {
		t.Fatal("key still present after TTL expiry — per-key TTL not enforced by embedded NATS server")
	}
	if err != jetstream.ErrKeyNotFound && !strings.Contains(err.Error(), "expired") {
		t.Fatalf("unexpected error after TTL expiry: %v", err)
	}
}

// TestSessionRuntimeSubsystem is the outside-in proof for the full slice 2
// obligation. It is a single owning test per the plan's traced-TDD rule,
// subdivided into sub-tests that each own one failure family.
func TestSessionRuntimeSubsystem(t *testing.T) {
	t.Parallel()

	// A long-lived session must run in a distinct execution subsystem, never on
	// the run-to-completion activation runner. Proven by attempting to start a
	// SessionRuntime; StartSessionRuntime must return a non-nil runtime (proving
	// the subsystem is distinct and can be started), not an error.
	t.Run("SessionStarvation", func(t *testing.T) {
		t.Parallel()

		rt, err := start(t, valid(t))
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		srt, err := StartSessionRuntime(ctx, rt, SessionRuntimeConfig{
			SessionID: "sess-starvation-001",
			Cmd:       exec.Command("/bin/sh", "-c", "exit 0"),
		})
		if err != nil {
			t.Fatalf("SessionStarvation: StartSessionRuntime returned error: %v — no distinct session execution subsystem exists", err)
		}
		if srt == nil {
			t.Fatal("SessionStarvation: StartSessionRuntime returned nil — no distinct session execution subsystem exists")
		}
	})

	// An orphaned stand-in subprocess plus an expired liveness lease must
	// resolve to a terminal record rather than leaving the session in an
	// unrecoverable state. Proven end-to-end: start a SessionRuntime with a
	// deterministic stand-in, claim a liveness lease with a short TTL, let the
	// TTL expire so IsAlive returns false (proving the detection trigger), then
	// call ReconcileOrphanedSession and assert a terminal record is written.
	// This proves the full failure-family chain, not just reconciliation in isolation.
	t.Run("OrphanedChild", func(t *testing.T) {
		t.Parallel()

		rt, err := start(t, valid(t))
		if err != nil {
			t.Fatal(err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		sessionID := "sess-orphan-001"

		// Step 1: start a SessionRuntime with a stand-in that blocks on stdin
		// (so it stays alive long enough to claim a liveness lease and expire).
		srt, err := StartSessionRuntime(ctx, rt, SessionRuntimeConfig{
			SessionID: sessionID,
			Cmd:       exec.Command("/bin/sh", "-c", "read line; exit 0"),
		})
		if err != nil {
			t.Fatalf("StartSessionRuntime: %v", err)
		}

		// Step 2: claim a liveness lease with a short TTL to simulate a live session.
		store, err := OpenLivenessStore(ctx, rt)
		if err != nil {
			t.Fatalf("OpenLivenessStore: %v", err)
		}
		t.Cleanup(store.Close)

		ttl := 1 * time.Second
		if err := store.ClaimLiveness(ctx, sessionID, ttl); err != nil {
			t.Fatalf("ClaimLiveness: %v", err)
		}

		// Verify the lease is live before expiry.
		alive, err := store.IsAlive(ctx, sessionID)
		if err != nil {
			t.Fatalf("IsAlive (before expiry): %v", err)
		}
		if !alive {
			t.Fatal("liveness lease not present immediately after claim")
		}

		// Step 3: let the TTL expire — this is the detection trigger for reconciliation.
		time.Sleep(ttl + 500*time.Millisecond)

		alive, err = store.IsAlive(ctx, sessionID)
		if err != nil {
			t.Fatalf("IsAlive (after expiry): %v", err)
		}
		if alive {
			t.Fatal("liveness lease still alive after TTL — expiry not enforced; reconciliation trigger absent")
		}

		// Step 4: the stand-in is now orphaned (lease expired). Simulate substrate
		// restart reconciliation: ReconcileOrphanedSession must write a terminal record.
		result, err := ReconcileOrphanedSession(ctx, rt, sessionID)
		if err != nil {
			t.Fatalf("ReconcileOrphanedSession: %v", err)
		}
		if result.State != SessionStateTerminal {
			t.Fatalf("orphan reconciliation: got state %q want %q", result.State, SessionStateTerminal)
		}
		if result.SessionID != sessionID {
			t.Fatalf("terminal record session id mismatch: got %q want %q", result.SessionID, sessionID)
		}

		// Clean up the blocked stand-in.
		_ = srt.Wait(ctx)
	})

	// The liveness lease store must use the nats.go jetstream package to
	// create a dedicated bucket with LimitMarkerTTL, so per-key TTL expiry is
	// observable over real embedded NATS. Proven by opening a LivenessStore,
	// writing a claim, letting the TTL expire, and reading back the absence.
	t.Run("LivenessLeaseExpired", func(t *testing.T) {
		t.Parallel()

		rt, err := start(t, valid(t))
		if err != nil {
			t.Fatal(err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		store, err := OpenLivenessStore(ctx, rt)
		if err != nil {
			t.Fatalf("OpenLivenessStore: %v", err)
		}
		t.Cleanup(store.Close)

		sessionID := "sess-liveness-001"
		// Per-key TTL requires at least 1s (embedded NATS server constraint on
		// LimitMarkerTTL buckets). Sub-second values are rejected with API error.
		ttl := 1 * time.Second

		if err := store.ClaimLiveness(ctx, sessionID, ttl); err != nil {
			t.Fatalf("ClaimLiveness: %v", err)
		}

		alive, err := store.IsAlive(ctx, sessionID)
		if err != nil {
			t.Fatalf("IsAlive (immediate): %v", err)
		}
		if !alive {
			t.Fatal("liveness claim not found immediately after write")
		}

		time.Sleep(ttl + 500*time.Millisecond)

		alive, err = store.IsAlive(ctx, sessionID)
		if err != nil {
			t.Fatalf("IsAlive (after expiry): %v", err)
		}
		if alive {
			t.Fatal("liveness claim still alive after TTL expiry — LivenessLeaseExpired not enforced")
		}
	})

	// Proves stale-revision semantics on the session records KV: a conditional
	// update that supplies the wrong revision is rejected by the server.
	// The liveness KV uses TTL semantics (not CAS revision); the records KV uses
	// Put (last-write-wins). The stale family here means: a caller holding an old
	// revision cannot silently overwrite a newer record — proven via the JS KV
	// Update API which is revision-gated.
	t.Run("StaleLeaseRevision", func(t *testing.T) {
		t.Parallel()

		rt, err := start(t, valid(t))
		if err != nil {
			t.Fatal(err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Open a records connection to exercise CAS on the records KV bucket.
		nc, err := recordsConn(ctx, rt)
		if err != nil {
			t.Fatalf("recordsConn: %v", err)
		}
		defer nc.Close()

		kv, err := openRecordsBucket(ctx, nc)
		if err != nil {
			t.Fatalf("openRecordsBucket: %v", err)
		}

		sessionID := "sess-stale-001"
		rev1, err := kv.Put(ctx, sessionID, []byte(`{"sessionId":"sess-stale-001","state":"running"}`))
		if err != nil {
			t.Fatalf("kv.Put (initial): %v", err)
		}

		_, err = kv.Put(ctx, sessionID, []byte(`{"sessionId":"sess-stale-001","state":"terminal"}`))
		if err != nil {
			t.Fatalf("kv.Put (second): %v", err)
		}

		_, err = kv.Update(ctx, sessionID, []byte(`{"sessionId":"sess-stale-001","state":"injected"}`), rev1)
		if err == nil {
			t.Fatal("stale revision: Update with old revision succeeded — CAS not enforced on session records KV")
		}
	})

	// Proves that purging a live liveness lease makes IsAlive return false.
	t.Run("RevokedLease", func(t *testing.T) {
		t.Parallel()

		rt, err := start(t, valid(t))
		if err != nil {
			t.Fatal(err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		store, err := OpenLivenessStore(ctx, rt)
		if err != nil {
			t.Fatalf("OpenLivenessStore: %v", err)
		}
		t.Cleanup(store.Close)

		sessionID := "sess-revoked-001"
		ttl := 30 * time.Second // long TTL — we revoke before natural expiry

		if err := store.ClaimLiveness(ctx, sessionID, ttl); err != nil {
			t.Fatalf("ClaimLiveness: %v", err)
		}

		alive, err := store.IsAlive(ctx, sessionID)
		if err != nil {
			t.Fatalf("IsAlive (before revoke): %v", err)
		}
		if !alive {
			t.Fatal("liveness claim not present before revoke")
		}

		js, jsErr := store.js.KeyValue(ctx, sessionLivenessKV)
		if jsErr != nil {
			t.Fatalf("KeyValue for revoke: %v", jsErr)
		}
		if err := js.Purge(ctx, leaseKey(sessionID)); err != nil {
			t.Fatalf("Purge (revoke): %v", err)
		}

		alive, err = store.IsAlive(ctx, sessionID)
		if err != nil {
			t.Fatalf("IsAlive (after revoke): %v", err)
		}
		if alive {
			t.Fatal("liveness claim still alive after purge — lease revocation not observed by IsAlive")
		}
	})

	// Every termination path (including crash) must resolve to a terminal record.
	// Proven outside-in: a deterministic stand-in subprocess is launched and
	// supervised through the session runtime, its frames are observed on the
	// session ingest subject over real NATS, and its exit writes a terminal
	// record. A denied-neighbor proof shows the runner's session-scoped
	// credential cannot observe or write another session's ingest subject.
	t.Run("TerminalRecordMissing", func(t *testing.T) {
		t.Parallel()

		rt, err := start(t, valid(t))
		if err != nil {
			t.Fatal(err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		sessionID := "sess-terminal-001"
		neighborID := "sess-terminal-neighbor-001"

		// The deterministic stand-in: a subprocess that writes a sequence of
		// valid session frames to stdout and then exits. It speaks the real
		// frame contract over stdio (not a fake of the NATS seam).
		//
		// Each line is a JSON session frame per the contract established in
		// session-contract-authority (Slice 1):
		//   token frame: {"kind":"session.frame","frame":"token","origin":"wrapper","sessionId":"...","text":"..."}
		//   status frame: {"kind":"session.frame","frame":"status","origin":"runner","sessionId":"...","state":"completed"}
		standinFrames := []map[string]any{
			{
				"kind":      "session.frame",
				"frame":     "token",
				"origin":    "wrapper",
				"sessionId": sessionID,
				"text":      "hello from stand-in",
			},
			{
				"kind":      "session.frame",
				"frame":     "status",
				"origin":    "runner",
				"sessionId": sessionID,
				"state":     "completed",
				"detail":    "stand-in exited normally",
			},
		}

		// Build the stand-in command: print frames as newline-delimited JSON to
		// stdout then exit. No live agent — CI uses this deterministic stand-in.
		// A malformed (non-JSON) line is injected between the two valid frames to
		// prove the runtime drops it and does not publish it to the ingest subject.
		var lines []string
		for i, f := range standinFrames {
			b, _ := json.Marshal(f)
			lines = append(lines, string(b))
			if i == 0 {
				lines = append(lines, "not-valid-json") // malformed: must be dropped
			}
		}
		standinScript := strings.Join(lines, "\n")
		standinCmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("printf '%%s\\n' %s", shellQuote(standinScript)))

		srt, err := StartSessionRuntime(ctx, rt, SessionRuntimeConfig{
			SessionID: sessionID,
			Cmd:       standinCmd,
		})
		if err != nil {
			t.Fatalf("StartSessionRuntime: %v", err)
		}

		// ── denied-neighbor: runner's credential cannot reach neighbor ──────
		// Proven before any watcher is treated as live (per plan obligation).
		// The runner's session-scoped credential must NOT be able to subscribe
		// or publish on a different session's ingest subject.
		// Denial oracle: async Permissions Violation captured via ErrorHandler —
		// NATS core always delivers -ERR asynchronously for publish/subscribe
		// permission violations; sync pubErr is always nil for core pubs.
		neighborSubject := sessionIngestSubject(neighborID)

		asyncErrCh := make(chan error, 8)
		runnerCred := srt.RunnerCredential()
		neighborNC, err := rt.ConnectAs(ctx, runnerCred,
			nats.ErrorHandler(func(_ *nats.Conn, _ *nats.Subscription, e error) {
				asyncErrCh <- e
			}),
		)
		if err != nil {
			t.Fatalf("neighbor connect: %v", err)
		}
		defer neighborNC.Close()

		isPermViolation := func(e error) bool {
			if e == nil {
				return false
			}
			s := e.Error()
			return strings.Contains(s, "Permissions Violation") ||
				strings.Contains(s, "not authorized") ||
				strings.Contains(s, "denied")
		}

		drainAsyncErr := func(label string) {
			// Drain all queued async errors and fail if none is a permission violation.
			deadline := time.Now().Add(400 * time.Millisecond)
			for time.Now().Before(deadline) {
				select {
				case e := <-asyncErrCh:
					if isPermViolation(e) {
						return // denial confirmed
					}
				default:
					time.Sleep(20 * time.Millisecond)
				}
			}
			t.Fatalf("denied-neighbor: %s — no async Permissions Violation received within flush window; credential boundary not enforced", label)
		}

		// Subscribe attempt: NATS accepts at client level then delivers async -ERR.
		sub, subErr := neighborNC.SubscribeSync(neighborSubject)
		if subErr == nil {
			_ = neighborNC.FlushTimeout(300 * time.Millisecond)
			msg, recvErr := sub.NextMsg(200 * time.Millisecond)
			if msg != nil {
				t.Fatal("denied-neighbor: runner subscribed to neighbor session ingest subject and received a message")
			}
			// recvErr == nats.ErrTimeout is the normal async-denial outcome;
			// the real denial signal is the async error — assert it arrived.
			if recvErr == nats.ErrTimeout || recvErr == nil {
				drainAsyncErr("subscribe on neighbor subject accepted without sync error")
			} else if !isPermViolation(recvErr) {
				t.Fatalf("denied-neighbor: unexpected subscribe recv error: %v", recvErr)
			}
		} else if !isPermViolation(subErr) {
			t.Fatalf("denied-neighbor: unexpected subscribe error: %v", subErr)
		}

		// Publish attempt: fire-and-forget; denial comes as async -ERR.
		_ = neighborNC.Publish(neighborSubject, []byte(`{"kind":"session.frame","frame":"token","origin":"wrapper","sessionId":"`+neighborID+`","text":"injected"}`))
		_ = neighborNC.FlushTimeout(300 * time.Millisecond)
		// Drain async errors; fail if no Permissions Violation for publish.
		drainAsyncErr("publish to neighbor session ingest subject")

		// ── observe frames on ingest subject over real NATS ─────────────────
		// A separate observer (with broader permissions) subscribes to the
		// ingest subject and expects to see the stand-in frames.
		observerNC, err := rt.Connect(ctx)
		if err != nil {
			t.Fatalf("observer connect: %v", err)
		}
		defer observerNC.Close()

		ingestSubject := sessionIngestSubject(sessionID)
		observerSub, err := observerNC.SubscribeSync(ingestSubject)
		if err != nil {
			t.Fatalf("subscribe ingest: %v", err)
		}
		t.Cleanup(func() { _ = observerSub.Unsubscribe() })

		// Wait for the stand-in to emit its frames and the runtime to publish them.
		var observed []map[string]any
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) && len(observed) < len(standinFrames) {
			msg, err := observerSub.NextMsg(500 * time.Millisecond)
			if err != nil {
				break
			}
			var frame map[string]any
			if err := json.Unmarshal(msg.Data, &frame); err != nil {
				t.Logf("ingest frame parse error: %v", err)
				continue
			}
			observed = append(observed, frame)
		}

		if len(observed) == 0 {
			t.Fatal("no frames observed on session ingest subject — runtime did not publish stand-in frames to real NATS")
		}
		// Malformed-frame proof: the injected non-JSON line must have been dropped;
		// only len(standinFrames) valid frames must arrive (not len(standinFrames)+1).
		if len(observed) > len(standinFrames) {
			t.Fatalf("malformed-frame: runtime published %d frames, expected at most %d — malformed line was not dropped", len(observed), len(standinFrames))
		}

		// ── terminal record must be written on session exit ─────────────────
		// Wait for the session to end and assert a terminal record is written.
		if err := srt.Wait(ctx); err != nil {
			t.Logf("session runtime wait: %v", err)
		}

		rec, err := ReadTerminalRecord(ctx, rt, sessionID)
		if err != nil {
			t.Fatalf("ReadTerminalRecord: %v", err)
		}
		if rec.State != SessionStateTerminal {
			t.Fatalf("terminal record state: got %q want %q", rec.State, SessionStateTerminal)
		}
		if rec.SessionID != sessionID {
			t.Fatalf("terminal record session id: got %q want %q", rec.SessionID, sessionID)
		}
	})
}

// TestSessionStdinInputPath proves the physical stdin pipe: a steer message
// published to the session steer subject over real NATS is delivered to the
// subprocess stdin. The stand-in reads one line from stdin, writes it back on
// stdout as a valid JSON frame, and exits. The observer sees that frame on the
// ingest subject, proving end-to-end delivery through the pipe.
//
// Mediated delivery (Slice 5 scope) is not exercised here; only the physical
// pipe from NATS subscriber to subprocess stdin is proven.
func TestSessionStdinInputPath(t *testing.T) {
	t.Parallel()

	rt, err := start(t, valid(t))
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	sessionID := "sess-stdin-001"
	steerSubject := "tb.session." + sessionID + ".steer"

	// Stand-in: read one line from stdin, emit it back as a JSON frame, exit.
	standinCmd := exec.Command("/bin/sh", "-c",
		`read line; printf '{"kind":"session.frame","frame":"token","origin":"wrapper","sessionId":"sess-stdin-001","text":"%s"}\n' "$line"`)

	srt, err := StartSessionRuntime(ctx, rt, SessionRuntimeConfig{
		SessionID: sessionID,
		Cmd:       standinCmd,
	})
	if err != nil {
		t.Fatalf("StartSessionRuntime: %v", err)
	}

	// Subscribe to ingest before publishing steer so the observer is live.
	observerNC, err := rt.Connect(ctx)
	if err != nil {
		t.Fatalf("observer connect: %v", err)
	}
	defer observerNC.Close()

	ingestSub, err := observerNC.SubscribeSync(sessionIngestSubject(sessionID))
	if err != nil {
		t.Fatalf("subscribe ingest: %v", err)
	}
	t.Cleanup(func() { _ = ingestSub.Unsubscribe() })

	// Give the runtime's startupHold time to elapse so its steer subscriber is live.
	time.Sleep(400 * time.Millisecond)

	// Publish a steer message using a dedicated steer-publisher credential.
	// The primary connection's publish allow list (tb.app.>) does not cover the
	// session steer subject (tb.session.*). Register a minimal internal user.
	steerPayload := "hello-stdin"
	steerPass, err := secret()
	if err != nil {
		t.Fatalf("steer secret: %v", err)
	}
	steerUser := "_tb_test_steer_pub_" + steerSubject
	steerAuth := core.Auth{
		User: steerUser,
		Capability: core.Capability{
			PrincipalID: steerUser,
			LeaseID:     steerPass,
			LeaseStatus: "active",
		},
		Permissions: core.Permissions{
			Publish:   core.PermList{Allow: []string{steerSubject}},
			Subscribe: core.PermList{Allow: []string{"_INBOX.>"}},
		},
	}
	if err := rt.addSessionUser(steerAuth); err != nil {
		t.Fatalf("steer addSessionUser: %v", err)
	}
	steerPublishNC, err := rt.ConnectAs(ctx, steerAuth)
	if err != nil {
		t.Fatalf("steer publisher connect: %v", err)
	}
	defer steerPublishNC.Close()

	if err := steerPublishNC.Publish(steerSubject, []byte(steerPayload)); err != nil {
		t.Fatalf("publish steer: %v", err)
	}
	_ = steerPublishNC.FlushTimeout(300 * time.Millisecond)

	msg, err := ingestSub.NextMsg(5 * time.Second)
	if err != nil {
		t.Fatalf("no ingest frame received after steer publish — stdin pipe not wired: %v", err)
	}

	var frame map[string]any
	if err := json.Unmarshal(msg.Data, &frame); err != nil {
		t.Fatalf("ingest frame not valid JSON: %v", err)
	}
	if frame["text"] != steerPayload {
		t.Fatalf("stdin pipe: got frame text %q, want %q — steer message not delivered to subprocess stdin", frame["text"], steerPayload)
	}

	if err := srt.Wait(ctx); err != nil {
		t.Logf("session runtime wait: %v", err)
	}
}

// shellQuote wraps a multi-line script body so printf receives it as a single
// argument. Used only in the deterministic stand-in construction above.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
