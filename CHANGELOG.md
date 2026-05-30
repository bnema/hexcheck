# Changelog

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
