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

## Commands

```text
aliasctl init       Create a configuration
aliasctl validate   Validate configuration
aliasctl inspect    Inspect resolved environment
aliasctl shell      Start temporary Bash environment
aliasctl run        Run a command with the environment
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

## Development

```bash
go test ./...
go test -race ./...
go vet ./...
go build -o aliasctl
```

**Status:** Active development.
