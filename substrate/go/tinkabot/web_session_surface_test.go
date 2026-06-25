package tinkabot

// TestWebSessionShell is the binary-side outside-in proof for Slice 7
// (web-session-surface): the shell HTTP server establishes the HttpOnly
// cookie session, gates the WebSocket upgrade on it, proxies the upgraded
// connection to the embedded NATS loopback WebSocket listener, and exchanges
// the cookie for an ephemeral bearer viewer credential at the mint endpoint.
// The end-to-end sub-test observes a mediated session and lands a steer
// intent on command acceptance through the minted credential.

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/lagz0ne/tinkabot/substrate/go/core"
	"github.com/lagz0ne/tinkabot/substrate/go/embednats"
)

// shellCookie fetches the shell index and returns the HttpOnly session cookie
// the server set.
func shellCookie(t *testing.T, shellURL string) *http.Cookie {
	t.Helper()
	res, err := http.Get(shellURL)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	for _, c := range res.Cookies() {
		if c.Name == "tb_shell" {
			return c
		}
	}
	t.Fatal("shell did not set the tb_shell HttpOnly cookie session on first contact")
	return nil
}

// wsUpgrade hand-rolls an HTTP/1.1 WebSocket upgrade against the shell WS
// route at path (the nats.go client cannot attach a Cookie header, and
// browsers do not reliably attach SameSite cookies to ws:// upgrades — the
// test plays both gating modes). It returns the response status line and any
// bytes that followed the response head within the read window (for a piped
// connection, the NATS INFO banner).
func wsUpgrade(t *testing.T, shellURL, path, cookie string) (string, []byte) {
	t.Helper()
	u, err := url.Parse(shellURL)
	if err != nil {
		t.Fatal(err)
	}
	conn, err := net.DialTimeout("tcp", u.Host, 3*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))

	keyRaw := make([]byte, 16)
	if _, err := rand.Read(keyRaw); err != nil {
		t.Fatal(err)
	}
	req := "GET " + path + " HTTP/1.1\r\n" +
		"Host: " + u.Host + "\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Key: " + base64.StdEncoding.EncodeToString(keyRaw) + "\r\n" +
		"Sec-WebSocket-Version: 13\r\n"
	if cookie != "" {
		req += "Cookie: tb_shell=" + cookie + "\r\n"
	}
	req += "\r\n"
	if _, err := conn.Write([]byte(req)); err != nil {
		t.Fatal(err)
	}

	buf := make([]byte, 4096)
	var got []byte
	for {
		n, err := conn.Read(buf)
		if n > 0 {
			got = append(got, buf[:n]...)
		}
		if err != nil || strings.Contains(string(got), "INFO") || (len(got) > 0 && !strings.Contains(string(got), "101") && strings.Contains(string(got), "\r\n\r\n")) {
			break
		}
	}
	head := string(got)
	statusLine := head
	if i := strings.Index(head, "\r\n"); i >= 0 {
		statusLine = head[:i]
	}
	return statusLine, got
}

func TestWebSessionShell(t *testing.T) {
	t.Parallel()

	app, err := boot(t, cfgFor(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	shell := app.Posture().Shell

	// CookieIssued: first contact sets an HttpOnly, SameSite session cookie —
	// the durable browser-side authority page script can never read.
	t.Run("CookieIssued", func(t *testing.T) {
		res, err := http.Get(shell.URL)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		raw := strings.Join(res.Header.Values("Set-Cookie"), "; ")
		if !strings.Contains(raw, "tb_shell=") {
			t.Fatalf("no tb_shell cookie session issued: %q", raw)
		}
		if !strings.Contains(raw, "HttpOnly") || !strings.Contains(raw, "SameSite=Strict") {
			t.Fatalf("cookie session must be HttpOnly and SameSite=Strict: %q", raw)
		}
	})

	// UngatedUpgrade: a WS upgrade without a cookie session is denied 401, and
	// a malformed/unknown cookie is denied 401 — output-parsed, never exit-code.
	t.Run("UngatedUpgrade", func(t *testing.T) {
		status, _ := wsUpgrade(t, shell.URL, "/session/ws", "")
		if !strings.Contains(status, "401") {
			t.Fatalf("UngatedUpgrade: upgrade without cookie session must be 401, got %q", status)
		}
		status2, _ := wsUpgrade(t, shell.URL, "/session/ws", "forged-token-xyz")
		if !strings.Contains(status2, "401") {
			t.Fatalf("UngatedUpgrade: upgrade with unknown cookie must be 401, got %q", status2)
		}
	})

	// CookieGatedUpgrade: with the issued cookie the upgrade completes (101)
	// and the piped connection speaks NATS — the INFO banner arrives through
	// the shell's proxy from the embedded loopback WebSocket listener. The
	// same cookie upgrades twice (DuplicateUpgrade: the session cookie is not
	// consumed by one upgrade).
	t.Run("CookieGatedUpgrade", func(t *testing.T) {
		cookie := shellCookie(t, shell.URL)
		status, body := wsUpgrade(t, shell.URL, "/session/ws", cookie.Value)
		if !strings.Contains(status, "101") {
			t.Fatalf("CookieGatedUpgrade: expected 101 Switching Protocols, got %q", status)
		}
		if !strings.Contains(string(body), "INFO") {
			t.Fatal("CookieGatedUpgrade: no NATS INFO banner through the proxied upgrade — the shell WS route must pipe to the embedded NATS WebSocket listener")
		}
		status2, body2 := wsUpgrade(t, shell.URL, "/session/ws", cookie.Value)
		if !strings.Contains(status2, "101") || !strings.Contains(string(body2), "INFO") {
			t.Fatalf("DuplicateUpgrade: second upgrade with the same cookie session must succeed, got %q", status2)
		}
	})

	// TicketGatedUpgrade: the mint endpoint derives a single-use upgrade
	// ticket from the cookie session (browsers do not reliably attach
	// SameSite cookies to ws:// handshakes). The ticket upgrades once without
	// any cookie, is consumed by use, and a forged ticket is denied.
	t.Run("TicketGatedUpgrade", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		const sid = "wss-ticket-001"
		mediator, err := embednats.StartFrameMediator(ctx, app.Runtime(), embednats.FrameMediatorConfig{SessionID: sid, QuotaMaxBytes: 1 << 20})
		if err != nil {
			t.Fatal(err)
		}
		defer mediator.Stop()

		cookie := shellCookie(t, shell.URL)
		req, err := http.NewRequest(http.MethodPost, shell.URL+"/session/viewer", strings.NewReader(`{"sessionId":"`+sid+`"}`))
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
			WSTicket string `json:"wsTicket"`
		}
		err = json.NewDecoder(res.Body).Decode(&grant)
		res.Body.Close()
		if err != nil || grant.WSTicket == "" {
			t.Fatalf("mint must return a wsTicket, err %v", err)
		}

		status, body := wsUpgrade(t, shell.URL, "/session/ws?t="+grant.WSTicket, "")
		if !strings.Contains(status, "101") || !strings.Contains(string(body), "INFO") {
			t.Fatalf("TicketGatedUpgrade: cookieless upgrade with a fresh ticket must succeed, got %q", status)
		}
		status2, _ := wsUpgrade(t, shell.URL, "/session/ws?t="+grant.WSTicket, "")
		if !strings.Contains(status2, "401") {
			t.Fatalf("TicketGatedUpgrade: a ticket is single-use — reuse must be 401, got %q", status2)
		}
		status3, _ := wsUpgrade(t, shell.URL, "/session/ws?t=forged-ticket", "")
		if !strings.Contains(status3, "401") {
			t.Fatalf("TicketGatedUpgrade: forged ticket must be 401, got %q", status3)
		}
	})

	// MintEndpoint: the cookie session is exchanged for an ephemeral bearer
	// viewer credential; no cookie and malformed bodies are denied with typed
	// statuses; the credential observes a mediated session end-to-end and its
	// steer intent lands on command acceptance — never on the steering subject.
	t.Run("MintEndpoint", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		const sid = "wss-shell-e2e-001"
		cookie := shellCookie(t, shell.URL)

		// The session stream must exist before a viewer deliver consumer can
		// bind to it: the mediator owns the stream.
		rt := app.Runtime()
		mediator, err := embednats.StartFrameMediator(ctx, rt, embednats.FrameMediatorConfig{SessionID: sid, QuotaMaxBytes: 1 << 20})
		if err != nil {
			t.Fatalf("MintEndpoint: StartFrameMediator: %v", err)
		}
		defer mediator.Stop()

		mint := func(body string, withCookie bool) (*http.Response, error) {
			req, err := http.NewRequest(http.MethodPost, shell.URL+"/session/viewer", strings.NewReader(body))
			if err != nil {
				return nil, err
			}
			req.Header.Set("Content-Type", "application/json")
			if withCookie {
				req.AddCookie(cookie)
			}
			return http.DefaultClient.Do(req)
		}

		// No cookie: denied.
		res, err := mint(`{"sessionId":"`+sid+`"}`, false)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != http.StatusUnauthorized {
			t.Fatalf("MintEndpoint: mint without cookie session must be 401, got %d", res.StatusCode)
		}

		// Malformed body: denied 400.
		res, err = mint(`{not json`, true)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != http.StatusBadRequest {
			t.Fatalf("MintEndpoint: malformed mint body must be 400, got %d", res.StatusCode)
		}
		res, err = mint(`{"sessionId":"bad.session"}`, true)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != http.StatusBadRequest {
			t.Fatalf("MintEndpoint: subject-unsafe mint session id must be 400, got %d", res.StatusCode)
		}

		// Valid: a bearer JWT plus the viewer's own deliver subject.
		res, err = mint(`{"sessionId":"`+sid+`"}`, true)
		if err != nil {
			t.Fatal(err)
		}
		var grant struct {
			JWT            string `json:"jwt"`
			DeliverSubject string `json:"deliverSubject"`
			StateSubject   string `json:"stateSubject"`
		}
		err = json.NewDecoder(res.Body).Decode(&grant)
		res.Body.Close()
		if err != nil || res.StatusCode != http.StatusOK {
			t.Fatalf("MintEndpoint: mint with cookie: status %d, decode err %v", res.StatusCode, err)
		}
		if grant.JWT == "" || !strings.HasPrefix(grant.DeliverSubject, "tb.session."+sid+".deliver.") {
			t.Fatalf("MintEndpoint: grant must carry a bearer JWT and the session deliver subject, got %+v", grant)
		}
		if !strings.HasPrefix(grant.StateSubject, "tb.app.browser.state.") || strings.Contains(grant.StateSubject, sid) {
			t.Fatalf("MintEndpoint: grant must carry an opaque viewer state prefix, got %+v", grant)
		}

		// The credential observes the mediated session: publish one frame
		// through ingest and watch it arrive on the deliver subject over the
		// embedded WebSocket listener.
		ws := rt.Posture().WebSocket
		if !ws.Enabled || ws.URL == "" {
			t.Fatalf("MintEndpoint: the binary must enable the loopback WebSocket listener, posture: %#v", ws)
		}
		viewerNC, err := nats.Connect(ws.URL,
			nats.UserJWT(
				func() (string, error) { return grant.JWT, nil },
				func([]byte) ([]byte, error) { return nil, nil },
			),
			nats.MaxReconnects(0),
		)
		if err != nil {
			t.Fatalf("MintEndpoint: bearer viewer connect over WS failed: %v", err)
		}
		defer viewerNC.Close()

		got := make(chan []byte, 8)
		if _, err := viewerNC.Subscribe(grant.DeliverSubject, func(m *nats.Msg) { got <- m.Data }); err != nil {
			t.Fatalf("MintEndpoint: viewer subscribe deliver subject: %v", err)
		}
		if err := viewerNC.Flush(); err != nil {
			t.Fatal(err)
		}

		ingestUC, err := rt.MintUser(embednats.AppAccount,
			principal("principal.test.ingest", "lease-test-ingest", core.Permissions{
				Publish:   core.PermList{Allow: []string{"tb.session." + sid + ".ingest"}},
				Subscribe: core.PermList{Allow: []string{"_INBOX.>"}},
			}), time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		ingestNC, err := rt.ConnectCreds(ctx, ingestUC.File)
		if err != nil {
			t.Fatal(err)
		}
		defer ingestNC.Close()
		frame, _ := json.Marshal(map[string]any{
			"kind": "session.frame", "frame": "token", "origin": "wrapper",
			"sessionId": sid, "text": "shell-e2e",
		})
		if err := ingestNC.Publish("tb.session."+sid+".ingest", frame); err != nil {
			t.Fatal(err)
		}

		select {
		case b := <-got:
			if !strings.Contains(string(b), "shell-e2e") {
				t.Fatalf("MintEndpoint: unexpected frame on deliver subject: %s", b)
			}
		case <-time.After(10 * time.Second):
			t.Fatal("MintEndpoint: mediated frame never arrived on the viewer deliver subject over WS")
		}

		// Steering rides command acceptance: a harness gateway subscriber
		// receives the viewer's intent on tb.app.browser.command.
		gwUC, err := rt.MintUser(embednats.AppAccount,
			principal("principal.test.gateway", "lease-test-gateway", core.Permissions{
				Subscribe: core.PermList{Allow: []string{"tb.app.browser.command", "_INBOX.>"}},
				Publish:   core.PermList{Allow: []string{"_INBOX.>"}},
			}), time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		gwNC, err := rt.ConnectCreds(ctx, gwUC.File)
		if err != nil {
			t.Fatal(err)
		}
		defer gwNC.Close()
		intents := make(chan []byte, 1)
		if _, err := gwNC.Subscribe("tb.app.browser.command", func(m *nats.Msg) {
			intents <- m.Data
			_ = m.Respond([]byte(`{"accepted":true}`))
		}); err != nil {
			t.Fatal(err)
		}
		if err := gwNC.Flush(); err != nil {
			t.Fatal(err)
		}

		steer := []byte(`{"kind":"session.steer_intent","intent":"steer","sessionId":"` + sid + `","text":"focus"}`)
		if _, err := viewerNC.Request("tb.app.browser.command", steer, 5*time.Second); err != nil {
			t.Fatalf("MintEndpoint: viewer steer intent via command acceptance failed: %v", err)
		}
		select {
		case b := <-intents:
			if !strings.Contains(string(b), "session.steer_intent") {
				t.Fatalf("MintEndpoint: gateway received unexpected payload: %s", b)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("MintEndpoint: steer intent never reached command acceptance")
		}
	})
}
