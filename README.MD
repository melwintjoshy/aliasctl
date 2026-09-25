# AliasCtl

A lightweight CLI for creating **temporary, project-scoped command environments** from YAML.

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
aliasctl allow      Trust a configuration for auto-activation
aliasctl tools check  Check installed tools against the configured versions
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

## Tools

Declare the tools the project's commands need, and AliasCtl checks the machine against them:

```yaml
tools:
  go: "1.22"                # any 1.22.x
  kubectl: ">=1.29 <1.31"   # every rule must hold
  jq: "*"                   # only needs to be installed
  mytool:
    version: "2.1"
    check: "mytool version --short"
```

```text
$ aliasctl tools check
TOOL       WANT          FOUND   STATUS    PATH
go         1.22          1.22.4  ok        /opt/homebrew/bin/go
kubectl    >=1.29 <1.31  1.28.1  mismatch  /usr/local/bin/kubectl
terraform  >=1.6         -       missing   -
```

- **Rules:** `*` (any version), a bare version that pins only the parts it names (`1.22` matches `1.22.x`, `1.22.4` is exact), or `>=`, `>`, `<=`, `<`, `=` comparisons separated by spaces.
- **How the version is read:** `go`, `kubectl`, `helm` and `java` have built-in checks. Every other tool runs `<name> --version`, and the first version number in the output is used. Use `check:` to override it; it follows the same rules as an alias (no shell syntax, `${VAR}` allowed).
- **Statuses:** `ok`, `missing` (not on `PATH`), `mismatch`, or `broken` (installed but exits with an error, prints no version, or takes longer than 5 seconds).
- **Where checks run:** `aliasctl tools check` exits non-zero on any problem, so it can gate CI. `aliasctl shell` prints a warning and opens anyway. `run`, `validate` and auto-activation never run checks, so scripts and prompts stay fast.
- AliasCtl only checks tools; it does not install them yet.

## Auto-activation

Instead of starting a subshell, you can have your own shell load `aliasctl.yaml` when you `cd` into a project and unload it when you leave:

```bash
eval "$(aliasctl hook bash)"   # in ~/.bashrc
eval "$(aliasctl hook zsh)"    # in ~/.zshrc
```

- A configuration is only loaded after you trust it with `aliasctl allow`, which records its path and a hash of its content. An unknown or edited file is refused with a one-line hint, so cloning a repository never runs its configuration on the next prompt.
- On leaving, project aliases, functions and variables are removed, and anything they shadowed (your own aliases, functions and previous variable values) is restored. The `(aliasctl:<name>)` prompt prefix is removed too.
- The configuration is picked up again on the next directory change, not when the file is edited in place.
- `aliasctl shell` still works inside an auto-activated directory.
- Only Bash and Zsh are supported for auto-activation.

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
