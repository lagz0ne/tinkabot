package embednats

import (
	"context"
	"time"

	"github.com/lagz0ne/tinkabot/substrate/go/core"
	jwt "github.com/nats-io/jwt/v2"
	"github.com/nats-io/nats.go/jetstream"
)

// sessionCookiesKV is the JetStream KV bucket backing shell cookie sessions.
const sessionCookiesKV = "tb-session-cookies"

// ViewerMintFailed is the typed kind for viewer credential mint failures.
const ViewerMintFailed Kind = "ViewerMintFailed"

// ViewerCred is the ephemeral browser-side grant: a bearer JWT (no signing
// seed ever exists browser-side) plus the viewer's own deliver subject. The
// UserPub is kept substrate-side for live revocation.
type ViewerCred struct {
	JWT            string
	DeliverSubject string
	UserPub        string
	Lease          core.Capability
}

// MintViewerCredential issues the web viewer grant for one session in
// TB_APP: bearer-mode, short-TTL, loopback-source-pinned, leaf-scoped to
//   - subscribe on the viewer's own deliver subject (fed by a substrate-bound
//     consumer via BindViewerDeliver) plus _INBOX.> for request replies,
//   - publish on tb.app.browser.command (command acceptance) only.
//
// No JetStream API authority, no session-subtree wildcard, and never the
// steering subject: the session runner stays the single steering writer, so
// viewer steering rides command acceptance into the mediated steering path.
func MintViewerCredential(rt *Runtime, sessionID string, ttl time.Duration) (ViewerCred, error) {
	if rt == nil || rt.op == nil {
		return ViewerCred{}, fail(ViewerMintFailed, "MintViewerCredential", "operator mode is not enabled", nil, nil)
	}
	if sessionID == "" {
		return ViewerCred{}, fail(ViewerMintFailed, "MintViewerCredential", "sessionID is required", nil, nil)
	}
	nonce, err := secret()
	if err != nil {
		return ViewerCred{}, fail(ViewerMintFailed, "MintViewerCredential", "deliver nonce could not be generated", nil, err)
	}
	deliver := "tb.session." + sessionID + ".deliver." + nonce
	leaseID, err := secret()
	if err != nil {
		return ViewerCred{}, fail(ViewerMintFailed, "MintViewerCredential", "lease id could not be generated", nil, err)
	}
	id := "viewer-" + sessionID
	creds, err := rt.MintUser(AppAccount, core.Auth{
		User: id,
		Capability: core.Capability{
			PrincipalID:   id,
			SessionID:     sessionID,
			CapabilityID:  "viewer-cap-" + sessionID,
			LeaseID:       leaseID,
			LeaseStatus:   "active",
			AppRevision:   "viewer.v1",
			SchemaVersion: "v1",
		},
		Permissions: core.Permissions{
			Publish:   core.PermList{Allow: []string{"tb.app.browser.command"}},
			Subscribe: core.PermList{Allow: []string{deliver, "_INBOX.>"}},
		},
	}, ttl)
	if err != nil {
		return ViewerCred{}, err
	}

	// Re-sign the minted user claims as a bearer token pinned to loopback
	// sources: the browser presents the JWT alone, and the credential is
	// self-enforcing loopback-bound regardless of who copies it.
	token, err := jwt.ParseDecoratedJWT(creds.File)
	if err != nil {
		return ViewerCred{}, fail(ViewerMintFailed, "MintViewerCredential", "minted creds could not be parsed", nil, err)
	}
	claims, err := jwt.DecodeUserClaims(token)
	if err != nil {
		return ViewerCred{}, fail(ViewerMintFailed, "MintViewerCredential", "user claims could not be decoded", nil, err)
	}
	claims.BearerToken = true
	claims.Limits.Src = jwt.CIDRList{"127.0.0.0/8", "::1/128"}
	rt.op.mu.Lock()
	acc := rt.op.accounts[AppAccount]
	rt.op.mu.Unlock()
	if acc == nil {
		return ViewerCred{}, fail(ViewerMintFailed, "MintViewerCredential", "AppAccount not found", nil, nil)
	}
	bearer, err := claims.Encode(acc.kp)
	if err != nil {
		return ViewerCred{}, fail(ViewerMintFailed, "MintViewerCredential", "bearer JWT could not be signed", nil, err)
	}
	return ViewerCred{JWT: bearer, DeliverSubject: deliver, UserPub: creds.UserPub, Lease: creds.Lease}, nil
}

// BindViewerDeliver creates the substrate-bound push consumer that feeds the
// viewer's deliver subject from the session output stream, delivering from
// the start of the stream (the snapshot-plus-tail attach shape). The
// JetStream API authority stays with this substrate principal; the viewer
// only ever subscribes the deliver subject.
func BindViewerDeliver(ctx context.Context, rt *Runtime, sessionID, deliver string) error {
	stream := "tb-session-out-" + sessionID
	nc, err := mintedConn(ctx, rt, "_tb_viewer_bind_"+sessionID, core.Permissions{
		Publish: core.PermList{Allow: []string{
			"$JS.API.INFO",
			"$JS.API.STREAM.INFO." + stream,
			"$JS.API.CONSUMER.CREATE." + stream,
			"$JS.API.CONSUMER.CREATE." + stream + ".>",
			"$JS.API.CONSUMER.DURABLE.CREATE." + stream + ".>",
		}},
		Subscribe: core.PermList{Allow: []string{"_INBOX.>"}},
	})
	if err != nil {
		return err
	}
	defer nc.Close()
	js, err := jetstream.New(nc)
	if err != nil {
		return err
	}
	_, err = js.CreateOrUpdateConsumer(ctx, stream, jetstream.ConsumerConfig{
		DeliverSubject: deliver,
		DeliverPolicy:  jetstream.DeliverAllPolicy,
		AckPolicy:      jetstream.AckNonePolicy,
	})
	return err
}

// withCookieKV runs op against the cookie session KV bucket over a minted
// least-authority connection, creating the bucket on first use.
func withCookieKV(rt *Runtime, op func(ctx context.Context, kv jetstream.KeyValue) error) error {
	ctx := context.Background()
	nc, err := mintedConn(ctx, rt, "_tb_cookies", core.Permissions{
		Publish:   core.PermList{Allow: kvWriteAPI(sessionCookiesKV, "$KV."+sessionCookiesKV+".>")},
		Subscribe: core.PermList{Allow: []string{"_INBOX.>", "$KV." + sessionCookiesKV + ".>"}},
	})
	if err != nil {
		return err
	}
	defer nc.Close()
	js, err := jetstream.New(nc)
	if err != nil {
		return err
	}
	kv, err := js.KeyValue(ctx, sessionCookiesKV)
	if err != nil {
		kv, err = js.CreateKeyValue(ctx, jetstream.KeyValueConfig{
			Bucket:  sessionCookiesKV,
			Storage: jetstream.MemoryStorage,
		})
		if err != nil {
			return err
		}
	}
	return op(ctx, kv)
}

// IssueSessionCookie mints an opaque shell cookie session token and registers
// it in the KV store. The shell HTTP server sets it as the HttpOnly cookie;
// it is the durable browser-side authority the mint endpoint and the WS
// upgrade gate both check.
func IssueSessionCookie(rt *Runtime) (string, error) {
	tok, err := secret()
	if err != nil {
		return "", err
	}
	err = withCookieKV(rt, func(ctx context.Context, kv jetstream.KeyValue) error {
		_, err := kv.Put(ctx, tok, []byte("shell"))
		return err
	})
	if err != nil {
		return "", err
	}
	return tok, nil
}

// ValidateCookieSession reports whether a shell cookie session token is live:
// empty, unknown, and revoked tokens are invalid.
func ValidateCookieSession(rt *Runtime, token string) bool {
	if token == "" {
		return false
	}
	return withCookieKV(rt, func(ctx context.Context, kv jetstream.KeyValue) error {
		_, err := kv.Get(ctx, token)
		return err
	}) == nil
}

// RevokeCookieSession deletes the cookie session token so it can no longer
// gate an upgrade or a viewer mint.
func RevokeCookieSession(rt *Runtime, token string) error {
	if token == "" {
		return nil
	}
	return withCookieKV(rt, func(ctx context.Context, kv jetstream.KeyValue) error {
		return kv.Delete(ctx, token)
	})
}
