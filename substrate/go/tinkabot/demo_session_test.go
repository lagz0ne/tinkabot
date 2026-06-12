package tinkabot

// TestDemoSession proves the demo observation example end-to-end through the
// real browser path: with Config.DemoSession set, the binary spawns a
// mediated stand-in session that ticks canonical token frames continuously,
// and a viewer minted through the shell HTTP surface (cookie -> bearer grant)
// observes those ticks over the embedded WebSocket listener on its deliver
// subject — exactly the composition the shell page renders.

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
)

func TestDemoSession(t *testing.T) {
	t.Parallel()

	cfg := cfgFor(t.TempDir())
	cfg.DemoSession = "demo-001"
	app, err := boot(t, cfg)
	if err != nil {
		t.Fatal(err)
	}
	shell := app.Posture().Shell

	// Browser path: cookie session, then the mint endpoint.
	cookie := shellCookie(t, shell.URL)
	req, err := http.NewRequest(http.MethodPost, shell.URL+"/session/viewer", strings.NewReader(`{"sessionId":"demo-001"}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	var grant struct {
		JWT            string `json:"jwt"`
		DeliverSubject string `json:"deliverSubject"`
	}
	err = json.NewDecoder(res.Body).Decode(&grant)
	res.Body.Close()
	if err != nil || res.StatusCode != http.StatusOK {
		t.Fatalf("demo viewer mint: status %d, decode err %v", res.StatusCode, err)
	}

	ws := app.Runtime().Posture().WebSocket
	viewerNC, err := nats.Connect(ws.URL,
		nats.UserJWT(
			func() (string, error) { return grant.JWT, nil },
			func([]byte) ([]byte, error) { return nil, nil },
		),
		nats.MaxReconnects(0),
	)
	if err != nil {
		t.Fatalf("bearer viewer connect: %v", err)
	}
	defer viewerNC.Close()

	ticks := make(chan string, 16)
	if _, err := viewerNC.Subscribe(grant.DeliverSubject, func(m *nats.Msg) {
		var f struct {
			Frame string `json:"frame"`
			Text  string `json:"text"`
		}
		if json.Unmarshal(m.Data, &f) == nil && f.Frame == "token" {
			ticks <- f.Text
		}
	}); err != nil {
		t.Fatal(err)
	}
	if err := viewerNC.Flush(); err != nil {
		t.Fatal(err)
	}

	// Two distinct ticks prove the emitter is continuous, not a one-shot.
	seen := map[string]bool{}
	deadline := time.After(15 * time.Second)
	for len(seen) < 2 {
		select {
		case txt := <-ticks:
			if !strings.Contains(txt, "tick") {
				t.Fatalf("unexpected token text from demo session: %q", txt)
			}
			seen[txt] = true
		case <-deadline:
			t.Fatalf("demo session produced %d distinct tick frames within 15s, want >= 2 — the continuous stand-in emitter is not running", len(seen))
		}
	}
}
