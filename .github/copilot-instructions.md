# biomuzak

A self-hosted music organizer, player, and streamer. Users upload audio files; the
server extracts metadata, stores files in MinIO, and serves a REST API plus a
Subsonic-compatible API for playback. A Python microservice generates 38-dim audio
feature embeddings used for "similar songs."

## Golden rules — red/green TDD (mandatory)

- Every change starts with a failing test. Write it, watch it fail, implement the
  minimum to make it pass, then refactor. Never modify behaviour without a red test
  first. Never report a task done while any test is red.
- Don't guess when the stack is high — a wrong direction can be expensive to roll
  back. Ask the user narrow questions in those cases, to share knowledge and decisions.

## Prime directives

1. **Per-user scoping is the security boundary.** Every library query is scoped to
   the requesting user, taken from the verified JWT claim and from nowhere else —
   never a body field, path segment, or query parameter. Admin-only routes are gated
   by the `is_admin` check in `pkg/middleware`. A query that returns another user's
   songs, or an admin route reachable without the admin check, is a defect.
2. **Parameterized SQL only.** Every query binds values with `$1, $2, ...`. No
   `fmt.Sprintf` into query text, ever.
3. **Data must never become code.** Every tunable is an environment variable with a
   documented default. Secrets (`JWT_SECRET`, `MINIO_*`, `TIDAL_*`) have no defaults
   that matter in production and are never logged or returned to a client.
4. **Migrations are immutable.** Schema changes are new numbered files in
   `db/migrations/`, applied in filename order on startup. Never edit an applied
   migration; never issue DDL from application code.
5. **Explicit errors.** Check and handle every error. Handlers degrade gracefully and
   return a sanitized message; internals never leak to the client.

## Tech stack

- **Backend**: Go 1.25 (`go.mod`), `chi` router, `pgx/v4` (Postgres), `golang-jwt`,
  `dhowden/tag`, `minio-go`. Module name is `go-postgres-example` (historical — keep it).
- **Database**: PostgreSQL 18 (`tensorchord/vchord-suite:pg18-latest`) with `pgvector`
  for the `vector(N)` type and `vchord` (`vchordrq`) for the dense index on 38-dim
  song embeddings.
- **Audio processor**: Python FastAPI + Essentia (`audio-processor/`).
- **Frontend**: React + Vite (`frontend/`).
- **Testing**: Go `testify` + `go-sqlmock`; Python `pytest`.

## Commands

```bash
go test -race ./...        # unit suite (offline, go-sqlmock, httptest)
go build ./...             # build
go vet ./...               # vet
pytest                     # audio-processor tests (run inside audio-processor/)
docker compose up          # full stack (postgres, backend, frontend, audio-processor, minio)
```

## Working agreements

- Go style and package layout: `.github/instructions/go.instructions.md` (authoritative).
- SQL and database rules: `.github/instructions/sql.instructions.md`.
- Vector search rules: `.github/instructions/vchord.instructions.md` (ground engine
  decisions in the official VectorChord docs it links).
- Audio-processor (Essentia/FastAPI) rules: `.github/instructions/audio-processor.instructions.md`.
- Frontend (React/Vite/Chakra) rules: `.github/instructions/frontend.instructions.md`.
- Testing rules: `.github/instructions/tests.instructions.md`.
- Docs sync: `.github/instructions/docs.instructions.md`.
- Keep files focused; prefer a new file over growing any file past ~800 lines.
- Vector search is bounded k-NN only (`ORDER BY embedding <=> $1 LIMIT k`) through
  `pkg/db/similar.go`; a distance comparison in `WHERE` is a defect.
- **Every user-visible change ⇒ one bullet under `## [Unreleased]` in
  `docs/CHANGELOG.md`** (Keep a Changelog).
- Use Conventional Commits (`feat(scope):`, `fix(scope):`, `refactor(scope):`, `docs:`).

## Review gate — the life of a change

A task is **not done** until the diff has been reviewed and every finding is fixed or
dismissed with a one-line reason. The three reviews are **advisory** — run them when
you want; they record a marker as evidence but do not block a commit (a WIP commit with
reviews pending is intentional and allowed). CI is the floor.

1. `go test -race ./...` and `go vet ./...` green.
2. `/pre-pr-review` — fresh-context Code Reviewer on the diff; **human** dispositions
   every finding (fix or reason).
3. `/security-review` — Security Reviewer; an open CRITICAL must be fixed or
   explicitly accepted.
4. `/performance-review` — Performance Reviewer; a claim without a measurement is
   marked "needs measurement", not accepted.
5. Scoped reviews (run when the diff touches that area):
   - `/audio-processor-review` — Code Reviewer scoped to `audio-processor/`
     against the 38-dim contract, normalization, and temp-file rules.
   - `/frontend-review` — Code Reviewer scoped to `frontend/` against the
     API-client, context-state, theming, and no-secrets rules.
6. `/risk-tier` — classifier proposes Low/Medium/High; **human** confirms or raises it.
   The change sets the tier, not the author.
7. Commit + push; CI is the floor. High tier ⇒ the author reads the diff and the
   evidence (test output), not just "green".

Reviewers are pinned agents (`.github/agents/`); what failure looks like here is
`REVIEW.md`.

## Resources

- Local run: configure `.env` (see `.env.example`), `docker compose up audio-processor`,
  then `go run cmd/server/main.go`.
- Key env: `DATABASE_URL` (or `DB_HOST`/`DB_PORT`/`POSTGRES_*`), `JWT_SECRET` (required,
  non-default), `PORT`, `AUDIO_PROCESSOR_URL`, `MINIO_*`, `CORS_ALLOWED_ORIGINS`,
  `ALLOW_REGISTRATION`, `TIDAL_CLIENT_ID`/`TIDAL_CLIENT_SECRET`.
