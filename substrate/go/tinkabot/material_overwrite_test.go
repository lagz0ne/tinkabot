package tinkabot

// TestMaterializerArtifactOverwrite pins the serving-time defect found live by
// the builder example: NATS object store purges the prior object's chunks when
// an artifact name is overwritten (nats.go object.go Put returns the purge
// error), so the materializer principal must hold PURGE on its own artifact
// bucket. Without it, every re-emit of an artifact under a stable name fails
// ArtifactWriteFailed even though the new content may already be published —
// data integrity, not just log noise. The boundary stays narrow: only the
// artifact OBJ stream, never the ledger or script streams.

import (
	"context"
	"testing"
	"time"

	"github.com/lagz0ne/tinkabot/substrate/go/core"
	"github.com/lagz0ne/tinkabot/substrate/go/embednats"
)

func TestMaterializerArtifactOverwrite(t *testing.T) {
	t.Parallel()
	app, err := boot(t, cfgFor(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	w := wiring()
	uc, err := app.Runtime().MintUser(embednats.AppAccount, principal("principal.test.materializer", "lease-test-mat", servicePerms(w)), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	nc, err := app.Runtime().ConnectCreds(ctx, uc.File)
	if err != nil {
		t.Fatal(err)
	}
	store, err := embednats.OpenKVMaterialStore(nc, w.MaterialBucket, w.ArtifactBucket)
	if err != nil {
		t.Fatal(err)
	}

	art := func(body string) core.MaterialArtifact {
		return core.MaterialArtifact{Name: "artifact/overwrite.txt", MediaType: "text/plain", Body: []byte(body)}
	}
	if err := store.SaveArtifact(art("one")); err != nil {
		t.Fatalf("first artifact write failed: %v", err)
	}
	// The overwrite is the bug surface: it purges the prior object's chunks.
	if err := store.SaveArtifact(art("two")); err != nil {
		t.Fatalf("artifact overwrite failed (PURGE on artifact bucket denied?): %v", err)
	}
	_, body, ok, err := store.LoadArtifact("artifact/overwrite.txt")
	if err != nil || !ok {
		t.Fatalf("overwritten artifact unreadable: ok=%v err=%v", ok, err)
	}
	if string(body) != "two" {
		t.Fatalf("overwrite did not take: got %q want %q", body, "two")
	}
}
