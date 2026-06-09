package edge

import "strings"

type GatewayMutationPolicy struct {
	SessionID               string
	LeaseID                 string
	LeaseStatus             string
	CSRFToken               string
	AllowedOrigin           string
	CurrentArtifactRevision string
}

type GatewayMutationRequest struct {
	SessionID        string `json:"sessionId"`
	LeaseID          string `json:"leaseId"`
	CSRFToken        string `json:"csrfToken"`
	Origin           string `json:"origin"`
	FetchSite        string `json:"fetchSite"`
	FetchMode        string `json:"fetchMode"`
	FetchDest        string `json:"fetchDest"`
	ArtifactRevision string `json:"artifactRevision"`
	CredentialedCORS bool   `json:"credentialedCors"`
}

type ServiceWorkerPolicy struct {
	SessionID      string
	ScriptURL      string
	Scope          string
	AllowedScope   string
	WorkerRevision string
	ArtifactScope  string
}

type ServiceWorkerRequest struct {
	SessionID        string
	ScriptURL        string
	Scope            string
	AllowedScope     string
	WorkerRevision   string
	GeneratedContent bool
}

func CheckGatewayMutation(pol GatewayMutationPolicy, req GatewayMutationRequest) error {
	if pol.SessionID == "" || req.SessionID != pol.SessionID || req.LeaseID != pol.LeaseID {
		return browserErr(GatewaySessionDenied, "CheckGatewayMutation", "gateway session or lease does not match", map[string]string{
			"sessionId": req.SessionID,
			"leaseId":   req.LeaseID,
		})
	}
	switch pol.LeaseStatus {
	case "revoked":
		return browserErr(RevokedLease, "CheckGatewayMutation", "browser gateway lease is revoked", map[string]string{"leaseId": pol.LeaseID})
	case "expired":
		return browserErr(ExpiredLease, "CheckGatewayMutation", "browser gateway lease is expired", map[string]string{"leaseId": pol.LeaseID})
	}
	if req.CredentialedCORS && req.Origin != pol.AllowedOrigin {
		return browserErr(CredentialedCORSDenied, "CheckGatewayMutation", "credentialed CORS is denied outside the trusted shell origin", map[string]string{
			"origin": req.Origin,
		})
	}
	if req.CSRFToken == "" || req.CSRFToken != pol.CSRFToken {
		return browserErr(CSRFDenied, "CheckGatewayMutation", "browser gateway CSRF token is invalid", nil)
	}
	if req.Origin != pol.AllowedOrigin {
		return browserErr(OriginDenied, "CheckGatewayMutation", "browser gateway origin is denied", map[string]string{
			"origin": req.Origin,
		})
	}
	if req.FetchSite != "same-origin" || !oneOf(req.FetchMode, "cors", "same-origin") || req.FetchDest != "empty" {
		return browserErr(FetchMetadataDenied, "CheckGatewayMutation", "browser gateway fetch metadata is denied", map[string]string{
			"site": req.FetchSite,
			"mode": req.FetchMode,
			"dest": req.FetchDest,
		})
	}
	if req.ArtifactRevision != pol.CurrentArtifactRevision {
		return browserErr(StaleRevision, "CheckGatewayMutation", "browser gateway artifact revision is stale", map[string]string{
			"expected": pol.CurrentArtifactRevision,
			"actual":   req.ArtifactRevision,
		})
	}
	return nil
}

func CheckServiceWorkerSetup(pol ServiceWorkerPolicy, req ServiceWorkerRequest) (map[string]string, error) {
	if req.GeneratedContent {
		return nil, browserErr(ServiceWorkerRegistrationDenied, "CheckServiceWorkerSetup", "generated content cannot register service workers", nil)
	}
	if pol.SessionID == "" || req.SessionID != pol.SessionID {
		return nil, browserErr(GatewaySessionDenied, "CheckServiceWorkerSetup", "service worker session does not match", map[string]string{
			"sessionId": req.SessionID,
		})
	}
	if req.ScriptURL != pol.ScriptURL {
		return nil, browserErr(ServiceWorkerScriptDenied, "CheckServiceWorkerSetup", "service worker script URL is denied", map[string]string{
			"scriptUrl": req.ScriptURL,
		})
	}
	if req.Scope != pol.Scope {
		return nil, browserErr(ServiceWorkerScopeDenied, "CheckServiceWorkerSetup", "service worker scope is denied", map[string]string{
			"scope": req.Scope,
		})
	}
	if req.AllowedScope != pol.AllowedScope || req.AllowedScope != req.Scope {
		return nil, browserErr(ServiceWorkerAllowedDenied, "CheckServiceWorkerSetup", "Service-Worker-Allowed scope is too broad", map[string]string{
			"allowedScope": req.AllowedScope,
		})
	}
	if overlap(req.Scope, pol.ArtifactScope) {
		return nil, browserErr(ServiceWorkerScopeDenied, "CheckServiceWorkerSetup", "service worker scope overlaps generated artifacts", map[string]string{
			"scope":         req.Scope,
			"artifactScope": pol.ArtifactScope,
		})
	}
	if req.WorkerRevision != pol.WorkerRevision {
		return nil, browserErr(ServiceWorkerRevisionDenied, "CheckServiceWorkerSetup", "service worker revision is stale", map[string]string{
			"expected": pol.WorkerRevision,
			"actual":   req.WorkerRevision,
		})
	}
	return map[string]string{
		"Service-Worker-Allowed": pol.AllowedScope,
		"Cache-Control":          "no-store",
		"X-Tinkabot-Worker-Rev":  pol.WorkerRevision,
	}, nil
}

func browserErr(kind ErrorKind, op, msg string, details map[string]string) *EdgeError {
	return err(kind, "BrowserEdge", op, msg, details)
}

func oneOf(value string, allowed ...string) bool {
	for _, item := range allowed {
		if value == item {
			return true
		}
	}
	return false
}

func overlap(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	return strings.HasPrefix(a, b) || strings.HasPrefix(b, a)
}
