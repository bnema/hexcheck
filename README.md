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
  no-infra-imports-in-usecase: error
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

## Standalone CLI

Because `hexcheck` is a `go/analysis` analyzer, standalone flags use the analyzer prefix:

```bash
hexcheck -hexcheck.config .hexcheck.yaml -hexcheck.root . ./...
```

During development:

```bash
go run ./cmd/hexcheck -hexcheck.config examples/hexcheck.yaml -hexcheck.root . ./...
```

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

## Agent configuration guide

[`SKILL.md`](SKILL.md) explains how an agent should configure `hexcheck` for a new repository, including non-standard layouts such as `core`/`boundaries`.

## Local development

```bash
make test
make check
HEXCHECK_SMOKE_REPO=/path/to/local/go/repo make smoke-local
```
