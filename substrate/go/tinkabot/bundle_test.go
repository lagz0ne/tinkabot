package tinkabot

// TestBundle proves the bundle surface: pointing the binary at a bundle dir
// serves a complete ephemeral app — manifest-declared scripts wired to
// triggers, boot effects materialized through the normal gate, and the
// frontend reachable as artifacts — while a bundle claiming durably-claimed
// authority is a typed load failure and nothing durable is mutated.

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lagz0ne/tinkabot/substrate/go/embednats"
	"github.com/nats-io/nats.go"
)

const clockBundle = "../../../examples/clock"

func TestBundle(t *testing.T) {
	t.Parallel()

	t.Run("AppServes", func(t *testing.T) {
		t.Parallel()
		cfg := cfgFor(t.TempDir())
		cfg.BundleDir = clockBundle
		app, err := boot(t, cfg)
		if err != nil {
			t.Fatal(err)
		}
		shell := app.Posture().Shell.URL

		hdr, body := waitFor200(t, shell+"/artifacts/bundle/clock/index.html", 15*time.Second)
		if ct := hdr.Get("Content-Type"); !strings.Contains(ct, "text/html") {
			t.Fatalf("artifact media type drift: %q", ct)
		}
		if csp := hdr.Get("Content-Security-Policy"); !strings.Contains(csp, "sandbox") {
			t.Fatalf("artifact served without sandbox policy: %q", csp)
		}
		if !strings.Contains(string(body), "tinkabot clock") {
			t.Fatalf("artifact body drift: %s", body)
		}

		_, pj := waitFor200(t, shell+"/projections/bundle.clock", 5*time.Second)
		first := unixOf(t, pj)

		// Nothing durable mutated: the bundle record never lands in the
		// durable script bucket.
		anc, err := nats.Connect(app.Posture().NATS.ClientURL, nats.UserCredentials(app.CredsFile(RoleAuthor)))
		if err != nil {
			t.Fatal(err)
		}
		defer anc.Close()
		js, err := anc.JetStream()
		if err != nil {
			t.Fatal(err)
		}
		kv, err := js.KeyValue("tb_scripts")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := kv.Get("s." + base64.RawURLEncoding.EncodeToString([]byte("scripts.clock.tick"))); err == nil {
			t.Fatal("bundle record leaked into the durable script bucket")
		}

		// Caller trigger round trip re-renders the projection.
		cnc, err := nats.Connect(app.Posture().NATS.ClientURL, nats.UserCredentials(app.CredsFile(RoleCaller)))
		if err != nil {
			t.Fatal(err)
		}
		defer cnc.Close()
		time.Sleep(1100 * time.Millisecond) // sequence is unix seconds; let it advance
		msg := nats.NewMsg("tb.bundle.clock.tick")
		msg.Header.Set(embednats.HeaderRequestID, "req-bundle-test-1")
		msg.Data = []byte("tick")
		reply, err := cnc.RequestMsg(msg, 5*time.Second)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(reply.Data), "accepted") {
			t.Fatalf("bundle trigger not accepted: %s", reply.Data)
		}
		deadline := time.Now().Add(10 * time.Second)
		for {
			_, pj := waitFor200(t, shell+"/projections/bundle.clock", 5*time.Second)
			if unixOf(t, pj) > first {
				break
			}
			if time.Now().After(deadline) {
				t.Fatal("projection did not advance after trigger")
			}
			time.Sleep(200 * time.Millisecond)
		}
	})

	t.Run("EphemeralAcrossRestart", func(t *testing.T) {
		t.Parallel()
		store := t.TempDir()
		cfg := cfgFor(store)
		cfg.BundleDir = clockBundle
		app1, err := boot(t, cfg)
		if err != nil {
			t.Fatal(err)
		}
		waitFor200(t, app1.Posture().Shell.URL+"/artifacts/bundle/clock/index.html", 15*time.Second)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := app1.Stop(ctx); err != nil {
			t.Fatal(err)
		}

		app2, err := boot(t, cfgFor(store))
		if err != nil {
			t.Fatal(err)
		}
		shell := app2.Posture().Shell.URL
		if code, _, _ := httpGet(t, shell+"/artifacts/bundle/clock/index.html"); code != http.StatusNotFound {
			t.Fatalf("bundle artifact survived a bundle-less restart: %d", code)
		}
		cnc, err := nats.Connect(app2.Posture().NATS.ClientURL, nats.UserCredentials(app2.CredsFile(RoleCaller)))
		if err != nil {
			t.Fatal(err)
		}
		defer cnc.Close()
		dead := nats.NewMsg("tb.bundle.clock.tick")
		dead.Header.Set(embednats.HeaderRequestID, "req-bundle-test-2")
		if _, err := cnc.RequestMsg(dead, time.Second); err == nil {
			t.Fatal("bundle trigger survived a bundle-less restart")
		}
		alive := nats.NewMsg(app2.Posture().Wiring.TriggerSubject)
		alive.Header.Set(embednats.HeaderRequestID, "req-bundle-test-3")
		if _, err := cnc.RequestMsg(alive, 5*time.Second); err != nil {
			t.Fatalf("manual trigger route is gone: %v", err)
		}
	})

	t.Run("ManifestCollision", func(t *testing.T) {
		t.Parallel()
		cases := []struct {
			name  string
			entry string
		}{
			{"ScriptKey", bundleEntry("scripts.app.main", "tb.bundle.t.run", `"bundle.t"`, "bundle/t/")},
			{"Trigger", bundleEntry("scripts.t.run", "tb.proof.runtime.execute", `"bundle.t"`, "bundle/t/")},
			{"Projection", bundleEntry("scripts.t.run", "tb.bundle.t.run", `"main"`, "bundle/t/")},
			{"ArtifactPrefix", bundleEntry("scripts.t.run", "tb.bundle.t.run", `"bundle.t"`, "artifact/")},
			{"ArtifactPrefixOverlap", bundleEntry("scripts.t.run", "tb.bundle.t.run", `"bundle.t"`, "art")},
			{"ReservedSubject", bundleEntry("scripts.t.run", "tb.session.t.run", `"bundle.t"`, "bundle/t/")},
			{
				"DuplicateInBundle",
				bundleEntry("scripts.t.run", "tb.bundle.t.run", `"bundle.t"`, "bundle/t/") + "," +
					bundleEntry("scripts.t.run", "tb.bundle.t.other", `"bundle.other"`, "bundle/other/"),
			},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				t.Parallel()
				cfg := cfgFor(t.TempDir())
				cfg.BundleDir = writeBundle(t, `{"kind":"bundle.manifest","name":"t","scripts":[`+c.entry+`]}`)
				_, err := boot(t, cfg)
				assertKind(t, err, BundleRejected)
			})
		}
	})

	t.Run("MalformedManifest", func(t *testing.T) {
		t.Parallel()
		cases := []struct {
			name     string
			manifest string
		}{
			{"UnknownField", `{"kind":"bundle.manifest","name":"t","zip":true,"scripts":[` +
				bundleEntry("scripts.t.run", "tb.bundle.t.run", `"bundle.t"`, "bundle/t/") + `]}`},
			{"MissingTrigger", `{"kind":"bundle.manifest","name":"t","scripts":[` +
				`{"scriptKey":"scripts.t.run","scriptRevision":1,"file":"scripts/noop.sh","command":"/bin/sh"}]}`},
			{"WrongKind", `{"kind":"script.record","name":"t","scripts":[]}`},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				t.Parallel()
				cfg := cfgFor(t.TempDir())
				cfg.BundleDir = writeBundle(t, c.manifest)
				_, err := boot(t, cfg)
				assertKind(t, err, BundleRejected)
			})
		}
		t.Run("MissingManifest", func(t *testing.T) {
			t.Parallel()
			cfg := cfgFor(t.TempDir())
			cfg.BundleDir = t.TempDir()
			_, err := boot(t, cfg)
			assertKind(t, err, BundleRejected)
		})
	})
}

func bundleEntry(key, trigger, projections, prefix string) string {
	return fmt.Sprintf(`{"scriptKey":%q,"scriptRevision":1,"file":"scripts/noop.sh","command":"/bin/sh","trigger":%q,"projections":[%s],"artifactPrefix":%q}`,
		key, trigger, projections, prefix)
}

func writeBundle(t *testing.T, manifest string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "scripts", "noop.sh"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bundle.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func httpGet(t *testing.T, url string) (int, http.Header, []byte) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, resp.Header, body
}

func waitFor200(t *testing.T, url string, timeout time.Duration) (http.Header, []byte) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		code, hdr, body := httpGet(t, url)
		if code == http.StatusOK {
			return hdr, body
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s stayed %d past %s", url, code, timeout)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func unixOf(t *testing.T, projection []byte) int64 {
	t.Helper()
	var p struct {
		Value struct {
			Unix int64 `json:"unix"`
		} `json:"value"`
	}
	if err := json.Unmarshal(projection, &p); err != nil {
		t.Fatalf("projection is not the stored record: %v: %s", err, projection)
	}
	return p.Value.Unix
}
