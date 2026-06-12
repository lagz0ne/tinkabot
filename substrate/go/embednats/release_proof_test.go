package embednats

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/lagz0ne/tinkabot/substrate/go/core"
	"github.com/nats-io/nats.go"
)

func TestActivationReleaseProofAcceptedSources(t *testing.T) {
	t.Parallel()
	t.Run("request reply cli", func(t *testing.T) {
		act, router, rt, nc, _ := routerRuntime(t, "fixtures/valid/activation-request-reply.json")
		route, out, err := router.RequestReply(nc, act)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { mustStop(t, route) })
		flush(t, nc)

		auth := core.Auth{User: act.SourcePrincipal.PrincipalID, Capability: core.Capability{LeaseID: act.SourceLease.LeaseID}}
		reply, err := natsCLI(rt, auth, "request", "--raw", "-H", HeaderRequestID+":req-rel-001", act.Source.Subject, "ping")
		if err != nil {
			t.Fatalf("CLI request failed: %v\n%s", err, reply)
		}
		if strings.TrimSpace(reply) != string(core.Accepted) {
			t.Fatalf("CLI reply drift: %q", reply)
		}
		res := waitResult(t, out)
		assertOutcome(t, releaseOutcome(t, "request-reply-cli", res.Record, res.Err), "ActivationLedger", string(core.Accepted))
	})

	t.Run("subject", func(t *testing.T) {
		act, router, nc, _ := routerHarness(t, "fixtures/valid/activation-source-subject.json")
		route, out, err := router.Subject(nc, act)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { mustStop(t, route) })
		flush(t, nc)

		msg := nats.NewMsg("tb.proof.runtime.execute")
		msg.Header.Set(HeaderMessageID, "msg-rel-001")
		if err := nc.PublishMsg(msg); err != nil {
			t.Fatal(err)
		}
		flush(t, nc)
		res := waitResult(t, out)
		assertOutcome(t, releaseOutcome(t, "subject", res.Record, res.Err), "ActivationLedger", string(core.Accepted))
	})

	t.Run("kv object stream", func(t *testing.T) {
		for _, c := range []struct {
			name    string
			fixture string
		}{
			{"kv", "fixtures/valid/activation-source-kv.json"},
			{"object", "fixtures/valid/activation-source-object.json"},
			{"stream", "fixtures/valid/activation-source-stream.json"},
		} {
			t.Run(c.name, func(t *testing.T) {
				res := acceptStoreSource(t, c.fixture)
				assertOutcome(t, releaseOutcome(t, c.name, res.Record, res.Err), "ActivationLedger", string(core.Accepted))
			})
		}
	})

	t.Run("schedule", func(t *testing.T) {
		rec := acceptScheduleSource(t)
		assertOutcome(t, releaseOutcome(t, "schedule", rec, nil), "ActivationLedger", string(core.Accepted))
	})
}

func TestActivationReleaseProofFailureAttribution(t *testing.T) {
	t.Parallel()
	t.Run("malformed", func(t *testing.T) {
		act, router, nc, _ := routerHarness(t, "fixtures/valid/activation-request-reply.json")
		route, out, err := router.RequestReply(nc, act)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { mustStop(t, route) })
		flush(t, nc)
		if _, err := nc.Request(act.Source.Subject, []byte("missing id"), time.Second); err != nil {
			t.Fatal(err)
		}
		res := waitResult(t, out)
		assertOutcome(t, releaseOutcome(t, "malformed", res.Record, res.Err), "LiveSourceRouter", string(SourceMalformed))
	})

	t.Run("denied neighbor", func(t *testing.T) {
		act := activation(t, read(t, "fixtures/valid/activation-source-subject.json"))
		bucket := releaseBucket(t)
		serverAuth := routerAuth(act, bucket)
		auth := serverAuth
		auth.Permissions.Subscribe.Deny = append(auth.Permissions.Subscribe.Deny, "tb.proof.runtime.denied")
		router, _, nc, _ := releaseRuntime(t, act, serverAuth, auth, bucket)
		route, out, err := router.Subject(nc, act)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { mustStop(t, route) })
		flush(t, nc)

		msg := nats.NewMsg("tb.proof.runtime.denied")
		msg.Header.Set(HeaderMessageID, "msg-rel-denied")
		if err := nc.PublishMsg(msg); err != nil {
			t.Fatal(err)
		}
		flush(t, nc)
		res := waitResult(t, out)
		assertOutcome(t, releaseOutcome(t, "denied", res.Record, res.Err), "SourceAuthority", string(core.DeniedNeighbor))
	})

	t.Run("duplicate", func(t *testing.T) {
		act, router, _, _ := routerHarness(t, "fixtures/valid/activation-source-subject.json")

		msg := nats.NewMsg("tb.proof.runtime.execute")
		msg.Header.Set(HeaderMessageID, "msg-rel-dup-direct")

		first, firstErr := router.AcceptSubject(act, msg)
		assertOutcome(t, releaseOutcome(t, "duplicate-first", first, firstErr), "ActivationLedger", string(core.Accepted))
		dup, dupErr := router.AcceptSubject(act, msg)
		assertOutcome(t, releaseOutcome(t, "duplicate", dup, dupErr), "ActivationLedger", string(core.Duplicate))
	})

	t.Run("stale cursor", func(t *testing.T) {
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
		sub, err := js.PullSubscribe(act.Source.Subject, releaseBucket(t)+"_seed", nats.BindStream(act.Source.Stream))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = sub.Unsubscribe() })
		flush(t, nc)
		msgs, err := sub.Fetch(2, nats.MaxWait(time.Second))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := router.AcceptStream(act, msgs[1]); err != nil {
			t.Fatal(err)
		}

		route, out, err := router.Stream(js, act)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { mustStop(t, route) })
		res := waitResult(t, out)
		assertOutcome(t, releaseOutcome(t, "stale", res.Record, res.Err), "ActivationLedger", string(core.StaleCursor))
	})

	t.Run("revoked lease", func(t *testing.T) {
		act := activation(t, edit(t, "fixtures/valid/activation-request-reply.json", func(doc map[string]any) {
			doc["sourceLease"].(map[string]any)["leaseStatus"] = "revoked"
		}))
		bucket := releaseBucket(t)
		auth := routerAuth(act, bucket)
		auth.Capability.LeaseStatus = "active"
		router, rt, nc, _ := releaseRuntime(t, act, auth, auth, bucket)
		route, out, err := router.RequestReply(nc, act)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { mustStop(t, route) })
		flush(t, nc)

		reply, err := natsCLI(rt, auth, "request", "--raw", "-H", HeaderRequestID+":req-rel-revoked", act.Source.Subject, "ping")
		if err != nil {
			t.Fatalf("CLI revoked request failed: %v\n%s", err, reply)
		}
		if strings.TrimSpace(reply) != string(core.LeaseRevoked) {
			t.Fatalf("revoked reply drift: %q", reply)
		}
		res := waitResult(t, out)
		assertOutcome(t, releaseOutcome(t, "revoked", res.Record, res.Err), "SourceAuthority", string(core.LeaseRevoked))
	})

	t.Run("loop suppressed", func(t *testing.T) {
		_, err := acceptScheduleLoop(t)
		assertOutcome(t, releaseOutcome(t, "loop", core.LedgerRecord{}, err), "ActivationLedger", string(core.LoopSuppressed))
	})
}

func acceptStoreSource(t *testing.T, fixture string) RouterResult {
	t.Helper()
	act, router, nc, js := routerHarness(t, fixture)
	switch act.Source.Kind {
	case "kv":
		kv, err := js.CreateKeyValue(&nats.KeyValueConfig{Bucket: act.Source.Bucket, Storage: nats.FileStorage})
		if err != nil {
			t.Fatal(err)
		}
		route, out, err := router.KV(kv, act)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { mustStop(t, route) })
		flush(t, nc)
		if _, err := kv.Put(act.Source.Key, []byte("state")); err != nil {
			t.Fatal(err)
		}
		return waitResult(t, out)
	case "object":
		obs, err := js.CreateObjectStore(&nats.ObjectStoreConfig{Bucket: act.Source.Bucket, Storage: nats.FileStorage})
		if err != nil {
			t.Fatal(err)
		}
		route, out, err := router.Object(js, act)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { mustStop(t, route) })
		flush(t, nc)
		if _, err := obs.PutBytes(act.Source.Name, []byte("bundle")); err != nil {
			t.Fatal(err)
		}
		return waitResult(t, out)
	case "stream":
		if _, err := js.AddStream(&nats.StreamConfig{Name: act.Source.Stream, Subjects: []string{act.Source.Subject}}); err != nil {
			t.Fatal(err)
		}
		route, out, err := router.Stream(js, act)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { mustStop(t, route) })
		flush(t, nc)
		if _, err := js.Publish(act.Source.Subject, []byte("event")); err != nil {
			t.Fatal(err)
		}
		return waitResult(t, out)
	default:
		t.Fatalf("unsupported fixture source: %s", act.Source.Kind)
		return RouterResult{}
	}
}

func acceptScheduleSource(t *testing.T) core.LedgerRecord {
	t.Helper()
	act, engine := embeddedScheduleEngine(t)
	rec, err := engine.Accept(act, embedTick(act))
	if err != nil {
		t.Fatal(err)
	}
	return rec
}

func acceptScheduleLoop(t *testing.T) (core.LedgerRecord, error) {
	t.Helper()
	act, engine := embeddedScheduleEngine(t)
	act.Chain.Hop = 5
	act.Chain.MaxHops = 5
	return engine.Accept(act, embedTick(act))
}

func embeddedScheduleEngine(t *testing.T) (core.Activation, *core.ScheduleEngine) {
	t.Helper()
	act := activation(t, read(t, "fixtures/valid/activation-source-schedule.json"))
	bucket := "tb_release_schedule_" + strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	ledgerBucket := bucket + "_ledger"
	auth := scheduleAuth(act, bucket, ledgerBucket)
	cfg := valid(t)
	cfg.Auth = auth
	rt, err := start(t, cfg)
	if err != nil {
		t.Fatal(err)
	}

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
	return act, engine
}

func releaseRuntime(t *testing.T, act core.Activation, rtAuth, auth core.Auth, bucket string) (*SourceRouter, *Runtime, *nats.Conn, nats.JetStreamContext) {
	t.Helper()
	cfg := valid(t)
	cfg.Auth = rtAuth
	rt, err := start(t, cfg)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	store, err := NewKVLedgerStore(ctx, rt, bucket)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(store.Close)
	router, err := NewSourceRouter(auth, core.NewDurableLedger(store))
	if err != nil {
		t.Fatal(err)
	}
	nc, err := rt.Connect(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(nc.Close)
	js, err := nc.JetStream()
	if err != nil {
		t.Fatal(err)
	}
	return router, rt, nc, js
}

func releaseBucket(t *testing.T) string {
	t.Helper()
	return strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' {
			return r
		}
		return '_'
	}, "tb_release_"+t.Name())
}

type proofOutcome struct {
	Scenario string
	Owner    string
	Kind     string
	Record   core.LedgerRecord
	Err      error
}

func releaseOutcome(t *testing.T, s string, rec core.LedgerRecord, err error) proofOutcome {
	t.Helper()
	out := proofOutcome{
		Scenario: s,
		Owner:    "ActivationLedger",
		Kind:     string(rec.Status),
		Record:   rec,
		Err:      err,
	}
	if err == nil {
		return out
	}

	var coreErr *core.Error
	if errors.As(err, &coreErr) {
		out.Owner = coreErr.Layer
		out.Kind = string(coreErr.Kind)
		return out
	}

	var liveErr *Error
	if errors.As(err, &liveErr) {
		out.Owner = liveErr.Layer
		out.Kind = string(liveErr.Kind)
		return out
	}

	t.Fatalf("unknown release proof error: %T: %v", err, err)
	return out
}

func assertOutcome(t *testing.T, got proofOutcome, owner, kind string) {
	t.Helper()
	if got.Owner != owner || got.Kind != kind {
		t.Fatalf("outcome drift: got=%#v owner=%s kind=%s", got, owner, kind)
	}
	if got.Scenario == "" {
		t.Fatalf("scenario missing: %#v", got)
	}
	switch core.Status(kind) {
	case core.Accepted, core.Duplicate:
		if got.Err != nil {
			t.Fatalf("unexpected accepted-path error: %#v", got)
		}
		if got.Record.Status != core.Status(kind) || got.Record.ActivationID == "" || got.Record.SourceID == "" || got.Record.ReplayCursor == "" {
			t.Fatalf("durable record missing: %#v", got)
		}
	default:
		if got.Err == nil {
			t.Fatalf("expected owned error: %#v", got)
		}
		if got.Record.ActivationID != "" {
			t.Fatalf("error outcome carried durable record: %#v", got)
		}
	}
}
