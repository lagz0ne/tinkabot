package core

import "testing"

func TestDurableLedgerAcceptsAllSourceCursors(t *testing.T) {
	cases := []struct {
		name   string
		path   string
		id     string
		kind   string
		pos    int64
		cursor string
	}{
		{"request", "fixtures/valid/activation-request-reply.json", "src-request-runtime", "request_reply", 0, "req-001"},
		{"command", "fixtures/valid/activation-command-acceptance.json", "src-command-browser_command", "command_acceptance", 0, "cmd-001"},
		{"subject", "fixtures/valid/activation-source-subject.json", "src-subject-runtime", "subject", 0, "msg-001"},
		{"kv", "fixtures/valid/activation-source-kv.json", "src-kv-material", "kv", 42, "kv:tb_proof_kv:42"},
		{"object", "fixtures/valid/activation-source-object.json", "src-object-artifacts", "object", 91, "obj:tb_proof_objects:91"},
		{"stream", "fixtures/valid/activation-source-stream.json", "src-stream-events", "stream", 100, "tb_proof_events:activation_source:100:8"},
		{"schedule", "fixtures/valid/activation-source-schedule.json", "src-schedule-daily", "schedule", 3, "sched-daily:tick-001:fence-003:3"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			store := NewMemoryLedgerStore()
			ledger := NewDurableLedger(store)
			act := activation(t, read(t, c.path))

			rec, err := ledger.Accept(act, Lease{ID: act.SourceLease.LeaseID, Status: "active"})
			if err != nil {
				t.Fatal(err)
			}
			if rec.Status != Accepted || rec.SourceID != c.id || rec.SourceKind != c.kind || rec.SourcePosition != c.pos || rec.SourceCursor != c.cursor {
				t.Fatalf("record drift: %#v", rec)
			}
			if rec.SourcePrincipalID != act.SourcePrincipal.PrincipalID || rec.SourceLeaseID != act.SourceLease.LeaseID {
				t.Fatalf("source authority drift: %#v", rec)
			}
			if rec.ReplayCursor != replayCursor(c.id, c.cursor) || rec.ReplayCursor == rec.SourceID+":"+rec.SourceCursor || store.AcceptedCount() != 1 {
				t.Fatalf("cursor persistence drift: rec=%#v count=%d", rec, store.AcceptedCount())
			}
		})
	}
}

func TestDurableLedgerReplayCursorEncodingAvoidsTextCollision(t *testing.T) {
	store := NewMemoryLedgerStore()
	ledger := NewDurableLedger(store)
	first := activation(t, edit(t, "fixtures/valid/activation-request-reply.json", func(doc map[string]any) {
		doc["activationId"] = "act:collision:1"
		doc["dedupeKey"] = "collision:1"
		sp := doc["sourcePrincipal"].(map[string]any)
		sp["principalId"] = "principal.source.collision.1"
		sp["sourceId"] = "a:b"
		sl := doc["sourceLease"].(map[string]any)
		sl["leaseId"] = "lease-collision-1"
		src := doc["source"].(map[string]any)
		src["requestId"] = "c"
	}))
	second := activation(t, edit(t, "fixtures/valid/activation-request-reply.json", func(doc map[string]any) {
		doc["activationId"] = "act:collision:2"
		doc["dedupeKey"] = "collision:2"
		sp := doc["sourcePrincipal"].(map[string]any)
		sp["principalId"] = "principal.source.collision.2"
		sp["sourceId"] = "a"
		sl := doc["sourceLease"].(map[string]any)
		sl["leaseId"] = "lease-collision-2"
		src := doc["source"].(map[string]any)
		src["requestId"] = "b:c"
	}))

	rec1, err := ledger.Accept(first, Lease{ID: "lease-collision-1", Status: "active"})
	if err != nil {
		t.Fatal(err)
	}
	rec2, err := ledger.Accept(second, Lease{ID: "lease-collision-2", Status: "active"})
	if err != nil {
		t.Fatal(err)
	}
	if rec1.ReplayCursor == rec2.ReplayCursor {
		t.Fatalf("replay cursor collision: %q", rec1.ReplayCursor)
	}
	replay, err := ledger.Replay("", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(replay) != 2 || replay[0].ActivationID != first.ActivationID || replay[1].ActivationID != second.ActivationID {
		t.Fatalf("collision replay drift: %#v", replay)
	}
}

func TestDurableLedgerAcceptDuplicateReplayAndRestart(t *testing.T) {
	store := NewMemoryLedgerStore()
	ledger := NewDurableLedger(store)
	lease := Lease{ID: "lease-source-stream-001", Status: "active"}
	act := activation(t, read(t, "fixtures/valid/activation-source-stream.json"))

	first, err := ledger.Accept(act, lease)
	if err != nil {
		t.Fatal(err)
	}
	if first.Status != Accepted || first.SourceID != "src-stream-events" || first.SourcePosition != 100 || first.ReplayCursor == "" {
		t.Fatalf("accept drift: %#v", first)
	}

	dup, err := ledger.Accept(act, lease)
	if err != nil {
		t.Fatal(err)
	}
	if dup.Status != Duplicate || store.AcceptedCount() != 1 {
		t.Fatalf("duplicate drift: dup=%#v count=%d", dup, store.AcceptedCount())
	}

	next := activation(t, edit(t, "fixtures/valid/activation-source-stream.json", func(doc map[string]any) {
		doc["activationId"] = "act:scripts.proof.observe:stream:101"
		doc["triggerId"] = "stream-seq-101"
		doc["dedupeKey"] = "stream:scripts.proof.observe:tb_proof_events:101"
		src := doc["source"].(map[string]any)
		src["streamSequence"] = float64(101)
		src["consumerSequence"] = float64(9)
	}))
	second, err := ledger.Accept(next, lease)
	if err != nil {
		t.Fatal(err)
	}

	replay, err := ledger.Replay(first.ReplayCursor, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(replay) != 1 || replay[0].ActivationID != second.ActivationID {
		t.Fatalf("replay drift: %#v", replay)
	}
	_, err = ledger.Replay("src-stream-events:missing", 10)
	assertKind(t, err, ReplayCursorFailed)

	restarted := NewDurableLedger(store)
	dup, err = restarted.Accept(act, lease)
	if err != nil {
		t.Fatal(err)
	}
	if dup.Status != Duplicate || store.AcceptedCount() != 2 {
		t.Fatalf("restart drift: dup=%#v count=%d", dup, store.AcceptedCount())
	}
	replay, err = restarted.Replay(first.ReplayCursor, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(replay) != 1 || replay[0].ActivationID != second.ActivationID {
		t.Fatalf("restart replay drift: %#v", replay)
	}
	stale := activation(t, edit(t, "fixtures/valid/activation-source-stale-cursor.json", func(doc map[string]any) {
		doc["dedupeKey"] = "stream:scripts.proof.observe:tb_proof_events:10:restart-stale"
	}))
	_, err = restarted.Accept(stale, lease)
	assertKind(t, err, StaleCursor)
}

func TestDurableLedgerLoopCursorLeaseAndConflict(t *testing.T) {
	t.Run("loop", func(t *testing.T) {
		store := NewMemoryLedgerStore()
		ledger := NewDurableLedger(store)
		act := activation(t, read(t, "fixtures/valid/activation-source-stream.json"))
		act.Chain.Hop = 5
		act.Chain.MaxHops = 5

		_, err := ledger.Accept(act, Lease{ID: "lease-source-stream-001", Status: "active"})
		assertKind(t, err, LoopSuppressed)
		if store.SuppressedCount() != 1 {
			t.Fatalf("suppression not recorded")
		}
	})

	t.Run("stale cursor", func(t *testing.T) {
		store := NewMemoryLedgerStore()
		ledger := NewDurableLedger(store)
		lease := Lease{ID: "lease-source-stream-001", Status: "active"}
		act := activation(t, read(t, "fixtures/valid/activation-source-stream.json"))
		if _, err := ledger.Accept(act, lease); err != nil {
			t.Fatal(err)
		}

		stale := activation(t, edit(t, "fixtures/valid/activation-source-stale-cursor.json", func(doc map[string]any) {
			doc["dedupeKey"] = "stream:scripts.proof.observe:tb_proof_events:10:stale"
		}))
		_, err := ledger.Accept(stale, lease)
		assertKind(t, err, StaleCursor)
	})

	t.Run("lease", func(t *testing.T) {
		store := NewMemoryLedgerStore()
		ledger := NewDurableLedger(store)
		act := activation(t, read(t, "fixtures/valid/activation-source-stream.json"))
		_, err := ledger.Accept(act, Lease{ID: "lease-source-stream-001", Status: "revoked"})
		assertKind(t, err, LeaseAcquireFailed)
		_, err = ledger.Accept(act, Lease{ID: "lease-other", Status: "active"})
		assertKind(t, err, LeaseAcquireFailed)
		_, err = ledger.Accept(act, Lease{Status: "active"})
		assertKind(t, err, LeaseAcquireFailed)
		missing := act
		missing.SourceLease.LeaseID = ""
		_, err = ledger.Accept(missing, Lease{ID: "lease-source-stream-001", Status: "active"})
		assertKind(t, err, LeaseAcquireFailed)
		if store.AcceptedCount() != 0 || store.SuppressedCount() != 0 {
			t.Fatalf("lease denial wrote records")
		}
	})

	t.Run("source principal kind", func(t *testing.T) {
		store := NewMemoryLedgerStore()
		ledger := NewDurableLedger(store)
		act := activation(t, read(t, "fixtures/valid/activation-source-stream.json"))
		act.SourcePrincipal.SourceKind = "kv"
		_, err := ledger.Accept(act, Lease{ID: "lease-source-stream-001", Status: "active"})
		assertKind(t, err, CursorFailure)
		if store.AcceptedCount() != 0 || store.SuppressedCount() != 0 {
			t.Fatalf("source principal mismatch wrote records")
		}
	})

	t.Run("conflict and replay failure", func(t *testing.T) {
		store := NewMemoryLedgerStore()
		store.WriteConflict = true
		ledger := NewDurableLedger(store)
		act := activation(t, read(t, "fixtures/valid/activation-source-stream.json"))
		_, err := ledger.Accept(act, Lease{ID: "lease-source-stream-001", Status: "active"})
		assertKind(t, err, WriteConflict)
		if store.AcceptedCount() != 0 {
			t.Fatalf("write conflict recorded accept")
		}

		store = NewMemoryLedgerStore()
		store.CursorFailed = true
		ledger = NewDurableLedger(store)
		_, err = ledger.Replay("", 10)
		assertKind(t, err, ReplayCursorFailed)
	})
}
