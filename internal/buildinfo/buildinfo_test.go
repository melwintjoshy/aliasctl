package buildinfo

import (
	"runtime/debug"
	"testing"
)

func TestFormat(t *testing.T) {
	vcs := []debug.BuildSetting{
		{Key: "vcs.revision", Value: "58fb2571234567890abcdef"},
		{Key: "vcs.time", Value: "2026-09-26T14:42:16Z"},
		{Key: "vcs.modified", Value: "true"},
	}

	cases := []struct {
		name                  string
		version, commit, date string
		info                  *debug.BuildInfo
		ok                    bool
		expected              string
	}{
		{
			name: "no build info", version: "dev", commit: "none", date: "unknown",
			expected: "dev (none, unknown)",
		},
		{
			name: "ldflags win over stamps", version: "v0.2.0", commit: "abc1234", date: "2026-09-27T00:00:00Z",
			info: &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}, Settings: vcs}, ok: true,
			expected: "v0.2.0 (abc1234, 2026-09-27T00:00:00Z)",
		},
		{
			name: "go install has the module version and no stamps", version: "dev", commit: "none", date: "unknown",
			info: &debug.BuildInfo{Main: debug.Module{Version: "v0.1.1"}}, ok: true,
			expected: "v0.1.1 (none, unknown)",
		},
		{
			name: "source build uses vcs stamps and marks a dirty tree", version: "dev", commit: "none", date: "unknown",
			info: &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}, Settings: vcs}, ok: true,
			expected: "dev (58fb257-dirty, 2026-09-26T14:42:16Z)",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := format(c.version, c.commit, c.date, c.info, c.ok); got != c.expected {
				t.Fatalf("expected %q, got %q", c.expected, got)
			}
		})
	}
}
