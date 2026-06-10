package contract

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

const ContractSchemaID = "tb.schema.base.contract_authority.v1"

type Registry struct {
	schemas map[string]*jsonschema.Schema
}

func Open(dir string) (*Registry, error) {
	raw, err := os.Open(filepath.Join(dir, "contract.schema.json"))
	if err != nil {
		return nil, fmt.Errorf("open schema: %w", err)
	}
	defer raw.Close()

	doc, err := jsonschema.UnmarshalJSON(raw)
	if err != nil {
		return nil, fmt.Errorf("read schema: %w", err)
	}

	c := jsonschema.NewCompiler()
	if err := c.AddResource(ContractSchemaID, doc); err != nil {
		return nil, fmt.Errorf("add schema: %w", err)
	}

	schema, err := c.Compile(ContractSchemaID)
	if err != nil {
		return nil, fmt.Errorf("compile schema: %w", err)
	}

	return &Registry{schemas: map[string]*jsonschema.Schema{
		ContractSchemaID: schema,
	}}, nil
}

func (r *Registry) Validate(id string, doc []byte) error {
	schema := r.schemas[id]
	if schema == nil {
		return fmt.Errorf("missing schema: %s", id)
	}

	value, err := jsonschema.UnmarshalJSON(bytes.NewReader(doc))
	if err != nil {
		return fmt.Errorf("read fixture: %w", err)
	}
	if err := schema.Validate(value); err != nil {
		return fmt.Errorf("validate fixture: %w", err)
	}
	return nil
}
