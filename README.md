# hexcheck

`hexcheck` is a Go architecture linter for hexagonal and clean architecture boundaries.

It is built on `golang.org/x/tools/go/analysis`, runs as a standalone CLI, and can be embedded in a custom `golangci-lint` binary through the Module Plugin System.

## What it checks

- dependency direction between configurable architecture roles
- framework or infrastructure types leaking into core/port APIs
- suspicious business logic in adapters and entrypoints
- missing generated mocks for port interfaces
- local fakes or concrete adapters in tests where generated mocks should be used

## Architecture roles

`hexcheck` does not require folders to be named `domain`, `usecase`, or `adapters`. A repository maps its own paths to roles in `.hexcheck.yaml`.

Supported roles:

- `core` — domain/core business logic
- `usecase` — application use cases and orchestration
- `ports` — interfaces/contracts
- `adapter` — infrastructure, persistence, external services, outbound adapters
- `entrypoint` — CLI, HTTP handlers, UI entrypoints, app bootstrap
- `ignore` — generated code, mocks, vendored paths

## Quick config

```yaml
version: 1
components:
  core:
    role: core
    paths:
      - internal/domain/**
      - internal/core/**
  usecases:
    role: usecase
    paths:
      - internal/application/usecase/**
      - internal/application/usecases/**
      - internal/usecase/**
      - internal/usecases/**
  ports:
    role: ports
    paths:
      - internal/application/port/**
      - internal/domain/repository/**
  adapters:
    role: adapter
    paths:
      - internal/infrastructure/**
      - internal/adapters/**
  entrypoints:
    role: entrypoint
    paths:
      - cmd/**
  generated:
    role: ignore
    paths:
      - '**/mocks/**'
      - '**/generated/**'
      - '**/*_templ.go'
      - '**/*_gen.go'

rules:
  no-adapter-imports-in-core: error
  no-usecase-imports-in-core: error
  no-infra-imports-in-usecase: error
  no-infra-imports-in-ports: error
  no-entrypoint-imports-in-adapter: error
  no-framework-types-in-core: error
  no-infra-types-in-ports: error
  no-adapter-to-adapter-imports: warn
  suspicious-business-logic-in-adapter: warn
  no-local-fakes-for-ports: warn
  missing-generated-mock-for-port: warn
  prefer-generated-mocks: warn

heuristics:
  businessLogicThreshold: 8
  businessLogicMinStrongSignals: 2
  businessLogicMinWeakSignals: 2
  businessLogicMaxFunctionNodes: 2000
  businessLogicMaxDiagnosticsPerPackage: 10
  businessLogicMode: audit # audit|ci
  businessLogicMinConfidence: medium # low|medium|high
  excludeTestFiles: true

mocking:
  generatedMockPaths:
    - internal/mocks/**
    - internal/application/mocks/**
    - internal/application/port/mocks/**
  generatedMockNamePatterns:
    - Mock{{Interface}}
    - '{{Interface}}Mock'
```

A fuller example lives in [`examples/hexcheck.yaml`](examples/hexcheck.yaml).

Business-logic diagnostics include a confidence level. `audit` mode reports findings at or above `businessLogicMinConfidence`; `ci` mode reports only high-confidence findings.

## Rules

| Rule | Default | Reports |
| --- | --- | --- |
| `no-adapter-imports-in-core` | error | core imports an adapter or entrypoint |
| `no-usecase-imports-in-core` | error | core imports a usecase |
| `no-infra-imports-in-usecase` | error | usecase imports an adapter or entrypoint |
| `no-infra-imports-in-ports` | error | ports import an adapter or entrypoint |
| `no-entrypoint-imports-in-adapter` | error | adapter imports an entrypoint |
| `no-framework-types-in-core` | error | exported core API exposes an adapter or framework type |
| `no-infra-types-in-ports` | error | port interface method exposes an adapter or framework type |
| `no-adapter-to-adapter-imports` | warn | adapter imports another adapter component |
| `suspicious-business-logic-in-adapter` | warn | adapter or entrypoint function looks like business logic |
| `no-local-fakes-for-ports` | warn | test defines a fake for a port that has a generated mock |
| `missing-generated-mock-for-port` | warn | port interface has no generated mock |
| `prefer-generated-mocks` | warn | usecase/core test imports a concrete adapter |

Type-leak rules look through pointers, slices, maps, channels, function signatures, inline structs, aliases, and generic type arguments.

The config is strict: unknown keys and unknown rule names are errors.

## Suppressing a finding

Prefer config `allow` entries for whole files or directories. For one line, use an inline directive with a reason:

```go
type Rows interface {
	//hexcheck:ignore no-infra-types-in-ports row type is part of the published contract
	Next() sql.Row
}
```

The directive covers its own line and the next one. It accepts a comma-separated rule list. A directive without a reason, or with an unknown rule, is reported as `invalid-ignore-directive`.

## Standalone CLI

```bash
hexcheck init              # write a starter .hexcheck.yaml from detected folders
hexcheck ./...             # check; config is discovered from the root upwards
hexcheck -config .hexcheck.yaml -root . ./...
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `-config` | discovered | path to `.hexcheck.yaml` |
| `-root` | current dir | project root for config-relative paths |
| `-module` | from `go.mod` | Go module path |
| `-fail-on` | `error` | lowest severity that fails: `error` or `warn` |
| `-json` | false | JSON output on stdout |
| `-test` | true | also analyze test files |
| `-version` | | print the version |

Each finding is prefixed with its severity, e.g. `[warn] no-adapter-to-adapter-imports: ...`.

Exit codes: `0` no failing findings, `1` load/config/analysis error, `2` usage error, `3` findings at or above `-fail-on`.

## golangci-lint module plugin

Build a custom golangci-lint binary using the module plugin system:

```bash
golangci-lint custom -c examples/custom-gcl.yml
```

Example builder config:

```yaml
version: v2.12.2
name: hex-golangci-lint
destination: ./bin
plugins:
  - module: github.com/bnema/hexcheck
    import: github.com/bnema/hexcheck/golangci
    version: v0.1.0
```

Enable it in a project:

```yaml
version: "2"
linters:
  enable:
    - hexcheck
  settings:
    custom:
      hexcheck:
        type: module
        description: Checks hexagonal architecture boundaries.
        settings:
          config: .hexcheck.yaml
```

golangci-lint reports every hexcheck finding as an issue, whatever its configured severity. To fail only on `error` rules, set the warn rules to `off` in `.hexcheck.yaml`, or exclude issues whose text matches `^\[warn\]` with `linters.exclusions.rules`.

## Agent configuration guide

[`SKILL.md`](SKILL.md) explains how an agent should configure `hexcheck` for a new repository, including non-standard layouts such as `core`/`boundaries`.

## Local development

```bash
make test        # go test -race
make check       # tidy, lint, test, and hexcheck on its own code
HEXCHECK_SMOKE_REPO=/path/to/local/go/repo make smoke-local
```
