package tinkabot

// Session observation surface. The shell HTTP server owns
// the HttpOnly cookie session, the viewer mint endpoint, and the cookie-gated
// WebSocket upgrade proxy onto the embedded loopback WebSocket listener. The
// cookie gates only the HTTP surface; the viewer credential travels in
// CONNECT, and NATS stays the only authorization engine for session subjects
// — neither half suffices alone.

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/lagz0ne/tinkabot/substrate/go/embednats"
)

const (
	shellCookieName = "tb_shell"
	viewerTTL       = 10 * time.Minute
)

// shellSession reports whether the request carries a valid shell cookie
// session.
func (a *App) shellSession(r *http.Request) bool {
	c, err := r.Cookie(shellCookieName)
	if err != nil || c.Value == "" {
		return false
	}
	return embednats.ValidateCookieSession(a.rt, c.Value)
}

// ensureShellCookie establishes the HttpOnly cookie session on first contact
// with the shell. Secure is added when the external TLS posture lands; at the
// loopback posture the cookie stays same-origin via SameSite=Strict.
func (a *App) ensureShellCookie(rw http.ResponseWriter, r *http.Request) {
	if a.shellSession(r) {
		return
	}
	tok, err := embednats.IssueSessionCookie(a.rt)
	if err != nil {
		return
	}
	http.SetCookie(rw, &http.Cookie{
		Name:     shellCookieName,
		Value:    tok,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
}

// mintViewer exchanges the cookie session for an ephemeral bearer viewer
// credential plus the viewer's deliver subject, binding the substrate-side
// consumer over the session output stream.
func (a *App) mintViewer(rw http.ResponseWriter, r *http.Request) {
	if !a.shellSession(r) {
		http.Error(rw, "shell cookie session required", http.StatusUnauthorized)
		return
	}
	var req struct {
		SessionID string `json:"sessionId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || !validShellSessionID(req.SessionID) {
		http.Error(rw, "malformed mint request", http.StatusBadRequest)
		return
	}
	viewer, err := embednats.MintViewerCredential(a.rt, req.SessionID, viewerTTL)
	if err != nil {
		http.Error(rw, "viewer mint failed", http.StatusInternalServerError)
		return
	}
	if err := embednats.BindViewerDeliver(r.Context(), a.rt, req.SessionID, viewer.DeliverSubject); err != nil {
		_ = a.rt.Revoke(embednats.AppAccount, viewer.UserPub)
		http.Error(rw, "session stream unavailable", http.StatusConflict)
		return
	}
	// Browsers do not reliably attach SameSite cookies to ws:// upgrades, so
	// the cookie-gated mint also derives a single-use upgrade ticket that
	// carries the cookie session's authority onto the WS route.
	ticket, err := embednats.IssueUpgradeTicket(a.rt)
	if err != nil {
		_ = a.rt.Revoke(embednats.AppAccount, viewer.UserPub)
		http.Error(rw, "upgrade ticket unavailable", http.StatusInternalServerError)
		return
	}
	rw.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(rw).Encode(map[string]string{
		"jwt":            viewer.JWT,
		"deliverSubject": viewer.DeliverSubject,
		"stateSubject":   viewer.StateSubject,
		"wsTicket":       ticket,
	})
}

// sessionWS gates the WebSocket upgrade on the cookie session — directly, or
// via a single-use upgrade ticket derived from it at the mint endpoint — and
// pipes the upgraded connection to the embedded NATS loopback WebSocket
// listener. Only the WebSocket handshake headers are replayed to the backend;
// neither the cookie nor the ticket crosses into NATS.
func (a *App) sessionWS(rw http.ResponseWriter, r *http.Request) {
	if !a.shellSession(r) && !embednats.RedeemUpgradeTicket(a.rt, r.URL.Query().Get("t")) {
		http.Error(rw, "shell cookie session or upgrade ticket required", http.StatusUnauthorized)
		return
	}
	ws := a.rt.Posture().WebSocket
	if !ws.Enabled || ws.URL == "" {
		http.Error(rw, "websocket surface unavailable", http.StatusBadGateway)
		return
	}
	target := strings.TrimPrefix(ws.URL, "ws://")
	back, err := net.Dial("tcp", target)
	if err != nil {
		http.Error(rw, "websocket backend unavailable", http.StatusBadGateway)
		return
	}
	hj, ok := rw.(http.Hijacker)
	if !ok {
		back.Close()
		http.Error(rw, "upgrade unsupported", http.StatusInternalServerError)
		return
	}
	client, buf, err := hj.Hijack()
	if err != nil {
		back.Close()
		return
	}

	req := "GET / HTTP/1.1\r\nHost: " + target + "\r\n"
	for _, h := range []string{"Upgrade", "Connection", "Sec-WebSocket-Key", "Sec-WebSocket-Version", "Sec-WebSocket-Protocol", "Sec-WebSocket-Extensions"} {
		if v := r.Header.Get(h); v != "" {
			req += h + ": " + v + "\r\n"
		}
	}
	req += "\r\n"
	if _, err := back.Write([]byte(req)); err != nil {
		client.Close()
		back.Close()
		return
	}

	go func() {
		_, _ = io.Copy(back, buf)
		_ = back.Close()
		_ = client.Close()
	}()
	_, _ = io.Copy(client, back)
	_ = client.Close()
	_ = back.Close()
}

func validShellSessionID(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-' || r == '_':
		default:
			return false
		}
	}
	return true
}
