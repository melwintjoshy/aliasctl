package schema

import (
	"encoding/json"
	"maps"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/melwintjoshy/aliasctl/config"
)

type node struct {
	Properties           map[string]json.RawMessage `json:"properties"`
	AdditionalProperties json.RawMessage            `json:"additionalProperties"`
	OneOf                []json.RawMessage          `json:"oneOf"`
}

func yamlKeys(t reflect.Type) []string {
	var keys []string

	for i := range t.NumField() {
		name, _, _ := strings.Cut(t.Field(i).Tag.Get("yaml"), ",")

		if name != "" && name != "-" {
			keys = append(keys, name)
		}
	}

	slices.Sort(keys)

	return keys
}

func decode(t *testing.T, data []byte) node {
	t.Helper()

	var n node

	if err := json.Unmarshal(data, &n); err != nil {
		t.Fatalf("schema is not valid JSON: %v", err)
	}

	return n
}

// fails when a config key is added without the schema, or the schema lists a key the loader would reject
func TestSchemaMatchesConfig(t *testing.T) {
	root := decode(t, JSON)

	if got, want := slices.Sorted(maps.Keys(root.Properties)), yamlKeys(reflect.TypeFor[config.Config]()); !slices.Equal(got, want) {
		t.Fatalf("top-level schema keys %v, config keys %v", got, want)
	}

	tools := decode(t, root.Properties["tools"])
	toolValue := decode(t, tools.AdditionalProperties)

	if len(toolValue.OneOf) != 2 {
		t.Fatalf("expected tools to allow a short and a long form, got %d forms", len(toolValue.OneOf))
	}

	long := decode(t, toolValue.OneOf[1])

	if got, want := slices.Sorted(maps.Keys(long.Properties)), yamlKeys(reflect.TypeFor[config.ToolSpec]()); !slices.Equal(got, want) {
		t.Fatalf("tool schema keys %v, ToolSpec keys %v", got, want)
	}
}
