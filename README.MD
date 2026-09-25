# AliasCtl

A lightweight CLI for creating **temporary, project-scoped command environments** from YAML.

AliasCtl lets you define variables, aliases, and shell functions in `aliasctl.yaml`, then launch a temporary Bash shell without modifying your parent shell.

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

### How the shell is built

- Your `~/.bashrc` is loaded first, then the AliasCtl definitions, so project aliases and functions win on conflicts.
- The prompt is prefixed with `(aliasctl:<name>)` and `ALIASCTL_ENV` is set to the environment name.
- Running `aliasctl shell` inside an AliasCtl shell is refused; exit the current one first.

### Variables in aliases

`${NAME}` in an alias is replaced with the value from `variables`, falling back to the process environment (for example `${HOME}`). An undefined name is a configuration error. Substituted values are not expanded again, and each argument keeps its boundaries, so values containing spaces or `;` are passed as plain text.

## Commands

```text
aliasctl init       Create a configuration
aliasctl validate   Validate configuration
aliasctl inspect    Inspect resolved environment
aliasctl shell      Start temporary Bash environment
aliasctl run        Run a command with the environment
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
