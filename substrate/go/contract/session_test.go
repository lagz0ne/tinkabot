package contract

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

var sessionRequired = []string{
	"session-record",
	"session-frame-token",
	"session-frame-chunk",
	"session-frame-status",
	"session-steer-intent",
	"session-stop-intent",
	"session-frame-malformed",
	"session-frame-unknown-kind",
	"session-record-missing-provenance",
}

func TestSessionContractParity(t *testing.T) {
	t.Parallel()
	dir := filepath.Join("..", "..", "..", "schemas", "base", "v1")

	reg, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(filepath.Join(dir, "session.cases.json"))
	if err != nil {
		t.Fatalf("session fixtures are not covered by the contract registry lane: %v", err)
	}

	var cases []parityCase
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}

	seen := map[string]bool{}
	for _, c := range cases {
		seen[c.CaseID] = true
	}
	for _, id := range sessionRequired {
		if !seen[id] {
			t.Fatalf("missing session contract case: %s", id)
		}
	}

	for _, c := range cases {
		t.Run(c.CaseID, func(t *testing.T) {
			doc, err := os.ReadFile(filepath.Join(dir, c.Fixture))
			if err != nil {
				t.Fatal(err)
			}

			err = reg.Validate(ContractSchemaID, doc)
			if c.Expect.Valid && err != nil {
				t.Fatalf("expected valid session fixture: %v", err)
			}
			if !c.Expect.Valid && err == nil {
				t.Fatal("expected invalid session fixture")
			}
		})
	}
}
