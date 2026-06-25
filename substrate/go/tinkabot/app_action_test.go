package tinkabot

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/lagz0ne/tinkabot/substrate/go/embednats"
)

func TestAppActionMalformedSubject(t *testing.T) {
	t.Parallel()
	for _, subj := range []string{
		"tb.app.demo.participants.alice.action.extra",
		"tb.app.Demo.participants.alice.action",
		"tb.app.demo.participants.Alice.action",
		"tb.app.demo.participants.alice.nope",
	} {
		resp := handleAppAction(nil, &nats.Msg{Subject: subj, Data: []byte(`{"actionId":"move-1"}`)})
		if resp.Status != "denied" || resp.Reason != "malformed-action" {
			t.Fatalf("malformed subject %q = %#v", subj, resp)
		}
	}
}

func TestParticipantAppActions(t *testing.T) {
	t.Parallel()
	store := t.TempDir()
	app, err := boot(t, cfgFor(store))
	if err != nil {
		t.Fatal(err)
	}
	alice, err := app.AdmitParticipant("demo", "alice")
	if err != nil {
		t.Fatal(err)
	}
	bob, err := app.AdmitParticipant("demo", "bob")
	if err != nil {
		t.Fatal(err)
	}

	owner, aliceEnv, bobEnv := tinkaletEnv(t), tinkaletEnv(t), tinkaletEnv(t)
	mustTinkalet(t, owner, "profile", "import", "local", "--store", store, "--name", "owner")
	mustTinkalet(t, owner, "profile", "use", "owner")
	mustTinkalet(t, aliceEnv, "profile", "import", "local", "--store", alice.StoreDir, "--name", "alice")
	mustTinkalet(t, aliceEnv, "profile", "use", "alice")
	mustTinkalet(t, bobEnv, "profile", "import", "local", "--store", bob.StoreDir, "--name", "bob")
	mustTinkalet(t, bobEnv, "profile", "use", "bob")

	code, out, errOut := runTinkalet(owner, "item", "create", "apps.demo.state.board", "--value", `{"turn":"alice","cells":[]}`, "--json")
	if code != 0 || errOut != "" {
		t.Fatalf("owner state create exit/stderr = %d/%q", code, errOut)
	}
	state := decodeItem(t, out)

	code, out, errOut = runTinkalet(aliceEnv, "item", "get", "apps.demo.state.board", "--json")
	if code != 0 || errOut != "" {
		t.Fatalf("participant state read exit/stderr = %d/%q", code, errOut)
	}
	if got := decodeItem(t, out); got.Key != "apps.demo.state.board" || got.Revision != state.Revision {
		t.Fatalf("participant state read drift: %#v", got)
	}
	code, out, errOut = runTinkalet(aliceEnv, "item", "get", "apps.other.state.board")
	assertParticipantDenied(t, code, out, errOut, "item apps.other.state.board denied get: denied-scope\n", app, alice, bob)

	code, out, errOut = runTinkalet(aliceEnv, "action", "submit", "move-other-state", "--state", "apps.other.state.board", "--base-revision", revString(state.Revision), "--value", `{"cell":"x1"}`)
	assertParticipantDenied(t, code, out, errOut, "action move-other-state denied submit: malformed-action\n", app, alice, bob)

	code, out, errOut = runTinkalet(aliceEnv, "action", "submit", "move-missing-state", "--state", "apps.demo.state.missing", "--base-revision", "1", "--value", `{"cell":"x2"}`)
	assertParticipantDenied(t, code, out, errOut, "action move-missing-state denied submit: item-not-found\n", app, alice, bob)

	code, out, errOut = runTinkalet(aliceEnv, "action", "submit", "move-1", "--state", "apps.demo.state.board", "--base-revision", revString(state.Revision), "--value", `{"cell":"a1"}`, "--json")
	if code != 0 || errOut != "" {
		t.Fatalf("action submit exit/stderr = %d/%q", code, errOut)
	}
	action := decodeItem(t, out)
	assertActionItem(t, action, "demo", "alice", "move-1", "apps.demo.state.board", state.Revision, `{"cell":"a1"}`)
	assertNoParticipantLeaks(t, out+errOut, app, alice, bob)

	code, out, errOut = runTinkalet(aliceEnv, "item", "get", "apps.demo.participants.alice.actions.move-1", "--json")
	if code != 0 || errOut != "" {
		t.Fatalf("participant action read exit/stderr = %d/%q", code, errOut)
	}
	assertActionItem(t, decodeItem(t, out), "demo", "alice", "move-1", "apps.demo.state.board", state.Revision, `{"cell":"a1"}`)

	code, out, errOut = runTinkalet(aliceEnv, "action", "submit", "move-1", "--state", "apps.demo.state.board", "--base-revision", revString(state.Revision), "--value", `{"cell":"a1"}`)
	assertParticipantDenied(t, code, out, errOut, "action move-1 denied submit: duplicate-action\n", app, alice, bob)

	code, out, errOut = runTinkalet(aliceEnv, "item", "create", "apps.demo.participants.alice.actions.raw", "--value", `{"bypass":true}`)
	assertParticipantDenied(t, code, out, errOut, "item apps.demo.participants.alice.actions.raw denied create: denied-scope\n", app, alice, bob)
	assertPublishDenied(t, app, alice, "$KV."+wiring().ItemBucket+".apps.demo.participants.alice.actions.raw")
	assertPublishDenied(t, app, alice, "tb.app.other.participants.alice.action")

	code, out, errOut = runTinkalet(owner, "item", "resolve", "apps.demo.state.board", "--revision", revString(state.Revision), "--value", `{"turn":"bob","cells":["a1"]}`, "--json")
	if code != 0 || errOut != "" {
		t.Fatalf("owner state resolve exit/stderr = %d/%q", code, errOut)
	}
	advanced := decodeItem(t, out)

	code, out, errOut = runTinkalet(aliceEnv, "action", "submit", "move-2", "--state", "apps.demo.state.board", "--base-revision", revString(state.Revision), "--value", `{"cell":"b1"}`)
	assertParticipantDenied(t, code, out, errOut, "action move-2 denied submit: stale-revision\n", app, alice, bob)
	code, out, errOut = runTinkalet(owner, "item", "get", "apps.demo.participants.alice.actions.move-2")
	if code != 1 || out != "" || errOut != "item apps.demo.participants.alice.actions.move-2 denied get: item-not-found\n" {
		t.Fatalf("stale action materialized: exit/stdout/stderr = %d/%q/%q", code, out, errOut)
	}

	code, out, errOut = runTinkalet(aliceEnv, "item", "resolve", "apps.demo.state.board", "--revision", revString(advanced.Revision), "--value", `{"bad":true}`)
	assertParticipantDenied(t, code, out, errOut, "item apps.demo.state.board denied resolve: denied-scope\n", app, alice, bob)

	code, out, errOut = runTinkalet(bobEnv, "action", "submit", "move-1", "--state", "apps.demo.state.board", "--base-revision", revString(advanced.Revision), "--value", `{"cell":"b2"}`, "--json")
	if code != 0 || errOut != "" {
		t.Fatalf("bob action submit exit/stderr = %d/%q", code, errOut)
	}
	assertActionItem(t, decodeItem(t, out), "demo", "bob", "move-1", "apps.demo.state.board", advanced.Revision, `{"cell":"b2"}`)

	if strings.Contains(out, "tb_items") || strings.Contains(out, "$KV") {
		t.Fatalf("action output leaked substrate details: %q", out)
	}
}

func TestBrowserParticipantActionBridge(t *testing.T) {
	t.Parallel()
	store := t.TempDir()
	app, err := boot(t, cfgFor(store))
	if err != nil {
		t.Fatal(err)
	}
	alice, err := app.AdmitParticipant("demo", "alice")
	if err != nil {
		t.Fatal(err)
	}
	bob, err := app.AdmitParticipant("demo", "bob")
	if err != nil {
		t.Fatal(err)
	}

	owner, aliceEnv := tinkaletEnv(t), tinkaletEnv(t)
	mustTinkalet(t, owner, "profile", "import", "local", "--store", store, "--name", "owner")
	mustTinkalet(t, owner, "profile", "use", "owner")
	mustTinkalet(t, aliceEnv, "profile", "import", "local", "--store", alice.StoreDir, "--name", "alice")
	mustTinkalet(t, aliceEnv, "profile", "use", "alice")

	code, out, errOut := runTinkalet(owner, "item", "create", "apps.demo.state.browser", "--value", `{"seq":0}`, "--json")
	if code != 0 || errOut != "" {
		t.Fatalf("state create exit/stderr = %d/%q", code, errOut)
	}
	state := decodeItem(t, out)

	browser, grant := browserGrantConn(t, app)
	defer browser.Close()
	raw := requestBrowserCommand(t, browser, browserCommand("participant_action", "cmd-browser-1", "demo", "alice", map[string]any{
		"actionId":     "browser-1",
		"stateKey":     "apps.demo.state.browser",
		"baseRevision": state.Revision,
		"value":        map[string]any{"seq": 1},
	}))
	resp := decodeBrowserActionResp(t, raw)
	if resp.Status != "accepted" || resp.Item == nil {
		t.Fatalf("browser action response drift: %s", raw)
	}
	assertActionItem(t, browserItemView(*resp.Item), "demo", "alice", "browser-1", "apps.demo.state.browser", state.Revision, `{"seq":1}`)
	assertNoParticipantLeaks(t, string(raw), app, alice, bob)

	actionRead := decodeBrowserActionResp(t, requestBrowserCommand(t, browser, browserCommand("participant_read", "cmd-read-action", "demo", "alice", map[string]any{
		"key": "apps.demo.participants.alice.actions.browser-1",
	})))
	if actionRead.Status != "accepted" || actionRead.Item == nil {
		t.Fatalf("browser action readback response drift: %#v", actionRead)
	}
	assertActionItem(t, browserItemView(*actionRead.Item), "demo", "alice", "browser-1", "apps.demo.state.browser", state.Revision, `{"seq":1}`)

	stateRead := decodeBrowserActionResp(t, requestBrowserCommand(t, browser, browserCommand("participant_read", "cmd-read-state", "demo", "alice", map[string]any{
		"key": "apps.demo.state.browser",
	})))
	if stateRead.Status != "accepted" || stateRead.Item == nil || stateRead.Item.Key != "apps.demo.state.browser" || stateRead.Item.Revision != state.Revision {
		t.Fatalf("browser state readback response drift: %#v", stateRead)
	}

	ctx := browserCommand("participant_watch", "cmd-watch-state", "demo", "alice", map[string]any{
		"key":      "apps.demo.state.browser",
		"delivery": grant.StateSubject,
	})["context"].(map[string]any)
	wantSubject, ok := browserStateSubject(grant.StateSubject, browserCommandContext{
		SessionID:        ctx["sessionId"].(string),
		CapabilityID:     ctx["capabilityId"].(string),
		ArtifactID:       ctx["artifactId"].(string),
		ArtifactRevision: ctx["artifactRevision"].(string),
		FrameID:          ctx["frameId"].(string),
		AppID:            ctx["appId"].(string),
		ParticipantID:    ctx["participantId"].(string),
		Chain: chainCtx{
			ChainID: ctx["chain"].(map[string]any)["chainId"].(string),
			RootID:  ctx["chain"].(map[string]any)["rootId"].(string),
			Hop:     ctx["chain"].(map[string]any)["hop"].(int),
			MaxHops: ctx["chain"].(map[string]any)["maxHops"].(int),
		},
	}, "apps.demo.state.browser")
	if !ok {
		t.Fatal("viewer state subject prefix rejected")
	}
	watched := make(chan browserStateEvent, 4)
	if _, err := browser.Subscribe(wantSubject, func(m *nats.Msg) {
		var ev browserStateEvent
		if json.Unmarshal(m.Data, &ev) == nil {
			watched <- ev
		}
	}); err != nil {
		t.Fatal(err)
	}
	if err := browser.Flush(); err != nil {
		t.Fatal(err)
	}
	watchResp := decodeBrowserActionResp(t, requestBrowserCommand(t, browser, browserCommand("participant_watch", "cmd-watch-state", "demo", "alice", map[string]any{
		"key":      "apps.demo.state.browser",
		"delivery": grant.StateSubject,
	})))
	if watchResp.Status != "accepted" || watchResp.DeliverySubject != wantSubject {
		t.Fatalf("browser watch response drift: %#v want %s", watchResp, wantSubject)
	}
	assertBrowserStateEvent(t, watched, "apps.demo.state.browser", state.Revision, `{"seq":0}`)
	missingDelivery := decodeBrowserActionResp(t, requestBrowserCommand(t, browser, browserCommand("participant_watch", "cmd-watch-missing-delivery", "demo", "alice", map[string]any{
		"key": "apps.demo.state.browser",
	})))
	if missingDelivery.Status != "denied" || missingDelivery.Reason != "malformed-action" {
		t.Fatalf("browser watch without minted delivery prefix drift: %#v", missingDelivery)
	}

	neighbor := decodeBrowserActionResp(t, requestBrowserCommand(t, browser, browserCommand("participant_read", "cmd-read-neighbor", "demo", "alice", map[string]any{
		"key": "apps.demo.participants.bob.actions.nope",
	})))
	if neighbor.Status != "denied" || neighbor.Reason != "denied-scope" {
		t.Fatalf("browser neighbor read response drift: %#v", neighbor)
	}

	dup := decodeBrowserActionResp(t, requestBrowserCommand(t, browser, browserCommand("participant_action", "cmd-browser-dup", "demo", "alice", map[string]any{
		"actionId":     "browser-1",
		"stateKey":     "apps.demo.state.browser",
		"baseRevision": state.Revision,
		"value":        map[string]any{"seq": 1},
	})))
	if dup.Status != "denied" || dup.Reason != "duplicate-action" {
		t.Fatalf("browser duplicate response drift: %#v", dup)
	}

	code, out, errOut = runTinkalet(owner, "item", "resolve", "apps.demo.state.browser", "--revision", revString(state.Revision), "--value", `{"seq":2}`, "--json")
	if code != 0 || errOut != "" {
		t.Fatalf("owner state resolve exit/stderr = %d/%q", code, errOut)
	}
	advanced := decodeItem(t, out)
	assertBrowserStateEvent(t, watched, "apps.demo.state.browser", advanced.Revision, `{"seq":2}`)
	stale := decodeBrowserActionResp(t, requestBrowserCommand(t, browser, browserCommand("participant_action", "cmd-browser-stale", "demo", "alice", map[string]any{
		"actionId":     "browser-stale",
		"stateKey":     "apps.demo.state.browser",
		"baseRevision": state.Revision,
		"value":        map[string]any{"seq": 3},
	})))
	if stale.Status != "denied" || stale.Reason != "stale-revision" || advanced.Revision <= state.Revision {
		t.Fatalf("browser stale response drift: %#v after %#v", stale, advanced)
	}

	code, out, errOut = runTinkalet(aliceEnv, "item", "get", "apps.demo.participants.alice.actions.browser-stale")
	if code != 1 || out != "" || errOut != "item apps.demo.participants.alice.actions.browser-stale denied get: item-not-found\n" {
		t.Fatalf("stale browser action materialized: exit/stdout/stderr = %d/%q/%q", code, out, errOut)
	}

	denied := decodeBrowserActionResp(t, requestBrowserCommand(t, browser, browserCommand("participant_action", "cmd-browser-bob", "demo", "alice", map[string]any{
		"actionId":      "bob-escape",
		"stateKey":      "apps.demo.state.browser",
		"baseRevision":  state.Revision,
		"participantId": "bob",
		"value":         map[string]any{"seq": 2},
	})))
	if denied.Status != "denied" || denied.Reason != "denied-scope" {
		t.Fatalf("browser wrong participant response drift: %#v", denied)
	}

	rawAuthority := decodeBrowserActionResp(t, requestBrowserCommand(t, browser, browserCommand("participant_action", "cmd-browser-raw", "demo", "alice", map[string]any{
		"actionId":     "browser-raw",
		"stateKey":     "apps.demo.state.browser",
		"baseRevision": state.Revision,
		"natsSubject":  "tb.internal.admin.delete",
		"password":     "not-allowed",
		"value":        map[string]any{"seq": 2},
	})))
	if rawAuthority.Status != "denied" || rawAuthority.Reason != "raw-authority" {
		t.Fatalf("browser raw authority response drift: %#v", rawAuthority)
	}

	unknown := decodeBrowserActionResp(t, requestBrowserCommand(t, browser, browserCommand("participant_delete", "cmd-browser-unknown", "demo", "alice", map[string]any{
		"key": "apps.demo.participants.alice.actions.browser-1",
	})))
	if unknown.Status != "denied" || unknown.Reason != "unknown-command" {
		t.Fatalf("browser unknown command response drift: %#v", unknown)
	}

	missingCtx := browserCommand("participant_read", "cmd-browser-missing-context", "demo", "alice", map[string]any{
		"key": "apps.demo.state.browser",
	})
	delete(missingCtx["context"].(map[string]any), "frameId")
	missing := decodeBrowserActionResp(t, requestBrowserCommand(t, browser, missingCtx))
	if missing.Status != "denied" || missing.Reason != "malformed-action" {
		t.Fatalf("browser missing context response drift: %#v", missing)
	}
}

func TestBrowserItemSubmitBridge(t *testing.T) {
	t.Parallel()
	store := t.TempDir()
	app, err := boot(t, cfgFor(store))
	if err != nil {
		t.Fatal(err)
	}
	owner := tinkaletEnv(t)
	mustTinkalet(t, owner, "profile", "import", "local", "--store", store, "--name", "owner")
	mustTinkalet(t, owner, "profile", "use", "owner")

	browser := browserConn(t, app)
	defer browser.Close()

	key := "artifacts.artifact-browser.results.choice"
	raw := requestBrowserCommand(t, browser, browserItemCommand("item_submit", "cmd-visual-1", map[string]any{
		"key":   key,
		"value": map[string]any{"choice": "diagram-a"},
	}))
	resp := decodeBrowserActionResp(t, raw)
	if resp.Status != "accepted" || resp.Item == nil || resp.Item.Key != key || resp.Item.Status != "resolved" || string(resp.Item.Value) != `{"choice":"diagram-a"}` {
		t.Fatalf("browser item submit response drift: %s", raw)
	}
	assertNoBrowserLeaks(t, string(raw), app)

	code, out, errOut := runTinkalet(owner, "item", "get", key, "--json")
	if code != 0 || errOut != "" {
		t.Fatalf("owner item get exit/stderr = %d/%q", code, errOut)
	}
	item := decodeItem(t, out)
	if item.Key != key || item.Revision != resp.Item.Revision || string(item.Value) != `{"choice":"diagram-a"}` {
		t.Fatalf("stored item drift: %#v vs %#v", item, resp.Item)
	}

	dup := decodeBrowserActionResp(t, requestBrowserCommand(t, browser, browserItemCommand("item_submit", "cmd-visual-dup", map[string]any{
		"key":   key,
		"value": map[string]any{"choice": "diagram-a"},
	})))
	if dup.Status != "denied" || dup.Reason != "duplicate-item" {
		t.Fatalf("duplicate item submit drift: %#v", dup)
	}

	updated := decodeBrowserActionResp(t, requestBrowserCommand(t, browser, browserItemCommand("item_submit", "cmd-visual-update", map[string]any{
		"key":              key,
		"expectedRevision": resp.Item.Revision,
		"value":            map[string]any{"choice": "diagram-b"},
	})))
	if updated.Status != "accepted" || updated.Item == nil || updated.Item.Revision <= resp.Item.Revision || string(updated.Item.Value) != `{"choice":"diagram-b"}` {
		t.Fatalf("guarded update drift: %#v", updated)
	}

	stale := decodeBrowserActionResp(t, requestBrowserCommand(t, browser, browserItemCommand("item_submit", "cmd-visual-stale", map[string]any{
		"key":              key,
		"expectedRevision": resp.Item.Revision,
		"value":            map[string]any{"choice": "diagram-c"},
	})))
	if stale.Status != "denied" || stale.Reason != "stale-revision" {
		t.Fatalf("stale item submit drift: %#v", stale)
	}

	escape := decodeBrowserActionResp(t, requestBrowserCommand(t, browser, browserItemCommand("item_submit", "cmd-visual-scope", map[string]any{
		"key":   "artifacts.other.results.choice",
		"value": map[string]any{"choice": "diagram-a"},
	})))
	if escape.Status != "denied" || escape.Reason != "denied-scope" {
		t.Fatalf("out-of-scope item submit drift: %#v", escape)
	}

	rawAuthority := decodeBrowserActionResp(t, requestBrowserCommand(t, browser, browserItemCommand("item_submit", "cmd-visual-raw", map[string]any{
		"key":         "artifacts.artifact-browser.results.raw",
		"natsSubject": "tb.internal.admin.delete",
		"value":       map[string]any{"choice": "diagram-a"},
	})))
	if rawAuthority.Status != "denied" || rawAuthority.Reason != "raw-authority" {
		t.Fatalf("raw authority item submit drift: %#v", rawAuthority)
	}
}

func browserCommand(command, id, appID, participantID string, payload map[string]any) map[string]any {
	return map[string]any{
		"kind":             "browser.command_intent",
		"type":             "content.intent",
		"command":          command,
		"commandId":        id,
		"expectedRevision": "artifact.rev.7",
		"payload":          payload,
		"context": map[string]any{
			"sessionId":        "session-browser",
			"capabilityId":     "cap-browser",
			"artifactId":       "artifact-browser",
			"artifactRevision": "artifact.rev.7",
			"frameId":          "frame-browser",
			"appId":            appID,
			"participantId":    participantID,
			"chain": map[string]any{
				"chainId": "chain-browser",
				"rootId":  "root-browser",
				"hop":     1,
				"maxHops": 5,
			},
		},
	}
}

func browserItemCommand(command, id string, payload map[string]any) map[string]any {
	msg := browserCommand(command, id, "", "", payload)
	ctx := msg["context"].(map[string]any)
	delete(ctx, "appId")
	delete(ctx, "participantId")
	return msg
}

func browserConn(t *testing.T, app *App) *nats.Conn {
	t.Helper()
	nc, _ := browserGrantConn(t, app)
	return nc
}

func browserGrantConn(t *testing.T, app *App) (*nats.Conn, embednats.ViewerCred) {
	t.Helper()
	grant, err := embednats.MintViewerCredential(app.rt, "session-browser", 10*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	ws := app.rt.Posture().WebSocket
	if !ws.Enabled || ws.URL == "" {
		t.Fatalf("websocket posture is unavailable: %#v", ws)
	}
	nc, err := nats.Connect(ws.URL,
		nats.UserJWT(
			func() (string, error) { return grant.JWT, nil },
			func([]byte) ([]byte, error) { return nil, nil },
		),
		nats.MaxReconnects(0),
	)
	if err != nil {
		t.Fatal(err)
	}
	return nc, grant
}

func requestBrowserCommand(t *testing.T, nc *nats.Conn, msg map[string]any) []byte {
	t.Helper()
	body, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	reply, err := nc.Request("tb.app.browser.command", body, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	return reply.Data
}

func decodeBrowserActionResp(t *testing.T, body []byte) appActionResp {
	t.Helper()
	var resp appActionResp
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("browser action response json: %v\n%s", err, body)
	}
	return resp
}

func assertBrowserStateEvent(t *testing.T, ch <-chan browserStateEvent, key string, rev uint64, value string) {
	t.Helper()
	select {
	case ev := <-ch:
		if ev.Kind != "tinkabot.browserState.v1" || ev.Source != "trusted-shell.nats-watch.push" || ev.Key != key || ev.Status == "" || ev.Revision != rev || string(ev.Value) != value {
			t.Fatalf("browser state event drift: %#v want key=%s rev=%d value=%s", ev, key, rev, value)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("browser state event %s rev %d did not arrive", key, rev)
	}
}

func browserItemView(item appActionItem) itemView {
	return itemView{
		Key:        item.Key,
		Status:     item.Status,
		Value:      item.Value,
		Revision:   item.Revision,
		Provenance: item.Provenance,
	}
}

func assertNoBrowserLeaks(t *testing.T, text string, app *App) {
	t.Helper()
	for _, leak := range []string{"tb_items", "$KV", "BEGIN NATS", "PRIVATE KEY", "nats://", ".creds", string(mustReadFile(t, app.CredsFile(RoleCaller)))} {
		if strings.Contains(text, leak) {
			t.Fatalf("browser response leaked %q: %s", leak, text)
		}
	}
}

func TestParticipantAppReducer(t *testing.T) {
	t.Parallel()
	store := t.TempDir()
	app, err := boot(t, cfgFor(store))
	if err != nil {
		t.Fatal(err)
	}
	alice, err := app.AdmitParticipant("demo", "alice")
	if err != nil {
		t.Fatal(err)
	}

	owner, aliceEnv := tinkaletEnv(t), tinkaletEnv(t)
	mustTinkalet(t, owner, "profile", "import", "local", "--store", store, "--name", "owner")
	mustTinkalet(t, owner, "profile", "use", "owner")
	mustTinkalet(t, aliceEnv, "profile", "import", "local", "--store", alice.StoreDir, "--name", "alice")
	mustTinkalet(t, aliceEnv, "profile", "use", "alice")

	code, out, errOut := runTinkalet(owner, "item", "create", "apps.demo.state.turn", "--value", `{"turn":"alice","log":[]}`, "--json")
	if code != 0 || errOut != "" {
		t.Fatalf("state create exit/stderr = %d/%q", code, errOut)
	}
	state := decodeItem(t, out)

	mustTinkalet(t, aliceEnv, "action", "submit", "move-apply", "--state", "apps.demo.state.turn", "--base-revision", revString(state.Revision), "--value", `{"move":"a"}`)
	mustTinkalet(t, owner, "item", "create", "apps.demo.participants.alice.actions.bad-state", "--value", `{"kind":"tinkabot.appAction.v1","appId":"demo","participantId":"alice","actionId":"bad-state","stateKey":"apps.other.state.turn","baseRevision":`+revString(state.Revision)+`,"payload":{}}`)
	code, out, errOut = runTinkalet(owner, "action", "apply", "apps.demo.participants.alice.actions.bad-state", "--value", `{"bad":true}`)
	if code != 1 || out != "" || errOut != "action apps.demo.participants.alice.actions.bad-state denied apply: malformed-action\n" {
		t.Fatalf("malformed action apply exit/stdout/stderr = %d/%q/%q", code, out, errOut)
	}

	code, out, errOut = runTinkalet(owner, "action", "apply", "apps.demo.participants.alice.actions.move-apply", "--value", `{"turn":"bob","log":["a"]}`, "--json")
	if code != 0 || errOut != "" {
		t.Fatalf("action apply exit/stderr = %d/%q", code, errOut)
	}
	receipt := decodeItem(t, out)
	assertActionReceipt(t, receipt, "apps.demo.participants.alice.actions.move-apply", "apps.demo.state.turn")

	code, out, errOut = runTinkalet(owner, "item", "get", "apps.demo.state.turn", "--json")
	if code != 0 || errOut != "" {
		t.Fatalf("state get exit/stderr = %d/%q", code, errOut)
	}
	applied := decodeItem(t, out)
	if applied.Revision <= state.Revision || applied.Status != "resolved" || string(applied.Value) != `{"turn":"bob","log":["a"]}` {
		t.Fatalf("applied state drift: %#v after %#v", applied, state)
	}

	code, out, errOut = runTinkalet(owner, "action", "apply", "apps.demo.participants.alice.actions.move-apply", "--value", `{"turn":"bob","log":["a","again"]}`)
	if code != 1 || out != "" || errOut != "action apps.demo.participants.alice.actions.move-apply denied apply: duplicate-action\n" {
		t.Fatalf("duplicate apply exit/stdout/stderr = %d/%q/%q", code, out, errOut)
	}
	code, out, errOut = runTinkalet(owner, "action", "reject", "apps.demo.participants.alice.actions.move-apply", "--reason", "wrong-turn")
	if code != 1 || out != "" || errOut != "action apps.demo.participants.alice.actions.move-apply denied reject: duplicate-action\n" {
		t.Fatalf("reject after apply exit/stdout/stderr = %d/%q/%q", code, out, errOut)
	}

	mustTinkalet(t, aliceEnv, "action", "submit", "move-stale", "--state", "apps.demo.state.turn", "--base-revision", revString(applied.Revision), "--value", `{"move":"b"}`)
	mustTinkalet(t, owner, "item", "resolve", "apps.demo.state.turn", "--revision", revString(applied.Revision), "--value", `{"turn":"alice","log":["a","owner"]}`)
	code, out, errOut = runTinkalet(owner, "action", "apply", "apps.demo.participants.alice.actions.move-stale", "--value", `{"turn":"bob","log":["a","b"]}`)
	if code != 1 || out != "" || errOut != "action apps.demo.participants.alice.actions.move-stale denied apply: stale-revision\n" {
		t.Fatalf("stale apply exit/stdout/stderr = %d/%q/%q", code, out, errOut)
	}
	code, out, errOut = runTinkalet(owner, "item", "get", "apps.demo.participants.alice.actions.move-stale.receipt")
	if code != 1 || out != "" || errOut != "item apps.demo.participants.alice.actions.move-stale.receipt denied get: item-not-found\n" {
		t.Fatalf("stale apply receipt materialized: %d/%q/%q", code, out, errOut)
	}

	code, out, errOut = runTinkalet(aliceEnv, "action", "apply", "apps.demo.participants.alice.actions.move-apply", "--value", `{"bad":true}`)
	assertParticipantDenied(t, code, out, errOut, "action apps.demo.participants.alice.actions.move-apply denied apply: denied-scope\n", app, alice)
	code, out, errOut = runTinkalet(aliceEnv, "action", "reject", "apps.demo.participants.alice.actions.move-apply", "--reason", "wrong-turn")
	assertParticipantDenied(t, code, out, errOut, "action apps.demo.participants.alice.actions.move-apply denied reject: denied-scope\n", app, alice)
}

func TestParticipantRealtimeWatchEnvelope(t *testing.T) {
	t.Parallel()
	store := t.TempDir()
	app, err := boot(t, cfgFor(store))
	if err != nil {
		t.Fatal(err)
	}
	alice, err := app.AdmitParticipant("demo", "alice")
	if err != nil {
		t.Fatal(err)
	}
	bob, err := app.AdmitParticipant("demo", "bob")
	if err != nil {
		t.Fatal(err)
	}

	owner, aliceEnv, bobEnv := tinkaletEnv(t), tinkaletEnv(t), tinkaletEnv(t)
	mustTinkalet(t, owner, "profile", "import", "local", "--store", store, "--name", "owner")
	mustTinkalet(t, owner, "profile", "use", "owner")
	mustTinkalet(t, aliceEnv, "profile", "import", "local", "--store", alice.StoreDir, "--name", "alice")
	mustTinkalet(t, aliceEnv, "profile", "use", "alice")
	mustTinkalet(t, bobEnv, "profile", "import", "local", "--store", bob.StoreDir, "--name", "bob")
	mustTinkalet(t, bobEnv, "profile", "use", "bob")

	code, out, errOut := runTinkalet(owner, "item", "create", "apps.demo.state.rate", "--value", `{"seq":0}`, "--json")
	if code != 0 || errOut != "" {
		t.Fatalf("state create exit/stderr = %d/%q", code, errOut)
	}
	state := decodeItem(t, out)
	state = resolveRateState(t, owner, state, 1)
	state = resolveRateState(t, owner, state, 2)

	code, out, errOut = runTinkalet(aliceEnv, "watch", "prefix", "apps.demo.state", "--limit", "3", "--timeout", "2s", "--json")
	if code != 0 || errOut != "" {
		t.Fatalf("participant state watch exit/stdout/stderr = %d/%q/%q", code, out, errOut)
	}
	stateEvents, err := parseWatchEvents(out)
	if err != nil {
		t.Fatal(err)
	}
	assertWatchKeys(t, stateEvents, "apps.demo.state.rate", 3)
	assertStrictRevisions(t, stateEvents)
	assertNoParticipantLeaks(t, out+errOut, app, alice, bob)

	mustTinkalet(t, aliceEnv, "action", "submit", "rate-1", "--state", "apps.demo.state.rate", "--base-revision", revString(state.Revision), "--value", `{"delta":1}`)
	code, out, errOut = runTinkalet(owner, "action", "apply", "apps.demo.participants.alice.actions.rate-1", "--value", `{"seq":3}`, "--json")
	if code != 0 || errOut != "" {
		t.Fatalf("rate action apply exit/stderr = %d/%q", code, errOut)
	}

	code, out, errOut = runTinkalet(aliceEnv, "watch", "prefix", "apps.demo.participants.alice.actions", "--limit", "2", "--timeout", "2s", "--json")
	if code != 0 || errOut != "" {
		t.Fatalf("participant action watch exit/stdout/stderr = %d/%q/%q", code, out, errOut)
	}
	actionEvents, err := parseWatchEvents(out)
	if err != nil {
		t.Fatal(err)
	}
	assertWatchKeys(t, actionEvents, "apps.demo.participants.alice.actions.rate-1", 1)
	assertWatchKeys(t, actionEvents, "apps.demo.participants.alice.actions.rate-1.receipt", 1)
	assertStrictRevisions(t, actionEvents)
	assertNoParticipantLeaks(t, out+errOut, app, alice, bob)

	code, out, errOut = runTinkalet(owner, "item", "get", "apps.demo.state.rate", "--json")
	if code != 0 || errOut != "" {
		t.Fatalf("state get exit/stderr = %d/%q", code, errOut)
	}
	latest := decodeItem(t, out)
	mustTinkalet(t, bobEnv, "action", "submit", "rate-1", "--state", "apps.demo.state.rate", "--base-revision", revString(latest.Revision), "--value", `{"delta":1}`)
	code, out, errOut = runTinkalet(aliceEnv, "watch", "prefix", "apps.demo.participants.bob.actions", "--limit", "1", "--timeout", "200ms", "--json")
	assertParticipantDenied(t, code, out, errOut, "watch apps.demo.participants.bob.actions denied prefix: denied-scope\n", app, alice, bob)
	assertPublishDenied(t, app, alice, "$JS.API.CONSUMER.CREATE.KV_"+wiring().ItemBucket+".unsafe.$KV."+wiring().ItemBucket+".>")
}

func TestParticipantRealtimeActionGapHarness(t *testing.T) {
	t.Parallel()
	runParticipantActionGapHarness(t, participantActionGapSpec{
		perParticipant: 24,
		interval:       25 * time.Millisecond,
		watchTimeout:   8 * time.Second,
	})
}

func TestParticipantRealtimeSustainedActionGapHarness(t *testing.T) {
	if os.Getenv("TINKABOT_REALTIME_SUSTAINED") != "1" {
		t.Skip("set TINKABOT_REALTIME_SUSTAINED=1 for the 60s sustained participant-rate proof")
	}
	runParticipantActionGapHarness(t, participantActionGapSpec{
		perParticipant: 600,
		interval:       100 * time.Millisecond,
		watchTimeout:   75 * time.Second,
	})
}

func TestParticipantRealtimeReconnectRestartCatchUp(t *testing.T) {
	store := t.TempDir()
	app, err := boot(t, cfgFor(store))
	if err != nil {
		t.Fatal(err)
	}
	alice, err := app.AdmitParticipant("demo", "alice")
	if err != nil {
		t.Fatal(err)
	}

	owner, aliceEnv := tinkaletEnv(t), tinkaletEnv(t)
	mustTinkalet(t, owner, "profile", "import", "local", "--store", store, "--name", "owner")
	mustTinkalet(t, owner, "profile", "use", "owner")
	mustTinkalet(t, aliceEnv, "profile", "import", "local", "--store", alice.StoreDir, "--name", "alice")
	mustTinkalet(t, aliceEnv, "profile", "use", "alice")

	state := createRateState(t, owner, "alice")
	submitRateActionViaTinkalet(t, aliceEnv, "alice", 0, state)
	code, out, errOut := runTinkalet(aliceEnv, "watch", "prefix", "apps.demo.participants.alice.actions", "--cursor", "alice-reconnect", "--limit", "1", "--timeout", "2s", "--json")
	if code != 0 || errOut != "" {
		t.Fatalf("seed watch exit/stdout/stderr = %d/%q/%q", code, out, errOut)
	}
	seed := decodeWatchEvent(t, out)
	assertReconnectReplay(t, []watchEvent{seed}, "alice", 0, 1, 0)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := app.Stop(ctx); err != nil {
		t.Fatal(err)
	}
	app2, err := boot(t, cfgFor(store))
	if err != nil {
		t.Fatal(err)
	}

	mustTinkalet(t, aliceEnv, "profile", "import", "local", "--store", alice.StoreDir, "--name", "alice")
	for i := 1; i <= 3; i++ {
		submitRateActionViaTinkalet(t, aliceEnv, "alice", i, state)
	}
	code, out, errOut = runTinkalet(aliceEnv, "watch", "prefix", "apps.demo.participants.alice.actions", "--cursor", "alice-reconnect", "--limit", "3", "--timeout", "2s", "--json")
	if code != 0 || errOut != "" {
		t.Fatalf("catch-up watch exit/stdout/stderr = %d/%q/%q", code, out, errOut)
	}
	events, err := parseWatchEvents(out)
	if err != nil {
		t.Fatal(err)
	}
	assertReconnectReplay(t, events, "alice", 1, 3, seed.Revision)
	assertNoParticipantLeaks(t, out+errOut, app2, alice)

	code, out, errOut = runTinkalet(aliceEnv, "watch", "prefix", "apps.demo.participants.alice.actions", "--cursor", "alice-reconnect", "--limit", "1", "--timeout", "200ms", "--json")
	if code != 1 || out != "" || errOut != "watch apps.demo.participants.alice.actions denied prefix: watch-timeout\n" {
		t.Fatalf("duplicate replay exit/stdout/stderr = %d/%q/%q", code, out, errOut)
	}

	if err := app2.RevokeParticipant(alice); err != nil {
		t.Fatal(err)
	}
	assertParticipantRecord(t, app2, alice, "revoked")
	assertParticipantCredsDenied(t, app2, mustReadFile(t, alice.CredsFile))
	code, out, errOut = runTinkalet(aliceEnv, "action", "submit", "rt-alice-after-revoke", "--state", state.Key, "--base-revision", revString(state.Revision), "--value", rateActionPayload("alice", 4))
	assertParticipantDenied(t, code, out, errOut, "action rt-alice-after-revoke denied submit: revoked-credentials\n", app2, alice)
}

func TestParticipantRealtimeTerminalResultMaterialization(t *testing.T) {
	t.Parallel()
	store := t.TempDir()
	app, err := boot(t, cfgFor(store))
	if err != nil {
		t.Fatal(err)
	}
	alice, err := app.AdmitParticipant("demo", "alice")
	if err != nil {
		t.Fatal(err)
	}
	bob, err := app.AdmitParticipant("demo", "bob")
	if err != nil {
		t.Fatal(err)
	}

	owner, aliceEnv, bobEnv := tinkaletEnv(t), tinkaletEnv(t), tinkaletEnv(t)
	mustTinkalet(t, owner, "profile", "import", "local", "--store", store, "--name", "owner")
	mustTinkalet(t, owner, "profile", "use", "owner")
	mustTinkalet(t, aliceEnv, "profile", "import", "local", "--store", alice.StoreDir, "--name", "alice")
	mustTinkalet(t, aliceEnv, "profile", "use", "alice")
	mustTinkalet(t, bobEnv, "profile", "import", "local", "--store", bob.StoreDir, "--name", "bob")
	mustTinkalet(t, bobEnv, "profile", "use", "bob")

	state := createTerminalState(t, owner)
	state = applyTerminalAction(t, owner, submitTerminalAction(t, aliceEnv, "alice", "term-a1", state.Revision, `{"delta":1}`), state, terminalState{Phase: "running", Progress: map[string]int{"alice": 1, "bob": 0}})
	state = applyTerminalAction(t, owner, submitTerminalAction(t, bobEnv, "bob", "term-b1", state.Revision, `{"delta":1}`), state, terminalState{Phase: "running", Progress: map[string]int{"alice": 1, "bob": 1}})
	final := terminalState{Phase: "finished", Progress: map[string]int{"alice": 2, "bob": 1}, Winner: "alice"}
	state = applyTerminalAction(t, owner, submitTerminalAction(t, aliceEnv, "alice", "term-finish", state.Revision, `{"delta":1,"finish":true}`), state, final)
	state = rejectTerminalAction(t, owner, submitTerminalAction(t, bobEnv, "bob", "term-late", state.Revision, `{"delta":1}`), state, "race-finished")

	code, out, errOut := runTinkalet(aliceEnv, "watch", "prefix", "apps.demo.state", "--limit", "4", "--timeout", "2s", "--json")
	if code != 0 || errOut != "" {
		t.Fatalf("terminal state watch exit/stdout/stderr = %d/%q/%q", code, out, errOut)
	}
	stateEvents, err := parseWatchEvents(out)
	if err != nil {
		t.Fatal(err)
	}
	assertTerminalStateMaterialized(t, stateEvents, state, final)
	assertNoParticipantLeaks(t, out+errOut, app, alice, bob)

	code, out, errOut = runTinkalet(aliceEnv, "watch", "prefix", "apps.demo.participants.alice.actions", "--limit", "6", "--timeout", "2s", "--json")
	if code != 0 || errOut != "" {
		t.Fatalf("alice receipt watch exit/stdout/stderr = %d/%q/%q", code, out, errOut)
	}
	aliceEvents, err := parseWatchEvents(out)
	if err != nil {
		t.Fatal(err)
	}
	assertTerminalReceiptAccounting(t, aliceEvents, "alice", []terminalReceipt{
		{ActionID: "term-a1", Outcome: "applied"},
		{ActionID: "term-finish", Outcome: "applied"},
	})
	assertNoParticipantLeaks(t, out+errOut, app, alice, bob)

	code, out, errOut = runTinkalet(bobEnv, "watch", "prefix", "apps.demo.participants.bob.actions", "--limit", "5", "--timeout", "2s", "--json")
	if code != 0 || errOut != "" {
		t.Fatalf("bob receipt watch exit/stdout/stderr = %d/%q/%q", code, out, errOut)
	}
	bobEvents, err := parseWatchEvents(out)
	if err != nil {
		t.Fatal(err)
	}
	assertTerminalReceiptAccounting(t, bobEvents, "bob", []terminalReceipt{
		{ActionID: "term-b1", Outcome: "applied"},
		{ActionID: "term-late", Outcome: "rejected", Reason: "race-finished"},
	})
	assertNoParticipantLeaks(t, out+errOut, app, alice, bob)
}

type participantActionGapSpec struct {
	perParticipant int
	interval       time.Duration
	watchTimeout   time.Duration
}

func runParticipantActionGapHarness(t *testing.T, spec participantActionGapSpec) {
	t.Helper()
	store := t.TempDir()
	app, err := boot(t, cfgFor(store))
	if err != nil {
		t.Fatal(err)
	}
	alice, err := app.AdmitParticipant("demo", "alice")
	if err != nil {
		t.Fatal(err)
	}
	bob, err := app.AdmitParticipant("demo", "bob")
	if err != nil {
		t.Fatal(err)
	}

	owner := tinkaletEnv(t)
	mustTinkalet(t, owner, "profile", "import", "local", "--store", store, "--name", "owner")
	mustTinkalet(t, owner, "profile", "use", "owner")

	participants := []struct {
		prof  ParticipantProfile
		state itemView
	}{
		{prof: alice, state: createRateState(t, owner, "alice")},
		{prof: bob, state: createRateState(t, owner, "bob")},
	}
	watches := make([]<-chan actionWatchResult, 0, len(participants))
	for _, p := range participants {
		watches = append(watches, startActionWatchCollector(t, app, p.prof, spec.perParticipant, spec.watchTimeout))
	}

	errs := make(chan error, len(participants))
	started := time.Now()
	for _, p := range participants {
		p := p
		go func() {
			errs <- submitRateActions(app, p.prof, p.state, spec.perParticipant, spec.interval)
		}()
	}
	for range participants {
		if err := <-errs; err != nil {
			t.Fatal(err)
		}
	}

	for i, p := range participants {
		res := <-watches[i]
		if res.err != nil {
			t.Fatal(res.err)
		}
		assertActionGapComplete(t, p.prof, spec.perParticipant, res)
	}
	t.Logf("participant action gap harness submitted %d actions across %d participants in %s", spec.perParticipant*len(participants), len(participants), time.Since(started))
}

func TestTurnBasedReferenceMission(t *testing.T) {
	t.Parallel()
	store := t.TempDir()
	app, err := boot(t, cfgFor(store))
	if err != nil {
		t.Fatal(err)
	}
	alice, err := app.AdmitParticipant("demo", "alice")
	if err != nil {
		t.Fatal(err)
	}
	bob, err := app.AdmitParticipant("demo", "bob")
	if err != nil {
		t.Fatal(err)
	}

	owner, aliceEnv, bobEnv := tinkaletEnv(t), tinkaletEnv(t), tinkaletEnv(t)
	mustTinkalet(t, owner, "profile", "import", "local", "--store", store, "--name", "owner")
	mustTinkalet(t, owner, "profile", "use", "owner")
	mustTinkalet(t, aliceEnv, "profile", "import", "local", "--store", alice.StoreDir, "--name", "alice")
	mustTinkalet(t, aliceEnv, "profile", "use", "alice")
	mustTinkalet(t, bobEnv, "profile", "import", "local", "--store", bob.StoreDir, "--name", "bob")
	mustTinkalet(t, bobEnv, "profile", "use", "bob")

	code, out, errOut := runTinkalet(owner, "item", "create", "apps.demo.state.board", "--value", `{"turn":"alice","cells":{}}`, "--json")
	if code != 0 || errOut != "" {
		t.Fatalf("state create exit/stderr = %d/%q", code, errOut)
	}
	state := decodeItem(t, out)
	initialRevision := state.Revision

	wrongTurn := submitTurnAction(t, bobEnv, "bob", "b-wrong-turn", state.Revision, "b1")
	state = rejectTurnAction(t, owner, wrongTurn, state, "wrong-turn")

	aliceA1 := submitTurnAction(t, aliceEnv, "alice", "a1", state.Revision, "a1")
	state = applyTurnAction(t, owner, aliceA1, state, "alice", "a1")

	code, out, errOut = runTinkalet(aliceEnv, "action", "submit", "a1", "--state", "apps.demo.state.board", "--base-revision", revString(state.Revision), "--value", turnPayload(t, "a1"))
	assertParticipantDenied(t, code, out, errOut, "action a1 denied submit: duplicate-action\n", app, alice, bob)

	code, out, errOut = runTinkalet(bobEnv, "action", "submit", "b-stale", "--state", "apps.demo.state.board", "--base-revision", revString(initialRevision), "--value", turnPayload(t, "b1"))
	assertParticipantDenied(t, code, out, errOut, "action b-stale denied submit: stale-revision\n", app, alice, bob)
	code, out, errOut = runTinkalet(owner, "item", "get", "apps.demo.participants.bob.actions.b-stale")
	if code != 1 || out != "" || errOut != "item apps.demo.participants.bob.actions.b-stale denied get: item-not-found\n" {
		t.Fatalf("stale action materialized: %d/%q/%q", code, out, errOut)
	}

	occupied := submitTurnAction(t, bobEnv, "bob", "b-occupied", state.Revision, "a1")
	state = rejectTurnAction(t, owner, occupied, state, "occupied-cell")

	state = applyTurnAction(t, owner, submitTurnAction(t, bobEnv, "bob", "b1", state.Revision, "b1"), state, "bob", "b1")
	state = applyTurnAction(t, owner, submitTurnAction(t, aliceEnv, "alice", "a2", state.Revision, "a2"), state, "alice", "a2")
	state = applyTurnAction(t, owner, submitTurnAction(t, bobEnv, "bob", "b2", state.Revision, "b2"), state, "bob", "b2")
	state = applyTurnAction(t, owner, submitTurnAction(t, aliceEnv, "alice", "a3", state.Revision, "a3"), state, "alice", "a3")

	board := decodeTurnBoard(t, state)
	if board.Winner != "alice" || board.Turn != "alice" ||
		board.Cells["a1"] != "alice" ||
		board.Cells["a2"] != "alice" ||
		board.Cells["a3"] != "alice" ||
		board.Cells["b1"] != "bob" ||
		board.Cells["b2"] != "bob" {
		t.Fatalf("final board drift: %#v in %#v", board, state)
	}
}

func resolveRateState(t *testing.T, owner []string, before itemView, seq int) itemView {
	t.Helper()
	code, out, errOut := runTinkalet(owner, "item", "resolve", "apps.demo.state.rate", "--revision", revString(before.Revision), "--value", `{"seq":`+strconv.Itoa(seq)+`}`, "--json")
	if code != 0 || errOut != "" {
		t.Fatalf("state resolve %d exit/stderr = %d/%q", seq, code, errOut)
	}
	after := decodeItem(t, out)
	if after.Revision <= before.Revision {
		t.Fatalf("state revision did not advance: before %#v after %#v", before, after)
	}
	return after
}

func createRateState(t *testing.T, owner []string, participant string) itemView {
	t.Helper()
	key := "apps.demo.state." + participant
	code, out, errOut := runTinkalet(owner, "item", "create", key, "--value", `{"participant":"`+participant+`","seq":0}`, "--json")
	if code != 0 || errOut != "" {
		t.Fatalf("state create %s exit/stderr = %d/%q", key, code, errOut)
	}
	return decodeItem(t, out)
}

const terminalStateKey = "apps.demo.state.terminal"

type terminalState struct {
	Phase    string         `json:"phase"`
	Progress map[string]int `json:"progress"`
	Winner   string         `json:"winner,omitempty"`
}

type terminalReceipt struct {
	ActionID string
	Outcome  string
	Reason   string
}

func createTerminalState(t *testing.T, owner []string) itemView {
	t.Helper()
	state := terminalState{Phase: "running", Progress: map[string]int{"alice": 0, "bob": 0}}
	code, out, errOut := runTinkalet(owner, "item", "create", terminalStateKey, "--value", terminalStatePayload(t, state), "--json")
	if code != 0 || errOut != "" {
		t.Fatalf("terminal state create exit/stderr = %d/%q", code, errOut)
	}
	item := decodeItem(t, out)
	if item.Key != terminalStateKey || string(item.Value) != terminalStatePayload(t, state) {
		t.Fatalf("terminal state create drift: %#v", item)
	}
	return item
}

func submitTerminalAction(t *testing.T, env []string, participantID, actionID string, baseRevision uint64, payload string) itemView {
	t.Helper()
	code, out, errOut := runTinkalet(env, "action", "submit", actionID, "--state", terminalStateKey, "--base-revision", revString(baseRevision), "--value", payload, "--json")
	if code != 0 || errOut != "" {
		t.Fatalf("terminal action submit %s exit/stderr = %d/%q", actionID, code, errOut)
	}
	item := decodeItem(t, out)
	assertActionItem(t, item, "demo", participantID, actionID, terminalStateKey, baseRevision, payload)
	return item
}

func applyTerminalAction(t *testing.T, owner []string, action, before itemView, next terminalState) itemView {
	t.Helper()
	payload := terminalStatePayload(t, next)
	code, out, errOut := runTinkalet(owner, "action", "apply", action.Key, "--value", payload, "--json")
	if code != 0 || errOut != "" {
		t.Fatalf("terminal action apply %s exit/stderr = %d/%q", action.Key, code, errOut)
	}
	assertActionReceipt(t, decodeItem(t, out), action.Key, terminalStateKey)
	after := getTerminalState(t, owner)
	if after.Revision <= before.Revision || after.Status != "resolved" || string(after.Value) != payload {
		t.Fatalf("terminal apply drift: before %#v after %#v want %s", before, after, payload)
	}
	return after
}

func rejectTerminalAction(t *testing.T, owner []string, action, before itemView, reason string) itemView {
	t.Helper()
	code, out, errOut := runTinkalet(owner, "action", "reject", action.Key, "--reason", reason, "--json")
	if code != 0 || errOut != "" {
		t.Fatalf("terminal action reject %s exit/stderr = %d/%q", action.Key, code, errOut)
	}
	assertActionRejectionReceipt(t, decodeItem(t, out), action.Key, terminalStateKey, reason)
	after := getTerminalState(t, owner)
	if after.Revision != before.Revision || string(after.Value) != string(before.Value) {
		t.Fatalf("terminal reject mutated state: before %#v after %#v", before, after)
	}
	return after
}

func getTerminalState(t *testing.T, owner []string) itemView {
	t.Helper()
	code, out, errOut := runTinkalet(owner, "item", "get", terminalStateKey, "--json")
	if code != 0 || errOut != "" {
		t.Fatalf("terminal state get exit/stderr = %d/%q", code, errOut)
	}
	return decodeItem(t, out)
}

func terminalStatePayload(t *testing.T, state terminalState) string {
	t.Helper()
	body, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func assertTerminalStateMaterialized(t *testing.T, events []watchEvent, final itemView, want terminalState) {
	t.Helper()
	if len(events) != 4 {
		t.Fatalf("terminal state event count = %d, want 4: %#v", len(events), events)
	}
	assertStrictRevisions(t, events)
	for _, ev := range events {
		if ev.Key != terminalStateKey {
			t.Fatalf("terminal state watch saw unexpected key: %#v", ev)
		}
	}
	last := events[len(events)-1]
	if last.Revision != final.Revision || last.Status != "resolved" || string(last.Value) != terminalStatePayload(t, want) {
		t.Fatalf("terminal final event drift: %#v final %#v want %s", last, final, terminalStatePayload(t, want))
	}
}

func assertTerminalReceiptAccounting(t *testing.T, events []watchEvent, participant string, wants []terminalReceipt) {
	t.Helper()
	wantEvents := 0
	for _, want := range wants {
		wantEvents += 2
		if want.Outcome == "applied" {
			wantEvents++
		}
	}
	if len(events) != wantEvents {
		t.Fatalf("terminal receipt event count = %d, want %d: %#v", len(events), wantEvents, events)
	}
	assertStrictRevisions(t, events)
	prefix := participantActionPrefix("demo", participant) + "."
	actionSeen := map[string]int{}
	finalReceipts := map[string]struct {
		Status  string
		Outcome string
		Reason  string
	}{}
	for _, ev := range events {
		if !strings.HasPrefix(ev.Key, prefix) {
			t.Fatalf("participant %s watch saw out-of-scope key: %#v", participant, ev)
		}
		id := strings.TrimPrefix(ev.Key, prefix)
		if !strings.HasSuffix(id, ".receipt") {
			if strings.Contains(id, ".") || ev.Status != "pending" {
				t.Fatalf("participant %s action event drift: %#v", participant, ev)
			}
			var act appActionValue
			if err := json.Unmarshal(ev.Value, &act); err != nil {
				t.Fatalf("participant %s action json: %v in %#v", participant, err, ev)
			}
			if act.AppID != "demo" || act.ParticipantID != participant || act.ActionID != id || act.StateKey != terminalStateKey {
				t.Fatalf("participant %s action value drift: %#v in %#v", participant, act, ev)
			}
			actionSeen[id]++
			continue
		}
		id = strings.TrimSuffix(id, ".receipt")
		var rec struct {
			Kind           string `json:"kind"`
			ActionKey      string `json:"actionKey"`
			StateKey       string `json:"stateKey"`
			ActionRevision uint64 `json:"actionRevision"`
			StateRevision  uint64 `json:"stateRevision"`
			Outcome        string `json:"outcome"`
			Reason         string `json:"reason"`
		}
		if err := json.Unmarshal(ev.Value, &rec); err != nil {
			t.Fatalf("participant %s receipt json: %v in %#v", participant, err, ev)
		}
		if rec.Kind != "tinkabot.appActionReceipt.v1" ||
			rec.ActionKey != prefix+id ||
			rec.StateKey != terminalStateKey ||
			rec.ActionRevision == 0 {
			t.Fatalf("participant %s receipt value drift: %#v in %#v", participant, rec, ev)
		}
		switch ev.Status {
		case "pending":
			if rec.Outcome != "applying" || rec.StateRevision != 0 || rec.Reason != "" {
				t.Fatalf("participant %s pending receipt drift: %#v in %#v", participant, rec, ev)
			}
		case "resolved", "denied":
			if rec.StateRevision == 0 {
				t.Fatalf("participant %s final receipt missing state revision: %#v in %#v", participant, rec, ev)
			}
			finalReceipts[id] = struct {
				Status  string
				Outcome string
				Reason  string
			}{Status: ev.Status, Outcome: rec.Outcome, Reason: rec.Reason}
		default:
			t.Fatalf("participant %s receipt status drift: %#v", participant, ev)
		}
	}
	for _, want := range wants {
		if actionSeen[want.ActionID] != 1 {
			t.Fatalf("participant %s action %s count = %d in %#v", participant, want.ActionID, actionSeen[want.ActionID], events)
		}
		got, ok := finalReceipts[want.ActionID]
		if !ok {
			t.Fatalf("participant %s action %s missing final receipt in %#v", participant, want.ActionID, events)
		}
		wantStatus := "resolved"
		if want.Outcome == "rejected" {
			wantStatus = "denied"
		}
		if got.Status != wantStatus || got.Outcome != want.Outcome || got.Reason != want.Reason {
			t.Fatalf("participant %s receipt %s = %#v, want status=%s outcome=%s reason=%s", participant, want.ActionID, got, wantStatus, want.Outcome, want.Reason)
		}
	}
}

func submitRateActionViaTinkalet(t *testing.T, env []string, participant string, i int, state itemView) itemView {
	t.Helper()
	actionID := rateActionID(participant, i)
	payload := rateActionPayload(participant, i)
	code, out, errOut := runTinkalet(env, "action", "submit", actionID, "--state", state.Key, "--base-revision", revString(state.Revision), "--value", payload, "--json")
	if code != 0 || errOut != "" {
		t.Fatalf("submit %s exit/stdout/stderr = %d/%q/%q", actionID, code, out, errOut)
	}
	item := decodeItem(t, out)
	assertActionItem(t, item, "demo", participant, actionID, state.Key, state.Revision, payload)
	return item
}

type actionWatchResult struct {
	seen      map[string]uint64
	revisions []uint64
	err       error
}

func startActionWatchCollector(t *testing.T, app *App, prof ParticipantProfile, want int, timeout time.Duration) <-chan actionWatchResult {
	t.Helper()
	nc, err := nats.Connect(
		app.Posture().NATS.ClientURL,
		nats.UserCredentials(prof.CredsFile),
		nats.NoReconnect(),
		nats.Timeout(2*time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		t.Fatal(err)
	}
	kv, err := js.KeyValue(wiring().ItemBucket)
	if err != nil {
		nc.Close()
		t.Fatal(err)
	}
	prefix := participantActionPrefix(prof.AppID, prof.ParticipantID)
	watcher, err := kv.WatchFiltered([]string{prefix + ".>"}, nats.IncludeHistory(), nats.IgnoreDeletes())
	if err != nil {
		nc.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		// Participant profiles cannot safely delete arbitrary KV consumers.
		nc.Close()
	})

	done := make(chan actionWatchResult, 1)
	go func() {
		seen := map[string]uint64{}
		revisions := make([]uint64, 0, want)
		timer := time.NewTimer(timeout)
		defer timer.Stop()
		for len(seen) < want {
			select {
			case err, ok := <-watcher.Error():
				if ok && err != nil {
					done <- actionWatchResult{err: err}
					return
				}
			case entry, ok := <-watcher.Updates():
				if !ok {
					done <- actionWatchResult{err: fmt.Errorf("participant %s watch closed", prof.ParticipantID)}
					return
				}
				if entry == nil {
					continue
				}
				key := entry.Key()
				if !strings.HasPrefix(key, prefix+".") {
					done <- actionWatchResult{err: fmt.Errorf("participant %s watch saw out-of-scope key %s", prof.ParticipantID, key)}
					return
				}
				id := strings.TrimPrefix(key, prefix+".")
				if strings.Contains(id, ".") {
					continue
				}
				if seen[id] != 0 {
					done <- actionWatchResult{err: fmt.Errorf("participant %s saw duplicate action id %s", prof.ParticipantID, id)}
					return
				}
				seen[id] = entry.Revision()
				revisions = append(revisions, entry.Revision())
			case <-timer.C:
				done <- actionWatchResult{seen: seen, revisions: revisions, err: fmt.Errorf("participant %s observed %d/%d action revisions", prof.ParticipantID, len(seen), want)}
				return
			}
		}
		done <- actionWatchResult{seen: seen, revisions: revisions}
	}()
	return done
}

func submitRateActions(app *App, prof ParticipantProfile, state itemView, count int, interval time.Duration) error {
	nc, err := nats.Connect(
		app.Posture().NATS.ClientURL,
		nats.UserCredentials(prof.CredsFile),
		nats.NoReconnect(),
		nats.Timeout(2*time.Second),
	)
	if err != nil {
		return err
	}
	defer nc.Close()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for i := 0; i < count; i++ {
		if i > 0 {
			<-ticker.C
		}
		actionID := rateActionID(prof.ParticipantID, i)
		val := json.RawMessage(`{"participant":"` + prof.ParticipantID + `","seq":` + strconv.Itoa(i) + `}`)
		body, err := json.Marshal(appActionReq{ActionID: actionID, StateKey: state.Key, BaseRevision: state.Revision, Value: val})
		if err != nil {
			return err
		}
		reply, err := nc.Request(participantActionSubject(prof.AppID, prof.ParticipantID), body, 2*time.Second)
		if err != nil {
			return err
		}
		var resp appActionResp
		if err := json.Unmarshal(reply.Data, &resp); err != nil {
			return err
		}
		if resp.Status != "accepted" || resp.Item == nil {
			return fmt.Errorf("action %s for %s = %#v", actionID, prof.ParticipantID, resp)
		}
	}
	return nil
}

func assertActionGapComplete(t *testing.T, prof ParticipantProfile, want int, res actionWatchResult) {
	t.Helper()
	for i := 0; i < want; i++ {
		id := rateActionID(prof.ParticipantID, i)
		if res.seen[id] == 0 {
			t.Fatalf("participant %s missing action %s in %#v", prof.ParticipantID, id, res.seen)
		}
	}
	for i := 1; i < len(res.revisions); i++ {
		if res.revisions[i] <= res.revisions[i-1] {
			t.Fatalf("participant %s revisions not strict: %#v", prof.ParticipantID, res.revisions)
		}
	}
}

func rateActionID(participant string, i int) string {
	return "rt-" + participant + "-" + strconv.Itoa(i)
}

func rateActionPayload(participant string, i int) string {
	return `{"participant":"` + participant + `","seq":` + strconv.Itoa(i) + `}`
}

func assertReconnectReplay(t *testing.T, events []watchEvent, participant string, start, count int, minRevision uint64) {
	t.Helper()
	if len(events) != count {
		t.Fatalf("reconnect replay count = %d, want %d: %#v", len(events), count, events)
	}
	prev := minRevision
	for i, ev := range events {
		wantKey := participantActionPrefix("demo", participant) + "." + rateActionID(participant, start+i)
		if ev.Key != wantKey || ev.Status != "pending" || ev.Source != "replay" {
			t.Fatalf("reconnect replay drift at %d: %#v want key=%s pending replay", i, ev, wantKey)
		}
		if ev.Revision <= prev {
			t.Fatalf("reconnect replay revisions not strict after %d: %#v", minRevision, events)
		}
		prev = ev.Revision
	}
}

func assertWatchKeys(t *testing.T, events []watchEvent, key string, count int) {
	t.Helper()
	got := 0
	for _, ev := range events {
		if ev.Key == key {
			got++
		}
	}
	if got != count {
		t.Fatalf("watch key %s count = %d, want %d in %#v", key, got, count, events)
	}
}

func assertStrictRevisions(t *testing.T, events []watchEvent) {
	t.Helper()
	for i := 1; i < len(events); i++ {
		if events[i].Revision <= events[i-1].Revision {
			t.Fatalf("watch revisions not strict: %#v", events)
		}
	}
}

func assertActionItem(t *testing.T, item itemView, appID, participantID, actionID, stateKey string, baseRevision uint64, payload string) {
	t.Helper()
	wantKey := "apps." + appID + ".participants." + participantID + ".actions." + actionID
	if item.Key != wantKey || item.Status != "pending" || item.Revision == 0 {
		t.Fatalf("action item drift: %#v", item)
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
		t.Fatal(err)
	}
	if val.Kind != "tinkabot.appAction.v1" ||
		val.AppID != appID ||
		val.ParticipantID != participantID ||
		val.ActionID != actionID ||
		val.StateKey != stateKey ||
		val.BaseRevision != baseRevision ||
		string(val.Payload) != payload {
		t.Fatalf("action value drift: %#v in %#v", val, item)
	}
}

func assertActionReceipt(t *testing.T, item itemView, actionKey, stateKey string) {
	t.Helper()
	if item.Key != actionKey+".receipt" || item.Status != "resolved" || item.Revision == 0 {
		t.Fatalf("action receipt drift: %#v", item)
	}
	var val struct {
		Kind          string `json:"kind"`
		ActionKey     string `json:"actionKey"`
		StateKey      string `json:"stateKey"`
		StateRevision uint64 `json:"stateRevision"`
	}
	if err := json.Unmarshal(item.Value, &val); err != nil {
		t.Fatal(err)
	}
	if val.Kind != "tinkabot.appActionReceipt.v1" || val.ActionKey != actionKey || val.StateKey != stateKey || val.StateRevision == 0 {
		t.Fatalf("receipt value drift: %#v", val)
	}
}

type turnBoard struct {
	Turn   string            `json:"turn"`
	Cells  map[string]string `json:"cells"`
	Winner string            `json:"winner,omitempty"`
}

type turnMove struct {
	Cell string `json:"cell"`
}

func submitTurnAction(t *testing.T, env []string, participantID, actionID string, baseRevision uint64, cell string) itemView {
	t.Helper()
	code, out, errOut := runTinkalet(env, "action", "submit", actionID, "--state", "apps.demo.state.board", "--base-revision", revString(baseRevision), "--value", turnPayload(t, cell), "--json")
	if code != 0 || errOut != "" {
		t.Fatalf("submit %s exit/stderr = %d/%q", actionID, code, errOut)
	}
	item := decodeItem(t, out)
	assertActionItem(t, item, "demo", participantID, actionID, "apps.demo.state.board", baseRevision, turnPayload(t, cell))
	return item
}

func applyTurnAction(t *testing.T, owner []string, action, before itemView, participantID, cell string) itemView {
	t.Helper()
	board := decodeTurnBoard(t, before)
	act, move := decodeTurnAction(t, action)
	if act.ParticipantID != participantID || move.Cell != cell {
		t.Fatalf("turn action drift: %#v move %#v", act, move)
	}
	if board.Winner != "" || board.Turn != participantID || board.Cells[cell] != "" {
		t.Fatalf("test attempted illegal apply: board %#v action %#v", board, act)
	}
	board.Cells[cell] = participantID
	if turnWinner(board, participantID) {
		board.Winner = participantID
	} else if participantID == "alice" {
		board.Turn = "bob"
	} else {
		board.Turn = "alice"
	}
	code, out, errOut := runTinkalet(owner, "action", "apply", action.Key, "--value", marshalTurnBoard(t, board), "--json")
	if code != 0 || errOut != "" {
		t.Fatalf("apply %s exit/stderr = %d/%q", action.Key, code, errOut)
	}
	assertActionReceipt(t, decodeItem(t, out), action.Key, "apps.demo.state.board")
	after := getTurnState(t, owner)
	if after.Revision <= before.Revision || string(after.Value) != marshalTurnBoard(t, board) {
		t.Fatalf("applied board drift: %#v want %s after %#v", decodeTurnBoard(t, after), marshalTurnBoard(t, board), before)
	}
	return after
}

func rejectTurnAction(t *testing.T, owner []string, action, before itemView, reason string) itemView {
	t.Helper()
	code, out, errOut := runTinkalet(owner, "action", "reject", action.Key, "--reason", reason, "--json")
	if code != 0 || errOut != "" {
		t.Fatalf("reject %s exit/stderr = %d/%q", action.Key, code, errOut)
	}
	assertActionRejectionReceipt(t, decodeItem(t, out), action.Key, "apps.demo.state.board", reason)
	code, out, errOut = runTinkalet(owner, "action", "apply", action.Key, "--value", string(before.Value))
	if code != 1 || out != "" || errOut != "action "+action.Key+" denied apply: duplicate-action\n" {
		t.Fatalf("apply after reject exit/stdout/stderr = %d/%q/%q", code, out, errOut)
	}
	after := getTurnState(t, owner)
	if after.Revision != before.Revision || string(after.Value) != string(before.Value) {
		t.Fatalf("rejected action mutated state: before %#v after %#v", before, after)
	}
	return after
}

func assertActionRejectionReceipt(t *testing.T, item itemView, actionKey, stateKey, reason string) {
	t.Helper()
	if item.Key != actionKey+".receipt" || item.Status != "denied" || item.Revision == 0 {
		t.Fatalf("rejection receipt drift: %#v", item)
	}
	var val struct {
		Kind           string `json:"kind"`
		ActionKey      string `json:"actionKey"`
		StateKey       string `json:"stateKey"`
		ActionRevision uint64 `json:"actionRevision"`
		StateRevision  uint64 `json:"stateRevision"`
		Outcome        string `json:"outcome"`
		Reason         string `json:"reason"`
	}
	if err := json.Unmarshal(item.Value, &val); err != nil {
		t.Fatal(err)
	}
	if val.Kind != "tinkabot.appActionReceipt.v1" ||
		val.ActionKey != actionKey ||
		val.StateKey != stateKey ||
		val.ActionRevision == 0 ||
		val.StateRevision == 0 ||
		val.Outcome != "rejected" ||
		val.Reason != reason {
		t.Fatalf("rejection value drift: %#v", val)
	}
}

func getTurnState(t *testing.T, owner []string) itemView {
	t.Helper()
	code, out, errOut := runTinkalet(owner, "item", "get", "apps.demo.state.board", "--json")
	if code != 0 || errOut != "" {
		t.Fatalf("state get exit/stderr = %d/%q", code, errOut)
	}
	return decodeItem(t, out)
}

func decodeTurnBoard(t *testing.T, item itemView) turnBoard {
	t.Helper()
	var board turnBoard
	if err := json.Unmarshal(item.Value, &board); err != nil {
		t.Fatalf("turn board json: %v in %#v", err, item)
	}
	if board.Cells == nil {
		board.Cells = map[string]string{}
	}
	return board
}

func decodeTurnAction(t *testing.T, item itemView) (appActionValue, turnMove) {
	t.Helper()
	var act appActionValue
	if err := json.Unmarshal(item.Value, &act); err != nil {
		t.Fatalf("action json: %v in %#v", err, item)
	}
	var move turnMove
	if err := json.Unmarshal(act.Payload, &move); err != nil {
		t.Fatalf("move json: %v in %#v", err, act)
	}
	return act, move
}

func turnPayload(t *testing.T, cell string) string {
	t.Helper()
	body, err := json.Marshal(turnMove{Cell: cell})
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func marshalTurnBoard(t *testing.T, board turnBoard) string {
	t.Helper()
	body, err := json.Marshal(board)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func turnWinner(board turnBoard, participantID string) bool {
	for _, line := range [][]string{
		{"a1", "a2", "a3"},
		{"b1", "b2", "b3"},
		{"c1", "c2", "c3"},
		{"a1", "b1", "c1"},
		{"a2", "b2", "c2"},
		{"a3", "b3", "c3"},
		{"a1", "b2", "c3"},
		{"a3", "b2", "c1"},
	} {
		if board.Cells[line[0]] == participantID && board.Cells[line[1]] == participantID && board.Cells[line[2]] == participantID {
			return true
		}
	}
	return false
}
