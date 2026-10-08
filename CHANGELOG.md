# Changelog

## v0.3.0

### Breaking

- The CLI uses plain flags: `-config`, `-root`, `-module` replace `-hexcheck.config`, `-hexcheck.root`, `-hexcheck.module`.
- The config is strict: unknown keys and unknown rule names in `rules`, `ruleSettings`, and `allow` are errors.
- Diagnostic messages are prefixed with their severity, e.g. `[error] no-adapter-imports-in-core: ...`.

### Added

- Rules `no-usecase-imports-in-core`, `no-infra-imports-in-ports`, and `no-entrypoint-imports-in-adapter`. `no-infra-imports-in-usecase` also covers entrypoint imports.
- Rule severity drives the CLI exit code: only `error` rules fail by default; `-fail-on warn` also fails on warnings.
- `//hexcheck:ignore <rule>[,<rule>] <reason>` inline suppression.
- `hexcheck init` writes a starter `.hexcheck.yaml` from detected folders.
- `-json` and `-version` CLI flags.
- Type-leak rules inspect channels, function signatures, inline structs, aliases, and generic type arguments.
- Release builds with GoReleaser.

### Fixed

- The config is loaded once per run instead of once per package.
- The generated-mock index is rebuilt on each run instead of being cached for the process lifetime.

## v0.1.0

Initial release of `hexcheck`.

- Adds a `go/analysis` analyzer for configurable hexagonal architecture checks.
- Adds a standalone `hexcheck` CLI.
- Adds golangci-lint Module Plugin integration.
- Adds `.hexcheck.yaml` component role mapping for `core`, `usecase`, `ports`, `adapter`, `entrypoint`, and `ignore`.
- Adds deterministic boundary rules for adapter imports, usecase imports, framework type leaks, port type leaks, and adapter coupling.
- Adds warning-level adapter business-logic heuristics with package-local AST/type analysis and performance guardrails.
- Adds mock discipline rules for missing generated mocks, local fakes, and concrete adapters in usecase/core tests.
- Adds example configuration and an agent-facing `SKILL.md` for configuring new repositories.
