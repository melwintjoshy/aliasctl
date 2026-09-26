// Package fakemise puts a stand-in mise on PATH for tests, since real mise downloads from the internet.
package fakemise

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// answers the few mise commands aliasctl uses; install writes a real tool that prints its version
const script = `#!/bin/sh
root="$FAKE_MISE_ROOT"
echo "$*" >> "$root/calls"

safe() { printf '%s' "$1" | tr ':/' '__'; }
bin_name() { n="${1##*/}"; printf '%s' "${n##*:}"; }

case "$1" in
  ls-remote)
    cat "$root/remote/$(safe "$2")" 2>/dev/null
    ;;
  ls)
    printf '{'
    first=1
    if [ -f "$root/installed" ]; then
      for name in $(cut -d' ' -f1 "$root/installed" | sort -u); do
        [ "$first" = 1 ] || printf ','
        first=0
        printf '"%s":[' "$name"
        sep=''
        for version in $(awk -v n="$name" '$1 == n { print $2 }' "$root/installed"); do
          printf '%s{"version":"%s","install_path":"%s","source":null}' "$sep" "$version" "$root/installs/$(safe "$name")/$version"
          sep=','
        done
        printf ']'
      done
    fi
    printf '}\n'
    ;;
  bin-paths)
    shift
    for spec in "$@"; do
      printf '%s\n' "$root/installs/$(safe "${spec%@*}")/${spec##*@}/bin"
    done
    ;;
  install)
    spec="$2"
    name="${spec%@*}"
    version="${spec##*@}"
    if grep -qx "$(safe "$name")" "$root/fail" 2>/dev/null; then
      echo "fake mise: install of $spec failed" >&2
      exit 1
    fi
    dir="$root/installs/$(safe "$name")/$version/bin"
    mkdir -p "$dir"
    printf '#!/bin/sh\necho "%s version %s"\n' "$(bin_name "$name")" "$version" > "$dir/$(bin_name "$name")"
    chmod +x "$dir/$(bin_name "$name")"
    echo "$name $version" >> "$root/installed"
    echo "installed $spec"
    ;;
  *)
    echo "fake mise: unsupported command: $*" >&2
    exit 2
    ;;
esac
`

type Fake struct {
	Root string
}

// Install puts the fake first on PATH for the rest of the test.
func Install(t testing.TB) *Fake {
	t.Helper()

	root := t.TempDir()
	bin := filepath.Join(root, "bin")

	for _, dir := range []string{bin, filepath.Join(root, "remote")} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}

	if err := os.WriteFile(filepath.Join(bin, "mise"), []byte(script), 0755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("FAKE_MISE_ROOT", root)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	return &Fake{Root: root}
}

func safe(name string) string {
	return strings.NewReplacer(":", "_", "/", "_").Replace(name)
}

// SetRemote sets what "mise ls-remote name" prints, one line per entry.
func (f *Fake) SetRemote(t testing.TB, name string, lines ...string) {
	t.Helper()

	content := strings.Join(lines, "\n") + "\n"

	if err := os.WriteFile(filepath.Join(f.Root, "remote", safe(name)), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

// FailInstall makes "mise install name@..." exit with an error.
func (f *Fake) FailInstall(t testing.TB, name string) {
	t.Helper()

	file, err := os.OpenFile(filepath.Join(f.Root, "fail"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}

	defer file.Close()

	if _, err := file.WriteString(safe(name) + "\n"); err != nil {
		t.Fatal(err)
	}
}

// Calls lists every mise invocation so far, as its argument string.
func (f *Fake) Calls(t testing.TB) []string {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(f.Root, "calls"))
	if os.IsNotExist(err) {
		return nil
	}

	if err != nil {
		t.Fatal(err)
	}

	return strings.Split(strings.TrimRight(string(data), "\n"), "\n")
}
