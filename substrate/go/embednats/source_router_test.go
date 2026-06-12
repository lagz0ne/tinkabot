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

func TestSourceRouterAcceptsLiveSourcesOverEmbeddedNATS(t *testing.T) {
	t.Parallel()
	t.Run("request reply", func(t *testing.T) {
		act, router, nc, _ := routerHarness(t, "fixtures/valid/activation-request-reply.json")
		route, out, err := router.RequestReply(nc, act)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { mustStop(t, route) })
		flush(t, nc)

		msg := nats.NewMsg(act.Source.Subject)
		msg.Header.Set(HeaderRequestID, "req-live-001")
		msg.Data = []byte("ping")
		reply, err := nc.RequestMsg(msg, time.Second)
		if err != nil {
			t.Fatal(err)
		}
		if string(reply.Data) != string(core.Accepted) {
			t.Fatalf("reply drift: %q", reply.Data)
		}

		rec := waitResult(t, out).Record
		if rec.Status != core.Accepted || rec.SourceCursor != "req-live-001" || rec.SourcePrincipalID != act.SourcePrincipal.PrincipalID {
			t.Fatalf("request record drift: %#v", rec)
		}
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
		msg.Header.Set(HeaderMessageID, "msg-live-001")
		if err := nc.PublishMsg(msg); err != nil {
			t.Fatal(err)
		}
		flush(t, nc)

		rec := waitResult(t, out).Record
		if rec.Status != core.Accepted || rec.SourceCursor == "" || rec.SourceID != act.SourcePrincipal.SourceID {
			t.Fatalf("subject record drift: %#v", rec)
		}
	})

	t.Run("kv", func(t *testing.T) {
		act, router, nc, js := routerHarness(t, "fixtures/valid/activation-source-kv.json")
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

		res := waitResult(t, out)
		if string(res.Payload) != "state" {
			t.Fatalf("KV payload drift: %q", res.Payload)
		}
		rec := res.Record
		if rec.Status != core.Accepted || rec.SourcePosition != 1 || !strings.Contains(rec.SourceCursor, act.Source.Key) {
			t.Fatalf("KV record drift: %#v", rec)
		}
	})

	t.Run("object", func(t *testing.T) {
		act, router, nc, js := routerHarness(t, "fixtures/valid/activation-source-object.json")
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

		rec := waitResult(t, out).Record
		if rec.Status != core.Accepted || rec.SourcePosition <= 0 || !strings.Contains(rec.SourceCursor, act.Source.Name) {
			t.Fatalf("object record drift: %#v", rec)
		}
	})

	t.Run("stream", func(t *testing.T) {
		act, router, nc, js := routerHarness(t, "fixtures/valid/activation-source-stream.json")
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

		rec := waitResult(t, out).Record
		if rec.Status != core.Accepted || rec.SourcePosition != 1 || rec.SourceCursor == "" {
			t.Fatalf("stream record drift: %#v", rec)
		}
	})
}

func TestSourceRouterRequestReplyFromNATSCLI(t *testing.T) {
	t.Parallel()
	act, router, rt, nc, _ := routerRuntime(t, "fixtures/valid/activation-request-reply.json")
	route, out, err := router.RequestReply(nc, act)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { mustStop(t, route) })
	flush(t, nc)

	auth := core.Auth{
		User: act.SourcePrincipal.PrincipalID,
		Capability: core.Capability{
			LeaseID: act.SourceLease.LeaseID,
		},
	}
	reply, err := natsCLI(rt, auth, "request", "--raw", "-H", HeaderRequestID+":req-cli-001", act.Source.Subject, "ping")
	if err != nil {
		t.Fatalf("CLI request failed: %v\n%s", err, reply)
	}
	if strings.TrimSpace(reply) != string(core.Accepted) {
		t.Fatalf("CLI reply drift: %q", reply)
	}
	if rec := waitResult(t, out).Record; rec.Status != core.Accepted || rec.SourceCursor != "req-cli-001" {
		t.Fatalf("CLI record drift: %#v", rec)
	}
}

func TestSourceRouterEdgeCases(t *testing.T) {
	t.Parallel()
	t.Run("router failures are typed", func(t *testing.T) {
		act := activation(t, read(t, "fixtures/valid/activation-request-reply.json"))
		router, err := NewSourceRouter(routerAuth(act, "tb_router_edges"), core.NewDurableLedger(core.NewMemoryLedgerStore()))
		if err != nil {
			t.Fatal(err)
		}
		_, _, err = router.RequestReply(nil, act)
		assertRouter(t, err, RequestReplyListenFailed)
		_, _, err = router.Subject(nil, act)
		assertRouter(t, err, SubjectSubscribeFailed)
		_, _, err = router.KV(nil, act)
		assertRouter(t, err, KVWatchFailed)
		_, _, err = router.Object(nil, act)
		assertRouter(t, err, ObjectWatchFailed)
		_, _, err = router.Stream(nil, act)
		assertRouter(t, err, StreamConsumeFailed)
		_, err = NewSourceRouter(routerAuth(act, "tb_router_edges"), nil)
		assertRouter(t, err, RouterConfigInvalid)
		var nilRouter *SourceRouter
		_, err = nilRouter.AcceptRequest(act, nats.NewMsg(act.Source.Subject))
		assertRouter(t, err, RouterCritical)
	})

	t.Run("malformed request frame", func(t *testing.T) {
		act, router, nc, _ := routerHarness(t, "fixtures/valid/activation-request-reply.json")
		route, out, err := router.RequestReply(nc, act)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { mustStop(t, route) })
		flush(t, nc)

		msg := nats.NewMsg(act.Source.Subject)
		msg.Data = []byte("missing id")
		reply, err := nc.RequestMsg(msg, time.Second)
		if err != nil {
			t.Fatal(err)
		}
		if string(reply.Data) != string(SourceMalformed) {
			t.Fatalf("malformed reply drift: %q", reply.Data)
		}
		assertRouter(t, waitResult(t, out).Err, SourceMalformed)
	})

	t.Run("source authority denial is propagated with attribution", func(t *testing.T) {
		act := activation(t, read(t, "fixtures/valid/activation-source-subject.json"))
		auth := routerAuth(act, "tb_router_denied")
		auth.Permissions.Subscribe.Deny = append(auth.Permissions.Subscribe.Deny, "tb.proof.runtime.denied")
		router, err := NewSourceRouter(auth, core.NewDurableLedger(core.NewMemoryLedgerStore()))
		if err != nil {
			t.Fatal(err)
		}

		msg := nats.NewMsg("tb.proof.runtime.denied")
		msg.Header.Set(HeaderMessageID, "msg-denied")
		_, err = router.AcceptSubject(act, msg)
		assertCore(t, err, core.DeniedNeighbor)
		var got *core.Error
		if !errors.As(err, &got) || got.Details["sourceId"] != act.SourcePrincipal.SourceID {
			t.Fatalf("denial attribution lost: %#v", got)
		}
	})

	t.Run("duplicate is ledger owned", func(t *testing.T) {
		act, router, _, _ := routerHarness(t, "fixtures/valid/activation-source-subject.json")

		msg := nats.NewMsg("tb.proof.runtime.execute")
		msg.Header.Set(HeaderMessageID, "msg-dup-direct")

		first, err := router.AcceptSubject(act, msg)
		if err != nil {
			t.Fatal(err)
		}
		if first.Status != core.Accepted {
			t.Fatalf("first duplicate record drift: %#v", first)
		}
		dup, err := router.AcceptSubject(act, msg)
		if err != nil {
			t.Fatal(err)
		}
		if dup.Status != core.Duplicate {
			t.Fatalf("second duplicate record drift: %#v", dup)
		}
	})

	t.Run("stale stream cursor is ledger owned", func(t *testing.T) {
		act, router, nc, js := routerHarness(t, "fixtures/valid/activation-source-stream.json")
		if _, err := js.AddStream(&nats.StreamConfig{Name: act.Source.Stream, Subjects: []string{act.Source.Subject}}); err != nil {
			t.Fatal(err)
		}
		sub, err := js.PullSubscribe(act.Source.Subject, act.Source.Consumer, nats.BindStream(act.Source.Stream))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = sub.Unsubscribe() })
		flush(t, nc)
		if _, err := js.Publish(act.Source.Subject, []byte("first")); err != nil {
			t.Fatal(err)
		}
		if _, err := js.Publish(act.Source.Subject, []byte("second")); err != nil {
			t.Fatal(err)
		}
		msgs, err := sub.Fetch(2, nats.MaxWait(time.Second))
		if err != nil {
			t.Fatal(err)
		}

		if _, err := router.AcceptStream(act, msgs[1]); err != nil {
			t.Fatal(err)
		}
		_, err = router.AcceptStream(act, msgs[0])
		assertCore(t, err, core.StaleCursor)
	})

	t.Run("revoked source lease is source-authority owned", func(t *testing.T) {
		act := activation(t, edit(t, "fixtures/valid/activation-request-reply.json", func(doc map[string]any) {
			doc["sourceLease"].(map[string]any)["leaseStatus"] = "revoked"
		}))
		auth := routerAuth(act, "tb_router_revoked")
		auth.Capability.LeaseStatus = "active"
		router, err := NewSourceRouter(auth, core.NewDurableLedger(core.NewMemoryLedgerStore()))
		if err != nil {
			t.Fatal(err)
		}

		msg := nats.NewMsg(act.Source.Subject)
		msg.Header.Set(HeaderRequestID, "req-revoked")
		_, err = router.AcceptRequest(act, msg)
		assertCore(t, err, core.LeaseRevoked)
	})
}

func routerHarness(t *testing.T, fixture string) (core.Activation, *SourceRouter, *nats.Conn, nats.JetStreamContext) {
	t.Helper()

	act, router, _, nc, js := routerRuntime(t, fixture)
	return act, router, nc, js
}

func routerRuntime(t *testing.T, fixture string) (core.Activation, *SourceRouter, *Runtime, *nats.Conn, nats.JetStreamContext) {
	t.Helper()

	act := activation(t, read(t, fixture))
	bucket := "tb_router_" + strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	auth := routerAuth(act, bucket)
	cfg := valid(t)
	cfg.Auth = auth
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
	return act, router, rt, nc, js
}

func routerAuth(act core.Activation, ledgerBucket string) core.Auth {
	src := routeSubject(act.Source)
	allow := []string{
		src,
		"$JS.API.>",
		"$KV." + ledgerBucket + ".>",
		"_INBOX.>",
	}
	if act.Source.Bucket != "" {
		allow = append(allow, "$KV."+act.Source.Bucket+".>", "$O."+act.Source.Bucket+".>")
	}
	if act.Source.Subject != "" {
		allow = append(allow, act.Source.Subject)
	}
	if act.Source.Pattern != "" {
		allow = append(allow, act.Source.Pattern)
	}
	if act.Source.Stream != "" {
		allow = append(allow, "tb.proof.events.>")
	}
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
			Publish:        core.PermList{Allow: allow, Deny: []string{"tb.internal.>"}},
			Subscribe:      core.PermList{Allow: allow, Deny: []string{"tb.internal.>"}},
			AllowResponses: core.AllowResponses{Max: 1, ExpiresMs: 30000},
		},
		Imports: map[string]core.Import{
			"source": {Kind: "subscribe", Subjects: []string{src}, Desc: "live source watch"},
		},
		Exports: []string{src},
		Exposure: map[string]core.Exposure{
			act.SourcePrincipal.AuthorityRef: {Kind: routeExposure(act.Source.Kind), Subject: src, Desc: "live source exposure"},
		},
	}
}

func routeSubject(src core.Source) string {
	switch src.Kind {
	case "kv":
		return "$KV." + src.Bucket + "." + src.Key
	case "object":
		return "$O." + src.Bucket + "." + src.Name
	case "subject":
		return src.Pattern
	default:
		return src.Subject
	}
}

func routeExposure(kind string) string {
	switch kind {
	case "request_reply":
		return "request_reply"
	case "subject":
		return "subject"
	case "kv":
		return "kv_watch"
	case "object":
		return "object_change"
	case "stream":
		return "stream"
	default:
		return kind
	}
}

func waitResult(t *testing.T, out <-chan RouterResult) RouterResult {
	t.Helper()
	select {
	case res := <-out:
		return res
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for router result")
		return RouterResult{}
	}
}

func flush(t *testing.T, nc *nats.Conn) {
	t.Helper()
	if err := nc.FlushTimeout(time.Second); err != nil {
		t.Fatal(err)
	}
}

func mustStop(t *testing.T, route *Route) {
	t.Helper()
	if err := route.Stop(); err != nil {
		t.Fatal(err)
	}
}

func assertRouter(t *testing.T, err error, kind Kind) {
	t.Helper()
	assertAdapter(t, err, kind)
	var got *Error
	if !errors.As(err, &got) || got.Layer != "LiveSourceRouter" {
		t.Fatalf("router attribution drift: %#v", got)
	}
}
