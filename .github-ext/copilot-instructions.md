# Magentia Knowledge Graph Service (MKGS)

> **Status: this tree is the legacy implementation and is scheduled for deletion.** `docs/BUILD_PLAN.md` specifies a repository that does not exist yet; build it from the plan, do not refactor this tree toward it. Delete this banner (and the matching note in the plan) when the plan is executed.

General-purpose knowledge ingestion & retrieval engine that **builds a graph**: an LLM pipeline extracts entities and typed relationships from source documents, attests every extracted fact to the chunk it came from, and retrieval fuses dense vectors (pgvector/vchord), sparse search (vchord_bm25) and graph traversal into cited answers. Served through a `net/http` JSON API, an admin web UI, an admin CLI, and an MCP server — one binary.

`docs/BUILD_PLAN.md` is the complete specification: read it before building anything. If code and plan disagree, fix the plan first.

## Golden rules — red/green TDD (mandatory)

- Every change starts with a failing test. Write it, watch it fail, implement the minimum to make it pass, then refactor. Never modify behaviour without a red test first. Never report a task done while any test is red.
- Don't guess when the stack is high - I mean taking a direction can cost high price to rollback. Ask the user narrow questions in those cases, in order to share knowledge and decisions.


## Prime directives

1. **The engine is domain-blind.** No entity type, relation type, country, regulation or taxonomy ever appears in Go code. Vocabulary arrives as per-scope data; extraction behaviour lives in prompt templates. Adding domain knowledge to a `switch` is a defect.
2. **Four axes, and they are not interchangeable.** Tenant and user come from a **verified JWT claim and from nowhere else** (Principal) — never a body field, path segment or query parameter; MKGS verifies principals and never manages them, so there is no `users` table, no password column and no login/token route, and `owner_user_id` is a bare `uuid` with no FK to a directory this database does not contain. Tenants are the security boundary (PostgreSQL `FORCE ROW LEVEL SECURITY`); scopes are retrieval domains within a tenant. Ownership (`tenant_id`, `owner_user_id`), identity (`UNIQUE (tenant_id, entity_type_id, normalized_name)`) and scoping (`source_scopes`/`node_scopes`) are separate concerns. Labelling is sufficient for scope, insufficient for identity, and unacceptable for authorization: an RLS policy that reads a label is a defect.
3. **Data must never become code.** Every tunable value is an environment knob with a documented default and a declared class: `boot` (frozen at start — DSNs, credentials, port, `EMBEDDING_DIM`) or `live` (overridable at runtime, audited). No magic literals. Secrets have no defaults — boot fails fast when one is missing — and a `live` secret is write-only over every surface.
4. **Provenance is mandatory.** Every node and edge cites at least one source chunk; answers cite their evidence and disclose gaps instead of speculating.
5. **Contract-first.** `openapi.json` and SQL migrations exist before the handlers and DDL they describe; a CI test checks both directions.

## Tech stack

- Go 1.26.1 (`go.mod`); stdlib `net/http.ServeMux` for routing (method + path patterns, `r.PathValue`); **no router and no ORM** — repositories are hand-written parameterized SQL through pgx.
- Direct dependencies, and nothing else without a decision recorded in `docs/BUILD_PLAN.md` §4.1: `jackc/pgx/v5`, `joho/godotenv` (dev only), `go.uber.org/zap`, `prometheus/client_golang`, `sony/gobreaker/v2`, `modelcontextprotocol/go-sdk`, `golang-jwt/jwt/v5`, `spf13/cobra` (admin CLI only, never on a request path).
- PostgreSQL 18 (`tensorchord/vchord-suite:pg18-latest`): **vchord is the elected dense engine** (pgvector is present only for the `vector(N)` type), vchord_bm25 (sparse), graph traversal in SQL.
- Admin web UI: vanilla JS in `static/admin/` (no build step). Admin CLI: `cobra` in `internal/admincli`, same service layer, not an HTTP client.

## Commands

All commands live in the `Makefile` — run `make help` and use the target; do not copy raw `go test`/`go build` flags into docs or scripts.

```bash
make test        # unit suite (offline, no DB, no network)
make test-race   # concurrency safety — required before commit
make verify      # fmt-check + vet + lint + test + route/spec sync
make lint        # golangci-lint
make test-int    # integration suite (live PostgreSQL, DATABASE_URL)
make up          # docker compose up (mkgs + MKGS-db)
```

## Working agreements

- Go style and package layout: `.github/instructions/go.instructions.md` (authoritative — do not duplicate here).
- Keep files focused; prefer a new file over growing any file past ~800 lines.
- Errors: explicit `if err != nil`. Handlers degrade gracefully and return a sanitized message; ingestion jobs mark the failing source failed and continue the batch.
- All model traffic goes through the one gateway client in `internal/llm` — call sites never see a transport, so the concurrency slot, circuit breaker and retries cannot be bypassed.
- Background work goes through `task.Go` (panic-safe by construction); raw `go func(` outside `internal/task` is a defect.
- Vector search is bounded k-NN only (`ORDER BY embedding <=> $1 LIMIT k`) through the one vchord helper in `internal/pg`; a distance comparison in `WHERE` is forbidden, and the only sanctioned bound is the index-side similarity filter (`<<=>> sphere($1::vector, $2)`), with an untrusted radius clamped in Go. A vector whose length is not `EMBEDDING_DIM` is rejected before SQL is built — Matryoshka truncation happens in `internal/llm`, once. Engine details: `.github/instructions/vchord.instructions.md`, `docs/VCHORD.md`.
- Embeddings are requested at `EMBEDDING_NATIVE_DIM` and truncated to `EMBEDDING_DIM` only when `EMBEDDING_MRL=true`; query and document vectors always share a width.
- Routes are registered on the stdlib mux as `"METHOD /path/{wildcard}"`; handlers use `respond`/`respondError` and never build an error body by hand.
- New endpoint ⇒ update `openapi.json` + `docs/OPERATIONS.md`. New knob ⇒ `docs/CONFIGURATION.md` + compose env + its `boot`|`live` class in the registry. **Every user-visible change ⇒ one bullet under `## [Unreleased]` in `CHANGELOG.md`** (Keep a Changelog).
- **Three surfaces, one contract.** An admin operation is a service function rendered by the HTTP handler, the admin UI and the CLI. `internal/admincli` and `internal/httpapi` never import each other. A verb reachable from one surface must be reachable from all three, and every state change writes its audit row in the same transaction.
- Run `make verify` before every commit. Use Conventional Commits (`feat(scope):`, `fix(scope):`, `refactor(scope):`, `docs:`).

## Review gate — the life of a change

A task is **not done** until the diff has been reviewed and every finding is fixed or dismissed with a one-line reason. The commit-gate hook enforces the one hard floor: `git commit` is denied only when `make verify` is red. The three reviews are **advisory** — run them when you want, they record a marker as evidence, but a missing marker does not block a commit (a WIP commit with reviews pending is intentional and allowed). The gate has two callers and one verdict — the agent hook (`.github/hooks/`) and `.githooks/pre-commit` for a human terminal; `make hooks-install` installs the latter.

1. `make verify` green (the hook re-checks at commit time — this is the only thing that blocks).
2. `/pre-pr-review` — fresh-context Code Reviewer on the diff; **human** dispositions every finding (fix or reason).
3. `/security-review` — Security Reviewer; an open CRITICAL must be fixed or explicitly accepted.
4. `/performance-review` — Performance Reviewer; a claim without a measurement is marked "needs measurement", not accepted.
5. `/risk-tier` — classifier proposes Low/Medium/High; **human** confirms or raises it. The change sets the tier, not the author; `CODEOWNERS` paths force High.
6. Commit + push; CI is the floor. High tier ⇒ the author reads the diff and the evidence (test output, eval baseline), not just "green".

Reviewers are pinned agents (`.github/agents/`, model pinned by name — `make review-models STRONG=...` re-pins the roster); what failure looks like here is `REVIEW.md`; repeated findings migrate into `REVIEW.md` or a `make check-*` guard; shipped defects become cases in `evals/review/`.
