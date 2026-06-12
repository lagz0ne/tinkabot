package embednats

// TestTrustedWrapperAuthority is the outside-in proof for Slice 4
// (trusted-wrapper-authority). One owning subtest per failure family:
//   - OverbroadMint: MintUser must deny a session-subtree wildcard grant.
//   - SelfDeclaredTrust: a wrapper connecting with its own minted cred must be
//     denied when it publishes to the steering subject it does not own.
//   - SteerAfterRevoke: a steer accepted before revocation must be denied at
//     apply time after the wrapper credential is revoked.
//   - FakeStatusImpersonation: a wrapper connecting with its own minted cred and
//     publishing a status frame to the session ingest subject must be rejected
//     by the republisher (FrameMediator) — the wrapper's credential grants only
//     ingest-publish and steering-subscribe; the republisher enforces origin.
//
import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/lagz0ne/tinkabot/substrate/go/core"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

func TestTrustedWrapperAuthority(t *testing.T) {
	t.Parallel()

	t.Run("OverbroadMint", func(t *testing.T) {
		t.Parallel()

		rt, err := start(t, operatorCfg(t, Loopback()))
		if err != nil {
			t.Fatal(err)
		}

		// A mint request carrying tb.session.> (session-subtree wildcard) must be
		// denied by the subject-breadth check. Existing _INBOX.>, $KV.*.>, and
		// JS-API wildcards are allowed; only session-subtree wildcards are denied.
		sessionID := "twa-overbroad-001"
		auth := principal("twa-overbroad-wrapper-"+sessionID, core.Permissions{
			Publish:   core.PermList{Allow: []string{"tb.session.>"}}, // session-subtree wildcard
			Subscribe: core.PermList{Allow: []string{"_INBOX.>", "tb.session." + sessionID + ".steer"}},
		})

		_, err = rt.MintUser(AppAccount, auth, time.Hour)
		if err == nil {
			t.Fatal("OverbroadMint: MintUser accepted a session-subtree wildcard grant — subject-breadth check does not exist yet; implement it in MintUser to deny tb.session.-prefixed wildcard patterns")
		}
		// The denial must be typed OverbroadMint.
		assertAdapter(t, err, OverbroadMint)
	})

	// OverbroadMintInfix: an infix wildcard such as tb.session.*.ingest grants
	// publish authority across every session's ingest subject.  The breadth check
	// must catch mid-path wildcards, not only terminal ones.
	t.Run("OverbroadMintInfix", func(t *testing.T) {
		t.Parallel()

		rt, err := start(t, operatorCfg(t, Loopback()))
		if err != nil {
			t.Fatal(err)
		}

		auth := principal("twa-overbroad-infix", core.Permissions{
			Publish:   core.PermList{Allow: []string{"tb.session.*.ingest"}}, // infix wildcard — spans all sessions
			Subscribe: core.PermList{Allow: []string{"_INBOX.>"}},
		})

		_, err = rt.MintUser(AppAccount, auth, time.Hour)
		if err == nil {
			t.Fatal("OverbroadMintInfix: MintUser accepted tb.session.*.ingest — infix wildcard grants cross-session ambient authority")
		}
		assertAdapter(t, err, OverbroadMint)
	})

	t.Run("SelfDeclaredTrust", func(t *testing.T) {
		t.Parallel()

		rt, err := start(t, operatorCfg(t, Loopback()))
		if err != nil {
			t.Fatal(err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		sessionID := "twa-self-declared-001"
		ingest := "tb.session." + sessionID + ".ingest"
		steer := "tb.session." + sessionID + ".steer"

		// MintTrustedWrapper issues a leaf-scoped credential for a trusted wrapper:
		// publish only on ingest, subscribe only on steer.
		wrapperCreds, err := MintTrustedWrapper(rt, sessionID)
		if err != nil {
			t.Fatalf("SelfDeclaredTrust: MintTrustedWrapper: %v", err)
		}

		nc, err := rt.ConnectCreds(ctx, wrapperCreds.File)
		if err != nil {
			t.Fatalf("SelfDeclaredTrust: connect wrapper: %v", err)
		}
		defer nc.Close()

		// asyncErr collects NATS async permission errors from the wrapper connection.
		asyncErr := make(chan error, 4)
		nc.SetErrorHandler(func(_ *nats.Conn, _ *nats.Subscription, e error) {
			asyncErr <- e
		})

		// The wrapper's credential must allow publishing to its ingest subject.
		if err := nc.Publish(ingest, []byte(`{"kind":"session.frame","frame":"token","origin":"wrapper","sessionId":"`+sessionID+`","text":"ok"}`)); err != nil {
			t.Fatalf("SelfDeclaredTrust: wrapper publish to ingest denied — credential must allow ingest publish: %v", err)
		}

		// The wrapper's credential must NOT allow publishing to the steering subject.
		// A publish to steer must result in a NATS permission denial (async error).
		if err := nc.Publish(steer, []byte(`{"kind":"steer","text":"inject"}`)); err != nil {
			t.Logf("SelfDeclaredTrust: sync publish denial (acceptable): %v", err)
		}
		if err := nc.FlushTimeout(300 * time.Millisecond); err != nil {
			t.Logf("SelfDeclaredTrust: flush: %v", err)
		}

		denied := false
		deadline := time.Now().Add(500 * time.Millisecond)
		for time.Now().Before(deadline) {
			select {
			case e := <-asyncErr:
				if isPermissionDenial(e) {
					denied = true
				}
			default:
				time.Sleep(20 * time.Millisecond)
			}
		}
		// The denial is enforced at the NATS server level via exact-subject grants;
		// no synchronous typed error is returned to the publisher.
		if !denied {
			t.Fatal("SelfDeclaredTrust: wrapper credential was not denied publish on the steering subject — MintTrustedWrapper must not grant steer publish authority")
		}
	})

	t.Run("SteerAfterRevoke", func(t *testing.T) {
		t.Parallel()

		rt, err := start(t, operatorCfg(t, Loopback()))
		if err != nil {
			t.Fatal(err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		sessionID := "twa-steer-revoke-001"

		wrapperCreds, err := MintTrustedWrapper(rt, sessionID)
		if err != nil {
			t.Fatalf("SteerAfterRevoke: MintTrustedWrapper: %v", err)
		}

		// Connect the wrapper.
		wrapperNC, err := rt.ConnectCreds(ctx, wrapperCreds.File)
		if err != nil {
			t.Fatalf("SteerAfterRevoke: connect wrapper: %v", err)
		}
		defer wrapperNC.Close()

		// Revoke the wrapper credential.
		if err := rt.Revoke(AppAccount, wrapperCreds.UserPub); err != nil {
			t.Fatalf("SteerAfterRevoke: revoke: %v", err)
		}
		// Wait for the server to enforce revocation.
		revokeDeadline := time.Now().Add(3 * time.Second)
		for wrapperNC.IsConnected() && time.Now().Before(revokeDeadline) {
			time.Sleep(20 * time.Millisecond)
		}

		// ApplySteerAfterRevoke re-checks the steerer's revocation status at
		// apply time. After revocation it must return a typed SteerAfterRevoke error.
		err = ApplySteerAfterRevoke(rt, wrapperCreds.UserPub)
		assertAdapter(t, err, SteerAfterRevoke)
	})

	// FakeStatusImpersonation: a trusted wrapper connecting with its own minted
	// leaf-scoped credential publishes a status frame (origin=wrapper,
	// frame=status) to the session ingest subject. The FrameMediator (Slice 3's
	// republisher) must reject it — status frames must originate from the runner.
	// This is the live credential proof: the wrapper connects with a real minted
	// credential, not a test-internal bypass.
	t.Run("FakeStatusImpersonation", func(t *testing.T) {
		t.Parallel()

		rt, err := start(t, operatorCfg(t, Loopback()))
		if err != nil {
			t.Fatal(err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		sessionID := "twa-fake-status-001"

		// Start the FrameMediator to prove that a wrapper-emitted status frame
		// arriving on ingest via a real minted credential is rejected.
		mediator, err := StartFrameMediator(ctx, rt, FrameMediatorConfig{
			SessionID:     sessionID,
			QuotaMaxBytes: 1 << 20,
		})
		if err != nil {
			t.Fatalf("FakeStatusImpersonation: StartFrameMediator: %v", err)
		}
		defer mediator.Stop()

		wrapperCreds, err := MintTrustedWrapper(rt, sessionID)
		if err != nil {
			t.Fatalf("FakeStatusImpersonation: MintTrustedWrapper: %v", err)
		}

		// Connect the wrapper using its own minted leaf-scoped credential.
		wrapperNC, err := rt.ConnectCreds(ctx, wrapperCreds.File)
		if err != nil {
			t.Fatalf("FakeStatusImpersonation: connect wrapper: %v", err)
		}
		defer wrapperNC.Close()

		// The wrapper publishes a fake status frame to its ingest subject.
		// Frame contract: status frames must have origin=runner; this has origin=wrapper.
		fakeStatus, _ := json.Marshal(map[string]any{
			"kind":      "session.frame",
			"frame":     "status",
			"origin":    "wrapper", // forged — only runner may emit status frames
			"sessionId": sessionID,
			"state":     "completed",
			"detail":    "fake-impersonated-status",
		})

		ingest := "tb.session." + sessionID + ".ingest"
		if err := wrapperNC.Publish(ingest, fakeStatus); err != nil {
			t.Fatalf("FakeStatusImpersonation: wrapper publish to ingest: %v", err)
		}
		if err := wrapperNC.FlushTimeout(300 * time.Millisecond); err != nil {
			t.Logf("FakeStatusImpersonation: flush: %v", err)
		}

		// Also publish a valid token frame so the output stream is not empty.
		validToken, _ := json.Marshal(map[string]any{
			"kind":      "session.frame",
			"frame":     "token",
			"origin":    "wrapper",
			"sessionId": sessionID,
			"text":      "valid-token",
		})
		if err := wrapperNC.Publish(ingest, validToken); err != nil {
			t.Fatalf("FakeStatusImpersonation: wrapper publish valid token: %v", err)
		}
		if err := wrapperNC.FlushTimeout(300 * time.Millisecond); err != nil {
			t.Logf("FakeStatusImpersonation: flush valid: %v", err)
		}

		// Wait briefly for the FrameMediator to process ingest messages.
		time.Sleep(300 * time.Millisecond)

		// Drain the output stream: the fake status frame must not appear.
		outStream := "tb-session-out-" + sessionID
		outJS := mediator.OutputJS()
		cons, err := outJS.OrderedConsumer(ctx, outStream, jetstream.OrderedConsumerConfig{})
		if err != nil {
			t.Fatalf("FakeStatusImpersonation: ordered consumer: %v", err)
		}

		msgs, err := drainConsumer(ctx, cons, 500*time.Millisecond)
		if err != nil {
			t.Fatalf("FakeStatusImpersonation: drain: %v", err)
		}

		for _, m := range msgs {
			var f map[string]any
			if err := json.Unmarshal(m, &f); err != nil {
				continue
			}
			if f["frame"] == "status" && f["origin"] == "wrapper" {
				t.Fatal("FakeStatusImpersonation: wrapper-emitted status frame reached the output stream via a real minted credential — the FrameMediator must reject it regardless of the credential the publisher holds; implement MintTrustedWrapper so this path is proven with a real credential")
			}
		}
	})

	// denied-neighbor: a wrapper credential minted for session A must be denied
	// publish and subscribe on session B's subjects — enforced by NATS over real
	// embedded server (exact-subject permission grant).
	t.Run("denied-neighbor", func(t *testing.T) {
		t.Parallel()

		rt, err := start(t, operatorCfg(t, Loopback()))
		if err != nil {
			t.Fatal(err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		sessionA := "twa-neighbor-A"
		sessionB := "twa-neighbor-B"

		// Mint a credential for session A.
		credsA, err := MintTrustedWrapper(rt, sessionA)
		if err != nil {
			t.Fatalf("denied-neighbor: MintTrustedWrapper sessionA: %v", err)
		}

		nc, err := rt.ConnectCreds(ctx, credsA.File)
		if err != nil {
			t.Fatalf("denied-neighbor: connect: %v", err)
		}
		defer nc.Close()

		asyncErr := make(chan error, 4)
		nc.SetErrorHandler(func(_ *nats.Conn, _ *nats.Subscription, e error) {
			asyncErr <- e
		})

		// Attempt to publish to session B's ingest — must be denied.
		ingestB := "tb.session." + sessionB + ".ingest"
		_ = nc.Publish(ingestB, []byte(`{"kind":"session.frame","frame":"token","origin":"wrapper","sessionId":"`+sessionB+`","text":"neighbor"}`))
		if err := nc.FlushTimeout(300 * time.Millisecond); err != nil {
			t.Logf("denied-neighbor: flush: %v", err)
		}

		denied := false
		deadline := time.Now().Add(500 * time.Millisecond)
		for time.Now().Before(deadline) {
			select {
			case e := <-asyncErr:
				if isPermissionDenial(e) {
					denied = true
				}
			default:
				time.Sleep(20 * time.Millisecond)
			}
		}
		if !denied {
			t.Fatal("denied-neighbor: session-A wrapper credential was not denied publish on session-B's ingest subject — MintTrustedWrapper must grant only exact ingest subject for its own session")
		}
	})

	// malformed: a malformed (non-JSON) ingest frame published by a valid wrapper
	// credential must be silently dropped by the FrameMediator — validFrame rejects
	// non-JSON frames and they must not appear on the output stream.
	t.Run("malformed", func(t *testing.T) {
		t.Parallel()

		rt, err := start(t, operatorCfg(t, Loopback()))
		if err != nil {
			t.Fatal(err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		sessionID := "twa-malformed-001"

		mediator, err := StartFrameMediator(ctx, rt, FrameMediatorConfig{
			SessionID:     sessionID,
			QuotaMaxBytes: 1 << 20,
		})
		if err != nil {
			t.Fatalf("malformed: StartFrameMediator: %v", err)
		}
		defer mediator.Stop()

		wrapperCreds, err := MintTrustedWrapper(rt, sessionID)
		if err != nil {
			t.Fatalf("malformed: MintTrustedWrapper: %v", err)
		}
		wrapperNC, err := rt.ConnectCreds(ctx, wrapperCreds.File)
		if err != nil {
			t.Fatalf("malformed: connect wrapper: %v", err)
		}
		defer wrapperNC.Close()

		ingest := "tb.session." + sessionID + ".ingest"
		// Publish a malformed (non-JSON) frame — must be dropped.
		if err := wrapperNC.Publish(ingest, []byte("not-json")); err != nil {
			t.Fatalf("malformed: publish: %v", err)
		}
		// Publish a valid token frame so the stream is non-empty if anything got through.
		valid, _ := json.Marshal(map[string]any{
			"kind": "session.frame", "frame": "token", "origin": "wrapper",
			"sessionId": sessionID, "text": "valid",
		})
		if err := wrapperNC.Publish(ingest, valid); err != nil {
			t.Fatalf("malformed: publish valid: %v", err)
		}
		if err := wrapperNC.FlushTimeout(300 * time.Millisecond); err != nil {
			t.Logf("malformed: flush: %v", err)
		}
		time.Sleep(300 * time.Millisecond)

		outStream := "tb-session-out-" + sessionID
		cons, err := mediator.OutputJS().OrderedConsumer(ctx, outStream, jetstream.OrderedConsumerConfig{})
		if err != nil {
			t.Fatalf("malformed: ordered consumer: %v", err)
		}
		msgs, err := drainConsumer(ctx, cons, 500*time.Millisecond)
		if err != nil {
			t.Fatalf("malformed: drain: %v", err)
		}
		// Only the valid token frame should appear; non-JSON is dropped.
		for _, m := range msgs {
			if string(m) == "not-json" {
				t.Fatal("malformed: non-JSON frame was not dropped by the FrameMediator")
			}
		}
		if len(msgs) == 0 {
			t.Fatal("malformed: valid token frame did not reach the output stream")
		}
	})

	// duplicate: two MintTrustedWrapper calls for the same session produce two
	// independent valid credentials (each is a distinct user key). Neither is
	// denied at mint time; the duplicate-protection contract for this surface is
	// credential-per-session isolation, not mint-deduplication.
	// N/A for "duplicate denial": MintTrustedWrapper does not deduplicate mints by
	// design — each call issues a fresh credential. Duplicate-credential denial is
	// enforced at the NATS layer through exact-subject grants; two credentials for
	// the same session do not create a privilege escalation. The duplicate family
	// for this surface is therefore proven by the absence of any extra authority:
	// a second wrapper credential for the same session has the same (not additive)
	// subject grants as the first.
	t.Run("duplicate", func(t *testing.T) {
		t.Parallel()

		rt, err := start(t, operatorCfg(t, Loopback()))
		if err != nil {
			t.Fatal(err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		sessionID := "twa-duplicate-001"

		creds1, err := MintTrustedWrapper(rt, sessionID)
		if err != nil {
			t.Fatalf("duplicate: first mint: %v", err)
		}
		creds2, err := MintTrustedWrapper(rt, sessionID)
		if err != nil {
			t.Fatalf("duplicate: second mint: %v", err)
		}

		// Both credentials must be distinct public keys.
		if creds1.UserPub == creds2.UserPub {
			t.Fatal("duplicate: two MintTrustedWrapper calls produced identical public keys — each call must generate a fresh keypair")
		}

		// Both credentials must connect and publish to ingest without denial.
		nc1, err := rt.ConnectCreds(ctx, creds1.File)
		if err != nil {
			t.Fatalf("duplicate: connect cred1: %v", err)
		}
		defer nc1.Close()
		nc2, err := rt.ConnectCreds(ctx, creds2.File)
		if err != nil {
			t.Fatalf("duplicate: connect cred2: %v", err)
		}
		defer nc2.Close()

		ingest := "tb.session." + sessionID + ".ingest"
		if err := nc1.Publish(ingest, []byte(`{"kind":"session.frame","frame":"token","origin":"wrapper","sessionId":"`+sessionID+`","text":"c1"}`)); err != nil {
			t.Fatalf("duplicate: cred1 publish: %v", err)
		}
		if err := nc2.Publish(ingest, []byte(`{"kind":"session.frame","frame":"token","origin":"wrapper","sessionId":"`+sessionID+`","text":"c2"}`)); err != nil {
			t.Fatalf("duplicate: cred2 publish: %v", err)
		}

		// Neither credential must be able to publish to the other session's subjects.
		asyncErr1 := make(chan error, 4)
		nc1.SetErrorHandler(func(_ *nats.Conn, _ *nats.Subscription, e error) { asyncErr1 <- e })
		_ = nc1.Publish("tb.session.other-session.ingest", []byte("spill"))
		if err := nc1.FlushTimeout(300 * time.Millisecond); err != nil {
			t.Logf("duplicate: flush: %v", err)
		}
		deadline := time.Now().Add(400 * time.Millisecond)
		denied := false
		for time.Now().Before(deadline) {
			select {
			case e := <-asyncErr1:
				if isPermissionDenial(e) {
					denied = true
				}
			default:
				time.Sleep(20 * time.Millisecond)
			}
		}
		if !denied {
			t.Fatal("duplicate: cred1 was not denied publish on a different session's ingest — duplicate credentials must not create additive authority")
		}
	})

	// stale: a wrapper credential revoked before its natural TTL expiry is denied
	// at apply time (SteerAfterRevoke) — the revocation-check at apply time treats
	// any revocation timestamp as stale, regardless of the original TTL.
	// This sub-test is the TTL-independent variant of SteerAfterRevoke: it confirms
	// that a recently minted (fresh, non-expired) credential that has been revoked
	// is treated as stale by ApplySteerAfterRevoke.
	t.Run("stale", func(t *testing.T) {
		t.Parallel()

		rt, err := start(t, operatorCfg(t, Loopback()))
		if err != nil {
			t.Fatal(err)
		}

		sessionID := "twa-stale-001"

		creds, err := MintTrustedWrapper(rt, sessionID)
		if err != nil {
			t.Fatalf("stale: MintTrustedWrapper: %v", err)
		}

		// Revoke the credential immediately after mint (well within its TTL).
		if err := rt.Revoke(AppAccount, creds.UserPub); err != nil {
			t.Fatalf("stale: revoke: %v", err)
		}

		// ApplySteerAfterRevoke must deny a revoked (stale) credential at apply time.
		err = ApplySteerAfterRevoke(rt, creds.UserPub)
		assertAdapter(t, err, SteerAfterRevoke)
	})

	// Loop suppression — N/A for this surface.
	// Loop suppression is a core activation-chain hop-limit concept: it applies to
	// the agent activation graph where a chain of hops can recurse. The
	// trusted-wrapper-authority surface is credential-minting and steer-apply only;
	// there is no activation chain and therefore no loop to suppress. This failure
	// family does not apply here and requires no test case or N/A sub-test.

	// attributed-failure: a permission denial on the trusted-wrapper surface carries
	// attribution traceable to the wrapper's lease fields. OverbroadMint is the
	// typed error returned by MintUser; its typed Kind and the error adapter carry
	// the subject that triggered the denial so failures can be attributed to the
	// minting party.
	t.Run("attributed-failure", func(t *testing.T) {
		t.Parallel()

		rt, err := start(t, operatorCfg(t, Loopback()))
		if err != nil {
			t.Fatal(err)
		}

		sessionID := "twa-attributed-001"
		auth := principal("twa-attr-wrapper-"+sessionID, core.Permissions{
			Publish:   core.PermList{Allow: []string{"tb.session." + sessionID + ".*"}}, // session-subtree wildcard
			Subscribe: core.PermList{Allow: []string{"_INBOX.>"}},
		})

		_, err = rt.MintUser(AppAccount, auth, time.Hour)
		if err == nil {
			t.Fatal("attributed-failure: MintUser accepted a session-subtree wildcard — OverbroadMint denial required")
		}
		// The typed Kind must be OverbroadMint — attribution is the Kind field.
		assertAdapter(t, err, OverbroadMint)
		// The Details map must carry the offending subject for attribution.
		var adapterErr *Error
		if errors.As(err, &adapterErr) {
			if adapterErr.Details["subject"] == "" {
				t.Fatalf("attributed-failure: OverbroadMint error carries no subject in Details — denial must be attributable to the triggering subject; got details: %v", adapterErr.Details)
			}
		} else {
			t.Fatalf("attributed-failure: error is not an *Error — cannot check attribution details: %T %v", err, err)
		}
	})
}

// isPermissionDenial returns true if the error text indicates a NATS permission
// denial (Permissions Violation, not authorized, or denied).
func isPermissionDenial(e error) bool {
	if e == nil {
		return false
	}
	s := e.Error()
	for _, sub := range []string{"Permissions Violation", "not authorized", "permissions violation", "denied"} {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
