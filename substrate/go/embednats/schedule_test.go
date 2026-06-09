package embednats

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/lagz0ne/tinkabot/substrate/go/core"
)

func TestEmbeddedScheduleStorePersistsRestartCatchUp(t *testing.T) {
	act := activation(t, read(t, "fixtures/valid/activation-source-schedule.json"))
	bucket := "tb_schedule_" + strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	ledgerBucket := bucket + "_ledger"
	auth := scheduleAuth(act, bucket, ledgerBucket)
	cfg := valid(t)
	cfg.Auth = auth
	rt, err := Start(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stop(t, rt) })

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	sched, err := NewKVScheduleStore(ctx, rt, bucket)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(sched.Close)
	ledgerStore, err := NewKVLedgerStore(ctx, rt, ledgerBucket)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(ledgerStore.Close)

	engine, err := core.NewScheduleEngine(auth, core.NewDurableLedger(ledgerStore), sched)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := engine.Accept(act, embedTick(act)); err != nil {
		t.Fatal(err)
	}

	restarted, err := core.NewScheduleEngine(auth, core.NewDurableLedger(ledgerStore), sched)
	if err != nil {
		t.Fatal(err)
	}
	tick2 := embedTick(act)
	tick2.TickID = "tick-002"
	tick2.Clock = "clock-main:0002"
	recs, err := restarted.CatchUp(act, []core.ScheduleTick{embedTick(act), tick2})
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 || recs[0].SourcePosition != 2 || recs[0].Status != core.Accepted {
		t.Fatalf("embedded schedule catch-up drift: %#v", recs)
	}
	replay, err := core.NewDurableLedger(ledgerStore).Replay("", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(replay) != 2 {
		t.Fatalf("embedded schedule replay drift: %#v", replay)
	}
}

func scheduleAuth(act core.Activation, scheduleBucket, ledgerBucket string) core.Auth {
	sub := "tb.schedule." + act.Source.ScheduleID + ".>"
	allow := []string{"$JS.API.>", "$KV." + scheduleBucket + ".>", "$KV." + ledgerBucket + ".>", sub, "_INBOX.>"}
	return core.Auth{
		User: act.SourcePrincipal.PrincipalID,
		Capability: core.Capability{
			PrincipalID:   act.SourcePrincipal.PrincipalID,
			SessionID:     "session-source",
			CapabilityID:  "cap-source",
			LeaseID:       act.SourceLease.LeaseID,
			LeaseStatus:   "active",
			AppRevision:   act.Provenance.AppRevision,
			SchemaVersion: act.Provenance.SchemaVersion,
		},
		Permissions: core.Permissions{
			Publish:        core.PermList{Allow: allow},
			Subscribe:      core.PermList{Allow: allow},
			AllowResponses: core.AllowResponses{Max: 1, ExpiresMs: 30000},
		},
		Imports: map[string]core.Import{
			"source": {Kind: "subscribe", Subjects: []string{sub}, Desc: "schedule source"},
		},
		Exports: []string{sub},
		Exposure: map[string]core.Exposure{
			act.SourcePrincipal.AuthorityRef: {Kind: "subject", Subject: sub, Desc: "schedule source exposure"},
		},
	}
}

func embedTick(act core.Activation) core.ScheduleTick {
	return core.ScheduleTick{
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
