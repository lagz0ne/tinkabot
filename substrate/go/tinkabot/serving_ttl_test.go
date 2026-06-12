package tinkabot

// TestServingCredsOutliveTheHour pins the serving-posture defect found live
// on 2026-06-12: every assembly-internal credential was minted with a 1h
// TTL, so at minute 60 the running binary's own store connections, ticker,
// and role creds files all expired and the shell served 502s. The binary's
// plane must outlive a real serving window; teardown revocation — not JWT
// expiry — is what ends minted authority with the process.

import (
	"os"
	"testing"
	"time"

	jwt "github.com/nats-io/jwt/v2"
)

func TestServingCredsOutliveTheHour(t *testing.T) {
	t.Parallel()
	app, err := boot(t, cfgFor(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	for _, role := range []string{RoleCaller, RoleObserver, RoleAuthor} {
		raw, err := os.ReadFile(app.CredsFile(role))
		if err != nil {
			t.Fatal(err)
		}
		token, err := jwt.ParseDecoratedJWT(raw)
		if err != nil {
			t.Fatal(err)
		}
		claims, err := jwt.DecodeUserClaims(token)
		if err != nil {
			t.Fatal(err)
		}
		if exp := time.Unix(claims.Expires, 0); time.Until(exp) < 24*time.Hour {
			t.Fatalf("%s creds expire within a serving day: %s", role, exp)
		}
	}
}
