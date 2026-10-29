package ompiglet

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sync"

	"go.yaml.in/yaml/v3"

	"github.com/tmih06/ompig/third_party/jsonschema"
)

// pigletSchemaJSON is the published closed Ompiglet JSON Schema v1. It mirrors
// the Go parser's accepted vocabulary; Go layers semantic validation (path
// safety, origin resolution, inheritance) on top of it.
//
//go:embed ompiglet.schema.json
var pigletSchemaJSON []byte

var (
	compiledSchema *jsonschema.Schema
	compileOnce    sync.Once
	compileErr     error
)

// SchemaJSON returns the published Ompiglet JSON Schema v1 bytes.
func SchemaJSON() []byte { return pigletSchemaJSON }

func pigletSchema() (*jsonschema.Schema, error) {
	compileOnce.Do(func() {
		var document any
		if err := json.Unmarshal(pigletSchemaJSON, &document); err != nil {
			compileErr = fmt.Errorf("decode Ompiglet schema: %w", err)
			return
		}
		compiler := jsonschema.NewCompiler()
		if err := compiler.AddResource("ompiglet.schema.json", document); err != nil {
			compileErr = fmt.Errorf("add Ompiglet schema: %w", err)
			return
		}
		compiledSchema, compileErr = compiler.Compile("ompiglet.schema.json")
	})
	return compiledSchema, compileErr
}

// ValidateAgainstSchema checks Ompiglet YAML against the published JSON Schema
// closed vocabulary. It is the schema half of the schema/parser agreement; Go
// parsing (ParseBytes) adds the semantic checks the vocabulary schema cannot
// express.
func ValidateAgainstSchema(yamlBytes []byte) error {
	schema, err := pigletSchema()
	if err != nil {
		return err
	}
	value, err := schemaInstance(yamlBytes)
	if err != nil {
		return err
	}
	return schema.Validate(value)
}

// schemaInstance decodes YAML and normalizes it to JSON-native types
// (map[string]any, []any, float64) so the JSON Schema validator sees the same
// value shape it was authored against.
func schemaInstance(yamlBytes []byte) (any, error) {
	var document any
	if err := yaml.Unmarshal(yamlBytes, &document); err != nil {
		return nil, fmt.Errorf("parse ompiglet YAML: %w", err)
	}
	encoded, err := json.Marshal(document)
	if err != nil {
		return nil, fmt.Errorf("normalize ompiglet YAML: %w", err)
	}
	var value any
	if err := json.Unmarshal(encoded, &value); err != nil {
		return nil, fmt.Errorf("normalize ompiglet YAML: %w", err)
	}
	return value, nil
}
