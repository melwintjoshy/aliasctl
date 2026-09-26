// Package mise talks to the mise CLI to find and install tool versions.
package mise

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strings"

	"github.com/melwintjoshy/aliasctl/versions"
)

// Release is a version as mise spells it, plus its parsed form for comparisons.
type Release struct {
	Raw     string
	Version versions.Version
}

// Backend is the subset of mise that aliasctl uses; tests swap in a fake.
type Backend interface {
	Available() bool
	Remote(name string) ([]Release, error)
	Installed() (map[string][]Release, error)
	BinPaths(specs []string) ([]string, error)
	Install(spec string, stdout, stderr io.Writer) error
}

// CLI runs the mise binary found on PATH.
type CLI struct{}

func (CLI) Available() bool {
	_, err := exec.LookPath("mise")
	return err == nil
}

// only plain releases count; prereleases and vendor-prefixed builds are skipped
var stablePattern = regexp.MustCompile(`^v?\d+(\.\d+)*$`)

func parseRelease(raw string) (Release, bool) {
	raw = strings.TrimSpace(raw)

	if !stablePattern.MatchString(raw) {
		return Release{}, false
	}

	version, ok := versions.Parse(raw)

	return Release{Raw: raw, Version: version}, ok
}

func (CLI) Remote(name string) ([]Release, error) {
	output, err := run("ls-remote", name)
	if err != nil {
		return nil, err
	}

	var releases []Release

	for _, line := range strings.Split(string(output), "\n") {
		if release, ok := parseRelease(line); ok {
			releases = append(releases, release)
		}
	}

	return releases, nil
}

// only "version" is read, to depend on as little of mise's json as possible
func (CLI) Installed() (map[string][]Release, error) {
	output, err := run("ls", "--installed", "--json")
	if err != nil {
		return nil, err
	}

	var records map[string][]struct {
		Version string `json:"version"`
	}

	if err := json.Unmarshal(output, &records); err != nil {
		return nil, fmt.Errorf("could not read mise ls output: %w", err)
	}

	installed := make(map[string][]Release, len(records))

	for name, entries := range records {
		for _, entry := range entries {
			if release, ok := parseRelease(entry.Version); ok {
				installed[name] = append(installed[name], release)
			}
		}
	}

	return installed, nil
}

func (CLI) BinPaths(specs []string) ([]string, error) {
	if len(specs) == 0 {
		return nil, nil
	}

	output, err := run(append([]string{"bin-paths"}, specs...)...)
	if err != nil {
		return nil, err
	}

	var paths []string

	for _, line := range strings.Split(string(output), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			paths = append(paths, line)
		}
	}

	return paths, nil
}

// install doesn't touch any mise.toml, so a user's own mise config is left alone
func (CLI) Install(spec string, stdout, stderr io.Writer) error {
	cmd := exec.Command("mise", "install", spec)

	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mise install %s: %w", spec, err)
	}

	return nil
}

func run(args ...string) ([]byte, error) {
	var stderr bytes.Buffer

	cmd := exec.Command("mise", args...)
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = err.Error()
		}

		return nil, fmt.Errorf("mise %s: %s", strings.Join(args, " "), detail)
	}

	return output, nil
}

// Best returns the highest release matching the rule, or false when none does.
func Best(releases []Release, constraint versions.Constraint) (Release, bool) {
	var best Release
	found := false

	for _, release := range releases {
		if constraint.Match(release.Version) && (!found || versions.Compare(release.Version, best.Version) > 0) {
			best, found = release, true
		}
	}

	return best, found
}
