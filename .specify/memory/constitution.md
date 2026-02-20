<!--
SYNC IMPACT REPORT
==================
Version change: (none) → 1.0.0
Bump rationale: Initial constitution — MINOR (new document establishing all principles and governance).

Modified principles: N/A (new file)

Added sections:
  - Core Principles (I–V)
  - Technical Standards
  - Development Workflow
  - Governance

Removed sections: N/A

Template alignment:
  - .specify/templates/plan-template.md      ✅ Constitution Check section is generic; aligns with principles as-is
  - .specify/templates/spec-template.md      ✅ Mandatory sections (User Scenarios, Requirements, Success Criteria) align with Principle II & IV
  - .specify/templates/tasks-template.md     ✅ Test-first task ordering (tests before implementation) aligns with Principle II
  - .specify/templates/agent-file-template.md ✅ No constitution-specific references; no updates needed

Follow-up TODOs:
  - TODO(RATIFICATION_DATE): Confirm official ratification date with the team if this differs from 2026-02-20.
-->

# goop Constitution

## Core Principles

### I. Code Quality

Every line of Go code MUST follow idiomatic Go conventions (`gofmt`, `golangci-lint` clean).
Functions MUST be focused, short, and carry a single clear responsibility.
Public APIs, non-trivial logic, and all exported symbols MUST have Go doc comments.
Magic values are prohibited; use named constants or typed enumerations.
Error values MUST be wrapped with context using `fmt.Errorf("...: %w", err)` so call-site
messages are traceable without a stack trace.
Code MUST NOT be merged unless it passes all static analysis gates defined in the CI pipeline.

**Rationale**: Go's tooling ecosystem enforces quality at low cost. Consistent style and
error propagation are the primary defense against maintenance debt in a multi-contributor
project, and they keep the reviewer's cognitive load manageable.

### II. Test-First Development (NON-NEGOTIABLE)

Test code MUST be written before implementation code (TDD Red-Green-Refactor).
Tests MUST fail for the right reason before any production code is added.
Unit tests MUST use Go's standard `testing` package with table-driven test cases (`[]struct{}`).
Coverage MUST remain at or above 80% for all packages; coverage regressions MUST be
justified in the PR description.
Benchmarks MUST be added alongside any function whose performance is a stated requirement.

**Rationale**: Test-first removes ambiguity about acceptance criteria and prevents
coverage from decaying over time. Table-driven tests are idiomatic Go and make
regression additions cheap.

### III. Integration & Contract Testing

Every new public interface or command boundary MUST have at least one integration test
that exercises the full call path from input to output.
Text-transformation scripts (the core of goop) MUST each have a contract test that
verifies: given a known input, the transformation produces the exact expected output.
Integration tests MUST live in a `tests/integration/` directory; contract tests in
`tests/contract/`.
Integration tests MUST NOT depend on external services or network access unless
explicitly tagged with `//go:build integration`.

**Rationale**: goop's correctness guarantee is that a transformation does exactly
what its name and documentation say. Contract tests make that guarantee machine-checkable.

### IV. UX Consistency

All user-facing commands, flags, and output formats MUST follow a single, documented
style guide (see `docs/ux-guide.md` — create if absent when first UI surface is added).
Error messages MUST be written in plain English, identify the offending input, and
suggest a corrective action where possible.
Output to `stdout` MUST be the transformation result only; diagnostic or informational
messages MUST go to `stderr`.
Exit codes MUST be consistent: `0` = success, `1` = user error (bad input/flags),
`2` = internal/system error.
Any breaking change to a command's interface MUST be deprecated for at least one minor
release before removal, with a warning printed to `stderr`.

**Rationale**: A text-transformation tool lives or dies by predictability. Users should
never be surprised by inconsistent flag names, mixed output channels, or silent failures.

### V. Simplicity & YAGNI

Abstractions MUST be introduced only when the same logic appears in three or more
distinct call sites (rule of three).
Dependencies MUST be justified; prefer the Go standard library unless an external
package demonstrably reduces complexity or risk.
No feature, configuration option, or code path MAY be added speculatively — all code
must trace to a stated requirement in a spec.
Performance optimizations MUST be backed by benchmark evidence; premature optimization
is a constitution violation.

**Rationale**: goop is a focused tool. Complexity accumulates quietly and its
principal cost is paid by future maintainers, not by the author who introduced it.

## Technical Standards

- **Language**: Go 1.22+ (update floor only on minor releases, documented in `go.mod`).
- **Linting**: `golangci-lint` with project-defined `.golangci.yml`; CI MUST fail on
  lint errors.
- **Formatting**: `gofmt` and `goimports`; unformatted code MUST NOT be merged.
- **Build**: `go build ./...` MUST succeed with zero warnings on all supported platforms.
- **Dependency management**: `go mod tidy` MUST leave no diff; any indirect dependency
  upgrade MUST be deliberate and recorded in the PR.
- **Platform targets**: Linux, macOS, Windows (amd64 + arm64 where applicable);
  platform-specific code MUST be isolated in `_linux.go` / `_darwin.go` / `_windows.go`
  files using Go build tags.

## Development Workflow

- **Branching**: Feature branches named `###-short-description`; merged via PR only.
- **PR requirements**: Green CI (build + lint + test + coverage gate), at least one
  approving review, and a Constitution Check section confirming no principle violations.
- **Commit messages**: Imperative mood, ≤72 chars subject; reference spec/task IDs
  where applicable (e.g., `feat(transform): add base64 encode [T014]`).
- **Quality gates** (all MUST pass before merge):
  1. `go vet ./...` — zero issues
  2. `golangci-lint run` — zero issues
  3. `go test ./... -race -coverprofile=coverage.out` — passes, ≥ 80% coverage
  4. `go build ./...` — zero errors
- **Releases**: Tagged with `vMAJOR.MINOR.PATCH`; changelog entry required per release;
  breaking changes allowed only on MAJOR bumps.

## Governance

This constitution supersedes all informal practices, README guidelines, and ad-hoc
conventions. In case of conflict, the constitution is authoritative.

**Amendment procedure**:
1. Author opens a PR that modifies this file with a rationale section explaining what
   changed and why.
2. The PR MUST include an updated Sync Impact Report (HTML comment at top of this file).
3. At least one reviewer MUST explicitly approve the amendment in the PR review.
4. The `CONSTITUTION_VERSION` MUST be bumped following semantic versioning rules
   (MAJOR: removals/redefinitions; MINOR: additions; PATCH: clarifications).
5. `LAST_AMENDED_DATE` MUST be updated to the merge date in ISO 8601 format.

**Compliance review**:
Every PR description MUST include a "Constitution Check" section confirming:
- No new complexity introduced without justification (Principle V).
- Tests written before implementation where code was added (Principle II).
- Lint and coverage gates pass (Technical Standards).
- UX surfaces (if any) follow the style guide (Principle IV).

**Versioning policy**:
`CONSTITUTION_VERSION` follows `MAJOR.MINOR.PATCH` semantics as defined above.
The version line below is the single source of truth; all tooling MUST read from it.

**Version**: 1.0.0 | **Ratified**: 2026-02-20 | **Last Amended**: 2026-02-20
