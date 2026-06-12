package embednats

// TestWebSessionSurface is the substrate-side outside-in proof for Slice 7
// (web-session-surface). One owning sub-test per substrate failure family:
//
//   - ViewerObserves:    a bearer viewer credential (no seed) connects over the
//     embedded NATS WebSocket and observes a mediated session through its own
//     deliver subject fed by a substrate-bound consumer — including frames
//     published before attach (snapshot-plus-tail).
//   - CrossSessionLeak:  a viewer credential for session A is denied subscribe
//     on session B's subjects over real NATS, and is denied publish on the
//     session steering subject (the runner stays the single steering writer).
//   - StaleViewerCred:   an expired viewer credential is denied connect; a
//     revoked one is disconnected and denied reconnect; renewal by re-mint
//     succeeds; a revoked cookie no longer gates an upgrade.
//   - ViewerMintFailed:  mint outside operator mode or without a session is a
//     typed, attributed failure.
//   - GateMatrixVacuous: scenario-matrix.json carries a web-session-surface
//     entry citing Go tests for all seven pinned families.
//   - UngrownManifest:   release/v1.json carries the session-v2 program and no
//     longer defers direct-browser-nats-websocket.
//
// The cookie-gated WS upgrade route itself is binary-owned and proven in
// substrate/go/tinkabot. FrameScopeEscape (hop two) is owned by the trusted
// shell's lease tests in apps/frontend.

import (
	"context"
	"encoding/json"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/lagz0ne/tinkabot/substrate/go/core"
)

// connectViewer dials the embedded NATS WebSocket listener with a bearer JWT
// only — no nkey seed ever exists browser-side, so the signature callback
// returns an empty signature and the server must accept the connection on the
// bearer claim alone.
func connectViewer(t *testing.T, rt *Runtime, jwt string) (*nats.Conn, error) {
	t.Helper()
	ws := rt.Posture().WebSocket
	if !ws.Enabled || ws.URL == "" {
		t.Fatalf("websocket posture is not enabled: %#v", ws)
	}
	return nats.Connect(ws.URL,
		nats.UserJWT(
			func() (string, error) { return jwt, nil },
			func([]byte) ([]byte, error) { return nil, nil },
		),
		nats.MaxReconnects(0),
	)
}

func TestWebSessionSurface(t *testing.T) {
	t.Parallel()

	// ViewerObserves: the allowed path. A mediated session produces frames;
	// the viewer attaches afterwards over WebSocket with a bearer credential
	// and still observes them through the substrate-bound deliver consumer.
	t.Run("ViewerObserves", func(t *testing.T) {
		t.Parallel()

		rt, err := start(t, operatorCfg(t, Loopback()))
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		const sid = "wss-observe-001"
		mediator, err := StartFrameMediator(ctx, rt, FrameMediatorConfig{SessionID: sid, QuotaMaxBytes: 1 << 20})
		if err != nil {
			t.Fatalf("ViewerObserves: StartFrameMediator: %v", err)
		}
		defer mediator.Stop()

		pub, err := mintedConn(ctx, rt, "_tb_test_ingest_"+sid, core.Permissions{
			Publish:   core.PermList{Allow: []string{"tb.session." + sid + ".ingest"}},
			Subscribe: core.PermList{Allow: []string{"_INBOX.>"}},
		})
		if err != nil {
			t.Fatalf("ViewerObserves: ingest publisher: %v", err)
		}
		defer pub.Close()
		frame, _ := json.Marshal(map[string]any{
			"kind": "session.frame", "frame": "token", "origin": "wrapper",
			"sessionId": sid, "text": "before-attach",
		})
		if err := pub.Publish("tb.session."+sid+".ingest", frame); err != nil {
			t.Fatal(err)
		}
		if err := pub.FlushTimeout(time.Second); err != nil {
			t.Fatal(err)
		}

		viewer, err := MintViewerCredential(rt, sid, 10*time.Minute)
		if err != nil {
			t.Fatalf("ViewerObserves: MintViewerCredential: %v", err)
		}
		if viewer.DeliverSubject == "" || !strings.HasPrefix(viewer.DeliverSubject, "tb.session."+sid+".deliver.") {
			t.Fatalf("ViewerObserves: viewer must get its own deliver subject under the session, got %q", viewer.DeliverSubject)
		}
		if strings.Contains(viewer.JWT, "SUAA") || strings.Contains(viewer.JWT, "-----BEGIN") {
			t.Fatal("ViewerObserves: viewer artifact must be a bare bearer JWT, never a creds file with a seed")
		}
		if err := BindViewerDeliver(ctx, rt, sid, viewer.DeliverSubject); err != nil {
			t.Fatalf("ViewerObserves: BindViewerDeliver: %v", err)
		}

		nc, err := connectViewer(t, rt, viewer.JWT)
		if err != nil {
			t.Fatalf("ViewerObserves: bearer WebSocket connect failed — the viewer credential must be usable without a seed over the WS listener: %v", err)
		}
		defer nc.Close()

		got := make(chan []byte, 16)
		if _, err := nc.Subscribe(viewer.DeliverSubject, func(m *nats.Msg) { got <- m.Data }); err != nil {
			t.Fatalf("ViewerObserves: subscribe own deliver subject: %v", err)
		}
		if err := nc.Flush(); err != nil {
			t.Fatal(err)
		}

		seen := func(text string, wait time.Duration) bool {
			deadline := time.After(wait)
			for {
				select {
				case b := <-got:
					if strings.Contains(string(b), text) {
						return true
					}
				case <-deadline:
					return false
				}
			}
		}
		if !seen("before-attach", 10*time.Second) {
			t.Fatal("ViewerObserves: frame published before attach was not delivered — the deliver subject must be fed by a substrate-bound consumer over the session stream (snapshot-plus-tail), not a live-only subject tap")
		}

		frame2, _ := json.Marshal(map[string]any{
			"kind": "session.frame", "frame": "token", "origin": "wrapper",
			"sessionId": sid, "text": "after-attach",
		})
		if err := pub.Publish("tb.session."+sid+".ingest", frame2); err != nil {
			t.Fatal(err)
		}
		if !seen("after-attach", 10*time.Second) {
			t.Fatal("ViewerObserves: live frame after attach was not delivered on the deliver subject")
		}
	})

	// CrossSessionLeak: denied-neighbor over real NATS, plus the steering
	// single-writer guard: the viewer must not hold steer-publish authority.
	t.Run("CrossSessionLeak", func(t *testing.T) {
		t.Parallel()

		rt, err := start(t, operatorCfg(t, Loopback()))
		if err != nil {
			t.Fatal(err)
		}

		const sidA, sidB = "wss-xleak-A", "wss-xleak-B"
		viewerA, err := MintViewerCredential(rt, sidA, 10*time.Minute)
		if err != nil {
			t.Fatal(err)
		}
		viewerB, err := MintViewerCredential(rt, sidB, 10*time.Minute)
		if err != nil {
			t.Fatal(err)
		}

		nc, err := connectViewer(t, rt, viewerA.JWT)
		if err != nil {
			t.Fatalf("CrossSessionLeak: viewer A connect: %v", err)
		}
		defer nc.Close()

		asyncErr := make(chan error, 8)
		nc.SetErrorHandler(func(_ *nats.Conn, _ *nats.Subscription, e error) { asyncErr <- e })

		deny := func(action string, do func() error) {
			t.Helper()
			if err := do(); err != nil {
				t.Logf("CrossSessionLeak: sync denial for %s (acceptable): %v", action, err)
			}
			_ = nc.FlushTimeout(300 * time.Millisecond)
			deadline := time.Now().Add(time.Second)
			for time.Now().Before(deadline) {
				select {
				case e := <-asyncErr:
					if isPermissionDenial(e) {
						return
					}
				default:
					time.Sleep(20 * time.Millisecond)
				}
			}
			t.Fatalf("CrossSessionLeak: %s was not denied — viewer authority must be leaf-scoped to its own session's deliver subject and command acceptance only", action)
		}

		deny("subscribe on session B's deliver subject", func() error {
			_, err := nc.Subscribe(viewerB.DeliverSubject, func(*nats.Msg) {})
			return err
		})
		deny("subscribe on session B's out subject", func() error {
			_, err := nc.Subscribe("tb.session."+sidB+".out", func(*nats.Msg) {})
			return err
		})
		deny("publish on own session's steering subject", func() error {
			return nc.Publish("tb.session."+sidA+".steer", []byte(`{"kind":"session.steer_intent","intent":"steer","sessionId":"`+sidA+`","text":"hijack"}`))
		})
	})

	// StaleViewerCred: expiry, revocation, reconnect denial, renewal by
	// re-mint, and cookie revocation.
	t.Run("StaleViewerCred", func(t *testing.T) {
		t.Parallel()

		rt, err := start(t, operatorCfg(t, Loopback()))
		if err != nil {
			t.Fatal(err)
		}

		const sid = "wss-stale-001"

		// Expired: mint with a 1s TTL and poll until the boundary second has
		// passed server-side (JWT expiry truncates to whole seconds) — the
		// credential must be denied within the window, never indefinitely
		// accepted.
		expired, err := MintViewerCredential(rt, sid, time.Second)
		if err != nil {
			t.Fatal(err)
		}
		time.Sleep(1100 * time.Millisecond)
		expiryDenied := false
		for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); {
			nc, err := connectViewer(t, rt, expired.JWT)
			if err != nil {
				expiryDenied = true
				break
			}
			nc.Close()
			time.Sleep(250 * time.Millisecond)
		}
		if !expiryDenied {
			t.Fatal("StaleViewerCred: expired viewer credential was still accepted 6s after a 1s TTL — short-TTL expiry must deny connect")
		}

		// Revoked: live connection is disconnected, reconnect is denied.
		viewer, err := MintViewerCredential(rt, sid, 10*time.Minute)
		if err != nil {
			t.Fatal(err)
		}
		nc, err := connectViewer(t, rt, viewer.JWT)
		if err != nil {
			t.Fatalf("StaleViewerCred: connect: %v", err)
		}
		defer nc.Close()
		if err := rt.Revoke(AppAccount, viewer.UserPub); err != nil {
			t.Fatalf("StaleViewerCred: revoke: %v", err)
		}
		deadline := time.Now().Add(3 * time.Second)
		for nc.IsConnected() && time.Now().Before(deadline) {
			time.Sleep(20 * time.Millisecond)
		}
		if nc.IsConnected() {
			t.Fatal("StaleViewerCred: revoked viewer connection stayed live — revocation must disconnect")
		}
		if nc2, err := connectViewer(t, rt, viewer.JWT); err == nil {
			nc2.Close()
			t.Fatal("StaleViewerCred: revoked viewer credential was accepted on reconnect")
		}

		// Renewal is a re-mint through the cookie path: a valid cookie session
		// gates a fresh mint, and the fresh credential connects.
		tok, err := IssueSessionCookie(rt)
		if err != nil {
			t.Fatal(err)
		}
		if !ValidateCookieSession(rt, tok) {
			t.Fatal("StaleViewerCred: cookie session must validate for renewal")
		}
		renewed, err := MintViewerCredential(rt, sid, 10*time.Minute)
		if err != nil {
			t.Fatal(err)
		}
		nc3, err := connectViewer(t, rt, renewed.JWT)
		if err != nil {
			t.Fatalf("StaleViewerCred: renewal connect failed: %v", err)
		}
		nc3.Close()

		// A revoked cookie must no longer gate an upgrade or a renewal.
		tok2, err := IssueSessionCookie(rt)
		if err != nil {
			t.Fatal(err)
		}
		if err := RevokeCookieSession(rt, tok2); err != nil {
			t.Fatal(err)
		}
		if ValidateCookieSession(rt, tok2) {
			t.Fatal("StaleViewerCred: revoked cookie was not denied")
		}
	})

	// ViewerMintFailed: attributed typed failure on mint misuse.
	t.Run("ViewerMintFailed", func(t *testing.T) {
		t.Parallel()

		rt, err := start(t, valid(t))
		if err != nil {
			t.Fatal(err)
		}
		_, err = MintViewerCredential(rt, "wss-attr-001", 10*time.Minute)
		assertAdapter(t, err, ViewerMintFailed)

		rtOp, err := start(t, operatorCfg(t, Loopback()))
		if err != nil {
			t.Fatal(err)
		}
		_, err = MintViewerCredential(rtOp, "", 10*time.Minute)
		assertAdapter(t, err, ViewerMintFailed)
	})

	// GateMatrixVacuous: the scenario matrix must name this surface with all
	// seven pinned families citing committed Go tests.
	t.Run("GateMatrixVacuous", func(t *testing.T) {
		t.Parallel()

		raw, err := os.ReadFile("../scenario-matrix.json")
		if err != nil {
			t.Fatalf("GateMatrixVacuous: cannot read scenario-matrix.json: %v", err)
		}
		var matrix map[string]map[string][]string
		if err := json.Unmarshal(raw, &matrix); err != nil {
			t.Fatalf("GateMatrixVacuous: cannot parse scenario-matrix.json: %v", err)
		}
		surface, ok := matrix["web-session-surface"]
		if !ok {
			t.Fatal("GateMatrixVacuous: 'web-session-surface' is not present in scenario-matrix.json — the gate:scenarios check must not pass vacuously")
		}
		required := []string{"allowed", "denied-neighbor", "malformed", "duplicate", "stale", "revoked", "attributed-failure"}
		for _, fam := range required {
			citations := surface[fam]
			if len(citations) == 0 {
				t.Errorf("GateMatrixVacuous: family %q has no Go test citation", fam)
			}
			for _, cite := range citations {
				if !strings.HasPrefix(cite, "TestWebSession") {
					t.Errorf("GateMatrixVacuous: citation %q for family %q must be a TestWebSession* Go test name", cite, fam)
				}
			}
		}
	})

	// UngrownManifest: the release manifest must cover the session program and
	// must have retired the direct-browser-nats-websocket deferral.
	t.Run("UngrownManifest", func(t *testing.T) {
		t.Parallel()

		raw, err := os.ReadFile("../../../release/v1.json")
		if err != nil {
			t.Fatalf("UngrownManifest: cannot read release/v1.json: %v", err)
		}
		var manifest struct {
			Milestones []struct {
				Milestone string `json:"milestone"`
			} `json:"milestones"`
			DeferredScope []string `json:"deferredScope"`
		}
		if err := json.Unmarshal(raw, &manifest); err != nil {
			t.Fatalf("UngrownManifest: cannot parse release/v1.json: %v", err)
		}
		found := false
		for _, m := range manifest.Milestones {
			if m.Milestone == "web-session-surface" {
				found = true
				break
			}
		}
		if !found {
			t.Error("UngrownManifest: release/v1.json has no 'web-session-surface' milestone entry")
		}
		if slices.Contains(manifest.DeferredScope, "direct-browser-nats-websocket") {
			t.Error("UngrownManifest: release/v1.json still defers direct-browser-nats-websocket — the deferral is retired by this slice with condition citations")
		}
	})
}
