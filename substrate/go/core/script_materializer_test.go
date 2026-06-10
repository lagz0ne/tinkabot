package core

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestScriptRuntimeMaterializesMediatedEffects(t *testing.T) {
	act := activation(t, read(t, "fixtures/valid/activation-command-acceptance.json"))
	store := NewMemoryMaterialStore()
	runner := ScriptRunnerFunc(func(inv ScriptInvocation) (ScriptRun, error) {
		if _, ok := inv.Env["NATS_URL"]; ok {
			t.Fatal("script received raw NATS URL")
		}
		if _, ok := inv.Env["NATS_CREDS"]; ok {
			t.Fatal("script received raw NATS credentials")
		}
		if _, ok := inv.Env["SUBJECT"]; ok {
			t.Fatal("script received raw subject authority")
		}
		if _, ok := inv.Env["REPLY_TO"]; ok {
			t.Fatal("script received raw reply authority")
		}
		if _, ok := inv.Env["PUBLISH_SUBJECT"]; ok {
			t.Fatal("script received raw publish authority")
		}
		if _, ok := inv.Env["PERMISSIONS"]; ok {
			t.Fatal("script received raw permission authority")
		}
		if _, ok := inv.Env["NKEY_SEED"]; ok {
			t.Fatal("script received raw nkey seed")
		}
		if _, ok := inv.Env["USER_JWT"]; ok {
			t.Fatal("script received raw JWT")
		}
		if _, ok := inv.Env["BEARER_PASSWORD"]; ok {
			t.Fatal("script received raw bearer password")
		}
		if inv.Env["SAFE_MODE"] != "script" {
			t.Fatalf("safe env missing: %#v", inv.Env)
		}
		return ScriptRun{Effects: []ScriptEffect{
			{
				Type:             ProjectionEffect,
				ProjectionID:     "main",
				SnapshotRevision: "snap-001",
				ArtifactRevision: "artifact.rev.7",
				Sequence:         1,
				Value:            json.RawMessage(`{"title":"from-script"}`),
			},
			{
				Type:             ArtifactEffect,
				ArtifactName:     "artifact/main.js",
				ArtifactRevision: "artifact.rev.7",
				MediaType:        "application/javascript",
				Body:             []byte("export default 1"),
			},
		}}, nil
	})
	rt, err := NewScriptRuntime(ScriptPolicy{
		AllowedProjections: []string{"main"},
		ArtifactPrefix:     "artifact/",
		Env: map[string]string{
			"NATS_URL":        "nats://example",
			"NATS_CREDS":      "raw.creds",
			"SUBJECT":         "tb.internal.escape",
			"REPLY_TO":        "_INBOX.x",
			"PUBLISH_SUBJECT": "tb.internal.escape",
			"PERMISSIONS":     "raw",
			"NKEY_SEED":       "SUA...",
			"USER_JWT":        "jwt",
			"BEARER_PASSWORD": "secret",
			"SAFE_MODE":       "script",
		},
	}, runner)
	if err != nil {
		t.Fatal(err)
	}

	acc := accepted(act)
	run, err := rt.Run(acc, scriptRecord(act))
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != "applied" || run.ActivationID != act.ActivationID || len(run.Effects) != 2 {
		t.Fatalf("run drift: %#v", run)
	}
	mat, err := NewMaterializer(store)
	if err != nil {
		t.Fatal(err)
	}
	if err := mat.Apply(MaterialContext{Accepted: acc, Record: scriptRecord(act)}, run.Effects); err != nil {
		t.Fatal(err)
	}
	proj, ok := store.Projection("main")
	if !ok || proj.Kind != "material.projection" || proj.ObservedAt == "" || proj.Provenance.Producer != "script-materializer" || proj.Sequence != 1 || string(proj.Value) != `{"title":"from-script"}` {
		t.Fatalf("projection drift: %#v", proj)
	}
	art, ok := store.Artifact("artifact/main.js")
	if !ok || art.Kind != "artifact.manifest" || art.ArtifactRevision != "artifact.rev.7" || string(art.Body) != "export default 1" {
		t.Fatalf("artifact drift: %#v", art)
	}
}

func TestScriptRuntimeAttributesFailures(t *testing.T) {
	act := activation(t, read(t, "fixtures/valid/activation-command-acceptance.json"))

	t.Run("revision", func(t *testing.T) {
		rt := runtimeWith(t, ScriptRun{})
		rec := scriptRecord(act)
		rec.Revision++
		_, err := rt.Run(accepted(act), rec)
		assertKind(t, err, ScriptRevisionMismatch)
	})

	t.Run("record kind", func(t *testing.T) {
		rt := runtimeWith(t, ScriptRun{})
		rec := scriptRecord(act)
		rec.Kind = "script.note"
		_, err := rt.Run(accepted(act), rec)
		assertKind(t, err, ProtocolFrameInvalid)
	})

	t.Run("facade denied", func(t *testing.T) {
		rt := runtimeWith(t, ScriptRun{Effects: []ScriptEffect{{Type: PublishEffect, Subject: "tb.internal.escape"}}})
		_, err := rt.Run(accepted(act), scriptRecord(act))
		assertKind(t, err, FacadeEffectDenied)
	})

	t.Run("raw projection vocabulary denied", func(t *testing.T) {
		for _, raw := range []string{"subject", "reply", "permission", "natsToken", "publish", "nkeySeed", "userJwt", "bearerPassword"} {
			rt := runtimeWith(t, ScriptRun{Effects: []ScriptEffect{{
				Type:             ProjectionEffect,
				ProjectionID:     "main",
				SnapshotRevision: "snap-001",
				ArtifactRevision: "artifact.rev.7",
				Sequence:         1,
				Value:            json.RawMessage(`{"` + raw + `":"tb.internal.escape"}`),
			}}})
			_, err := rt.Run(accepted(act), scriptRecord(act))
			assertKind(t, err, FacadeEffectDenied)
		}
	})

	t.Run("projection conflict", func(t *testing.T) {
		store := NewMemoryMaterialStore()
		if err := store.SaveProjection(MaterialProjection{ProjectionID: "main", Sequence: 2, Value: json.RawMessage(`{"title":"new"}`)}); err != nil {
			t.Fatal(err)
		}
		runner := ScriptRunnerFunc(func(ScriptInvocation) (ScriptRun, error) {
			return ScriptRun{Effects: []ScriptEffect{{
				Type:             ProjectionEffect,
				ProjectionID:     "main",
				SnapshotRevision: "snap-old",
				ArtifactRevision: "artifact.rev.7",
				Sequence:         1,
				Value:            json.RawMessage(`{"title":"old"}`),
			}}}, nil
		})
		rt, err := NewScriptRuntime(ScriptPolicy{AllowedProjections: []string{"main"}}, runner)
		if err != nil {
			t.Fatal(err)
		}
		run, err := rt.Run(accepted(act), scriptRecord(act))
		if err != nil {
			t.Fatal(err)
		}
		mat, err := NewMaterializer(store)
		if err != nil {
			t.Fatal(err)
		}
		err = mat.Apply(MaterialContext{Accepted: accepted(act), Record: scriptRecord(act)}, run.Effects)
		assertKind(t, err, ProjectionConflict)
	})

	t.Run("process failure", func(t *testing.T) {
		rt := runtimeErr(t, errors.New("exit 1"))
		_, err := rt.Run(accepted(act), scriptRecord(act))
		assertKind(t, err, ScriptProcessFailed)
	})

	t.Run("cleanup failure", func(t *testing.T) {
		rt := runtimeWith(t, ScriptRun{CleanupErr: errors.New("cleanup failed")})
		_, err := rt.Run(accepted(act), scriptRecord(act))
		assertKind(t, err, CleanupFailed)
	})

	t.Run("artifact malformed", func(t *testing.T) {
		rt := runtimeWith(t, ScriptRun{Effects: []ScriptEffect{{Type: ArtifactEffect, ArtifactRevision: "artifact.rev.7", MediaType: "text/plain", Body: []byte("missing name")}}})
		_, err := rt.Run(accepted(act), scriptRecord(act))
		assertKind(t, err, ProtocolFrameInvalid)
	})

	t.Run("artifact write failure", func(t *testing.T) {
		mat, err := NewMaterializer(failingMaterialStore{})
		if err != nil {
			t.Fatal(err)
		}
		err = mat.Apply(MaterialContext{Accepted: accepted(act), Record: scriptRecord(act)}, []ScriptEffect{{
			Type:             ArtifactEffect,
			ArtifactName:     "artifact/main.js",
			ArtifactRevision: "artifact.rev.7",
			MediaType:        "application/javascript",
			Body:             []byte("export default 1"),
		}})
		assertKind(t, err, ArtifactWriteFailed)
	})

	t.Run("materializer raw activation", func(t *testing.T) {
		mat, err := NewMaterializer(NewMemoryMaterialStore())
		if err != nil {
			t.Fatal(err)
		}
		err = mat.Apply(MaterialContext{Accepted: AcceptedActivation{Activation: act}, Record: scriptRecord(act)}, nil)
		assertKind(t, err, ExecutionStateInvalid)
	})

	t.Run("raw activation", func(t *testing.T) {
		rt := runtimeWith(t, ScriptRun{})
		_, err := rt.Run(AcceptedActivation{Activation: act}, scriptRecord(act))
		assertKind(t, err, ExecutionStateInvalid)
	})
}

func runtimeWith(t *testing.T, run ScriptRun) *ScriptRuntime {
	t.Helper()
	rt, err := NewScriptRuntime(ScriptPolicy{AllowedProjections: []string{"main"}}, ScriptRunnerFunc(func(ScriptInvocation) (ScriptRun, error) {
		return run, nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	return rt
}

func runtimeErr(t *testing.T, runErr error) *ScriptRuntime {
	t.Helper()
	rt, err := NewScriptRuntime(ScriptPolicy{AllowedProjections: []string{"main"}}, ScriptRunnerFunc(func(ScriptInvocation) (ScriptRun, error) {
		return ScriptRun{}, runErr
	}))
	if err != nil {
		t.Fatal(err)
	}
	return rt
}

type failingMaterialStore struct{}

func (failingMaterialStore) SaveProjection(MaterialProjection) error {
	return nil
}

func (failingMaterialStore) SaveArtifact(MaterialArtifact) error {
	return fail(ArtifactWriteFailed, "Materializer", "SaveArtifact", "artifact write failed", nil)
}

func scriptRecord(act Activation) ScriptRecord {
	return ScriptRecord{
		Kind:     "script.record",
		Key:      act.ScriptKey,
		Revision: act.ScriptRevision,
		Process: Process{
			Command:   "/bin/sh",
			Args:      []string{"-c", "true"},
			Cwd:       ".",
			RPC:       FramedStdio,
			TimeoutMs: 1000,
			Resource:  Resource{CPUMillis: 100, MemoryMB: 64},
			Kill:      "process.kill",
			Cleanup:   "workdir.delete",
			Identity:  "principal.script.001",
		},
	}
}

func accepted(act Activation) AcceptedActivation {
	return AcceptedActivation{
		Activation: act,
		Record: LedgerRecord{
			ActivationID: act.ActivationID,
			SourceID:     act.SourcePrincipal.SourceID,
			SourceKind:   act.Source.Kind,
			DedupeKey:    act.DedupeKey,
			ChainID:      act.Chain.ChainID,
			Status:       Accepted,
		},
	}
}
