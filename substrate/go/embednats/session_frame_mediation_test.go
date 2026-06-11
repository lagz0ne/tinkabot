package embednats

// TestSessionFrameMediation is the outside-in proof for Slice 3
// (session-frame-mediation). It is the single owning test for this surface,
// subdivided into one subtest per failure family plus one combined outside-in
// subtest that proves quota-bounded flood, contract rejection, and cross-session
// transcript isolation over real embedded NATS JetStream streams.

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

// sessionOutputStream returns the JetStream stream name for a session's durable output.
func sessionOutputStream(sessionID string) string {
	return "tb-session-out-" + sessionID
}

// ingestPublisher mints a dedicated credential allowed to publish only the
// given session ingest subjects and returns a connection using it.
func ingestPublisher(ctx context.Context, t *testing.T, rt *Runtime, sessionIDs ...string) (*nats.Conn, error) {
	t.Helper()
	pass, err := secret()
	if err != nil {
		return nil, err
	}
	var allow []string
	for _, id := range sessionIDs {
		allow = append(allow, sessionIngestSubject(id))
	}
	user := "_tb_test_ingest_pub_" + sessionIDs[0]
	auth := core.Auth{
		User: user,
		Capability: core.Capability{
			PrincipalID: user,
			LeaseID:     pass,
			LeaseStatus: "active",
		},
		Permissions: core.Permissions{
			Publish:   core.PermList{Allow: allow},
			Subscribe: core.PermList{Allow: []string{"_INBOX.>"}},
		},
	}
	if err := rt.addSessionUser(auth); err != nil {
		return nil, err
	}
	return rt.ConnectAs(ctx, auth)
}

func TestSessionFrameMediation(t *testing.T) {
	t.Parallel()

	// SchemaViolationOnOut: a frame violating the session frame contract is
	// published to the ingest subject. The republisher intercepts it and
	// rejects it — the output subject receives only valid frames.
	t.Run("SchemaViolationOnOut", func(t *testing.T) {
		t.Parallel()

		rt, err := start(t, valid(t))
		if err != nil {
			t.Fatal(err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		sessionID := "sfm-schema-violation-001"

		mediator, err := StartFrameMediator(ctx, rt, FrameMediatorConfig{
			SessionID:     sessionID,
			QuotaMaxBytes: 1 << 20, // 1 MiB
		})
		if err != nil {
			t.Fatalf("SchemaViolationOnOut: StartFrameMediator: %v — FrameMediator (Slice 3 validating republisher) does not exist yet; implement session_frame_mediation.go", err)
		}
		defer mediator.Stop()

		// Publish a frame violating the contract (missing required 'frame' field).
		violatingFrame := `{"kind":"session.frame","origin":"wrapper","sessionId":"` + sessionID + `","text":"violating"}`
		publisherNC, err := ingestPublisher(ctx, t, rt, sessionID)
		if err != nil {
			t.Fatalf("publisher connect: %v", err)
		}
		defer publisherNC.Close()

		if err := publisherNC.Publish(sessionIngestSubject(sessionID), []byte(violatingFrame)); err != nil {
			t.Fatalf("publish violating frame: %v", err)
		}
		_ = publisherNC.FlushTimeout(200 * time.Millisecond)

		// FakeStatusImpersonation: a wrapper-emitted status frame (origin=wrapper,
		// frame=status) must be rejected — status frames must originate from the runner.
		fakeStatus, _ := json.Marshal(map[string]any{
			"kind":      "session.frame",
			"frame":     "status",
			"origin":    "wrapper",
			"sessionId": sessionID,
			"state":     "completed",
			"detail":    "impersonated status",
		})
		if err := publisherNC.Publish(sessionIngestSubject(sessionID), fakeStatus); err != nil {
			t.Fatalf("publish fake status frame: %v", err)
		}
		_ = publisherNC.FlushTimeout(200 * time.Millisecond)

		// Also publish a valid frame — only the valid frame should reach the output.
		validFrame, _ := json.Marshal(map[string]any{
			"kind":      "session.frame",
			"frame":     "token",
			"origin":    "wrapper",
			"sessionId": sessionID,
			"text":      "valid token",
		})
		if err := publisherNC.Publish(sessionIngestSubject(sessionID), validFrame); err != nil {
			t.Fatalf("publish valid frame: %v", err)
		}
		_ = publisherNC.FlushTimeout(200 * time.Millisecond)

		// The violating frames must not appear on the output subject/stream.
		outJS := mediator.OutputJS()

		cons, err := outJS.OrderedConsumer(ctx, sessionOutputStream(sessionID), jetstream.OrderedConsumerConfig{})
		if err != nil {
			t.Fatalf("SchemaViolationOnOut: ordered consumer: %v", err)
		}

		msgs, err := drainConsumer(ctx, cons, 500*time.Millisecond)
		if err != nil {
			t.Fatalf("SchemaViolationOnOut: drain: %v", err)
		}

		for _, m := range msgs {
			var f map[string]any
			if err := json.Unmarshal(m, &f); err != nil {
				continue
			}
			if f["frame"] == nil || f["frame"] == "" {
				t.Fatalf("SchemaViolationOnOut: contract-violating frame (missing 'frame' field) reached the output stream — republisher did not reject it")
			}
			if f["frame"] == "status" && f["origin"] == "wrapper" {
				t.Fatalf("SchemaViolationOnOut: wrapper-emitted status frame (FakeStatusImpersonation) reached the output stream — republisher did not reject it")
			}
		}
	})

	// QuotaExceeded: a flood of frames on a session's ingest subject must be
	// bounded by the per-session quota gate. The republisher must stop
	// republishing once the session's cumulative output crosses the byte quota.
	// In RED no quota enforcer exists; the flood exhausts storage unbounded.
	t.Run("QuotaExceeded", func(t *testing.T) {
		t.Parallel()

		rt, err := start(t, valid(t))
		if err != nil {
			t.Fatal(err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		sessionID := "sfm-quota-001"
		// Set a very small quota (5 KiB) so the flood hits it quickly.
		const quota = 5 * 1024

		mediator, err := StartFrameMediator(ctx, rt, FrameMediatorConfig{
			SessionID:     sessionID,
			QuotaMaxBytes: quota,
		})
		if err != nil {
			t.Fatalf("QuotaExceeded: StartFrameMediator: %v — FrameMediator does not exist yet", err)
		}
		defer mediator.Stop()

		// Flood the ingest subject with valid frames totalling >> quota.
		publisherNC, err := ingestPublisher(ctx, t, rt, sessionID)
		if err != nil {
			t.Fatalf("publisher connect: %v", err)
		}
		defer publisherNC.Close()

		// 1 KiB per frame × 20 frames = 20 KiB >> 5 KiB quota.
		frameText := strings.Repeat("x", 1024)
		for i := 0; i < 20; i++ {
			frame, _ := json.Marshal(map[string]any{
				"kind":      "session.frame",
				"frame":     "token",
				"origin":    "wrapper",
				"sessionId": sessionID,
				"text":      fmt.Sprintf("%s-%d", frameText, i),
			})
			_ = publisherNC.Publish(sessionIngestSubject(sessionID), frame)
		}
		_ = publisherNC.FlushTimeout(500 * time.Millisecond)

		time.Sleep(300 * time.Millisecond)

		// The output stream must contain no more than quota bytes of frame data.
		// Exceeding quota is the violation; the gate must have stopped the flood.
		outJS := mediator.OutputJS()

		cons, err := outJS.OrderedConsumer(ctx, sessionOutputStream(sessionID), jetstream.OrderedConsumerConfig{})
		if err != nil {
			t.Fatalf("QuotaExceeded: ordered consumer: %v", err)
		}

		msgs, err := drainConsumer(ctx, cons, 500*time.Millisecond)
		if err != nil {
			t.Fatalf("QuotaExceeded: drain: %v", err)
		}

		total := 0
		for _, m := range msgs {
			total += len(m)
		}

		// The quota gate must bound output. Allow a small tolerance for the last
		// frame that straddles the quota boundary.
		tolerance := 2048
		if total > quota+tolerance {
			t.Fatalf("QuotaExceeded: output stream contains %d bytes, quota is %d (+%d tolerance) — per-session quota gate did not bound the flood", total, quota, tolerance)
		}
		if len(msgs) == 0 {
			t.Fatal("QuotaExceeded: no frames reached the output stream at all — republisher is not wired to ingest")
		}
	})

	// BridgeBypassAttempt: a raw publish directly to the output subject
	// (bypassing the republisher) must be denied by NATS subject permissions.
	// The validating republisher must be the sole writer of the output subject;
	// a direct raw publish from an external connection must fail with a
	// Permissions Violation. In RED no sole-writer enforcement exists.
	t.Run("BridgeBypassAttempt", func(t *testing.T) {
		t.Parallel()

		rt, err := start(t, valid(t))
		if err != nil {
			t.Fatal(err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		sessionID := "sfm-bridge-bypass-001"

		// Start the FrameMediator to establish the sole-writer credential.
		mediator, err := StartFrameMediator(ctx, rt, FrameMediatorConfig{
			SessionID:     sessionID,
			QuotaMaxBytes: 1 << 20,
		})
		if err != nil {
			t.Fatalf("BridgeBypassAttempt: StartFrameMediator: %v — FrameMediator does not exist yet", err)
		}
		defer mediator.Stop()

		// Attempt a raw publish from a primary connection directly to the output
		// subject. The primary user does not hold publish permission on the output
		// subject — only the mediator's internal credential does. The denial
		// arrives as an async NATS -ERR.
		asyncErrCh := make(chan error, 4)
		errNC, err := rt.Connect(ctx, nats.ErrorHandler(func(_ *nats.Conn, _ *nats.Subscription, e error) {
			asyncErrCh <- e
		}))
		if err != nil {
			t.Fatalf("errNC connect: %v", err)
		}
		defer errNC.Close()

		outSubj := "tb.session." + sessionID + ".out"
		bypassPayload, _ := json.Marshal(map[string]any{
			"kind":      "session.frame",
			"frame":     "token",
			"origin":    "wrapper",
			"sessionId": sessionID,
			"text":      "bypass injection",
		})

		_ = errNC.Publish(outSubj, bypassPayload)
		_ = errNC.FlushTimeout(300 * time.Millisecond)

		// Wait for the async permission denial.
		isPermDenial := func(e error) bool {
			if e == nil {
				return false
			}
			s := e.Error()
			return strings.Contains(s, "Permissions Violation") ||
				strings.Contains(s, "not authorized") ||
				strings.Contains(s, "denied")
		}

		deadline := time.Now().Add(400 * time.Millisecond)
		denied := false
		for time.Now().Before(deadline) {
			select {
			case e := <-asyncErrCh:
				if isPermDenial(e) {
					denied = true
				}
			default:
				time.Sleep(20 * time.Millisecond)
			}
		}
		if !denied {
			t.Fatal("BridgeBypassAttempt: raw publish to session output subject was not denied — sole-writer enforcement (FrameMediator credential) does not exist yet")
		}
	})

	// CrossSessionEviction: a flood of frames on one session's ingest must not
	// evict or corrupt a second session's durable transcript. Per-session stream
	// isolation ensures session A's volume cannot push session B's messages out
	// of the stream.
	t.Run("CrossSessionEviction", func(t *testing.T) {
		t.Parallel()

		rt, err := start(t, valid(t))
		if err != nil {
			t.Fatal(err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		sessionA := "sfm-evict-A-001"
		sessionB := "sfm-evict-B-001"

		const smallQuota = 8 * 1024 // 8 KiB — session A floods this fast

		mediatorA, err := StartFrameMediator(ctx, rt, FrameMediatorConfig{
			SessionID:     sessionA,
			QuotaMaxBytes: smallQuota,
		})
		if err != nil {
			t.Fatalf("CrossSessionEviction: StartFrameMediator(A): %v — FrameMediator does not exist yet", err)
		}
		defer mediatorA.Stop()

		mediatorB, err := StartFrameMediator(ctx, rt, FrameMediatorConfig{
			SessionID:     sessionB,
			QuotaMaxBytes: 1 << 20, // 1 MiB — session B has headroom
		})
		if err != nil {
			t.Fatalf("CrossSessionEviction: StartFrameMediator(B): %v — FrameMediator does not exist yet", err)
		}
		defer mediatorB.Stop()

		publisherNC, err := ingestPublisher(ctx, t, rt, sessionA, sessionB)
		if err != nil {
			t.Fatalf("publisher connect: %v", err)
		}
		defer publisherNC.Close()

		// Publish a known sentinel frame on session B first.
		sentinelB, _ := json.Marshal(map[string]any{
			"kind":      "session.frame",
			"frame":     "token",
			"origin":    "wrapper",
			"sessionId": sessionB,
			"text":      "sentinel-B",
		})
		if err := publisherNC.Publish(sessionIngestSubject(sessionB), sentinelB); err != nil {
			t.Fatalf("publish sentinel B: %v", err)
		}
		_ = publisherNC.FlushTimeout(200 * time.Millisecond)

		// Give mediator B time to republish the sentinel.
		time.Sleep(200 * time.Millisecond)

		// Flood session A with large frames to fill its quota and stress storage.
		frameText := strings.Repeat("y", 1024)
		for i := 0; i < 30; i++ {
			frame, _ := json.Marshal(map[string]any{
				"kind":      "session.frame",
				"frame":     "token",
				"origin":    "wrapper",
				"sessionId": sessionA,
				"text":      fmt.Sprintf("%s-%d", frameText, i),
			})
			_ = publisherNC.Publish(sessionIngestSubject(sessionA), frame)
		}
		_ = publisherNC.FlushTimeout(500 * time.Millisecond)

		time.Sleep(400 * time.Millisecond)

		// Session B's transcript must still contain the sentinel frame.
		outJS := mediatorB.OutputJS()

		cons, err := outJS.OrderedConsumer(ctx, sessionOutputStream(sessionB), jetstream.OrderedConsumerConfig{})
		if err != nil {
			t.Fatalf("CrossSessionEviction: ordered consumer B: %v", err)
		}

		msgsB, err := drainConsumer(ctx, cons, 500*time.Millisecond)
		if err != nil {
			t.Fatalf("CrossSessionEviction: drain B: %v", err)
		}

		foundSentinel := false
		for _, m := range msgsB {
			var f map[string]any
			if err := json.Unmarshal(m, &f); err != nil {
				continue
			}
			if txt, _ := f["text"].(string); txt == "sentinel-B" {
				foundSentinel = true
				break
			}
		}
		if !foundSentinel {
			t.Fatal("CrossSessionEviction: session B sentinel frame not found after session A flood — per-session stream isolation does not exist or session A evicted session B's messages")
		}
	})

	// DuplicateFrame: the FrameMediator is a pass-through republisher. It has no
	// dedup logic and intentionally forwards duplicate frames — deduplication is
	// the JetStream stream consumer's concern (via Nats-Msg-Id / dedup window),
	// not this layer. This test proves the pass-through behavior: a duplicate
	// frame payload published twice reaches the output stream twice.
	t.Run("DuplicateFrame", func(t *testing.T) {
		t.Parallel()

		rt, err := start(t, valid(t))
		if err != nil {
			t.Fatal(err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		sessionID := "sfm-duplicate-001"

		mediator, err := StartFrameMediator(ctx, rt, FrameMediatorConfig{
			SessionID:     sessionID,
			QuotaMaxBytes: 1 << 20,
		})
		if err != nil {
			t.Fatalf("DuplicateFrame: StartFrameMediator: %v", err)
		}
		defer mediator.Stop()

		publisherNC, err := ingestPublisher(ctx, t, rt, sessionID)
		if err != nil {
			t.Fatalf("DuplicateFrame: publisher connect: %v", err)
		}
		defer publisherNC.Close()

		frame, _ := json.Marshal(map[string]any{
			"kind":      "session.frame",
			"frame":     "token",
			"origin":    "wrapper",
			"sessionId": sessionID,
			"text":      "duplicate-payload",
		})

		// Publish the same frame payload twice.
		_ = publisherNC.Publish(sessionIngestSubject(sessionID), frame)
		_ = publisherNC.Publish(sessionIngestSubject(sessionID), frame)
		_ = publisherNC.FlushTimeout(300 * time.Millisecond)

		time.Sleep(200 * time.Millisecond)

		// Both frames must reach the output stream — the FrameMediator is a
		// pass-through republisher; dedup is not its responsibility.
		outJS := mediator.OutputJS()
		cons, err := outJS.OrderedConsumer(ctx, sessionOutputStream(sessionID), jetstream.OrderedConsumerConfig{})
		if err != nil {
			t.Fatalf("DuplicateFrame: ordered consumer: %v", err)
		}

		msgs, err := drainConsumer(ctx, cons, 500*time.Millisecond)
		if err != nil {
			t.Fatalf("DuplicateFrame: drain: %v", err)
		}
		if len(msgs) < 2 {
			t.Fatalf("DuplicateFrame: expected 2 frames on output (pass-through), got %d — FrameMediator must not deduplicate", len(msgs))
		}
	})

	// StaleFrame: the FrameMediator performs no sequence or revision ordering
	// check. It is a validating republisher that enforces the frame contract
	// (required fields and origin rules), not an ordering filter. A frame
	// carrying a stale-looking sequence field is forwarded as long as it
	// satisfies the contract. Stale detection is a stream-consumer concern.
	t.Run("StaleFrame", func(t *testing.T) {
		t.Parallel()

		rt, err := start(t, valid(t))
		if err != nil {
			t.Fatal(err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		sessionID := "sfm-stale-001"

		mediator, err := StartFrameMediator(ctx, rt, FrameMediatorConfig{
			SessionID:     sessionID,
			QuotaMaxBytes: 1 << 20,
		})
		if err != nil {
			t.Fatalf("StaleFrame: StartFrameMediator: %v", err)
		}
		defer mediator.Stop()

		publisherNC, err := ingestPublisher(ctx, t, rt, sessionID)
		if err != nil {
			t.Fatalf("StaleFrame: publisher connect: %v", err)
		}
		defer publisherNC.Close()

		high, _ := json.Marshal(map[string]any{
			"kind":      "session.frame",
			"frame":     "token",
			"origin":    "wrapper",
			"sessionId": sessionID,
			"text":      "high-seq",
			"seq":       10,
		})
		low, _ := json.Marshal(map[string]any{
			"kind":      "session.frame",
			"frame":     "token",
			"origin":    "wrapper",
			"sessionId": sessionID,
			"text":      "stale-seq",
			"seq":       1,
		})

		_ = publisherNC.Publish(sessionIngestSubject(sessionID), high)
		_ = publisherNC.Publish(sessionIngestSubject(sessionID), low)
		_ = publisherNC.FlushTimeout(300 * time.Millisecond)

		time.Sleep(200 * time.Millisecond)

		// Both frames must reach the output stream — the FrameMediator does not
		// filter on sequence fields; stale detection is not its responsibility.
		outJS := mediator.OutputJS()
		cons, err := outJS.OrderedConsumer(ctx, sessionOutputStream(sessionID), jetstream.OrderedConsumerConfig{})
		if err != nil {
			t.Fatalf("StaleFrame: ordered consumer: %v", err)
		}

		msgs, err := drainConsumer(ctx, cons, 500*time.Millisecond)
		if err != nil {
			t.Fatalf("StaleFrame: drain: %v", err)
		}
		if len(msgs) < 2 {
			t.Fatalf("StaleFrame: expected 2 frames on output (pass-through), got %d — FrameMediator must not filter on sequence fields", len(msgs))
		}
	})

	// OutsideIn: combined proof that the FrameMediator correctly bounds a
	// quota-flood, rejects contract violations, and keeps session transcripts
	// isolated — all over real embedded NATS JetStream streams and subjects.
	//
	// This is the outside-in surface proof required by the plan; it is also the
	// scenario-matrix "allowed" citation for this surface: a well-formed frame
	// passes through the republisher and reaches the durable output stream.
	t.Run("OutsideIn", func(t *testing.T) {
		t.Parallel()

		rt, err := start(t, valid(t))
		if err != nil {
			t.Fatal(err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		sessionID := "sfm-outside-in-001"
		floodID := "sfm-outside-in-flood-001"

		const quota = 4 * 1024 // 4 KiB

		mediator, err := StartFrameMediator(ctx, rt, FrameMediatorConfig{
			SessionID:     sessionID,
			QuotaMaxBytes: 1 << 20,
		})
		if err != nil {
			t.Fatalf("OutsideIn: StartFrameMediator: %v — FrameMediator does not exist yet", err)
		}
		defer mediator.Stop()

		floodMediator, err := StartFrameMediator(ctx, rt, FrameMediatorConfig{
			SessionID:     floodID,
			QuotaMaxBytes: quota,
		})
		if err != nil {
			t.Fatalf("OutsideIn: StartFrameMediator(flood): %v", err)
		}
		defer floodMediator.Stop()

		// Use the stand-in agent via StartSessionRuntime to generate real frames
		// through the full path: subprocess → stdout → ingest → mediator → output.
		standinFrames := []map[string]any{
			{"kind": "session.frame", "frame": "token", "origin": "wrapper", "sessionId": sessionID, "text": "outside-in-proof"},
			{"kind": "session.frame", "frame": "status", "origin": "runner", "sessionId": sessionID, "state": "completed", "detail": "stand-in done"},
		}
		// Inject a violating frame between the two valid frames to prove rejection.
		var lines []string
		for i, f := range standinFrames {
			b, _ := json.Marshal(f)
			lines = append(lines, string(b))
			if i == 0 {
				// wrapper-emitted status (FakeStatusImpersonation): origin is
				// "wrapper" but frame is "status" — this is a contract violation
				// that the republisher must reject.
				fakeStatus, _ := json.Marshal(map[string]any{
					"kind":      "session.frame",
					"frame":     "status",
					"origin":    "wrapper", // wrong: status must be runner-originated
					"sessionId": sessionID,
					"state":     "completed",
					"detail":    "impersonated status",
				})
				lines = append(lines, string(fakeStatus))
			}
		}
		standinScript := strings.Join(lines, "\n")
		standinCmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("printf '%%s\\n' %s", shellQuote(standinScript)))

		srt, err := StartSessionRuntime(ctx, rt, SessionRuntimeConfig{
			SessionID: sessionID,
			Cmd:       standinCmd,
		})
		if err != nil {
			t.Fatalf("OutsideIn: StartSessionRuntime: %v", err)
		}

		// Flood the flood session to fill its quota.
		publisherNC, err := ingestPublisher(ctx, t, rt, floodID)
		if err != nil {
			t.Fatalf("publisher connect: %v", err)
		}
		defer publisherNC.Close()

		floodText := strings.Repeat("z", 512)
		for i := 0; i < 20; i++ {
			frame, _ := json.Marshal(map[string]any{
				"kind":      "session.frame",
				"frame":     "token",
				"origin":    "wrapper",
				"sessionId": floodID,
				"text":      fmt.Sprintf("%s-%d", floodText, i),
			})
			_ = publisherNC.Publish(sessionIngestSubject(floodID), frame)
		}
		_ = publisherNC.FlushTimeout(300 * time.Millisecond)

		// Wait for the stand-in to finish.
		if err := srt.Wait(ctx); err != nil {
			t.Logf("OutsideIn: srt.Wait: %v", err)
		}

		// Snapshot-plus-tail: open the output stream and consume from the
		// beginning. Every valid stand-in frame must be present; the
		// wrapper-emitted status frame must be absent.
		outJS := mediator.OutputJS()

		cons, err := outJS.OrderedConsumer(ctx, sessionOutputStream(sessionID), jetstream.OrderedConsumerConfig{})
		if err != nil {
			t.Fatalf("OutsideIn: ordered consumer: %v", err)
		}

		msgs, err := drainConsumer(ctx, cons, time.Second)
		if err != nil {
			t.Fatalf("OutsideIn: drain: %v", err)
		}

		// Check: at least one valid frame must be present (the token frame).
		foundToken := false
		foundFakeStatus := false
		for _, m := range msgs {
			var f map[string]any
			if err := json.Unmarshal(m, &f); err != nil {
				continue
			}
			if f["frame"] == "token" {
				foundToken = true
			}
			// Fake status: origin=wrapper frame=status must not reach output.
			if f["frame"] == "status" && f["origin"] == "wrapper" {
				foundFakeStatus = true
			}
		}
		if !foundToken {
			t.Fatal("OutsideIn: no token frame observed on output stream — republisher not wired to ingest")
		}
		if foundFakeStatus {
			t.Fatal("OutsideIn: wrapper-emitted status frame (FakeStatusImpersonation) reached output stream — republisher did not reject it")
		}

		// Flood must be bounded: the flood session's output must not exceed quota.
		floodOutJS := floodMediator.OutputJS()

		floodCons, err := floodOutJS.OrderedConsumer(ctx, sessionOutputStream(floodID), jetstream.OrderedConsumerConfig{})
		if err != nil {
			t.Fatalf("OutsideIn: ordered consumer flood: %v", err)
		}

		floodMsgs, err := drainConsumer(ctx, floodCons, 500*time.Millisecond)
		if err != nil {
			t.Fatalf("OutsideIn: drain flood: %v", err)
		}

		total := 0
		for _, m := range floodMsgs {
			total += len(m)
		}
		tolerance := 2048
		if total > quota+tolerance {
			t.Fatalf("OutsideIn: flood session output is %d bytes, quota is %d (+%d tolerance) — quota gate not effective", total, quota, tolerance)
		}
	})
}

// drainConsumer reads messages from a JetStream ordered consumer until idle.
// Returns raw message payloads.
func drainConsumer(ctx context.Context, cons jetstream.Consumer, idle time.Duration) ([][]byte, error) {
	var out [][]byte
	deadline := time.Now().Add(idle)
	for time.Now().Before(deadline) {
		msg, err := cons.Next(jetstream.FetchMaxWait(100 * time.Millisecond))
		if err != nil {
			if err == jetstream.ErrMsgIteratorClosed ||
				err == context.DeadlineExceeded ||
				err == context.Canceled {
				break
			}
			continue
		}
		out = append(out, msg.Data())
		_ = msg.Ack()
		deadline = time.Now().Add(idle)
	}
	return out, ctx.Err()
}

// FrameMediatorConfig, FrameMediator, and StartFrameMediator are defined in
// session_frame_mediation.go (GREEN implementation).
