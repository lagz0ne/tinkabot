package embednats

// TestBundleAccountSeam proves the account-per-bundle isolation primitives:
// a runtime-minted account is a hard subject namespace with its own
// JetStream plane — the same subject does not cross, the same bucket name
// holds unrelated state — and a service export imported into TB_APP is the
// only crossing, under the importer's chosen local name.

import (
	"errors"
	"testing"
	"time"

	"github.com/lagz0ne/tinkabot/substrate/go/core"
	"github.com/nats-io/nats.go"
)

func TestBundleAccountSeam(t *testing.T) {
	t.Parallel()
	rt, err := start(t, operatorCfg(t, Loopback()))
	if err != nil {
		t.Fatal(err)
	}
	const acct = "TB_BUNDLE_T"
	const subject = "tb.bundle.t.run"
	if err := rt.MintAccount(acct); err != nil {
		t.Fatal(err)
	}
	if err := rt.MintAccount(acct); err == nil {
		t.Fatal("duplicate account mint must fail typed")
	}
	if err := rt.ExportService("TB_NOWHERE", subject); err == nil {
		t.Fatal("export from an unknown account must fail typed")
	}

	// The responder lives inside the bundle account.
	resp := mint(t, rt, acct, principal("principal.bundle.t.responder", core.Permissions{
		Subscribe:      core.PermList{Allow: []string{subject}},
		AllowResponses: core.AllowResponses{Max: 1, ExpiresMs: 30000},
	}))
	rnc := connect(t, rt, resp)
	if _, err := rnc.Subscribe(subject, func(m *nats.Msg) { _ = m.Respond([]byte("pong")) }); err != nil {
		t.Fatal(err)
	}
	if err := rnc.Flush(); err != nil {
		t.Fatal(err)
	}

	caller := mint(t, rt, AppAccount, principal("principal.app.caller", appPerms(subject)))
	cnc := connect(t, rt, caller)

	t.Run("SameSubjectNoCrossing", func(t *testing.T) {
		if _, err := cnc.Request(subject, []byte("ping"), 500*time.Millisecond); err == nil {
			t.Fatal("request crossed the account boundary without an import")
		}
	})

	t.Run("ImportedServiceRoundTrip", func(t *testing.T) {
		if err := rt.ExportService(acct, subject); err != nil {
			t.Fatal(err)
		}
		if err := rt.ImportService(AppAccount, acct, subject, ""); err != nil {
			t.Fatal(err)
		}
		// The claims push propagates asynchronously; poll until the import
		// routes.
		deadline := time.Now().Add(5 * time.Second)
		for {
			reply, err := cnc.Request(subject, []byte("ping"), time.Second)
			if err == nil && string(reply.Data) == "pong" {
				return
			}
			if time.Now().After(deadline) {
				t.Fatalf("imported service did not answer: %v", err)
			}
			time.Sleep(50 * time.Millisecond)
		}
	})

	t.Run("JetStreamIsolated", func(t *testing.T) {
		jsPerms := core.Permissions{
			Publish:   core.PermList{Allow: []string{"$JS.API.>", "$KV.shadow.>"}},
			Subscribe: core.PermList{Allow: []string{"_INBOX.>"}},
		}
		bun := connect(t, rt, mint(t, rt, acct, principal("principal.bundle.t.store", jsPerms)))
		bjs, err := bun.JetStream()
		if err != nil {
			t.Fatal(err)
		}
		if _, err := bjs.CreateKeyValue(&nats.KeyValueConfig{Bucket: "shadow", Storage: nats.MemoryStorage}); err != nil {
			t.Fatal(err)
		}

		app := connect(t, rt, mint(t, rt, AppAccount, principal("principal.app.store", jsPerms)))
		ajs, err := app.JetStream()
		if err != nil {
			t.Fatal(err)
		}
		if _, err := ajs.KeyValue("shadow"); !errors.Is(err, nats.ErrBucketNotFound) {
			t.Fatalf("bundle bucket visible across the account boundary: %v", err)
		}
	})
}
