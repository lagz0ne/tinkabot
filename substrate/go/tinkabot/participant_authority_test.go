package tinkabot

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
)

func TestParticipantAuthority(t *testing.T) {
	t.Parallel()
	store := t.TempDir()
	app, err := boot(t, cfgFor(store))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := app.AdmitParticipant("Demo", "alice"); err == nil {
		t.Fatal("uppercase app token was accepted")
	}
	if _, err := app.AdmitParticipant("demo", "Alice"); err == nil {
		t.Fatal("uppercase participant token was accepted")
	}

	alice, err := app.AdmitParticipant("demo", "alice")
	if err != nil {
		t.Fatal(err)
	}
	bob, err := app.AdmitParticipant("demo", "bob")
	if err != nil {
		t.Fatal(err)
	}
	oldAlice := alice
	oldAlicePub := alice.UserPub
	oldAliceCreds := append([]byte(nil), mustReadFile(t, alice.CredsFile)...)
	alice, err = app.AdmitParticipant("demo", "alice")
	if err != nil {
		t.Fatal(err)
	}
	if alice.UserPub == oldAlicePub {
		t.Fatalf("participant rotation reused user pub %q", alice.UserPub)
	}
	assertParticipantCredsDenied(t, app, oldAliceCreds)
	if err := app.RevokeParticipant(oldAlice); err != nil {
		t.Fatal(err)
	}
	assertParticipantRecord(t, app, alice, "active")

	owner, aliceEnv, bobEnv := tinkaletEnv(t), tinkaletEnv(t), tinkaletEnv(t)
	mustTinkalet(t, owner, "profile", "import", "local", "--store", store, "--name", "owner")
	mustTinkalet(t, owner, "profile", "use", "owner")
	mustTinkalet(t, aliceEnv, "profile", "import", "local", "--store", alice.StoreDir, "--name", "alice")
	mustTinkalet(t, aliceEnv, "profile", "use", "alice")
	mustTinkalet(t, bobEnv, "profile", "import", "local", "--store", bob.StoreDir, "--name", "bob")
	mustTinkalet(t, bobEnv, "profile", "use", "bob")

	code, out, errOut := runTinkalet(aliceEnv, "profile", "list")
	if code != 0 || errOut != "" || !strings.Contains(out, "* alice participant app-participant\n") {
		t.Fatalf("participant profile list = %d/%q/%q", code, out, errOut)
	}
	assertNoParticipantLeaks(t, out+errOut, app, alice, bob)
	assertParticipantRecord(t, app, alice, "active")

	code, out, errOut = runTinkalet(owner, "item", "create", "apps.demo.state.authority", "--value", `{"turn":"alice"}`, "--json")
	if code != 0 || errOut != "" {
		t.Fatalf("owner state create exit/stderr = %d/%q", code, errOut)
	}
	state := decodeItem(t, out)
	mustTinkalet(t, aliceEnv, "action", "submit", "intent-1", "--state", "apps.demo.state.authority", "--base-revision", revString(state.Revision), "--value", `{"intent":"alpha"}`)
	assertParticipantReadDenied(t, app, alice, bob.RecordKey)
	code, out, errOut = runTinkalet(aliceEnv, "item", "create", "apps.demo.participants.bob.actions.intent-1", "--value", `{"intent":"cross-scope"}`)
	assertParticipantDenied(t, code, out, errOut, "item apps.demo.participants.bob.actions.intent-1 denied create: denied-scope\n", app, alice, bob)
	code, out, errOut = runTinkalet(aliceEnv, "item", "create", "apps.demo.participants.alice.actions.direct", "--value", `{"intent":"direct"}`)
	assertParticipantDenied(t, code, out, errOut, "item apps.demo.participants.alice.actions.direct denied create: denied-scope\n", app, alice, bob)

	assertPublishDenied(t, app, alice, "$KV."+wiring().ItemBucket+".apps.demo.participants.alice.actions.raw")
	assertPublishDenied(t, app, alice, "tb.app.demo.participants.bob.action")
	assertPublishDenied(t, app, alice, "$KV."+wiring().ItemBucket+".apps.other.participants.alice.actions.raw")
	assertPublishDenied(t, app, alice, "$KV."+wiring().ConfigBucket+".participant.raw")
	assertPublishDenied(t, app, alice, "$KV."+wiring().ScheduleBucket+".participant.raw")

	if err := app.RevokeParticipant(alice); err != nil {
		t.Fatal(err)
	}
	assertParticipantRecord(t, app, alice, "revoked")
	assertParticipantCredsDenied(t, app, mustReadFile(t, alice.CredsFile))
	code, out, errOut = runTinkalet(aliceEnv, "action", "submit", "intent-2", "--state", "apps.demo.state.authority", "--base-revision", revString(state.Revision), "--value", `{"intent":"after-revoke"}`)
	assertParticipantDenied(t, code, out, errOut, "action intent-2 denied submit: revoked-credentials\n", app, alice, bob)
	mustTinkalet(t, bobEnv, "action", "submit", "intent-2", "--state", "apps.demo.state.authority", "--base-revision", revString(state.Revision), "--value", `{"intent":"still-live"}`)
}

func assertParticipantRecord(t *testing.T, app *App, prof ParticipantProfile, status string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	nc, err := app.Runtime().ConnectCreds(ctx, app.Creds(RoleCaller).File)
	if err != nil {
		t.Fatal(err)
	}
	defer nc.Close()
	js, err := nc.JetStream()
	if err != nil {
		t.Fatal(err)
	}
	kv, err := js.KeyValue(wiring().ItemBucket)
	if err != nil {
		t.Fatal(err)
	}
	entry, err := kv.Get(prof.RecordKey)
	if err != nil {
		t.Fatal(err)
	}
	var rec participantRecord
	if err := json.Unmarshal(entry.Value(), &rec); err != nil {
		t.Fatal(err)
	}
	if rec.Kind != "tinkabot.participant.v1" ||
		rec.AppID != prof.AppID ||
		rec.ParticipantID != prof.ParticipantID ||
		rec.UserPub != prof.UserPub ||
		rec.LeaseID != prof.LeaseID ||
		rec.Status != status {
		t.Fatalf("participant record drift: %#v for %#v", rec, prof)
	}
	if status == "revoked" && rec.RevokedAt == "" {
		t.Fatalf("revoked participant missing audit time: %#v", rec)
	}
}

func assertParticipantCredsDenied(t *testing.T, app *App, creds []byte) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	nc, err := app.Runtime().ConnectCreds(ctx, creds)
	if err == nil {
		nc.Close()
		t.Fatal("revoked participant creds connected")
	}
	assertNoParticipantLeaks(t, err.Error(), app)
}

func assertParticipantReadDenied(t *testing.T, app *App, prof ParticipantProfile, key string) {
	t.Helper()
	errs := make(chan error, 1)
	nc, err := nats.Connect(
		app.Posture().NATS.ClientURL,
		nats.UserCredentials(prof.CredsFile),
		nats.NoReconnect(),
		nats.Timeout(2*time.Second),
		nats.ErrorHandler(func(_ *nats.Conn, _ *nats.Subscription, err error) {
			select {
			case errs <- err:
			default:
			}
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer nc.Close()
	js, err := nc.JetStream()
	if err != nil {
		t.Fatal(err)
	}
	kv, err := js.KeyValue(wiring().ItemBucket)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := kv.Get(key); err == nil {
		t.Fatalf("participant read %s was not denied", key)
	} else {
		assertNoParticipantLeaks(t, err.Error(), app, prof)
	}
	select {
	case err := <-errs:
		if !permissionError(err) {
			t.Fatalf("participant read denial for %s = %v", key, err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("participant read %s did not surface a NATS permission error", key)
	}
}

func assertPublishDenied(t *testing.T, app *App, prof ParticipantProfile, subj string) {
	t.Helper()
	errs := make(chan error, 1)
	nc, err := nats.Connect(
		app.Posture().NATS.ClientURL,
		nats.UserCredentials(prof.CredsFile),
		nats.NoReconnect(),
		nats.Timeout(2*time.Second),
		nats.ErrorHandler(func(_ *nats.Conn, _ *nats.Subscription, err error) {
			select {
			case errs <- err:
			default:
			}
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer nc.Close()
	if err := nc.Publish(subj, []byte(`{"bad":true}`)); err != nil {
		return
	}
	_ = nc.Flush()
	select {
	case err := <-errs:
		if !permissionError(err) {
			t.Fatalf("raw publish denial = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("raw publish to %s was not denied", subj)
	}
}

func permissionError(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "permission") || strings.Contains(msg, "authorization")
}

func assertParticipantDenied(t *testing.T, code int, out, errOut, want string, app *App, profiles ...ParticipantProfile) {
	t.Helper()
	if code != 1 || out != "" || errOut != want {
		t.Fatalf("exit/stdout/stderr = %d/%q/%q, want 1/empty/%q", code, out, errOut, want)
	}
	assertNoParticipantLeaks(t, out+errOut, app, profiles...)
}

func assertNoParticipantLeaks(t *testing.T, text string, app *App, profiles ...ParticipantProfile) {
	t.Helper()
	for _, leak := range []string{"tb_items", "$KV", "caller.creds", string(mustReadFile(t, app.CredsFile(RoleCaller)))} {
		if strings.Contains(text, leak) {
			t.Fatalf("participant output leaked %q: %s", leak, text)
		}
	}
	for _, prof := range profiles {
		if prof.CredsFile != "" && strings.Contains(text, string(mustReadFile(t, prof.CredsFile))) {
			t.Fatalf("participant output leaked profile creds: %s", text)
		}
	}
}
