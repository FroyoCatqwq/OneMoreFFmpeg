package contracts_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/stretchr/testify/require"
)

// TestGoldenExamplesValidateAgainstJSONSchema compiles the real JSON Schema
// (docs/reference/schemas/contract.schema.json) with santhosh-tekuri/jsonschema
// and validates every golden example against it. The hand-written
// validateDataFields checks field semantics; this adds structural/type
// validation from a single machine-readable contract document.
func TestGoldenExamplesValidateAgainstJSONSchema(t *testing.T) {
	root := repoRoot(t)
	schemaPath := filepath.Join(root, "docs", "reference", "schemas", "contract.schema.json")
	schemaDoc, err := jsonschema.UnmarshalJSON(mustOpen(t, schemaPath))
	require.NoError(t, err, "schema JSON")

	c := jsonschema.NewCompiler()
	require.NoError(t, c.AddResource("contract.json", schemaDoc))
	schema, err := c.Compile("contract.json")
	require.NoError(t, err, "compile contract schema")

	examples := filepath.Join(root, "docs", "reference", "schemas", "examples")
	entries, err := os.ReadDir(examples)
	require.NoError(t, err)
	checked := 0
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		value, err := jsonschema.UnmarshalJSON(mustOpen(t, filepath.Join(examples, entry.Name())))
		require.NoError(t, err, entry.Name())
		require.NoError(t, schema.Validate(value), "golden example %s does not satisfy contract.schema.json", entry.Name())
		checked++
	}
	require.GreaterOrEqual(t, checked, 20, "expected all golden examples to be checked")
}

func mustOpen(t *testing.T, path string) *os.File {
	t.Helper()
	f, err := os.Open(path)
	require.NoError(t, err, path)
	t.Cleanup(func() { _ = f.Close() })
	return f
}
