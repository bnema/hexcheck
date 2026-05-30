---
name: hexcheck
description: Use when configuring or running hexcheck in a Go repository to enforce hexagonal architecture boundaries, adapter business-logic warnings, and mock discipline.
---

# Hexcheck Configuration Skill

Use `hexcheck` to lint Go repositories for hexagonal / clean architecture violations.

## Core idea

`hexcheck` does not require folders to have fixed names. It maps project-specific paths to architecture roles through `.hexcheck.yaml`.

Supported roles:

- `core` — domain entities, value objects, domain services, core business logic
- `usecase` — application services/use cases/orchestration
- `ports` — interfaces/contracts that core/usecases depend on
- `adapter` — infrastructure, persistence, external services, outbound adapters
- `entrypoint` — CLI, HTTP handlers, UI entrypoints, app bootstrap
- `ignore` — generated code, mocks, vendored or ignored paths

Component names are arbitrary. The `role` is what matters.

## Example mappings

A repo with `domain`, `application`, and `infrastructure`:

```yaml
components:
  domain:
    role: core
    paths:
      - internal/domain/**
  usecases:
    role: usecase
    paths:
      - internal/application/usecase/**
      - internal/application/usecases/**
  ports:
    role: ports
    paths:
      - internal/application/port/**
  infrastructure:
    role: adapter
    paths:
      - internal/infrastructure/**
  cli:
    role: entrypoint
    paths:
      - cmd/**
      - internal/cli/**
```

A repo that uses `core` and `boundaries`:

```yaml
components:
  core:
    role: core
    paths:
      - internal/core/**
  usecases:
    role: usecase
    paths:
      - internal/usecases/**
  inboundBoundaries:
    role: entrypoint
    paths:
      - internal/boundaries/in/**
  outboundBoundaries:
    role: adapter
    paths:
      - internal/boundaries/out/**
  contracts:
    role: ports
    paths:
      - internal/boundaries/ports/**
```

## Recommended defaults

Start strict on deterministic rules and warning-first on heuristics:

```yaml
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
```

## Business-logic heuristic

`suspicious-business-logic-in-adapter` is cumulative and package-local. It should not scan the whole repository. It combines strong and weak evidence:

- business keywords such as `Validate`, `Migrate`, `Resolve`, `Score`, `Restore`, `Purge`, `Update`
- policy constants and branching
- domain/core mutation or method calls
- multiple port/usecase collaborators when combined with business wording

Use `excludePaths` for adapter areas where warnings are not useful:

```yaml
ruleSettings:
  suspicious-business-logic-in-adapter:
    excludePaths:
      - internal/ui/generated/**
```

## Generated code

Always ignore generated code and mocks:

```yaml
components:
  generated:
    role: ignore
    paths:
      - '**/mocks/**'
      - '**/generated/**'
      - '**/*_templ.go'
      - '**/*_gen.go'
```

`hexcheck` also ignores files with a standard `// Code generated ... DO NOT EDIT.` comment.

## Mock discipline

Configure generated mock locations so `hexcheck` can distinguish missing mocks from local fake misuse:

```yaml
mocking:
  generatedMockPaths:
    - internal/mocks/**
    - internal/application/mocks/**
    - internal/application/port/mocks/**
    - internal/domain/repository/mocks/**
  generatedMockNamePatterns:
    - Mock{{Interface}}
    - '{{Interface}}Mock'
```

Expected behavior:

- `missing-generated-mock-for-port` flags mockable interfaces without generated mocks.
- `no-local-fakes-for-ports` flags local fakes only when a generated mock exists.
- Intentional exceptions should use project config `allow` entries or linter suppression comments.

## Running

Standalone:

```bash
hexcheck -hexcheck.config .hexcheck.yaml -hexcheck.root . ./...
```

With a custom golangci-lint binary, use the module plugin config from `examples/custom-gcl.yml` and `examples/golangci.yml`.

## Agent checklist

When configuring a new repo:

1. Read architecture docs (`AGENTS.md`, `README`, docs) before guessing paths.
2. Map local folder names to roles; do not force folder renames.
3. Add generated/mocks ignore paths.
4. Add mock paths and name patterns.
5. Run `hexcheck` once and inspect counts by rule.
6. Tune `ruleSettings.*.excludePaths` for known technical adapters only after reviewing examples.
7. Keep deterministic rules as `error`; keep heuristic rules as `warn` until signal is proven.
