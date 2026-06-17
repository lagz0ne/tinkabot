package tinkabot

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestTinkaletItemRecords(t *testing.T) {
	t.Parallel()
	store := t.TempDir()
	app, err := boot(t, cfgFor(store))
	if err != nil {
		t.Fatal(err)
	}
	env := tinkaletEnv(t)
	mustTinkalet(t, env, "profile", "import", "local", "--store", store, "--name", "local")
	mustTinkalet(t, env, "profile", "use", "local")

	code, out, errOut := runTinkalet(env, "item", "create", "deploy/123", "--value", `{"env":"staging"}`, "--json")
	if code != 0 || errOut != "" {
		t.Fatalf("create exit/stderr = %d/%q", code, errOut)
	}
	created := decodeItem(t, out)
	if created.Key != "deploy/123" || created.Status != "pending" || created.Revision == 0 || string(created.Value) != `{"env":"staging"}` {
		t.Fatalf("created item drift: %#v", created)
	}
	if strings.Contains(out, "tb_items") || strings.Contains(out, "$KV") || strings.Contains(out, string(mustReadFile(t, app.CredsFile(RoleCaller)))) {
		t.Fatalf("item json leaked substrate details: %q", out)
	}

	code, out, errOut = runTinkalet(env, "item", "create", "deploy/123")
	if code != 1 || out != "" || errOut != "item deploy/123 denied create: duplicate-item\n" {
		t.Fatalf("duplicate exit/stdout/stderr = %d/%q/%q", code, out, errOut)
	}

	code, out, errOut = runTinkalet(env, "item", "resolve", "deploy/123", "--revision", "0", "--value", `{"approved":false}`)
	if code != 1 || out != "" || errOut != "item deploy/123 denied resolve: stale-revision\n" {
		t.Fatalf("stale resolve exit/stdout/stderr = %d/%q/%q", code, out, errOut)
	}

	waitDone := make(chan itemView, 1)
	waitErr := make(chan string, 1)
	go func() {
		code, out, errOut := runTinkalet(env, "item", "wait", "deploy/123", "--for", "resolved", "--timeout", "5s", "--json")
		if code != 0 || errOut != "" {
			waitErr <- "wait exit/stderr = " + errOut
			return
		}
		var got itemView
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			waitErr <- "wait item json: " + err.Error()
			return
		}
		waitDone <- got
	}()
	time.Sleep(150 * time.Millisecond)

	code, out, errOut = runTinkalet(env, "item", "resolve", "deploy/123", "--revision", revString(created.Revision), "--value", `{"approved":true}`, "--json")
	if code != 0 || errOut != "" {
		t.Fatalf("resolve exit/stderr = %d/%q", code, errOut)
	}
	resolved := decodeItem(t, out)
	if resolved.Status != "resolved" || resolved.Revision <= created.Revision || string(resolved.Value) != `{"approved":true}` {
		t.Fatalf("resolved item drift: %#v after %#v", resolved, created)
	}

	select {
	case got := <-waitDone:
		if got.Status != "resolved" || got.Revision != resolved.Revision || string(got.Value) != `{"approved":true}` {
			t.Fatalf("wait item drift: %#v want %#v", got, resolved)
		}
	case err := <-waitErr:
		t.Fatal(err)
	case <-time.After(6 * time.Second):
		t.Fatal("item wait did not unblock")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := app.Stop(ctx); err != nil {
		t.Fatal(err)
	}
	app2, err := boot(t, cfgFor(store))
	if err != nil {
		t.Fatal(err)
	}
	_ = app2
	mustTinkalet(t, env, "profile", "import", "local", "--store", store, "--name", "local")

	code, out, errOut = runTinkalet(env, "item", "get", "deploy/123", "--json")
	if code != 0 || errOut != "" {
		t.Fatalf("get after restart exit/stderr = %d/%q", code, errOut)
	}
	afterRestart := decodeItem(t, out)
	if afterRestart.Status != "resolved" || string(afterRestart.Value) != `{"approved":true}` {
		t.Fatalf("restart lost item: %#v", afterRestart)
	}
}

type itemView struct {
	Key      string          `json:"key"`
	Status   string          `json:"status"`
	Value    json.RawMessage `json:"value"`
	Revision uint64          `json:"revision"`
}

func decodeItem(t *testing.T, out string) itemView {
	t.Helper()
	var got itemView
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("item json: %v\n%s", err, out)
	}
	return got
}

func revString(rev uint64) string {
	out, _ := json.Marshal(rev)
	return string(out)
}
