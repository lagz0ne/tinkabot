package tinkabot

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestMultitenantTwoAppScopeIsolation(t *testing.T) {
	t.Parallel()
	store := t.TempDir()
	app, err := boot(t, cfgFor(store))
	if err != nil {
		t.Fatal(err)
	}

	demoAlice, err := app.AdmitParticipant("demo", "alice")
	if err != nil {
		t.Fatal(err)
	}
	demoBob, err := app.AdmitParticipant("demo", "bob")
	if err != nil {
		t.Fatal(err)
	}
	otherAlice, err := app.AdmitParticipant("other", "alice")
	if err != nil {
		t.Fatal(err)
	}
	otherBob, err := app.AdmitParticipant("other", "bob")
	if err != nil {
		t.Fatal(err)
	}

	owner := tinkaletEnv(t)
	mustTinkalet(t, owner, "profile", "import", "local", "--store", store, "--name", "owner")
	mustTinkalet(t, owner, "profile", "use", "owner")
	demoAliceEnv := importParticipant(t, demoAlice, "demo-alice")
	demoBobEnv := importParticipant(t, demoBob, "demo-bob")
	otherAliceEnv := importParticipant(t, otherAlice, "other-alice")
	otherBobEnv := importParticipant(t, otherBob, "other-bob")

	demoState := createScopedState(t, owner, "apps.demo.state.board", `{"app":"demo","turn":"alice"}`)
	otherState := createScopedState(t, owner, "apps.other.state.board", `{"app":"other","turn":"alice"}`)

	assertScopedRead(t, demoAliceEnv, demoState)
	assertScopedRead(t, demoBobEnv, demoState)
	assertScopedRead(t, otherAliceEnv, otherState)
	assertScopedRead(t, otherBobEnv, otherState)

	assertScopedAction(t, demoAliceEnv, "demo", "alice", "demo-a1", demoState, `{"move":"a1"}`)
	assertScopedAction(t, demoBobEnv, "demo", "bob", "demo-b1", demoState, `{"move":"b1"}`)
	assertScopedAction(t, otherAliceEnv, "other", "alice", "other-a1", otherState, `{"move":"a1"}`)
	assertScopedAction(t, otherBobEnv, "other", "bob", "other-b1", otherState, `{"move":"b1"}`)
	assertScopedWatch(t, demoAliceEnv, "apps.demo.state", demoState.Key, app, demoAlice, demoBob, otherAlice, otherBob)
	assertScopedWatch(t, demoBobEnv, "apps.demo.state", demoState.Key, app, demoAlice, demoBob, otherAlice, otherBob)
	assertScopedWatch(t, otherAliceEnv, "apps.other.state", otherState.Key, app, demoAlice, demoBob, otherAlice, otherBob)
	assertScopedWatch(t, otherBobEnv, "apps.other.state", otherState.Key, app, demoAlice, demoBob, otherAlice, otherBob)

	code, out, errOut := runTinkalet(demoAliceEnv, "item", "get", otherState.Key)
	assertParticipantDenied(t, code, out, errOut, "item apps.other.state.board denied get: denied-scope\n", app, demoAlice, otherAlice)
	code, out, errOut = runTinkalet(otherAliceEnv, "item", "get", demoState.Key)
	assertParticipantDenied(t, code, out, errOut, "item apps.demo.state.board denied get: denied-scope\n", app, otherAlice, demoAlice)
	code, out, errOut = runTinkalet(demoBobEnv, "item", "get", otherState.Key)
	assertParticipantDenied(t, code, out, errOut, "item apps.other.state.board denied get: denied-scope\n", app, demoBob, otherBob)
	code, out, errOut = runTinkalet(otherBobEnv, "item", "get", demoState.Key)
	assertParticipantDenied(t, code, out, errOut, "item apps.demo.state.board denied get: denied-scope\n", app, otherBob, demoBob)

	code, out, errOut = runTinkalet(demoAliceEnv, "watch", "prefix", "apps.other.state", "--limit", "1", "--timeout", "200ms", "--json")
	assertParticipantDenied(t, code, out, errOut, "watch apps.other.state denied prefix: denied-scope\n", app, demoAlice, otherAlice)
	code, out, errOut = runTinkalet(otherAliceEnv, "watch", "prefix", "apps.demo.state", "--limit", "1", "--timeout", "200ms", "--json")
	assertParticipantDenied(t, code, out, errOut, "watch apps.demo.state denied prefix: denied-scope\n", app, otherAlice, demoAlice)
	code, out, errOut = runTinkalet(demoBobEnv, "watch", "prefix", "apps.other.state", "--limit", "1", "--timeout", "200ms", "--json")
	assertParticipantDenied(t, code, out, errOut, "watch apps.other.state denied prefix: denied-scope\n", app, demoBob, otherBob)
	code, out, errOut = runTinkalet(otherBobEnv, "watch", "prefix", "apps.demo.state", "--limit", "1", "--timeout", "200ms", "--json")
	assertParticipantDenied(t, code, out, errOut, "watch apps.demo.state denied prefix: denied-scope\n", app, otherBob, demoBob)

	code, out, errOut = runTinkalet(demoAliceEnv, "action", "submit", "demo-cross", "--state", otherState.Key, "--base-revision", revString(otherState.Revision), "--value", `{"move":"x"}`)
	assertParticipantDenied(t, code, out, errOut, "action demo-cross denied submit: malformed-action\n", app, demoAlice, otherAlice)
	code, out, errOut = runTinkalet(otherAliceEnv, "action", "submit", "other-cross", "--state", demoState.Key, "--base-revision", revString(demoState.Revision), "--value", `{"move":"x"}`)
	assertParticipantDenied(t, code, out, errOut, "action other-cross denied submit: malformed-action\n", app, otherAlice, demoAlice)
	code, out, errOut = runTinkalet(demoBobEnv, "action", "submit", "demo-bob-cross", "--state", otherState.Key, "--base-revision", revString(otherState.Revision), "--value", `{"move":"x"}`)
	assertParticipantDenied(t, code, out, errOut, "action demo-bob-cross denied submit: malformed-action\n", app, demoBob, otherBob)
	code, out, errOut = runTinkalet(otherBobEnv, "action", "submit", "other-bob-cross", "--state", demoState.Key, "--base-revision", revString(demoState.Revision), "--value", `{"move":"x"}`)
	assertParticipantDenied(t, code, out, errOut, "action other-bob-cross denied submit: malformed-action\n", app, otherBob, demoBob)

	code, out, errOut = runTinkalet(demoAliceEnv, "trigger", "bundle.clock.tick", "--request-id", "multi-trigger-deny")
	assertParticipantDenied(t, code, out, errOut, "profile demo-alice denied bundle.clock.tick: denied-scope\n", app, demoAlice, otherAlice)
	code, out, errOut = runTinkalet(demoBobEnv, "trigger", "bundle.clock.tick", "--request-id", "multi-trigger-deny-demo-bob")
	assertParticipantDenied(t, code, out, errOut, "profile demo-bob denied bundle.clock.tick: denied-scope\n", app, demoBob, otherBob)
	code, out, errOut = runTinkalet(otherAliceEnv, "trigger", "bundle.clock.tick", "--request-id", "multi-trigger-deny-other-alice")
	assertParticipantDenied(t, code, out, errOut, "profile other-alice denied bundle.clock.tick: denied-scope\n", app, otherAlice, demoAlice)
	code, out, errOut = runTinkalet(otherBobEnv, "trigger", "bundle.clock.tick", "--request-id", "multi-trigger-deny-bob")
	assertParticipantDenied(t, code, out, errOut, "profile other-bob denied bundle.clock.tick: denied-scope\n", app, otherBob, demoBob)

	assertParticipantReadDenied(t, app, demoAlice, otherAlice.RecordKey)
	assertParticipantReadDenied(t, app, otherAlice, demoAlice.RecordKey)
	assertParticipantReadDenied(t, app, demoBob, otherBob.RecordKey)
	assertParticipantReadDenied(t, app, otherBob, demoBob.RecordKey)
	assertPublishDenied(t, app, demoAlice, "tb.app.other.participants.alice.action")
	assertPublishDenied(t, app, otherAlice, "tb.app.demo.participants.alice.action")
	assertPublishDenied(t, app, demoBob, "tb.app.other.participants.bob.action")
	assertPublishDenied(t, app, otherBob, "tb.app.demo.participants.bob.action")
	assertPublishDenied(t, app, demoAlice, "$KV."+wiring().ItemBucket+".apps.other.participants.alice.actions.raw")
	assertPublishDenied(t, app, otherAlice, "$KV."+wiring().ItemBucket+".apps.demo.participants.alice.actions.raw")
	assertPublishDenied(t, app, demoBob, "$KV."+wiring().ItemBucket+".apps.other.participants.bob.actions.raw")
	assertPublishDenied(t, app, otherBob, "$KV."+wiring().ItemBucket+".apps.demo.participants.bob.actions.raw")
	assertPublishDenied(t, app, demoAlice, "tb.bundle.clock.tick")
	assertPublishDenied(t, app, demoBob, "tb.bundle.clock.tick")
	assertPublishDenied(t, app, otherAlice, "tb.bundle.clock.tick")
	assertPublishDenied(t, app, otherBob, "tb.bundle.clock.tick")

	if err := app.RevokeParticipant(demoAlice); err != nil {
		t.Fatal(err)
	}
	assertParticipantRecord(t, app, demoAlice, "revoked")
	assertParticipantCredsDenied(t, app, mustReadFile(t, demoAlice.CredsFile))
	code, out, errOut = runTinkalet(demoAliceEnv, "action", "submit", "demo-after-revoke", "--state", demoState.Key, "--base-revision", revString(demoState.Revision), "--value", `{"move":"revoked"}`)
	assertParticipantDenied(t, code, out, errOut, "action demo-after-revoke denied submit: revoked-credentials\n", app, demoAlice, demoBob, otherAlice)

	assertScopedAction(t, demoBobEnv, "demo", "bob", "demo-bob-live", demoState, `{"move":"b2"}`)
	assertScopedAction(t, otherAliceEnv, "other", "alice", "other-after-demo-revoke", otherState, `{"move":"a2"}`)
	assertScopedAction(t, otherBobEnv, "other", "bob", "other-bob-after-demo-revoke", otherState, `{"move":"b2"}`)
	assertParticipantRecord(t, app, demoBob, "active")
	assertParticipantRecord(t, app, otherAlice, "active")
	assertParticipantRecord(t, app, otherBob, "active")
}

func TestMultitenantConcurrentRestartCatchUp(t *testing.T) {
	store := t.TempDir()
	app, err := boot(t, cfgFor(store))
	if err != nil {
		t.Fatal(err)
	}

	demoAlice, err := app.AdmitParticipant("demo", "alice")
	if err != nil {
		t.Fatal(err)
	}
	demoBob, err := app.AdmitParticipant("demo", "bob")
	if err != nil {
		t.Fatal(err)
	}
	otherAlice, err := app.AdmitParticipant("other", "alice")
	if err != nil {
		t.Fatal(err)
	}
	otherBob, err := app.AdmitParticipant("other", "bob")
	if err != nil {
		t.Fatal(err)
	}

	owner := tinkaletEnv(t)
	mustTinkalet(t, owner, "profile", "import", "local", "--store", store, "--name", "owner")
	mustTinkalet(t, owner, "profile", "use", "owner")
	parts := []isoParticipant{
		{name: "demo-alice", appID: "demo", participantID: "alice", prof: demoAlice, env: importParticipant(t, demoAlice, "demo-alice")},
		{name: "demo-bob", appID: "demo", participantID: "bob", prof: demoBob, env: importParticipant(t, demoBob, "demo-bob")},
		{name: "other-alice", appID: "other", participantID: "alice", prof: otherAlice, env: importParticipant(t, otherAlice, "other-alice")},
		{name: "other-bob", appID: "other", participantID: "bob", prof: otherBob, env: importParticipant(t, otherBob, "other-bob")},
	}
	for i := range parts {
		key := fmt.Sprintf("apps.%s.state.rate-%s", parts[i].appID, parts[i].participantID)
		val := fmt.Sprintf(`{"app":"%s","participant":"%s","seq":0}`, parts[i].appID, parts[i].participantID)
		parts[i].state = createScopedState(t, owner, key, val)
	}

	const preActions = 2
	const postActions = 2
	if err := submitISOActions(parts, 0, preActions); err != nil {
		t.Fatal(err)
	}
	for _, p := range parts {
		assertISOReplay(t, app, p, p.cursor(), 0, preActions, 0, isoProfiles(parts)...)
	}
	assertISODenials(t, app, parts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := app.Stop(ctx); err != nil {
		t.Fatal(err)
	}
	app2, err := boot(t, cfgFor(store))
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range parts {
		mustTinkalet(t, p.env, "profile", "import", "local", "--store", p.prof.StoreDir, "--name", p.name)
		mustTinkalet(t, p.env, "profile", "use", p.name)
	}

	if err := submitISOActions(parts, preActions, postActions); err != nil {
		t.Fatal(err)
	}
	assertISODenials(t, app2, parts)
	for _, p := range parts {
		assertISOReplay(t, app2, p, p.cursor(), preActions, postActions, 0, isoProfiles(parts)...)
		prefix := participantActionPrefix(p.appID, p.participantID)
		code, out, errOut := runTinkalet(p.env, "watch", "prefix", prefix, "--cursor", p.cursor(), "--limit", "1", "--timeout", "200ms", "--json")
		assertParticipantDenied(t, code, out, errOut, "watch "+prefix+" denied prefix: watch-timeout\n", app2, isoProfiles(parts)...)
	}

	if err := app2.RevokeParticipant(demoAlice); err != nil {
		t.Fatal(err)
	}
	assertParticipantRecord(t, app2, demoAlice, "revoked")
	code, out, errOut := runTinkalet(parts[0].env, "action", "submit", "iso-after-revoke", "--state", parts[0].state.Key, "--base-revision", revString(parts[0].state.Revision), "--value", `{"after":"revoke"}`)
	assertParticipantDenied(t, code, out, errOut, "action iso-after-revoke denied submit: revoked-credentials\n", app2, demoAlice, demoBob, otherAlice, otherBob)
	for _, p := range parts[1:] {
		id := "iso-live-" + p.appID + "-" + p.participantID
		payload := fmt.Sprintf(`{"app":"%s","participant":"%s","live":true}`, p.appID, p.participantID)
		code, out, errOut = runTinkalet(p.env, "action", "submit", id, "--state", p.state.Key, "--base-revision", revString(p.state.Revision), "--value", payload, "--json")
		if code != 0 || errOut != "" {
			t.Fatalf("%s live action exit/stdout/stderr = %d/%q/%q", p.name, code, out, errOut)
		}
		assertActionItem(t, decodeItem(t, out), p.appID, p.participantID, id, p.state.Key, p.state.Revision, payload)
	}
}

func importParticipant(t *testing.T, prof ParticipantProfile, name string) []string {
	t.Helper()
	env := tinkaletEnv(t)
	mustTinkalet(t, env, "profile", "import", "local", "--store", prof.StoreDir, "--name", name)
	mustTinkalet(t, env, "profile", "use", name)
	return env
}

type isoParticipant struct {
	name          string
	appID         string
	participantID string
	prof          ParticipantProfile
	env           []string
	state         itemView
}

func (p isoParticipant) cursor() string {
	return "iso-" + p.appID + "-" + p.participantID
}

func submitISOActions(parts []isoParticipant, start, count int) error {
	var wg sync.WaitGroup
	errs := make(chan error, len(parts))
	for _, p := range parts {
		p := p
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := start; i < start+count; i++ {
				if err := submitISOAction(p, i); err != nil {
					errs <- err
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

func submitISOAction(p isoParticipant, i int) error {
	id := isoActionID(p.appID, p.participantID, i)
	payload := isoActionPayload(p.appID, p.participantID, i)
	code, out, errOut := runTinkalet(p.env, "action", "submit", id, "--state", p.state.Key, "--base-revision", revString(p.state.Revision), "--value", payload, "--json")
	if code != 0 || errOut != "" {
		return fmt.Errorf("%s action %s exit/stdout/stderr = %d/%q/%q", p.name, id, code, out, errOut)
	}
	if isoRawLeak(out + errOut) {
		return fmt.Errorf("%s action %s leaked raw authority: %s%s", p.name, id, out, errOut)
	}
	var item itemView
	if err := json.Unmarshal([]byte(out), &item); err != nil {
		return fmt.Errorf("%s action %s json: %w in %q", p.name, id, err, out)
	}
	return validateISOAction(item, p, id, payload)
}

func validateISOAction(item itemView, p isoParticipant, actionID, payload string) error {
	wantKey := participantActionPrefix(p.appID, p.participantID) + "." + actionID
	if item.Key != wantKey || item.Status != "pending" || item.Revision == 0 {
		return fmt.Errorf("action item drift: %#v want key %s", item, wantKey)
	}
	var val struct {
		Kind          string          `json:"kind"`
		AppID         string          `json:"appId"`
		ParticipantID string          `json:"participantId"`
		ActionID      string          `json:"actionId"`
		StateKey      string          `json:"stateKey"`
		BaseRevision  uint64          `json:"baseRevision"`
		Payload       json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal(item.Value, &val); err != nil {
		return err
	}
	if val.Kind != "tinkabot.appAction.v1" ||
		val.AppID != p.appID ||
		val.ParticipantID != p.participantID ||
		val.ActionID != actionID ||
		val.StateKey != p.state.Key ||
		val.BaseRevision != p.state.Revision ||
		string(val.Payload) != payload {
		return fmt.Errorf("action value drift: %#v in %#v", val, item)
	}
	return nil
}

func assertISOReplay(t *testing.T, app *App, p isoParticipant, cursor string, start, count int, minRevision uint64, profiles ...ParticipantProfile) []watchEvent {
	t.Helper()
	target := participantActionPrefix(p.appID, p.participantID)
	code, out, errOut := runTinkalet(p.env, "watch", "prefix", target, "--cursor", cursor, "--limit", fmt.Sprint(count), "--timeout", "2s", "--json")
	if code != 0 || errOut != "" {
		t.Fatalf("%s replay watch exit/stdout/stderr = %d/%q/%q", p.name, code, out, errOut)
	}
	assertNoParticipantLeaks(t, out+errOut, app, profiles...)
	events, err := parseWatchEvents(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != count {
		t.Fatalf("%s replay count = %d, want %d: %#v", p.name, len(events), count, events)
	}
	prefix := participantActionPrefix(p.appID, p.participantID) + "."
	want := map[string]struct{}{}
	for i := 0; i < count; i++ {
		want[isoActionID(p.appID, p.participantID, start+i)] = struct{}{}
	}
	seen := map[string]struct{}{}
	prev := minRevision
	for _, ev := range events {
		if !strings.HasPrefix(ev.Key, prefix) || ev.Status != "pending" || ev.Source != "replay" {
			t.Fatalf("%s replay drift: %#v want prefix=%s pending replay", p.name, ev, prefix)
		}
		id := strings.TrimPrefix(ev.Key, prefix)
		if _, ok := want[id]; !ok {
			t.Fatalf("%s replay saw unexpected action %s in %#v", p.name, id, events)
		}
		if _, ok := seen[id]; ok {
			t.Fatalf("%s replay saw duplicate action %s in %#v", p.name, id, events)
		}
		seen[id] = struct{}{}
		if ev.Revision <= prev {
			t.Fatalf("%s replay revisions not strict after %d: %#v", p.name, minRevision, events)
		}
		prev = ev.Revision
	}
	for id := range want {
		if _, ok := seen[id]; !ok {
			t.Fatalf("%s replay missing action %s in %#v", p.name, id, events)
		}
	}
	return events
}

func assertISODenials(t *testing.T, app *App, parts []isoParticipant) {
	t.Helper()
	byScope := map[string]isoParticipant{}
	for _, p := range parts {
		byScope[p.appID+":"+p.participantID] = p
	}
	for _, p := range parts {
		otherApp := "other"
		if p.appID == "other" {
			otherApp = "demo"
		}
		other := byScope[otherApp+":"+p.participantID]
		neighborID := "bob"
		if p.participantID == "bob" {
			neighborID = "alice"
		}
		code, out, errOut := runTinkalet(p.env, "item", "get", other.state.Key)
		assertParticipantDenied(t, code, out, errOut, "item "+other.state.Key+" denied get: denied-scope\n", app, isoProfiles(parts)...)
		code, out, errOut = runTinkalet(p.env, "watch", "prefix", "apps."+otherApp+".state", "--limit", "1", "--timeout", "200ms", "--json")
		assertParticipantDenied(t, code, out, errOut, "watch apps."+otherApp+".state denied prefix: denied-scope\n", app, isoProfiles(parts)...)
		code, out, errOut = runTinkalet(p.env, "watch", "prefix", participantActionPrefix(p.appID, neighborID), "--limit", "1", "--timeout", "200ms", "--json")
		assertParticipantDenied(t, code, out, errOut, "watch "+participantActionPrefix(p.appID, neighborID)+" denied prefix: denied-scope\n", app, isoProfiles(parts)...)
		crossID := "iso-cross-" + p.appID + "-" + p.participantID
		code, out, errOut = runTinkalet(p.env, "action", "submit", crossID, "--state", other.state.Key, "--base-revision", revString(other.state.Revision), "--value", `{"cross":true}`)
		assertParticipantDenied(t, code, out, errOut, "action "+crossID+" denied submit: malformed-action\n", app, isoProfiles(parts)...)
		code, out, errOut = runTinkalet(p.env, "item", "create", participantActionPrefix(p.appID, p.participantID)+".raw", "--value", `{"bypass":true}`)
		assertParticipantDenied(t, code, out, errOut, "item "+participantActionPrefix(p.appID, p.participantID)+".raw denied create: denied-scope\n", app, isoProfiles(parts)...)
		code, out, errOut = runTinkalet(p.env, "trigger", "bundle.clock.tick", "--request-id", "iso-"+p.appID+"-"+p.participantID)
		assertParticipantDenied(t, code, out, errOut, "profile "+p.name+" denied bundle.clock.tick: denied-scope\n", app, isoProfiles(parts)...)
		assertParticipantReadDenied(t, app, p.prof, other.prof.RecordKey)
		assertPublishDenied(t, app, p.prof, "tb.app."+otherApp+".participants."+p.participantID+".action")
		assertPublishDenied(t, app, p.prof, "$KV."+wiring().ItemBucket+".apps."+otherApp+".participants."+p.participantID+".actions.raw")
		assertPublishDenied(t, app, p.prof, "tb.bundle.clock.tick")
	}
}

func isoProfiles(parts []isoParticipant) []ParticipantProfile {
	out := make([]ParticipantProfile, 0, len(parts))
	for _, p := range parts {
		out = append(out, p.prof)
	}
	return out
}

func isoActionID(appID, participantID string, i int) string {
	return "iso-" + appID + "-" + participantID + "-" + fmt.Sprint(i)
}

func isoActionPayload(appID, participantID string, i int) string {
	return fmt.Sprintf(`{"app":"%s","participant":"%s","seq":%d}`, appID, participantID, i)
}

func isoRawLeak(text string) bool {
	lower := strings.ToLower(text)
	for _, token := range []string{"tb_items", "$kv", "$js.api", "begin nats", "private key", ".creds", "credential", "jwt", "nkey", "bearer", "token", "nats://", "tb.app.", "tb.bundle."} {
		if strings.Contains(lower, token) {
			return true
		}
	}
	return false
}

func createScopedState(t *testing.T, env []string, key, value string) itemView {
	t.Helper()
	code, out, errOut := runTinkalet(env, "item", "create", key, "--value", value, "--json")
	if code != 0 || errOut != "" {
		t.Fatalf("state create %s exit/stdout/stderr = %d/%q/%q", key, code, out, errOut)
	}
	return decodeItem(t, out)
}

func assertScopedRead(t *testing.T, env []string, want itemView) {
	t.Helper()
	code, out, errOut := runTinkalet(env, "item", "get", want.Key, "--json")
	if code != 0 || errOut != "" {
		t.Fatalf("state read %s exit/stdout/stderr = %d/%q/%q", want.Key, code, out, errOut)
	}
	got := decodeItem(t, out)
	if got.Key != want.Key || got.Revision != want.Revision || string(got.Value) != string(want.Value) {
		t.Fatalf("state read drift: got %#v want %#v", got, want)
	}
}

func assertScopedAction(t *testing.T, env []string, appID, participantID, actionID string, state itemView, value string) itemView {
	t.Helper()
	code, out, errOut := runTinkalet(env, "action", "submit", actionID, "--state", state.Key, "--base-revision", revString(state.Revision), "--value", value, "--json")
	if code != 0 || errOut != "" {
		t.Fatalf("action %s exit/stdout/stderr = %d/%q/%q", actionID, code, out, errOut)
	}
	item := decodeItem(t, out)
	assertActionItem(t, item, appID, participantID, actionID, state.Key, state.Revision, value)
	return item
}

func assertScopedWatch(t *testing.T, env []string, prefix, key string, app *App, profiles ...ParticipantProfile) {
	t.Helper()
	code, out, errOut := runTinkalet(env, "watch", "prefix", prefix, "--limit", "1", "--timeout", "2s", "--json")
	if code != 0 || errOut != "" {
		t.Fatalf("watch %s exit/stdout/stderr = %d/%q/%q", prefix, code, out, errOut)
	}
	events, err := parseWatchEvents(out)
	if err != nil {
		t.Fatal(err)
	}
	assertWatchKeys(t, events, key, 1)
	assertNoParticipantLeaks(t, out+errOut, app, profiles...)
}
