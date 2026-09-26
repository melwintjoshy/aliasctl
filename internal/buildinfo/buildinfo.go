// Package buildinfo reports which build of aliasctl is running.
package buildinfo

import (
	"fmt"
	"runtime/debug"
)

// set by -ldflags -X on release builds; a source build keeps the defaults
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// String is what --version prints, for example "v0.2.0 (58fb257, 2026-09-26T14:42:16Z)".
func String() string {
	info, ok := debug.ReadBuildInfo()

	return format(Version, Commit, Date, info, ok)
}

// ldflags win; otherwise the module version and vcs stamps that go install and go build record
func format(version, commit, date string, info *debug.BuildInfo, ok bool) string {
	if !ok {
		return fmt.Sprintf("%s (%s, %s)", version, commit, date)
	}

	if version == "dev" && info.Main.Version != "" && info.Main.Version != "(devel)" {
		version = info.Main.Version
	}

	stamped := commit == "none"
	modified := false

	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			if stamped {
				commit = shortRevision(setting.Value)
			}
		case "vcs.time":
			if date == "unknown" {
				date = setting.Value
			}
		case "vcs.modified":
			modified = setting.Value == "true"
		}
	}

	// only a vcs-stamped commit can be dirty; a release build says exactly what it was built from
	if stamped && modified && commit != "none" {
		commit += "-dirty"
	}

	return fmt.Sprintf("%s (%s, %s)", version, commit, date)
}

func shortRevision(revision string) string {
	if len(revision) > 7 {
		return revision[:7]
	}

	return revision
}
