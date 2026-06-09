package core

import "testing"

func TestScheduleEngineAcceptsDueTickAndCatchUp(t *testing.T) {
	act := activation(t, read(t, "fixtures/valid/activation-source-schedule.json"))
	engine, sched, ledger := scheduleEngine(t, act)

	rec, err := engine.Accept(act, tickFrom(act))
	if err != nil {
		t.Fatal(err)
	}
	if rec.Status != Accepted || rec.SourceKind != "schedule" || rec.SourcePosition != 1 || rec.SourceCursor != "sched-daily:tick-001:fence-003:3:clock-main:0001" {
		t.Fatalf("schedule accept drift: %#v", rec)
	}
	if rec.SourcePrincipalID != act.SourcePrincipal.PrincipalID || rec.SourceLeaseID != act.SourceLease.LeaseID {
		t.Fatalf("schedule authority drift: %#v", rec)
	}

	restarted, err := NewScheduleEngine(scheduleAuthFor(act), NewDurableLedger(ledger), sched)
	if err != nil {
		t.Fatal(err)
	}
	tick2 := tickFrom(act)
	tick2.TickID = "tick-002"
	tick2.Clock = "clock-main:0002"
	recs, err := restarted.CatchUp(act, []ScheduleTick{tickFrom(act), tick2})
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 || recs[0].Status != Accepted || recs[0].SourcePosition != 2 || recs[0].SourceCursor != "sched-daily:tick-002:fence-003:3:clock-main:0002" {
		t.Fatalf("catch-up drift: %#v", recs)
	}

	tick3 := tickFrom(act)
	tick3.TickID = "tick-003"
	tick3.Clock = "clock-main:0003"
	rec, err = restarted.Accept(act, tick3)
	if err != nil {
		t.Fatal(err)
	}
	if rec.SourcePosition != 3 || rec.SourceCursor != "sched-daily:tick-003:fence-003:3:clock-main:0003" {
		t.Fatalf("same-leader cursor drift: %#v", rec)
	}
}

func TestScheduleEngineDenials(t *testing.T) {
	t.Run("config", func(t *testing.T) {
		act := activation(t, read(t, "fixtures/valid/activation-source-schedule.json"))
		engine, _, _ := scheduleEngine(t, act)
		tick := tickFrom(act)
		tick.TickID = ""
		_, err := engine.Accept(act, tick)
		assertKind(t, err, ScheduleConfigInvalid)

		act.Source.OwnerPrincipalID = "principal.other"
		_, err = engine.Accept(act, tickFrom(act))
		assertKind(t, err, ScheduleConfigInvalid)
	})

	t.Run("duplicate", func(t *testing.T) {
		act := activation(t, read(t, "fixtures/valid/activation-source-schedule.json"))
		engine, _, _ := scheduleEngine(t, act)
		if _, err := engine.Accept(act, tickFrom(act)); err != nil {
			t.Fatal(err)
		}
		_, err := engine.Accept(act, tickFrom(act))
		assertKind(t, err, ScheduleTickDuplicate)
	})

	t.Run("clock", func(t *testing.T) {
		act := activation(t, read(t, "fixtures/valid/activation-source-schedule.json"))
		engine, _, _ := scheduleEngine(t, act)
		if _, err := engine.Accept(act, tickFrom(act)); err != nil {
			t.Fatal(err)
		}
		tick := tickFrom(act)
		tick.TickID = "tick-002"
		_, err := engine.Accept(act, tick)
		assertKind(t, err, ClockInvalid)

		tick.Clock = "bad"
		_, err = engine.Accept(act, tick)
		assertKind(t, err, ClockInvalid)
	})

	t.Run("lease", func(t *testing.T) {
		act := activation(t, read(t, "fixtures/valid/activation-source-schedule.json"))
		missing := act
		missing.SourceLease.LeaseID = ""
		engine, _, _ := scheduleEngine(t, act)
		_, err := engine.Accept(missing, tickFrom(missing))
		assertKind(t, err, ScheduleLeaseMissing)

		revoked := act
		revoked.SourceLease.LeaseStatus = "revoked"
		_, err = engine.Accept(revoked, tickFrom(revoked))
		assertKind(t, err, LeaseRevoked)
	})

	t.Run("leader", func(t *testing.T) {
		act := activation(t, read(t, "fixtures/valid/activation-source-schedule.json"))
		engine, _, _ := scheduleEngine(t, act)
		if _, err := engine.Accept(act, tickFrom(act)); err != nil {
			t.Fatal(err)
		}
		tick := tickFrom(act)
		tick.TickID = "tick-002"
		tick.Clock = "clock-main:0002"
		tick.LeaderEpoch = 2
		_, err := engine.Accept(act, tick)
		assertKind(t, err, ScheduleLeaseLost)
	})

	t.Run("loop", func(t *testing.T) {
		act := activation(t, read(t, "fixtures/valid/activation-source-schedule.json"))
		act.Chain.Hop = 5
		act.Chain.MaxHops = 5
		engine, _, _ := scheduleEngine(t, act)
		_, err := engine.Accept(act, tickFrom(act))
		assertKind(t, err, LoopSuppressed)
		_, err = engine.Accept(act, tickFrom(act))
		assertKind(t, err, ScheduleTickDuplicate)
	})
}

func TestScheduleEngineStoreFailures(t *testing.T) {
	act := activation(t, read(t, "fixtures/valid/activation-source-schedule.json"))
	_, err := NewScheduleEngine(sourceAuth(act), nil, NewMemoryScheduleStore())
	assertKind(t, err, ScheduleConfigInvalid)
	_, err = NewScheduleEngine(sourceAuth(act), NewDurableLedger(NewMemoryLedgerStore()), nil)
	assertKind(t, err, ScheduleConfigInvalid)

	recoverFailed := NewMemoryScheduleStore()
	recoverFailed.RecoverFailed = true
	engine, err := NewScheduleEngine(scheduleAuthFor(act), NewDurableLedger(NewMemoryLedgerStore()), recoverFailed)
	if err != nil {
		t.Fatal(err)
	}
	_, err = engine.Accept(act, tickFrom(act))
	assertKind(t, err, RestartRecoveryFailed)

	writeFailed := NewMemoryScheduleStore()
	writeFailed.WriteFailed = true
	engine, err = NewScheduleEngine(scheduleAuthFor(act), NewDurableLedger(NewMemoryLedgerStore()), writeFailed)
	if err != nil {
		t.Fatal(err)
	}
	_, err = engine.Accept(act, tickFrom(act))
	assertKind(t, err, CatchUpFailed)
}

func scheduleEngine(t *testing.T, act Activation) (*ScheduleEngine, *MemoryScheduleStore, *MemoryLedgerStore) {
	t.Helper()
	sched := NewMemoryScheduleStore()
	ledger := NewMemoryLedgerStore()
	engine, err := NewScheduleEngine(scheduleAuthFor(act), NewDurableLedger(ledger), sched)
	if err != nil {
		t.Fatal(err)
	}
	return engine, sched, ledger
}

func scheduleAuthFor(act Activation) Auth {
	auth := sourceAuth(act)
	if act.Source.Kind != "schedule" {
		return auth
	}
	subject := "tb.schedule." + act.Source.ScheduleID + ".>"
	auth.Permissions.Subscribe.Allow = append(auth.Permissions.Subscribe.Allow, subject)
	auth.Imports["source"] = Import{Kind: "subscribe", Subjects: []string{subject}, Desc: "Schedule source observation aperture."}
	auth.Exports = []string{subject}
	auth.Exposure[act.SourcePrincipal.AuthorityRef] = Exposure{Kind: "subject", Subject: subject, Desc: "Schedule source exposure."}
	return auth
}

func tickFrom(act Activation) ScheduleTick {
	return ScheduleTick{
		TickID:       act.Source.TickID,
		DueAt:        act.Source.DueAt,
		AcquiredAt:   act.Source.AcquiredAt,
		ExpiresAt:    act.Source.ExpiresAt,
		ClockID:      act.Source.ClockID,
		Clock:        act.Source.Clock,
		LeaderEpoch:  act.Source.LeaderEpoch,
		FencingToken: act.Source.FencingToken,
	}
}
