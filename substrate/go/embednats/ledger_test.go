package embednats

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lagz0ne/tinkabot/substrate/go/core"
)

func TestEmbeddedLedgerUsesJetStreamKV(t *testing.T) {
	ledger, store := embeddedLedger(t, "tb_ledger")
	lease := core.Lease{ID: "lease-source-stream-001", Status: "active"}
	act := activation(t, read(t, "fixtures/valid/activation-source-stream.json"))

	first, err := ledger.Accept(act, lease)
	if err != nil {
		t.Fatal(err)
	}
	dup, err := ledger.Accept(act, lease)
	if err != nil {
		t.Fatal(err)
	}
	if first.Status != core.Accepted || dup.Status != core.Duplicate {
		t.Fatalf("accept drift: first=%#v dup=%#v", first, dup)
	}

	next := activation(t, edit(t, "fixtures/valid/activation-source-stream.json", func(doc map[string]any) {
		doc["activationId"] = "act:scripts.proof.observe:stream:101"
		doc["dedupeKey"] = "stream:scripts.proof.observe:tb_proof_events:101"
		src := doc["source"].(map[string]any)
		src["streamSequence"] = float64(101)
		src["consumerSequence"] = float64(9)
	}))
	second, err := ledger.Accept(next, lease)
	if err != nil {
		t.Fatal(err)
	}

	restarted := core.NewDurableLedger(store)
	dup, err = restarted.Accept(act, lease)
	if err != nil {
		t.Fatal(err)
	}
	if dup.Status != core.Duplicate {
		t.Fatalf("restart duplicate drift: %#v", dup)
	}
	replay, err := restarted.Replay(first.ReplayCursor, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(replay) != 1 || replay[0].ActivationID != second.ActivationID {
		t.Fatalf("embedded replay drift: %#v", replay)
	}

	stale := activation(t, edit(t, "fixtures/valid/activation-source-stale-cursor.json", func(doc map[string]any) {
		doc["dedupeKey"] = "stream:scripts.proof.observe:tb_proof_events:10:embedded-stale"
	}))
	_, err = restarted.Accept(stale, lease)
	assertCore(t, err, core.StaleCursor)
}

func TestEmbeddedLedgerAcceptsCanonicalSourceKinds(t *testing.T) {
	ledger, _ := embeddedLedger(t, "tb_ledger_sources")
	fixtures := []string{
		"fixtures/valid/activation-request-reply.json",
		"fixtures/valid/activation-command-acceptance.json",
		"fixtures/valid/activation-source-subject.json",
		"fixtures/valid/activation-source-kv.json",
		"fixtures/valid/activation-source-object.json",
		"fixtures/valid/activation-source-stream.json",
		"fixtures/valid/activation-source-schedule.json",
	}

	for _, fixture := range fixtures {
		t.Run(fixture, func(t *testing.T) {
			act := activation(t, read(t, fixture))
			rec, err := ledger.Accept(act, core.Lease{ID: act.SourceLease.LeaseID, Status: "active"})
			if err != nil {
				t.Fatal(err)
			}
			if rec.Status != core.Accepted || rec.SourceID != act.SourcePrincipal.SourceID || rec.SourceKind != act.Source.Kind || rec.ReplayCursor == "" {
				t.Fatalf("embedded source drift: %#v", rec)
			}
		})
	}
}

func embeddedLedger(t *testing.T, bucket string) (*core.DurableLedger, *KVLedgerStore) {
	t.Helper()
	cfg := valid(t)
	cfg.Auth.Permissions.Publish.Allow = []string{"$JS.API.>", "$KV." + bucket + ".>"}
	cfg.Auth.Permissions.Subscribe.Allow = []string{"_INBOX.>", "$KV." + bucket + ".>"}

	rt, err := Start(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stop(t, rt) })

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	store, err := NewKVLedgerStore(ctx, rt, bucket)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(store.Close)
	return core.NewDurableLedger(store), store
}

func read(t *testing.T, fixture string) []byte {
	t.Helper()
	doc, err := os.ReadFile(filepath.Join("..", "..", "..", "schemas", "endgame", "v1", fixture))
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

func edit(t *testing.T, fixture string, fn func(map[string]any)) []byte {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal(read(t, fixture), &doc); err != nil {
		t.Fatal(err)
	}
	fn(doc)
	out, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func activation(t *testing.T, doc []byte) core.Activation {
	t.Helper()
	var act core.Activation
	if err := json.Unmarshal(doc, &act); err != nil {
		t.Fatal(err)
	}
	return act
}
