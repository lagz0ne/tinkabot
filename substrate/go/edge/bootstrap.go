package edge

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lagz0ne/tinkabot/substrate/go/contract"
)

type ErrorKind string

const (
	ContractInvalid              ErrorKind = "ContractInvalid"
	RevokedLease                 ErrorKind = "RevokedLease"
	ExpiredLease                 ErrorKind = "ExpiredLease"
	StaleRevision                ErrorKind = "StaleRevision"
	ArtifactDigestMismatch       ErrorKind = "ArtifactDigestMismatch"
	ArtifactGatewayPolicyInvalid ErrorKind = "ArtifactGatewayPolicyInvalid"
	CredentialDescriptorInvalid  ErrorKind = "CredentialDescriptorInvalid"
	SubstrateEdgeCritical        ErrorKind = "SubstrateEdgeCritical"
)

type EdgeError struct {
	Kind      ErrorKind
	Layer     string
	Operation string
	Message   string
	Details   map[string]string
}

func (e *EdgeError) Error() string {
	return fmt.Sprintf("%s.%s: %s", e.Layer, e.Kind, e.Message)
}

type BootstrapOptions struct {
	CredentialRef   string
	ObjectNamespace string
	ExpectedDigest  string
}

type Bootstrap struct {
	SchemaID         string
	AppRevision      string
	PrincipalID      string
	SessionID        string
	CapabilityID     string
	LeaseID          string
	ArtifactRevision string
	Credential       *WorkerCredential
	Gateway          ArtifactGatewayPolicy
}

type WorkerCredential struct {
	Kind             string
	Ref              string
	PrincipalID      string
	SessionID        string
	CapabilityID     string
	LeaseID          string
	SchemaVersion    string
	AppRevision      string
	ArtifactRevision string
	PublishAllow     []string
	PublishDeny      []string
	SubscribeAllow   []string
	SubscribeDeny    []string
}

type ArtifactGatewayPolicy struct {
	ArtifactID       string
	ArtifactRevision string
	Digest           string
	MediaType        string
	ObjectRef        string
	ObjectNamespace  string
	CSPPolicy        string
	FramePolicy      string
	SandboxPolicy    string
}

func BuildBootstrap(reg *contract.Registry, authDoc, artifactDoc []byte, opts BootstrapOptions) (*Bootstrap, error) {
	if reg == nil {
		return nil, err(SubstrateEdgeCritical, "GoSubstrate", "BuildBootstrap", "contract registry is required", nil)
	}
	if err := reg.Validate(contract.ContractSchemaID, authDoc); err != nil {
		return nil, wrap(ContractInvalid, "ContractAuthority", "validateAuthPolicy", "auth policy contract is invalid", err)
	}

	var policy authPolicy
	if err := json.Unmarshal(authDoc, &policy); err != nil {
		return nil, wrap(ContractInvalid, "ContractAuthority", "decodeAuthPolicy", "auth policy JSON is invalid", err)
	}
	if policy.Kind != "auth.policy" {
		return nil, err(ContractInvalid, "ContractAuthority", "decodeAuthPolicy", "expected auth.policy", nil)
	}
	if e := authorize(policy); e != nil {
		return nil, e
	}

	if err := reg.Validate(contract.ContractSchemaID, artifactDoc); err != nil {
		return nil, wrap(ContractInvalid, "ContractAuthority", "validateArtifactManifest", "artifact manifest contract is invalid", err)
	}

	var artifact artifactManifest
	if err := json.Unmarshal(artifactDoc, &artifact); err != nil {
		return nil, wrap(ContractInvalid, "ContractAuthority", "decodeArtifactManifest", "artifact manifest JSON is invalid", err)
	}
	if artifact.Kind != "artifact.manifest" {
		return nil, err(ContractInvalid, "ContractAuthority", "decodeArtifactManifest", "expected artifact.manifest", nil)
	}

	gateway, e := gatewayPolicy(artifact, opts)
	if e != nil {
		return nil, e
	}

	ref := strings.TrimSpace(opts.CredentialRef)
	if ref == "" {
		return nil, err(CredentialDescriptorInvalid, "GoSubstrate", "BuildBootstrap", "credential ref is required", map[string]string{
			"field": "credentialRef",
		})
	}

	cap := policy.Capability
	cred := &WorkerCredential{
		Kind:             "browser.worker_nats",
		Ref:              ref,
		PrincipalID:      cap.PrincipalID,
		SessionID:        cap.SessionID,
		CapabilityID:     cap.CapabilityID,
		LeaseID:          cap.LeaseID,
		SchemaVersion:    cap.SchemaVersion,
		AppRevision:      cap.AppRevision,
		ArtifactRevision: artifact.ArtifactRevision,
		PublishAllow:     copyStrings(policy.Permissions.Publish.Allow),
		PublishDeny:      copyStrings(policy.Permissions.Publish.Deny),
		SubscribeAllow:   copyStrings(policy.Permissions.Subscribe.Allow),
		SubscribeDeny:    copyStrings(policy.Permissions.Subscribe.Deny),
	}

	return &Bootstrap{
		SchemaID:         policy.Provenance.SchemaID,
		AppRevision:      cap.AppRevision,
		PrincipalID:      cap.PrincipalID,
		SessionID:        cap.SessionID,
		CapabilityID:     cap.CapabilityID,
		LeaseID:          cap.LeaseID,
		ArtifactRevision: artifact.ArtifactRevision,
		Credential:       cred,
		Gateway:          gateway,
	}, nil
}

func authorize(policy authPolicy) *EdgeError {
	cap := policy.Capability
	details := map[string]string{
		"principalId":           cap.PrincipalID,
		"sessionId":             cap.SessionID,
		"capabilityId":          cap.CapabilityID,
		"leaseId":               cap.LeaseID,
		"leaseStatus":           cap.LeaseStatus,
		"appRevision":           cap.AppRevision,
		"schemaVersion":         cap.SchemaVersion,
		"provenanceAppRevision": policy.Provenance.AppRevision,
	}

	switch cap.LeaseStatus {
	case "revoked":
		return err(RevokedLease, "ManagedAuth", "authorizeBootstrap", "capability lease is revoked", details)
	case "expired":
		return err(ExpiredLease, "ManagedAuth", "authorizeBootstrap", "capability lease is expired", details)
	}
	if policy.Provenance.AppRevision != cap.AppRevision || policy.Provenance.SchemaVersion != cap.SchemaVersion {
		return err(StaleRevision, "ManagedAuth", "authorizeBootstrap", "policy provenance does not match capability revision", details)
	}
	return nil
}

func gatewayPolicy(artifact artifactManifest, opts BootstrapOptions) (ArtifactGatewayPolicy, *EdgeError) {
	namespace := opts.ObjectNamespace
	if namespace == "" {
		namespace = "obj://frontend/"
	}
	if opts.ExpectedDigest != "" && artifact.Digest != opts.ExpectedDigest {
		return ArtifactGatewayPolicy{}, err(ArtifactDigestMismatch, "ArtifactGateway", "gatewayPolicy", "artifact digest does not match expected digest", map[string]string{
			"field":    "digest",
			"expected": opts.ExpectedDigest,
			"actual":   artifact.Digest,
		})
	}
	if !strings.HasPrefix(artifact.ObjectRef, namespace) {
		return ArtifactGatewayPolicy{}, err(ArtifactGatewayPolicyInvalid, "ArtifactGateway", "gatewayPolicy", "artifact object ref is outside the allowed namespace", map[string]string{
			"field":     "objectRef",
			"namespace": namespace,
			"objectRef": artifact.ObjectRef,
		})
	}
	if artifact.CSPPolicy == "" {
		return ArtifactGatewayPolicy{}, missingGatewayPolicy("cspPolicy")
	}
	if artifact.FramePolicy == "" {
		return ArtifactGatewayPolicy{}, missingGatewayPolicy("framePolicy")
	}
	if !strings.HasPrefix(artifact.SandboxPolicy, "sandbox.browser.") {
		return ArtifactGatewayPolicy{}, err(ArtifactGatewayPolicyInvalid, "ArtifactGateway", "gatewayPolicy", "artifact sandbox policy is not browser scoped", map[string]string{
			"field":         "sandboxPolicy",
			"sandboxPolicy": artifact.SandboxPolicy,
		})
	}

	return ArtifactGatewayPolicy{
		ArtifactID:       artifact.ArtifactID,
		ArtifactRevision: artifact.ArtifactRevision,
		Digest:           artifact.Digest,
		MediaType:        artifact.MediaType,
		ObjectRef:        artifact.ObjectRef,
		ObjectNamespace:  namespace,
		CSPPolicy:        artifact.CSPPolicy,
		FramePolicy:      artifact.FramePolicy,
		SandboxPolicy:    artifact.SandboxPolicy,
	}, nil
}

func missingGatewayPolicy(field string) *EdgeError {
	return err(ArtifactGatewayPolicyInvalid, "ArtifactGateway", "gatewayPolicy", "artifact gateway policy is incomplete", map[string]string{
		"field": field,
	})
}

func err(kind ErrorKind, layer, operation, message string, details map[string]string) *EdgeError {
	if details == nil {
		details = map[string]string{}
	}
	return &EdgeError{
		Kind:      kind,
		Layer:     layer,
		Operation: operation,
		Message:   message,
		Details:   details,
	}
}

func wrap(kind ErrorKind, layer, operation, message string, cause error) *EdgeError {
	return err(kind, layer, operation, fmt.Sprintf("%s: %v", message, cause), nil)
}

func copyStrings(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	out := make([]string, len(items))
	copy(out, items)
	return out
}

type authPolicy struct {
	Kind        string     `json:"kind"`
	Provenance  provenance `json:"provenance"`
	Capability  capability `json:"capability"`
	Permissions permission `json:"permissions"`
}

type permission struct {
	Publish   permList `json:"publish"`
	Subscribe permList `json:"subscribe"`
}

type permList struct {
	Allow []string `json:"allow"`
	Deny  []string `json:"deny"`
}

type provenance struct {
	SchemaID      string `json:"schemaId"`
	SchemaVersion string `json:"schemaVersion"`
	AppRevision   string `json:"appRevision"`
	CreatedAt     string `json:"createdAt"`
	Producer      string `json:"producer"`
}

type capability struct {
	PrincipalID   string `json:"principalId"`
	SessionID     string `json:"sessionId"`
	CapabilityID  string `json:"capabilityId"`
	LeaseID       string `json:"leaseId"`
	LeaseStatus   string `json:"leaseStatus"`
	AppRevision   string `json:"appRevision"`
	SchemaVersion string `json:"schemaVersion"`
}

type artifactManifest struct {
	Kind             string     `json:"kind"`
	ArtifactID       string     `json:"artifactId"`
	ArtifactRevision string     `json:"artifactRevision"`
	Digest           string     `json:"digest"`
	MediaType        string     `json:"mediaType"`
	ObjectRef        string     `json:"objectRef"`
	SandboxPolicy    string     `json:"sandboxPolicy"`
	FramePolicy      string     `json:"framePolicy"`
	CSPPolicy        string     `json:"cspPolicy"`
	Provenance       provenance `json:"provenance"`
}
