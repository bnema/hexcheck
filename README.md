# hexcheck

`hexcheck` is a Go architecture linter for hexagonal and clean architecture boundaries.

It is built on `golang.org/x/tools/go/analysis`, runs as a standalone CLI, and can be embedded in a custom `golangci-lint` binary through the Module Plugin System.

## Goals

- map project packages to configurable architecture components
- catch deterministic boundary violations in CI
- warn about suspicious business logic in adapters and entrypoints
- warn when tests bypass generated interface mocks with local fakes or real adapters
- smoke test the analyzer against real Go repositories such as Dumber

## Quick example

```yaml
version: 1
components:
  core:
    role: core
    paths:
      - internal/domain/**
  usecase:
    role: usecase
    paths:
      - internal/application/usecase/**
  ports:
    role: ports
    paths:
      - internal/application/port/**
  adapters:
    role: adapter
    paths:
      - internal/infrastructure/**
      - internal/ui/**
      - internal/cli/**
  entrypoints:
    role: entrypoint
    paths:
      - cmd/**

rules:
  no-adapter-imports-in-core: error
  no-infra-imports-in-usecase: error
  no-framework-types-in-core: error
  no-infra-types-in-ports: error
  no-adapter-to-adapter-imports: warn
  suspicious-business-logic-in-adapter: warn
  no-local-fakes-for-ports: warn
  prefer-generated-mocks: warn
```

## Standalone CLI

```bash
hexcheck --config .hexcheck.yaml ./...
```

## golangci-lint module plugin

Build a custom golangci-lint binary:

```yaml
# .custom-gcl.yml
version: v2.12.2
name: hex-golangci-lint
destination: ./bin
plugins:
  - module: github.com/bnema/hexcheck
    import: github.com/bnema/hexcheck/golangci
    version: v0.1.0
```

Enable it:

```yaml
# .golangci.yml
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

## Local development

```bash
make test
make check
HEXCHECK_DUMBER_PATH=/home/brice/dev/projects/dumber make smoke-dumber
```
