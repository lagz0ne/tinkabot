package embednats

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/lagz0ne/tinkabot/substrate/go/apps/wrapper"
	"github.com/lagz0ne/tinkabot/substrate/go/core"
)

// steerIntent is the canonical session.steer_intent payload for sid.
func steerIntent(sid, text string) []byte {
	b, _ := json.Marshal(map[string]any{
		"kind":      "session.steer_intent",
		"intent":    "steer",
		"sessionId": sid,
		"text":      text,
	})
	return b
}

// outFrames drains the session output stream until pred is satisfied over the
// accumulated frames or the deadline passes, returning the accumulated frames.
func outFrames(ctx context.Context, t *testing.T, js jetstream.JetStream, sid string, pred func([][]byte) bool, wait time.Duration) [][]byte {
	t.Helper()
	deadline := time.Now().Add(wait)
	var msgs [][]byte
	for {
		cons, err := js.OrderedConsumer(ctx, "tb-session-out-"+sid, jetstream.OrderedConsumerConfig{})
		if err == nil {
			msgs, _ = drainConsumer(ctx, cons, 300*time.Millisecond)
		}
		if pred(msgs) || time.Now().After(deadline) {
			return msgs
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func hasFrame(msgs [][]byte, pred func(map[string]json.RawMessage) bool) bool {
	for _, m := range msgs {
		var f map[string]json.RawMessage
		if json.Unmarshal(m, &f) != nil {
			continue
		}
		if pred(f) {
			return true
		}
	}
	return false
}

func str(f map[string]json.RawMessage, key string) string {
	var s string
	_ = json.Unmarshal(f[key], &s)
	return s
}

// canonicalToken reports whether the canonical wrapper token frames for sid,
// joined in stream order, carry text. The agent may split a word across
// streaming deltas, so the oracle is the joined transcript, not one frame.
func canonicalToken(sid, text string) func([][]byte) bool {
	return func(msgs [][]byte) bool {
		var transcript strings.Builder
		for _, m := range msgs {
			var f map[string]json.RawMessage
			if json.Unmarshal(m, &f) != nil {
				continue
			}
			if str(f, "kind") == "session.frame" && str(f, "frame") == "token" &&
				str(f, "origin") == "wrapper" && str(f, "sessionId") == sid {
				transcript.WriteString(str(f, "text"))
			}
		}
		return strings.Contains(transcript.String(), text)
	}
}

// chunkBodyType reports whether msgs contains a canonical wrapper chunk frame
// whose body (the verbatim event line as a string) is an event of the given
// top-level type.
func chunkBodyType(sid, typ string) func([][]byte) bool {
	return func(msgs [][]byte) bool {
		return hasFrame(msgs, func(f map[string]json.RawMessage) bool {
			if str(f, "kind") != "session.frame" || str(f, "frame") != "chunk" || str(f, "sessionId") != sid {
				return false
			}
			var body struct {
				Type string `json:"type"`
			}
			return json.Unmarshal([]byte(str(f, "body")), &body) == nil && body.Type == typ
		})
	}
}

// mintSteerPublisher mints a least-authority publisher for the session steer
// subject, mirroring the runner principal that owns steering writes.
func mintSteerPublisher(t *testing.T, rt *Runtime, sid string) []byte {
	t.Helper()
	creds, err := rt.MintUser(AppAccount, core.Auth{
		User: "steerer-" + sid,
		Capability: core.Capability{
			PrincipalID:   "steerer-" + sid,
			SessionID:     sid,
			CapabilityID:  "steer-cap-" + sid,
			LeaseID:       "steer-lease-" + sid,
			LeaseStatus:   "active",
			AppRevision:   "wrapper.v1",
			SchemaVersion: "v1",
		},
		Permissions: core.Permissions{
			Publish:   core.PermList{Allow: []string{"tb.session." + sid + ".steer"}},
			Subscribe: core.PermList{Allow: []string{"_INBOX.>"}},
		},
	}, time.Hour)
	if err != nil {
		t.Fatalf("mint steer publisher: %v", err)
	}
	return creds.File
}

func writeCreds(t *testing.T, name string, file []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, file, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestAgentWrapperMediated is the slice-6 outside-in proof minus the live
// agent: the real wrapper loop runs a real subprocess that replays recorded
// claude stream-json lines (one malformed line first) and then echoes its
// stdin, connects with a MintTrustedWrapper credential, and is observed on
// the mediated output stream. It proves: canonical token/chunk frames arrive
// through the FrameMediator, a malformed line is survived (frames after it
// still flow), and a canonical steer intent published to the steer subject
// reaches the subprocess stdin translated to the claude user-message shape.
func TestAgentWrapperMediated(t *testing.T) {
	t.Parallel()

	rt, err := start(t, operatorCfg(t, Loopback()))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	const sid = "wrap-mediated-001"

	mediator, err := StartFrameMediator(ctx, rt, FrameMediatorConfig{SessionID: sid, QuotaMaxBytes: 1 << 20})
	if err != nil {
		t.Fatalf("StartFrameMediator: %v", err)
	}
	defer mediator.Stop()

	recorded, err := os.ReadFile(filepath.Join("..", "apps", "wrapper", "testdata", "recorded.jsonl"))
	if err != nil {
		t.Fatalf("recorded fixtures: %v", err)
	}
	linesFile := filepath.Join(t.TempDir(), "lines.jsonl")
	if err := os.WriteFile(linesFile, append([]byte("{malformed line first\n"), recorded...), 0o600); err != nil {
		t.Fatal(err)
	}

	creds, err := MintTrustedWrapper(rt, sid)
	if err != nil {
		t.Fatalf("MintTrustedWrapper: %v", err)
	}
	credsPath := writeCreds(t, "wrapper.creds", creds.File)

	// Replay the recorded lines, then echo stdin back to stdout so the steer
	// translation is observable through the same mediated path.
	cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("cat %s; exec cat", linesFile))
	h, err := wrapper.StartWrapper(ctx, wrapper.WrapperConfig{
		NATSUrl:   rt.Posture().ClientURL,
		CredsFile: credsPath,
		SessionID: sid,
		Cmd:       cmd,
	})
	if err != nil {
		t.Fatalf("StartWrapper: %v", err)
	}
	defer func() {
		cancel()
		_ = h.Wait(context.Background())
	}()

	js := mediator.OutputJS()

	// Canonical token after the malformed first line proves both the frame
	// shape and parse-failure survival.
	if msgs := outFrames(ctx, t, js, sid, canonicalToken(sid, "pong"), 15*time.Second); !canonicalToken(sid, "pong")(msgs) {
		t.Fatalf("no canonical token frame with recorded text on the mediated output stream (got %d frames)", len(msgs))
	}
	if msgs := outFrames(ctx, t, js, sid, chunkBodyType(sid, "result"), 10*time.Second); !chunkBodyType(sid, "result")(msgs) {
		t.Fatalf("no canonical chunk frame carrying the raw result event (got %d frames)", len(msgs))
	}

	// Steer: canonical intent in, claude-shaped user message out (echoed by
	// the subprocess, re-published as a chunk through mediation).
	steerNC, err := rt.ConnectCreds(ctx, mintSteerPublisher(t, rt, sid))
	if err != nil {
		t.Fatalf("steer publisher connect: %v", err)
	}
	defer steerNC.Close()
	if err := steerNC.Publish("tb.session."+sid+".steer", steerIntent(sid, "switch to the parity test")); err != nil {
		t.Fatalf("steer publish: %v", err)
	}

	userEcho := func(msgs [][]byte) bool {
		return hasFrame(msgs, func(f map[string]json.RawMessage) bool {
			if str(f, "frame") != "chunk" {
				return false
			}
			var body struct {
				Type    string `json:"type"`
				Message struct {
					Content []struct {
						Text string `json:"text"`
					} `json:"content"`
				} `json:"message"`
			}
			if json.Unmarshal([]byte(str(f, "body")), &body) != nil || body.Type != "user" {
				return false
			}
			return len(body.Message.Content) == 1 && body.Message.Content[0].Text == "switch to the parity test"
		})
	}
	if msgs := outFrames(ctx, t, js, sid, userEcho, 15*time.Second); !userEcho(msgs) {
		t.Fatalf("steer intent was not translated to a claude user message on the wrapper stdin (got %d frames)", len(msgs))
	}
}

// TestAgentWrapperLocalE2E is the live real-runner proof: the actual claude
// CLI under the real wrapper, observed and steered over real NATS through the
// mediated path. It is local-only by the no-live-agent-in-CI carried decision:
// set TB_E2E_CLAUDE=1 to run.
func TestAgentWrapperLocalE2E(t *testing.T) {
	t.Parallel()
	if os.Getenv("TB_E2E_CLAUDE") == "" {
		t.Skip("local-only live agent proof: set TB_E2E_CLAUDE=1 to run")
	}

	rt, err := start(t, operatorCfg(t, Loopback()))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	const sid = "wrap-e2e-001"

	mediator, err := StartFrameMediator(ctx, rt, FrameMediatorConfig{SessionID: sid, QuotaMaxBytes: 8 << 20})
	if err != nil {
		t.Fatalf("StartFrameMediator: %v", err)
	}
	defer mediator.Stop()

	creds, err := MintTrustedWrapper(rt, sid)
	if err != nil {
		t.Fatalf("MintTrustedWrapper: %v", err)
	}
	credsPath := writeCreds(t, "wrapper.creds", creds.File)

	cmd := exec.Command("claude",
		"--print", "--verbose",
		"--input-format", "stream-json",
		"--output-format", "stream-json",
		"--include-partial-messages")
	h, err := wrapper.StartWrapper(ctx, wrapper.WrapperConfig{
		NATSUrl:   rt.Posture().ClientURL,
		CredsFile: credsPath,
		SessionID: sid,
		Cmd:       cmd,
	})
	if err != nil {
		t.Fatalf("StartWrapper: %v", err)
	}
	defer func() {
		cancel()
		_ = h.Wait(context.Background())
	}()

	steerNC, err := rt.ConnectCreds(ctx, mintSteerPublisher(t, rt, sid))
	if err != nil {
		t.Fatalf("steer publisher connect: %v", err)
	}
	defer steerNC.Close()

	js := mediator.OutputJS()

	// First steer is the opening prompt of the long-lived session.
	if err := steerNC.Publish("tb.session."+sid+".steer", steerIntent(sid, "Reply with exactly the word: pong")); err != nil {
		t.Fatal(err)
	}
	if msgs := outFrames(ctx, t, js, sid, canonicalToken(sid, "pong"), 2*time.Minute); !canonicalToken(sid, "pong")(msgs) {
		t.Fatalf("live claude run produced no canonical token frame with the requested text (got %d frames)", len(msgs))
	}

	// Second steer mid-session proves the session is steerable while alive.
	if err := steerNC.Publish("tb.session."+sid+".steer", steerIntent(sid, "Now reply with exactly the word: maple")); err != nil {
		t.Fatal(err)
	}
	if msgs := outFrames(ctx, t, js, sid, canonicalToken(sid, "maple"), 2*time.Minute); !canonicalToken(sid, "maple")(msgs) {
		for i, m := range msgs {
			s := string(m)
			if len(s) > 200 {
				s = s[:200]
			}
			t.Logf("frame %d: %s", i, s)
		}
		t.Fatalf("mid-session steer produced no canonical token frame with the steered text (got %d frames)", len(msgs))
	}
}
