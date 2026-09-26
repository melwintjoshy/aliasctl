# AliasCtl

A lightweight CLI for creating temporary, project-scoped shell environments from YAML.

AliasCtl lets you define variables, aliases, and shell functions in `aliasctl.yaml`, then launch a temporary Bash, Zsh or Fish shell without modifying your parent shell.

## Installation

### From source

```bash
git clone <repository-url>
cd aliasctl
go build -o aliasctl
```

Or install directly with Go:

```bash
go install github.com/melwintjoshy/aliasctl@latest
```

## Example

```yaml
name: k8s

variables:
  NAMESPACE: app-prod

aliases:
  k: kubectl
  kgp: "kubectl get pods -n ${NAMESPACE}"

functions:
  deploy: |
    echo "Building..."
    go build ./...
```

Start the environment:

```bash
aliasctl shell
```

Inside the temporary shell:

```bash
kgp
deploy
```

Exit the shell and the parent environment remains unchanged.

### Choosing a shell

`shell` and `run` take `--shell bash|zsh|fish`, defaulting to the basename of `$SHELL` and falling back to Bash.

- **zsh** uses the same alias and function syntax as Bash. Your `.zshenv` and `.zshrc` are loaded from `$ZDOTDIR` (or `$HOME`) first, including when `.zshenv` moves `ZDOTDIR`.
- **fish** gets aliases as fish functions, and `functions` bodies are never fed to it, because they are Bash code. Put fish bodies under `functions_fish`:

```yaml
functions:
  deploy: |
    go build ./... && echo "built $1"
functions_fish:
  deploy: |
    go build ./...; and echo "built $argv[1]"
```

### How the shell is built

- Your `~/.bashrc` is loaded first (or `~/.bash_profile` when there is no `~/.bashrc`), then the AliasCtl definitions, so project aliases and functions win on conflicts.
- The prompt is prefixed with `(aliasctl:<name>)` and `ALIASCTL_ENV` is set to the environment name. The prefix is re-applied from `PROMPT_COMMAND`, so prompts that rebuild `PS1` (starship, git-prompt) keep it.
- Running `aliasctl shell` inside an AliasCtl shell is refused; exit the current one first.

### Variables in aliases

`${NAME}` in an alias is replaced with the value from `variables`, falling back to the process environment (for example `${HOME}`). An undefined name is a configuration error. Because of that fallback, a config that relies on environment variables can validate on one machine and fail on another. Substituted values are not expanded again, and each argument keeps its boundaries, so values containing spaces or `;` are passed as plain text.

Aliases are structured commands, not shell code: globs, `~`, `$VAR`, pipes, redirects and `&&` are not expanded, so unquoted shell syntax in an alias is rejected. Quote it to pass it as plain text, or use a function when you need the shell.

Quoting follows POSIX rules: a backslash is literal inside single quotes, and inside double quotes it only escapes `$`, `` ` ``, `"`, `\` and newline. So `printf "[%s]\n"` keeps its `\n`. Earlier versions dropped every backslash, which turned `"\n"` into `n`.

## Commands

```text
aliasctl init       Create a configuration
aliasctl validate   Validate configuration
aliasctl inspect    Inspect resolved environment
aliasctl shell      Start temporary shell environment
aliasctl run        Run a command with the environment
aliasctl export     Print the environment as shell code for eval
aliasctl hook       Print a snippet that auto-activates on cd
aliasctl allow      Trust a configuration for auto-activation (--list, --prune)
aliasctl deny       Stop trusting a configuration
aliasctl tools check    Check installed tools against the configured versions
aliasctl tools install  Install missing or mismatched tools with mise
aliasctl schema     Print the JSON schema for aliasctl.yaml
```

`aliasctl run` also understands the configuration's names: an alias runs its command directly with any extra arguments appended, and a function runs in Bash with its arguments as `$1`, `$2`, …

```bash
aliasctl run kgp -o wide
aliasctl run deploy
```

Use a specific configuration:

```bash
aliasctl --config dev.yaml shell
```

### Seeing what runs

`shell --print` and `run --print` show the generated command, the environment changes and every generated file, and exit without running anything. The temporary directory appears as `<tmp>`. Include this output when reporting a shell-specific problem.

```bash
aliasctl shell --print --shell zsh
aliasctl run --print kgp -o wide
```

### Config format and editor support

`version: 1` at the top of `aliasctl.yaml` is optional; a file written for a newer format fails with a message to upgrade AliasCtl instead of a confusing field error. For autocomplete and inline validation in editors that use the YAML language server (such as VS Code with the YAML extension), add this as the first line:

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/melwintjoshy/aliasctl/main/schema/aliasctl.schema.json
```

`aliasctl schema` prints the same schema. `aliasctl validate` remains the source of truth.

## Tools

Declare the tools the project's commands need, and AliasCtl checks the machine against them:

```yaml
tools:
  go: "1.22"                # any 1.22.x
  kubectl: ">=1.29 <1.31"   # every rule must hold
  jq: "*"                   # only needs to be installed
  mytool:
    version: "2.1"
    check: "./bin/mytool version --short"
  swift:
    version: "*"
    timeout: 20s            # slow to start, so give it longer than the default 5s
```

```text
$ aliasctl tools check
TOOL       WANT          FOUND   STATUS    PATH                    NOTE
go         1.22          1.22.4  ok        /opt/homebrew/bin/go
kubectl    >=1.29 <1.31  1.28.1  mismatch  /usr/local/bin/kubectl
terraform  >=1.6         -       missing   -
java       *             -       broken    /usr/bin/java           Unable to locate a Java Runtime.
```

- **Rules:** `*` (any version), a bare version that pins only the parts it names (`1.22` matches `1.22.x`, `1.22.4` is exact), or `>=`, `>`, `<=`, `<`, `=` comparisons separated by spaces. A comparison sees missing parts as zero, so `=1.22` means exactly `1.22.0`; use the bare form for a prefix match. A leading `v` is ignored, so `>=v1.29` works. Quote every rule: `*` and `>` are YAML syntax on their own.
- **How the version is read:** `go`, `kubectl`, `helm` and `java` have built-in checks. Every other tool runs `<name> --version`, and the first version number in the output is used. Use `check:` when that is the wrong number (`swift --version` prints the driver version first) or the flag differs; it follows the same rules as an alias (no shell syntax, `${VAR}` allowed), and a relative path is taken from the directory holding `aliasctl.yaml`. Java 8 reports itself as `1.8.0`, so its rule is `1.8`, not `8`.
- **Statuses:** `ok`, `missing` (not on `PATH`), `mismatch`, `broken` (installed but exits with an error or prints no version), or `timeout` (no answer within 5 seconds, or the tool's own `timeout:`).
- **Where checks run:** `aliasctl tools check` exits non-zero on any problem, so it can gate CI. `aliasctl shell` prints a warning and opens anyway, after waiting for the checks, so a hanging tool delays the prompt by its timeout. `run`, `validate` and auto-activation never run checks, so scripts and prompts stay fast.
- **Installing:** `aliasctl tools install` uses [mise](https://mise.jdx.dev) to install every tool that is missing or the wrong version. It picks the highest mise release that matches the rule, shows the plan, and asks before installing (`--yes` to skip the question, `--dry-run` to only show the plan). Without a terminal and without `--yes` it refuses, since the repository's config decides what gets downloaded. Set `mise:` when mise knows the tool by another name, for example `mise: "aqua:hashicorp/terraform"`. mise's own config files are never changed.
- **Using installed versions:** when mise is installed, `shell`, `run`, auto-activation and `tools check` put the matching mise-installed version of each tool first on `PATH`, choosing the highest installed version that fits the rule. Tools without a matching mise install use whatever is already on `PATH`. If your shell rc runs `mise activate`, mise rewrites `PATH` on every prompt and may override these entries; `aliasctl tools check` shows which binary is actually used.

## Auto-activation

Instead of starting a subshell, you can have your own shell load `aliasctl.yaml` when you `cd` into a project and unload it when you leave:

```bash
eval "$(aliasctl hook bash)"   # in ~/.bashrc
eval "$(aliasctl hook zsh)"    # in ~/.zshrc
```

- A configuration is only loaded after you trust it with `aliasctl allow`, which records its path and a hash of its content. An unknown or edited file is refused with a one-line hint, so cloning a repository never runs its configuration on the next prompt.
- `aliasctl allow --list` shows every trusted configuration as `ok`, `changed` (edited since it was allowed) or `missing` (file deleted). `aliasctl allow --prune` forgets the missing ones, and `aliasctl deny` stops trusting one; it unloads on the next directory change.
- On leaving, project aliases, functions and variables are removed, and anything they shadowed (your own aliases, functions and previous variable values) is restored. Tool directories are taken off `PATH` again, while anything you added to `PATH` meanwhile (a virtualenv, `nvm use`) stays. The `(aliasctl:<name>)` prompt prefix is removed too.
- **Editing a loaded configuration:** the hook notices on the next prompt. The previous trusted version stays loaded and you get a one-line hint; after `aliasctl allow`, the next prompt loads the new version. Until then nothing runs per prompt: the hook only tries again after the file or the allow list changes.
- `aliasctl shell` still works inside an auto-activated directory.
- Only Bash and Zsh are supported for auto-activation.

### What trust covers

The hash covers the bytes of `aliasctl.yaml` and nothing else, and the file is loaded from exactly the bytes that were checked. Trust does not cover:

- scripts or files that a function sources or runs
- `${VAR}` values taken from your process environment instead of the config's `variables`
- the tool binaries that aliases call, or what `mise:` backends download
- anything the environment already had before the configuration loaded

Review those the same way you would review any script you run from a repository.

## Architecture

```text
YAML → Loader → Validator → Resolver → Environment → Bash → Temporary Shell
```

Aliases are internally resolved into structured commands, while functions remain shell code.

Configuration is validated strictly: unknown keys, invalid names, and a name defined as both an alias and a function are rejected. At least one alias or function is required.

## Development

```bash
go test ./...
go test -race ./...
go vet ./...
go build -o aliasctl
```

**Status:** Active development.
