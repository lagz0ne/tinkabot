package edge

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/lagz0ne/tinkabot/substrate/go/contract"
)

func TestSubstrateConsumesContractsIntoBootstrapDescriptor(t *testing.T) {
	t.Parallel()
	reg := registry(t)
	boot, err := BuildBootstrap(reg, read(t, "fixtures/valid/auth-policy.json"), read(t, "fixtures/valid/artifact-manifest.json"), BootstrapOptions{
		CredentialRef:   "credential.browser.worker.001",
		ObjectNamespace: "obj://frontend/",
		ExpectedDigest:  "sha256:5a1e",
	})
	if err != nil {
		t.Fatal(err)
	}

	if boot.SchemaID != contract.ContractSchemaID {
		t.Fatalf("schema id drift: %s", boot.SchemaID)
	}
	if boot.AppRevision != "app.rev.1" {
		t.Fatalf("app revision drift: %s", boot.AppRevision)
	}
	if boot.PrincipalID != "principal.browser.001" || boot.SessionID != "session-001" {
		t.Fatalf("identity drift: %#v", boot)
	}
	if boot.Credential == nil {
		t.Fatal("expected worker credential descriptor")
	}
	if boot.Credential.Ref != "credential.browser.worker.001" {
		t.Fatalf("credential ref drift: %s", boot.Credential.Ref)
	}
	if boot.Credential.ArtifactRevision != "artifact.rev.7" {
		t.Fatalf("artifact revision drift: %s", boot.Credential.ArtifactRevision)
	}
	if boot.Gateway.ObjectRef != "obj://frontend/artifact-001/rev-7/bundle.js" {
		t.Fatalf("object ref drift: %s", boot.Gateway.ObjectRef)
	}
	if boot.Gateway.CSPPolicy != "csp.subapp.v1" || boot.Gateway.FramePolicy != "frame.subapp.v1" || boot.Gateway.SandboxPolicy != "sandbox.browser.subapp.v1" {
		t.Fatalf("gateway policy drift: %#v", boot.Gateway)
	}
}

func TestSubstrateDeniesLeaseBeforeCredentialDescriptor(t *testing.T) {
	t.Parallel()
	reg := registry(t)
	cases := []struct {
		name    string
		fixture string
		kind    ErrorKind
	}{
		{"revoked", "fixtures/valid/auth-policy-revoked-lease.json", RevokedLease},
		{"expired", "fixtures/valid/auth-policy-expired-lease.json", ExpiredLease},
		{"provenance", "fixtures/valid/auth-policy-provenance-mismatch.json", StaleRevision},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			boot, err := BuildBootstrap(reg, read(t, c.fixture), read(t, "fixtures/valid/artifact-manifest.json"), validOptions())
			if boot != nil {
				t.Fatalf("credential descriptor was created before denial: %#v", boot.Credential)
			}

			var got *EdgeError
			if !errors.As(err, &got) {
				t.Fatalf("expected EdgeError, got %T: %v", err, err)
			}
			if got.Kind != c.kind {
				t.Fatalf("kind mismatch: got %s want %s", got.Kind, c.kind)
			}
			if got.Layer != "ManagedAuth" {
				t.Fatalf("layer mismatch: %s", got.Layer)
			}
		})
	}
}

func TestSubstrateRejectsMissingCredentialDescriptorRef(t *testing.T) {
	t.Parallel()
	reg := registry(t)
	boot, err := BuildBootstrap(reg, read(t, "fixtures/valid/auth-policy.json"), read(t, "fixtures/valid/artifact-manifest.json"), BootstrapOptions{
		ObjectNamespace: "obj://frontend/",
		ExpectedDigest:  "sha256:5a1e",
	})
	if boot != nil {
		t.Fatalf("bootstrap created without credential ref: %#v", boot)
	}

	var got *EdgeError
	if !errors.As(err, &got) {
		t.Fatalf("expected EdgeError, got %T: %v", err, err)
	}
	if got.Kind != CredentialDescriptorInvalid {
		t.Fatalf("kind mismatch: got %s", got.Kind)
	}
	if got.Details["field"] != "credentialRef" {
		t.Fatalf("detail mismatch: %#v", got.Details)
	}
}

func TestArtifactGatewayPolicyRejectsUnsafeManifest(t *testing.T) {
	t.Parallel()
	reg := registry(t)

	if _, err := BuildBootstrap(reg, read(t, "fixtures/valid/auth-policy.json"), read(t, "fixtures/valid/artifact-manifest.json"), validOptions()); err != nil {
		t.Fatalf("valid manifest rejected: %v", err)
	}

	cases := []struct {
		name   string
		edit   func(map[string]any)
		kind   ErrorKind
		detail string
	}{
		{"digest", func(doc map[string]any) { doc["digest"] = "sha256:bad" }, ArtifactDigestMismatch, "digest"},
		{"namespace", func(doc map[string]any) { doc["objectRef"] = "obj://control/artifact-001/rev-7/bundle.js" }, ArtifactGatewayPolicyInvalid, "objectRef"},
		{"csp", func(doc map[string]any) { delete(doc, "cspPolicy") }, ArtifactGatewayPolicyInvalid, "cspPolicy"},
		{"frame", func(doc map[string]any) { delete(doc, "framePolicy") }, ArtifactGatewayPolicyInvalid, "framePolicy"},
		{"sandbox", func(doc map[string]any) { doc["sandboxPolicy"] = "sandbox.none" }, ArtifactGatewayPolicyInvalid, "sandboxPolicy"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := BuildBootstrap(reg, read(t, "fixtures/valid/auth-policy.json"), manifest(t, c.edit), validOptions())
			var got *EdgeError
			if !errors.As(err, &got) {
				t.Fatalf("expected EdgeError, got %T: %v", err, err)
			}
			if got.Kind != c.kind {
				t.Fatalf("kind mismatch: got %s want %s", got.Kind, c.kind)
			}
			if got.Details["field"] != c.detail {
				t.Fatalf("detail mismatch: %#v", got.Details)
			}
		})
	}
}

func registry(t *testing.T) *contract.Registry {
	t.Helper()
	reg, err := contract.Open(schemaDir())
	if err != nil {
		t.Fatal(err)
	}
	return reg
}

func validOptions() BootstrapOptions {
	return BootstrapOptions{
		CredentialRef:   "credential.browser.worker.001",
		ObjectNamespace: "obj://frontend/",
		ExpectedDigest:  "sha256:5a1e",
	}
}

func read(t *testing.T, fixture string) []byte {
	t.Helper()
	doc, err := os.ReadFile(filepath.Join(schemaDir(), fixture))
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

func manifest(t *testing.T, edit func(map[string]any)) []byte {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal(read(t, "fixtures/valid/artifact-manifest.json"), &doc); err != nil {
		t.Fatal(err)
	}
	edit(doc)
	out, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func schemaDir() string {
	return filepath.Join("..", "..", "..", "schemas", "base", "v1")
}
