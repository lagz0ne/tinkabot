package embednats

// TestSteeringAcceptance is the outside-in proof for Slice 5 (steering-acceptance).
// Sub-tests cover all seven scenario-matrix families for the steering surface:
//
//   allowed:            SteerOutOfOrder (two steers accepted with server-assigned cursors)
//   duplicate:          DuplicateReplay (same dedup key accepted then returns Duplicate)
//   denied-neighbor:    DeniedNeighbor  (principal B cannot steer session A's subject)
//   malformed:          Malformed       (steer missing required identity field rejected)
//   stale:              Stale           (steer with stale stream cursor returns StaleCursor)
//   revoked:            Revoked         (steer from revoked lease denied at apply time)
//   attributed-failure: DeniedNeighbor  (denied error carries sourceId attribution)
//   stop-ordering:      StopOrderedAgainstPending (Stop drains pending results, no loss)

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/lagz0ne/tinkabot/substrate/go/core"
	"github.com/nats-io/nats.go"
)

func TestSteeringAcceptance(t *testing.T) {
	t.Parallel()

	t.Run("SteerDropped", func(t *testing.T) {
		t.Parallel()

		act, router, nc, _ := routerHarness(t, "fixtures/valid/activation-source-subject.json")

		const bufferSize = 16
		route, out, err := router.Subject(nc, act)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { mustStop(t, route) })
		flush(t, nc)

		// Publish bufferSize+1 steers. The first bufferSize fill the channel;
		// the 17th (overflow) must also arrive because send() no longer drops.
		for i := range bufferSize {
			msg := nats.NewMsg(act.Source.Pattern)
			msg.Header.Set(HeaderMessageID, fmt.Sprintf("steer-fill-%03d", i))
			if err := nc.PublishMsg(msg); err != nil {
				t.Fatal(err)
			}
		}
		flush(t, nc)
		time.Sleep(200 * time.Millisecond)

		overflow := nats.NewMsg(act.Source.Pattern)
		overflow.Header.Set(HeaderMessageID, "steer-overflow-001")
		if err := nc.PublishMsg(overflow); err != nil {
			t.Fatal(err)
		}
		flush(t, nc)
		time.Sleep(100 * time.Millisecond)

		// Drain all results.
		seen := map[string]bool{}
		deadline := time.Now().Add(600 * time.Millisecond)
		for time.Now().Before(deadline) {
			select {
			case res := <-out:
				if res.Err == nil {
					seen[res.Record.SourceCursor] = true
				}
			default:
				time.Sleep(20 * time.Millisecond)
			}
		}

		// All bufferSize+1 steers must be delivered — no steer silently dropped.
		if len(seen) != bufferSize+1 {
			t.Fatalf(
				"SteerDropped: expected %d steers delivered, got %d — "+
					"send() still drops on full channel; "+
					"fix send() to use a goroutine so all steers reach the consumer",
				bufferSize+1, len(seen),
			)
		}
		// Verify cursors are server-assigned numeric strings (stream sequences).
		for cursor := range seen {
			if _, parseErr := strconv.ParseUint(cursor, 10, 64); parseErr != nil {
				t.Fatalf(
					"SteerDropped: cursor %q is not a server-assigned sequence — "+
						"Subject() must route through JetStream so cursors are numeric",
					cursor,
				)
			}
		}
	})

	t.Run("SteerOutOfOrder", func(t *testing.T) {
		t.Parallel()

		act, router, nc, _ := routerHarness(t, "fixtures/valid/activation-source-subject.json")

		route, out, err := router.Subject(nc, act)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { mustStop(t, route) })
		flush(t, nc)

		// Publish in "wrong" client-label order. Server assigns sequences 1, 2.
		msg1 := nats.NewMsg(act.Source.Pattern)
		msg1.Header.Set(HeaderMessageID, "steer-seq-second") // client says "second"
		if err := nc.PublishMsg(msg1); err != nil {
			t.Fatal(err)
		}
		msg2 := nats.NewMsg(act.Source.Pattern)
		msg2.Header.Set(HeaderMessageID, "steer-seq-first") // client says "first"
		if err := nc.PublishMsg(msg2); err != nil {
			t.Fatal(err)
		}
		flush(t, nc)

		r1 := waitResult(t, out)
		r2 := waitResult(t, out)

		cursor1 := r1.Record.SourceCursor
		cursor2 := r2.Record.SourceCursor

		// Cursors must be server-assigned numeric strings (stream sequences),
		// not the client-chosen "steer-seq-*" MessageID values.
		seq1, err1 := strconv.ParseUint(cursor1, 10, 64)
		seq2, err2 := strconv.ParseUint(cursor2, 10, 64)
		if err1 != nil || err2 != nil {
			t.Fatalf(
				"SteerOutOfOrder: cursors %q and %q are not server-assigned sequences — "+
					"Subject() must route through JetStream",
				cursor1, cursor2,
			)
		}
		// Server sequence is monotonically increasing: first arrival < second arrival.
		if seq1 >= seq2 {
			t.Fatalf(
				"SteerOutOfOrder: server sequence is not monotone: seq1=%d seq2=%d "+
					"(cursors: %q, %q)",
				seq1, seq2, cursor1, cursor2,
			)
		}
	})

	t.Run("LedgerScanUnbounded", func(t *testing.T) {
		t.Parallel()

		const historyDepth = 30
		bucket := "tb_steer_scan_" + strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())

		ledger, store := embeddedLedger(t, bucket)

		baseAct := activation(t, read(t, "fixtures/valid/activation-source-subject.json"))
		lease := core.Lease{ID: baseAct.SourceLease.LeaseID, Status: "active"}

		for i := range historyDepth {
			act := activation(t, edit(t, "fixtures/valid/activation-source-subject.json", func(doc map[string]any) {
				srcID := fmt.Sprintf("src-scan-%03d", i)
				doc["sourcePrincipal"].(map[string]any)["sourceId"] = srcID
				doc["source"].(map[string]any)["pattern"] = fmt.Sprintf("tb.scan.subject.%03d.>", i)
				doc["source"].(map[string]any)["observedSubject"] = fmt.Sprintf("tb.scan.subject.%03d.hit", i)
				doc["source"].(map[string]any)["messageId"] = fmt.Sprintf("msg-scan-%03d", i)
				doc["dedupeKey"] = fmt.Sprintf("subject:scripts.proof.observe:tb_proof_events:tb.scan.subject.%03d.>:msg-scan-%03d", i, i)
				doc["activationId"] = fmt.Sprintf("act:scripts.proof.observe:tb_proof_events:%03d", i)
			}))
			if _, err := ledger.Accept(act, lease); err != nil {
				t.Fatalf("fill accept[%d]: %v", i, err)
			}
		}

		lastSourceID := fmt.Sprintf("src-scan-%03d", historyDepth-1)
		indexKey := "s." + keyEnc(lastSourceID)

		// The index key must exist — SaveAccepted now writes it.
		if _, err := store.kv.Get(indexKey); err != nil {
			t.Fatalf(
				"LedgerScanUnbounded: per-source index key %q not found in KV bucket after %d accepts — "+
					"SaveAccepted must write the 's.<sourceID>' index key",
				indexKey, historyDepth,
			)
		}

		// Source() must return the correct record via the direct O(1) keyed read.
		rec, ok, err := store.Source(lastSourceID)
		if err != nil {
			t.Fatalf("LedgerScanUnbounded: Source(%q): unexpected error: %v", lastSourceID, err)
		}
		if !ok {
			t.Fatalf("LedgerScanUnbounded: Source(%q): record not found after %d accepts", lastSourceID, historyDepth)
		}
		if rec.SourceID != lastSourceID {
			t.Fatalf("LedgerScanUnbounded: Source(%q): got SourceID=%q", lastSourceID, rec.SourceID)
		}
	})

	t.Run("NonIdempotentReplay", func(t *testing.T) {
		t.Parallel()

		act, router, nc, _ := routerHarness(t, "fixtures/valid/activation-source-subject.json")

		route, out, err := router.Subject(nc, act)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { mustStop(t, route) })
		flush(t, nc)

		const steerContent = `{"kind":"steer","text":"hello from replay attack"}`

		msg1 := nats.NewMsg(act.Source.Pattern)
		msg1.Header.Set(HeaderMessageID, "steer-replay-v1")
		msg1.Data = []byte(steerContent)
		if err := nc.PublishMsg(msg1); err != nil {
			t.Fatal(err)
		}
		msg2 := nats.NewMsg(act.Source.Pattern)
		msg2.Header.Set(HeaderMessageID, "steer-replay-v2") // same content, different header
		msg2.Data = []byte(steerContent)
		if err := nc.PublishMsg(msg2); err != nil {
			t.Fatal(err)
		}
		flush(t, nc)

		r1 := waitResult(t, out)
		r2 := waitResult(t, out)

		if r1.Record.Status != core.Accepted || r2.Record.Status != core.Accepted {
			t.Fatalf(
				"NonIdempotentReplay: expected both steers Accepted (each has a unique "+
					"server-assigned sequence); got r1.Status=%s r2.Status=%s",
				r1.Record.Status, r2.Record.Status,
			)
		}

		for i, r := range []RouterResult{r1, r2} {
			if _, parseErr := strconv.ParseUint(r.Record.SourceCursor, 10, 64); parseErr != nil {
				t.Fatalf(
					"NonIdempotentReplay: r%d.SourceCursor=%q is not a server-assigned sequence — "+
						"dedup key must be bound to a server-assigned component",
					i+1, r.Record.SourceCursor,
				)
			}
		}

		if r1.Record.SourceCursor == r2.Record.SourceCursor {
			t.Fatalf(
				"NonIdempotentReplay: both steers have the same cursor %q — "+
					"server-assigned sequences must be unique per publish",
				r1.Record.SourceCursor,
			)
		}
	})

	// DuplicateReplay: the same activation submitted twice over the steering path
	// returns core.Duplicate on the second call.
	t.Run("DuplicateReplay", func(t *testing.T) {
		t.Parallel()

		act, router, _, _ := routerHarness(t, "fixtures/valid/activation-source-subject.json")

		msg := nats.NewMsg(act.Source.Pattern)
		msg.Header.Set(HeaderMessageID, "steer-dup-replay-001")

		first, err := router.AcceptSubject(act, msg)
		if err != nil {
			t.Fatal(err)
		}
		if first.Status != core.Accepted {
			t.Fatalf("DuplicateReplay: first accept status=%s, want Accepted", first.Status)
		}
		dup, err := router.AcceptSubject(act, msg)
		if err != nil {
			t.Fatal(err)
		}
		if dup.Status != core.Duplicate {
			t.Fatalf("DuplicateReplay: second accept status=%s, want Duplicate — ledger dedup must apply", dup.Status)
		}
	})

	// StopOrderedAgainstPending: calling route.Stop() while results are buffered
	// does not discard them. Results already written to the output channel before
	// Stop() returns are still retrievable by the consumer.
	t.Run("StopOrderedAgainstPending", func(t *testing.T) {
		t.Parallel()

		act, router, nc, _ := routerHarness(t, "fixtures/valid/activation-source-subject.json")

		route, out, err := router.Subject(nc, act)
		if err != nil {
			t.Fatal(err)
		}
		flush(t, nc)

		const n = 4
		for i := range n {
			msg := nats.NewMsg(act.Source.Pattern)
			msg.Header.Set(HeaderMessageID, fmt.Sprintf("steer-pending-%03d", i))
			if err := nc.PublishMsg(msg); err != nil {
				t.Fatal(err)
			}
		}
		flush(t, nc)
		// Allow the subscription goroutine to write results into the buffered channel.
		time.Sleep(300 * time.Millisecond)

		// Stop before draining. Results already in the buffered channel must survive.
		if err := route.Stop(); err != nil {
			t.Fatal(err)
		}

		seen := 0
		deadline := time.Now().Add(500 * time.Millisecond)
		for time.Now().Before(deadline) {
			select {
			case res, ok := <-out:
				if !ok {
					goto done
				}
				if res.Err == nil {
					seen++
				}
			default:
				time.Sleep(10 * time.Millisecond)
			}
		}
	done:
		if seen != n {
			t.Fatalf("StopOrderedAgainstPending: expected %d results after Stop(), got %d — pending steers were lost", n, seen)
		}
	})

	// DeniedNeighbor: a steer from a principal whose subscribe aperture does not
	// cover the source subject is denied with core.DeniedNeighbor, and the error
	// carries sourceId attribution (attributed-failure proof).
	t.Run("DeniedNeighbor", func(t *testing.T) {
		t.Parallel()

		act := activation(t, read(t, "fixtures/valid/activation-source-subject.json"))
		auth := routerAuth(act, "tb_steer_denied")
		// Deny the source pattern so AuthorizeSource returns DeniedNeighbor.
		auth.Permissions.Subscribe.Deny = append(auth.Permissions.Subscribe.Deny, act.Source.Pattern)
		router, err := NewSourceRouter(auth, core.NewDurableLedger(core.NewMemoryLedgerStore()))
		if err != nil {
			t.Fatal(err)
		}

		msg := nats.NewMsg(act.Source.Pattern)
		msg.Header.Set(HeaderMessageID, "steer-denied-neighbor-001")
		_, gotErr := router.AcceptSubject(act, msg)
		assertCore(t, gotErr, core.DeniedNeighbor)

		// attributed-failure: the error must carry sourceId attribution.
		var coreErr *core.Error
		if !errors.As(gotErr, &coreErr) || coreErr.Details["sourceId"] != act.SourcePrincipal.SourceID {
			t.Fatalf("DeniedNeighbor: attribution missing — want sourceId=%q in Details, got %#v", act.SourcePrincipal.SourceID, coreErr)
		}
	})

	// Revoked: a steer whose activation carries a revoked lease is denied at apply
	// time with core.LeaseRevoked.
	t.Run("Revoked", func(t *testing.T) {
		t.Parallel()

		act := activation(t, edit(t, "fixtures/valid/activation-source-subject.json", func(doc map[string]any) {
			doc["sourceLease"].(map[string]any)["leaseStatus"] = "revoked"
		}))
		auth := routerAuth(act, "tb_steer_revoked")
		auth.Capability.LeaseStatus = "active"
		router, err := NewSourceRouter(auth, core.NewDurableLedger(core.NewMemoryLedgerStore()))
		if err != nil {
			t.Fatal(err)
		}

		msg := nats.NewMsg(act.Source.Pattern)
		msg.Header.Set(HeaderMessageID, "steer-revoked-001")
		_, gotErr := router.AcceptSubject(act, msg)
		assertCore(t, gotErr, core.LeaseRevoked)
	})

	// Malformed: a steer with a missing required identity field is rejected with
	// SourceMalformed before reaching the ledger.
	t.Run("Malformed", func(t *testing.T) {
		t.Parallel()

		act, router, _, _ := routerHarness(t, "fixtures/valid/activation-source-subject.json")

		// A plain-NATS message with no MessageID header and no JetStream metadata
		// fails the malformed() check (subject kind requires MessageID for plain path).
		msg := nats.NewMsg(act.Source.Pattern)
		// Deliberately omit HeaderMessageID — no Metadata(), StreamSequence stays 0.
		_, gotErr := router.AcceptSubject(act, msg)
		assertRouter(t, gotErr, SourceMalformed)
	})

	// Stale: a steer replaying a cursor older than the accepted high-water mark
	// returns core.StaleCursor from the ledger.
	t.Run("Stale", func(t *testing.T) {
		t.Parallel()

		act, router, nc, js := routerHarness(t, "fixtures/valid/activation-source-stream.json")
		if _, err := js.AddStream(&nats.StreamConfig{Name: act.Source.Stream, Subjects: []string{act.Source.Subject}}); err != nil {
			t.Fatal(err)
		}
		if _, err := js.Publish(act.Source.Subject, []byte("first")); err != nil {
			t.Fatal(err)
		}
		if _, err := js.Publish(act.Source.Subject, []byte("second")); err != nil {
			t.Fatal(err)
		}
		sub, err := js.PullSubscribe(act.Source.Subject, act.Source.Consumer, nats.BindStream(act.Source.Stream))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = sub.Unsubscribe() })
		flush(t, nc)
		msgs, err := sub.Fetch(2, nats.MaxWait(time.Second))
		if err != nil {
			t.Fatal(err)
		}
		// Accept the later message first to advance the cursor.
		if _, err := router.AcceptStream(act, msgs[1]); err != nil {
			t.Fatal(err)
		}
		// Now accept the earlier message — cursor is behind the high-water mark.
		_, gotErr := router.AcceptStream(act, msgs[0])
		assertCore(t, gotErr, core.StaleCursor)
	})
}
