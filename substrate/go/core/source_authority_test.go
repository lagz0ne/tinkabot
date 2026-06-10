package core

import "testing"

func TestSourceAuthorityAllowsCanonicalSources(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		fixture string
		subject string
	}{
		{"request", "fixtures/valid/activation-request-reply.json", "tb.proof.runtime.execute"},
		{"command", "fixtures/valid/activation-command-acceptance.json", "tb.proof.runtime.execute"},
		{"subject", "fixtures/valid/activation-source-subject.json", "tb.proof.runtime.execute"},
		{"kv", "fixtures/valid/activation-source-kv.json", "$KV.tb_proof_kv.scripts/proof/state"},
		{"object", "fixtures/valid/activation-source-object.json", "$O.tb_proof_objects.artifact/main.js"},
		{"stream", "fixtures/valid/activation-source-stream.json", "tb.proof.events.created"},
		{"schedule", "fixtures/valid/activation-source-schedule.json", "tb.schedule.sched-daily.tick-001"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			act := activation(t, read(t, c.fixture))
			grant, err := AuthorizeSource(sourceAuth(act), act)
			if err != nil {
				t.Fatal(err)
			}
			if grant.SourceID != act.SourcePrincipal.SourceID || grant.PrincipalID != act.SourcePrincipal.PrincipalID || grant.LeaseID != act.SourceLease.LeaseID {
				t.Fatalf("source identity drift: %#v", grant)
			}
			if grant.Subject != c.subject || grant.Exposure.Subject == "" {
				t.Fatalf("subject/exposure drift: %#v", grant)
			}
			if len(grant.Imports) == 0 || len(grant.Exports) == 0 {
				t.Fatalf("imports/exports were not preserved: %#v", grant)
			}
			if grant.Event.Kind != "activation.source.authorized" || grant.Event.Layer != "SourceAuthority" || grant.Event.Provenance["sourceId"] != act.SourcePrincipal.SourceID {
				t.Fatalf("event drift: %#v", grant.Event)
			}
		})
	}
}

func TestSourceAuthorityDeniesNeighborAndDenyOverAllow(t *testing.T) {
	t.Parallel()
	act := activation(t, read(t, "fixtures/valid/activation-source-denied-neighbor.json"))
	auth := sourceAuth(act)
	_, err := AuthorizeSource(auth, act)
	assertKind(t, err, DeniedNeighbor)

	act = activation(t, edit(t, "fixtures/valid/activation-source-subject.json", func(doc map[string]any) {
		src := doc["source"].(map[string]any)
		src["observedSubject"] = "tb.proof.runtime.secret"
	}))
	auth = sourceAuth(act)
	auth.Permissions.Subscribe.Deny = append(auth.Permissions.Subscribe.Deny, "tb.proof.runtime.secret")
	_, err = AuthorizeSource(auth, act)
	assertKind(t, err, DeniedNeighbor)
}

func TestSourceAuthorityDeniesWildcardAndResponseDrift(t *testing.T) {
	t.Parallel()
	act := activation(t, read(t, "fixtures/valid/activation-source-wildcard-overreach.json"))
	_, err := AuthorizeSource(sourceAuth(act), act)
	assertKind(t, err, WildcardOverreach)

	act = activation(t, edit(t, "fixtures/valid/activation-source-subject.json", func(doc map[string]any) {
		src := doc["source"].(map[string]any)
		src["observedSubject"] = "tb.proof.events.created"
	}))
	_, err = AuthorizeSource(sourceAuth(act), act)
	assertKind(t, err, DeniedNeighbor)

	act = activation(t, read(t, "fixtures/valid/activation-request-reply.json"))
	auth := sourceAuth(act)
	auth.Permissions.AllowResponses = AllowResponses{}
	_, err = AuthorizeSource(auth, act)
	assertKind(t, err, PermissionCompileFailed)
}

func TestSourceAuthorityDeniesLeaseAndRevisionDrift(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		edit func(*Activation, *Auth)
		kind Kind
	}{
		{"principal", func(act *Activation, auth *Auth) { auth.User = "principal.source.other" }, SourceAuthDenied},
		{"mismatched source kind", func(act *Activation, auth *Auth) { act.SourcePrincipal.SourceKind = "kv" }, SourceAuthDenied},
		{"revoked", func(act *Activation, auth *Auth) { act.SourceLease.LeaseStatus = "revoked" }, LeaseRevoked},
		{"expired", func(act *Activation, auth *Auth) { act.SourceLease.LeaseStatus = "expired" }, LeaseExpired},
		{"stale app", func(act *Activation, auth *Auth) { act.SourceLease.AppRevision = "app.rev.old" }, StaleChain},
		{"stale schema", func(act *Activation, auth *Auth) { act.SourceLease.SchemaVersion = "v0" }, StaleChain},
		{"stale script", func(act *Activation, auth *Auth) { act.SourceLease.ScriptRevision = 6 }, StaleChain},
		{"missing exposure", func(act *Activation, auth *Auth) { delete(auth.Exposure, act.SourcePrincipal.AuthorityRef) }, SourceAuthDenied},
		{"missing export", func(act *Activation, auth *Auth) { auth.Exports = nil }, PermissionCompileFailed},
		{"advanced import", func(act *Activation, auth *Auth) { auth.Imports["raw"] = Import{Kind: "raw_nats"} }, PermissionCompileFailed},
		{"advanced exposure", func(act *Activation, auth *Auth) {
			auth.Exposure[act.SourcePrincipal.AuthorityRef] = Exposure{Kind: "stream", Subject: act.Source.Subject}
		}, PermissionCompileFailed},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			act := activation(t, read(t, "fixtures/valid/activation-request-reply.json"))
			auth := sourceAuth(act)
			c.edit(&act, &auth)
			_, err := AuthorizeSource(auth, act)
			assertKind(t, err, c.kind)
			if err != nil {
				ev := err.(*Error).Event()
				if ev.Layer != "SourceAuthority" || ev.Operation != "AuthorizeSource" || ev.Provenance["sourceId"] != act.SourcePrincipal.SourceID || ev.Provenance["origin"] != "activation-source-authority" {
					t.Fatalf("denial attribution drift: %#v", ev)
				}
			}
		})
	}
}

func sourceAuth(act Activation) Auth {
	subject := sourceSubjectForTest(act.Source)
	expKind := map[string]string{
		"request_reply":      "request_reply",
		"command_acceptance": "request_reply",
		"subject":            "subject",
		"kv":                 "kv_watch",
		"object":             "object_change",
		"stream":             "stream",
		"schedule":           "subject",
	}[act.Source.Kind]
	ref := act.SourcePrincipal.AuthorityRef
	return Auth{
		User: act.SourcePrincipal.PrincipalID,
		Capability: Capability{
			PrincipalID:   act.SourcePrincipal.PrincipalID,
			SessionID:     "session-source-001",
			CapabilityID:  "cap-source-001",
			LeaseID:       act.SourceLease.LeaseID,
			LeaseStatus:   "active",
			AppRevision:   act.SourceLease.AppRevision,
			SchemaVersion: act.SourceLease.SchemaVersion,
			Provenance:    act.Provenance,
		},
		Permissions: Permissions{
			Subscribe: PermList{
				Allow: []string{
					"tb.proof.runtime.*",
					"$KV.tb_proof_kv.>",
					"$O.tb_proof_objects.>",
					"tb.proof.events.*",
					"tb.schedule.sched-daily.>",
				},
				Deny: []string{"tb.proof.runtime.denied"},
			},
			AllowResponses: AllowResponses{Max: 1, ExpiresMs: 30000},
		},
		Imports: map[string]Import{
			"source": {Kind: "subscribe", Subjects: []string{subject}, Desc: "Source observation aperture."},
		},
		Exports: []string{subject},
		Exposure: map[string]Exposure{
			ref: {Kind: expKind, Subject: subject, Desc: "Source exposure."},
		},
	}
}

func sourceSubjectForTest(src Source) string {
	switch src.Kind {
	case "request_reply", "command_acceptance":
		return src.Subject
	case "subject":
		return src.ObservedSubject
	case "kv":
		return "$KV." + src.Bucket + "." + src.Key
	case "object":
		return "$O." + src.Bucket + "." + src.Name
	case "stream":
		return src.Subject
	case "schedule":
		return "tb.schedule." + src.ScheduleID + "." + src.TickID
	default:
		return ""
	}
}
