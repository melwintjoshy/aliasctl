package mise

import (
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/melwintjoshy/aliasctl/internal/fakemise"
	"github.com/melwintjoshy/aliasctl/versions"
)

func raws(releases []Release) []string {
	out := make([]string, len(releases))

	for i, release := range releases {
		out[i] = release.Raw
	}

	return out
}

func TestRemoteSkipsPrereleasesAndVendorBuilds(t *testing.T) {
	fake := fakemise.Install(t)

	fake.SetRemote(t, "go", "1.21.13", "1.22.0", "1.22rc1", "1.22.6", "1.23.0-beta.1", "", "v1.23.2")
	fake.SetRemote(t, "java", "temurin-21.0.2+13", "21.0.2", "corretto-21")

	releases, err := CLI{}.Remote("go")
	if err != nil {
		t.Fatal(err)
	}

	if got := raws(releases); !reflect.DeepEqual(got, []string{"1.21.13", "1.22.0", "1.22.6", "v1.23.2"}) {
		t.Fatalf("unexpected releases %v", got)
	}

	releases, _ = CLI{}.Remote("java")

	if got := raws(releases); !reflect.DeepEqual(got, []string{"21.0.2"}) {
		t.Fatalf("unexpected java releases %v", got)
	}
}

func TestInstallThenInstalledAndBinPaths(t *testing.T) {
	fake := fakemise.Install(t)

	backend := CLI{}

	if !backend.Available() {
		t.Fatal("expected the fake mise to be found")
	}

	for _, spec := range []string{"go@1.22.6", "go@1.21.13", "aqua:hashicorp/terraform@1.9.8"} {
		if err := backend.Install(spec, io.Discard, io.Discard); err != nil {
			t.Fatal(err)
		}
	}

	installed, err := backend.Installed()
	if err != nil {
		t.Fatal(err)
	}

	if got := raws(installed["go"]); !reflect.DeepEqual(got, []string{"1.22.6", "1.21.13"}) {
		t.Fatalf("unexpected installed go %v", got)
	}

	if got := raws(installed["aqua:hashicorp/terraform"]); !reflect.DeepEqual(got, []string{"1.9.8"}) {
		t.Fatalf("unexpected installed terraform %v", got)
	}

	paths, err := backend.BinPaths([]string{"go@1.22.6", "aqua:hashicorp/terraform@1.9.8"})
	if err != nil {
		t.Fatal(err)
	}

	expected := []string{
		filepath.Join(fake.Root, "installs", "go", "1.22.6", "bin"),
		filepath.Join(fake.Root, "installs", "aqua_hashicorp_terraform", "1.9.8", "bin"),
	}

	if !reflect.DeepEqual(paths, expected) {
		t.Fatalf("expected %v, got %v", expected, paths)
	}

	if _, err := os.Stat(filepath.Join(expected[1], "terraform")); err != nil {
		t.Fatalf("expected the installed binary to exist: %v", err)
	}
}

func TestInstallFailureIsReported(t *testing.T) {
	fake := fakemise.Install(t)
	fake.FailInstall(t, "go")

	err := CLI{}.Install("go@1.22.6", io.Discard, io.Discard)

	if err == nil || err.Error() != "mise install go@1.22.6: exit status 1" {
		t.Fatalf("unexpected error %v", err)
	}
}

func TestAvailableWithoutMise(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	if (CLI{}).Available() {
		t.Fatal("expected mise to be unavailable")
	}
}

func TestBest(t *testing.T) {
	releases := []Release{
		{Raw: "1.21.13", Version: versions.Version{1, 21, 13}},
		{Raw: "1.22.6", Version: versions.Version{1, 22, 6}},
		{Raw: "1.22.10", Version: versions.Version{1, 22, 10}},
		{Raw: "1.23.2", Version: versions.Version{1, 23, 2}},
	}

	tests := []struct {
		rule     string
		expected string
	}{
		{"*", "1.23.2"},
		{"1.22", "1.22.10"},
		{">=1.22 <1.23", "1.22.10"},
		{"<1.22", "1.21.13"},
		{"1.22.6", "1.22.6"},
		{">=2", ""},
	}

	for _, tt := range tests {
		constraint, err := versions.ParseConstraint(tt.rule)
		if err != nil {
			t.Fatal(err)
		}

		best, ok := Best(releases, constraint)

		if tt.expected == "" {
			if ok {
				t.Fatalf("%q: expected no match, got %s", tt.rule, best.Raw)
			}

			continue
		}

		if !ok || best.Raw != tt.expected {
			t.Fatalf("%q: expected %s, got %s (ok=%v)", tt.rule, tt.expected, best.Raw, ok)
		}
	}
}
