package edge

import (
	"errors"
	"testing"
)

func TestGatewayMutationPolicyRejectsUntrustedBrowserRequests(t *testing.T) {
	pol := validGatewayMutationPolicy()
	if err := CheckGatewayMutation(pol, validGatewayMutationRequest()); err != nil {
		t.Fatalf("valid gateway request rejected: %v", err)
	}

	cases := []struct {
		name string
		pol  func(GatewayMutationPolicy) GatewayMutationPolicy
		req  func(GatewayMutationRequest) GatewayMutationRequest
		kind ErrorKind
	}{
		{
			name: "csrf",
			req:  func(r GatewayMutationRequest) GatewayMutationRequest { r.CSRFToken = "csrf-bad"; return r },
			kind: CSRFDenied,
		},
		{
			name: "origin",
			req: func(r GatewayMutationRequest) GatewayMutationRequest {
				r.Origin = "https://generated.invalid"
				return r
			},
			kind: OriginDenied,
		},
		{
			name: "fetch-metadata",
			req:  func(r GatewayMutationRequest) GatewayMutationRequest { r.FetchSite = "cross-site"; return r },
			kind: FetchMetadataDenied,
		},
		{
			name: "credentialed-generated-origin-cors",
			req: func(r GatewayMutationRequest) GatewayMutationRequest {
				r.Origin = "null"
				r.CredentialedCORS = true
				return r
			},
			kind: CredentialedCORSDenied,
		},
		{
			name: "stale-artifact",
			req:  func(r GatewayMutationRequest) GatewayMutationRequest { r.ArtifactRevision = "artifact.rev.6"; return r },
			kind: StaleRevision,
		},
		{
			name: "revoked",
			pol:  func(p GatewayMutationPolicy) GatewayMutationPolicy { p.LeaseStatus = "revoked"; return p },
			kind: RevokedLease,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := pol
			r := validGatewayMutationRequest()
			if c.pol != nil {
				p = c.pol(p)
			}
			if c.req != nil {
				r = c.req(r)
			}

			err := CheckGatewayMutation(p, r)
			var got *EdgeError
			if !errors.As(err, &got) {
				t.Fatalf("expected EdgeError, got %T: %v", err, err)
			}
			if got.Kind != c.kind {
				t.Fatalf("kind mismatch: got %s want %s", got.Kind, c.kind)
			}
			if got.Layer != "BrowserEdge" {
				t.Fatalf("layer mismatch: %s", got.Layer)
			}
		})
	}
}

func TestServiceWorkerSetupRejectsUnsafeScopeAndHeaders(t *testing.T) {
	pol := validServiceWorkerPolicy()
	headers, err := CheckServiceWorkerSetup(pol, validServiceWorkerRequest())
	if err != nil {
		t.Fatalf("valid service worker setup rejected: %v", err)
	}
	if headers["Service-Worker-Allowed"] != "/__tinkabot_session/session-001/" {
		t.Fatalf("allowed scope drift: %#v", headers)
	}

	cases := []struct {
		name string
		pol  func(ServiceWorkerPolicy) ServiceWorkerPolicy
		req  func(ServiceWorkerRequest) ServiceWorkerRequest
		kind ErrorKind
	}{
		{
			name: "wrong-scope",
			req:  func(r ServiceWorkerRequest) ServiceWorkerRequest { r.Scope = "/"; return r },
			kind: ServiceWorkerScopeDenied,
		},
		{
			name: "broad-allowed-scope",
			req:  func(r ServiceWorkerRequest) ServiceWorkerRequest { r.AllowedScope = "/"; return r },
			kind: ServiceWorkerAllowedDenied,
		},
		{
			name: "stale-worker",
			req:  func(r ServiceWorkerRequest) ServiceWorkerRequest { r.WorkerRevision = "worker.rev.0"; return r },
			kind: ServiceWorkerRevisionDenied,
		},
		{
			name: "generated-registration",
			req:  func(r ServiceWorkerRequest) ServiceWorkerRequest { r.GeneratedContent = true; return r },
			kind: ServiceWorkerRegistrationDenied,
		},
		{
			name: "artifact-overlap",
			pol: func(p ServiceWorkerPolicy) ServiceWorkerPolicy {
				p.ArtifactScope = "/__tinkabot_session/session-001/artifacts/"
				return p
			},
			kind: ServiceWorkerScopeDenied,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := pol
			r := validServiceWorkerRequest()
			if c.pol != nil {
				p = c.pol(p)
			}
			if c.req != nil {
				r = c.req(r)
			}

			_, err := CheckServiceWorkerSetup(p, r)
			var got *EdgeError
			if !errors.As(err, &got) {
				t.Fatalf("expected EdgeError, got %T: %v", err, err)
			}
			if got.Kind != c.kind {
				t.Fatalf("kind mismatch: got %s want %s", got.Kind, c.kind)
			}
			if got.Layer != "BrowserEdge" {
				t.Fatalf("layer mismatch: %s", got.Layer)
			}
		})
	}
}

func validGatewayMutationPolicy() GatewayMutationPolicy {
	return GatewayMutationPolicy{
		SessionID:               "session-001",
		LeaseID:                 "lease-001",
		LeaseStatus:             "active",
		CSRFToken:               "csrf-001",
		AllowedOrigin:           "https://app.localhost",
		CurrentArtifactRevision: "artifact.rev.7",
	}
}

func validGatewayMutationRequest() GatewayMutationRequest {
	return GatewayMutationRequest{
		SessionID:        "session-001",
		LeaseID:          "lease-001",
		CSRFToken:        "csrf-001",
		Origin:           "https://app.localhost",
		FetchSite:        "same-origin",
		FetchMode:        "cors",
		FetchDest:        "empty",
		ArtifactRevision: "artifact.rev.7",
		CredentialedCORS: false,
	}
}

func validServiceWorkerPolicy() ServiceWorkerPolicy {
	return ServiceWorkerPolicy{
		SessionID:      "session-001",
		ScriptURL:      "/__tinkabot_session/session-001/sw.js",
		Scope:          "/__tinkabot_session/session-001/",
		AllowedScope:   "/__tinkabot_session/session-001/",
		WorkerRevision: "worker.rev.1",
		ArtifactScope:  "/__tinkabot_artifacts/session-001/",
	}
}

func validServiceWorkerRequest() ServiceWorkerRequest {
	return ServiceWorkerRequest{
		SessionID:      "session-001",
		ScriptURL:      "/__tinkabot_session/session-001/sw.js",
		Scope:          "/__tinkabot_session/session-001/",
		AllowedScope:   "/__tinkabot_session/session-001/",
		WorkerRevision: "worker.rev.1",
	}
}
