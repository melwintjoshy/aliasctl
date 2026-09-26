package app

import (
	"slices"

	"github.com/melwintjoshy/aliasctl/mise"
	"github.com/melwintjoshy/aliasctl/resolver"
	"github.com/melwintjoshy/aliasctl/versions"
)

// ActivateTools puts the mise-installed version of each declared tool in front of PATH.
// tools without a matching mise install are left to whatever PATH already has.
func ActivateTools(env *resolver.Environment, backend mise.Backend) error {
	if len(env.Tools) == 0 || !backend.Available() {
		return nil
	}

	installed, err := backend.Installed()
	if err != nil {
		return err
	}

	var specs []string

	for _, tool := range env.Tools {
		constraint, err := versions.ParseConstraint(tool.Rule)
		if err != nil {
			continue
		}

		if best, ok := mise.Best(installed[tool.Mise], constraint); ok {
			specs = append(specs, tool.Mise+"@"+best.Raw)
		}
	}

	// one call for every tool, so activation costs two mise runs however many tools there are
	paths, err := backend.BinPaths(specs)
	if err != nil {
		return err
	}

	env.PathPrepend = uniqueInOrder(paths)

	return nil
}

// two tools can share a bin dir, and slices.Compact would only drop repeats that sit next to each other
func uniqueInOrder(paths []string) []string {
	var unique []string

	for _, path := range paths {
		if !slices.Contains(unique, path) {
			unique = append(unique, path)
		}
	}

	return unique
}
