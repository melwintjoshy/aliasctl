// Package schema holds the JSON schema for aliasctl.yaml, embedded so it ships with the binary.
package schema

import _ "embed"

//go:embed aliasctl.schema.json
var JSON []byte
