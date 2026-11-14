# pi-specific wiring (biomuzak)

The repo's agent instructions live in `.github/` — the single source of truth shared
with VS Code Copilot. `AGENTS.md` is a symlink to `.github/copilot-instructions.md`;
do not fork the content, edit the `.github/` files.

## Glob-scoped instructions — enforce them manually

Copilot applies `.github/instructions/*.instructions.md` by `applyTo` globs; pi has no
glob-triggered context, so the mapping is: **before writing, reviewing, or refactoring a
file matching a row, read that instruction file first** (if not already in context) and
hold the work to it. Review findings must cite the file, not restate it.

| Touching | Read first |
|---|---|
| `*.go`, `go.mod`, `go.sum` | `.github/instructions/go.instructions.md` |
| SQL anywhere: `pkg/db/`, `db/migrations/`, `*.sql`, embedded query text | `.github/instructions/sql.instructions.md` **and** `.github/instructions/vchord.instructions.md` |
| `audio-processor/**` (`.py`, `pyproject.toml`, `Dockerfile`) | `.github/instructions/audio-processor.instructions.md` |
| `frontend/**` (`.js`, `.jsx`, `vite.config.js`, `package.json`, `nginx.conf`) | `.github/instructions/frontend.instructions.md` |
| `*_test.go`, `testdata/`, `tests/` | `.github/instructions/tests.instructions.md` |
| any doc-affecting change (`.md`, compose files, `Dockerfile`) | `.github/instructions/docs.instructions.md` |
| any review of a diff / PR / branch | `.github/instructions/code-review-biomuzak.instructions.md` and `REVIEW.md` |

## Review gate in pi

The review skills (`/pre-pr-review`, `/security-review`, `/performance-review`,
`/audio-processor-review`, `/frontend-review`, `/risk-tier`) are loaded from
`.github/skills/` and say "invoke the X agent as a subagent". In pi that means the
`run_agent` tool: a fresh, isolated, **read-only** pi subprocess running the pinned
agent from `.github/agents/`. Available agents: `Code Reviewer`, `Performance Reviewer`,
`Security Reviewer`, `Risk Classifier`.

- Pass the base ref and diff range in `task`; the agent reads `REVIEW.md` itself.
- Runs stream progress, stay cancellable with Esc, and never block the TUI — if a
  review looks hung, it is not; the latest reviewer turn is shown in the tool update.
- A run that produced no answer is a failure; never write a review marker for an
  empty review.
