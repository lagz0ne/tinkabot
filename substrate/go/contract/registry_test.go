package contract

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type parityCase struct {
	CaseID  string `json:"caseId"`
	Fixture string `json:"fixture"`
	Expect  struct {
		Valid bool `json:"valid"`
	} `json:"expect"`
}

func TestEndgameContractParity(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	dir := filepath.Join(root, "schemas", "endgame", "v1")

	reg, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(filepath.Join(dir, "parity.cases.json"))
	if err != nil {
		t.Fatal(err)
	}

	var cases []parityCase
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}
	if len(cases) == 0 {
		t.Fatal("expected parity cases")
	}

	for _, c := range cases {
		t.Run(c.CaseID, func(t *testing.T) {
			doc, err := os.ReadFile(filepath.Join(dir, c.Fixture))
			if err != nil {
				t.Fatal(err)
			}

			err = reg.Validate("tb.schema.endgame.contract_authority.v1", doc)
			if c.Expect.Valid && err != nil {
				t.Fatalf("expected valid fixture: %v", err)
			}
			if !c.Expect.Valid && err == nil {
				t.Fatal("expected invalid fixture")
			}
		})
	}
}
