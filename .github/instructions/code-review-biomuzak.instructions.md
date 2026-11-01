---
description: 'biomuzak-specific code review instructions for the self-hosted music organizer, player, and streamer'
applyTo: '**'
excludeAgent: ["coding-agent"]
---

# biomuzak Code Review Instructions

Comprehensive code review guidelines for biomuzak, a self-hosted music organizer,
player, and streamer built in Go with a Python audio-processor microservice and a React
frontend. This file is the project-specific specialization of
`.github/instructions/code-review-generic.instructions.md`; read both. `REVIEW.md` (repo
root) is the single source of truth for what matters in this repo.

## Review Language

When performing a code review, respond in **English**.

## Review Priorities

When performing a code review, prioritize issues in the following order:

### 🔴 CRITICAL (Block merge)
- **Security**: Vulnerabilities, exposed secrets, authentication/authorization issues
- **Correctness**: Logic errors, data corruption risks, race conditions
- **Breaking Changes**: API contract changes without versioning
- **Data Loss**: Risk of data loss or corruption
- **biomuzak Prime Directives Violated**: per-user scoping bypass, SQL built by
  concatenation, secrets in defaults or logs, edited applied migration, unbounded vector
  search

### 🟡 IMPORTANT (Requires discussion)
- **Code Quality**: Severe violations of SOLID principles, excessive duplication
- **Test Coverage**: Missing tests for critical paths or new functionality
- **Performance**: Obvious performance bottlenecks (N+1 queries, memory leaks)
- **Architecture**: Significant deviations from established patterns
- **Documentation Sync**: Docs not updated to match code changes

### 🟢 SUGGESTION (Non-blocking improvements)
- **Readability**: Poor naming, complex logic that could be simplified
- **Optimization**: Performance improvements without functional impact
- **Best Practices**: Minor deviations from conventions
- **Documentation**: Missing or incomplete comments/documentation

## biomuzak-Specific Review Checklist

When performing a code review for biomuzak changes, verify the following:

### Prime Directives (Non-negotiable)
- [ ] **Per-user scoping** (Directive 1): Every library query is scoped to the
  requesting user from the verified JWT claim; never from a body field, path segment, or
  query parameter. Admin-only routes require the `is_admin` check in `pkg/middleware`.
- [ ] **Parameterized SQL only** (Directive 2): `$1, $2, ...` bindings; no
  `fmt.Sprintf` into query text.
- [ ] **No magic literals** (Directive 3): Every tunable is an environment variable with
  a documented default. Secrets (`JWT_SECRET`, `MINIO_*`, `TIDAL_*`) have no meaningful
  defaults and are never logged or returned to a client.
- [ ] **Migrations immutable** (Directive 4): Schema change is a new numbered file in
  `db/migrations/`; no DDL from application code; no editing an applied migration.
- [ ] **Explicit errors** (Directive 5): Every error is checked; handlers return
  sanitized messages and never leak internals.

### Architecture & Patterns
- [ ] **Vector Search Bounds**: Only bounded k-NN through `pkg/db/similar.go`. No
  distance comparison in `WHERE` clause. Similarity filters are index-side only and an
  untrusted radius is clamped in Go.
- [ ] **Embedding Width**: Embeddings are 38-dim; a vector whose length is not 38 is
  rejected before SQL is built.
- [ ] **SQL Repositories**: Hand-written parameterized SQL through pgx/v4. No ORM.
  Parameters properly bound.
- [ ] **HTTP Routes**: Routes registered on the `chi` router. Handlers return sanitized
  errors; auth and admin middleware applied where required.

### Go Style
- [ ] **Follows `.github/instructions/go.instructions.md`**: Idiomatic Go, consistent
  with repository patterns
- [ ] **Package Scope**: Files focused; no file grows past ~800 lines
- [ ] **Red/Green TDD**: Behavior change starts with failing test. No behavior changes
  without red test first.
- [ ] **Error Wrapping**: Uses `fmt.Errorf(...%w, err)` for proper error chains

### Database & SQL
- [ ] **Follows `.github/instructions/sql.instructions.md`**: PostgreSQL 18 patterns,
  vchord usage
- [ ] **Migrations**: New numbered file in `db/migrations/`, immutable once merged
- [ ] **Per-user scoping**: Every library query carries a `user_id` predicate
- [ ] **No Hardcoded Limits**: Vector search limit, pagination, batch sizes are from
  environment variables

### Testing
- [ ] **Follows `.github/instructions/tests.instructions.md`**: Red/green TDD,
  table-driven tests, fixture policy
- [ ] **Offline unit tests**: `go-sqlmock` and `httptest.Server`; no live Postgres,
  MinIO, or audio-processor
- [ ] **Testdata**: Fixtures stored in `testdata/`, never inline; no large audio files
  committed
- [ ] **Test Isolation**: No test pollution; tests pass independently and in any order
- [ ] **Race Detection**: Passes `go test -race ./...` for concurrency safety

### Configuration & Secrets
- [ ] **No Secret Defaults**: Secrets fail fast when missing; never have meaningful
  default values
- [ ] **Secrets Write-Only**: `JWT_SECRET`, `MINIO_*`, `TIDAL_*` never logged or
  returned to a client
- [ ] **`.env.example` Sync**: New env var added to `.env.example` and `docs/DOCKER.md`

### Documentation Sync
- [ ] **Docs-Sync Table**: Checked against `.github/instructions/docs.instructions.md`;
  all required docs updated
- [ ] **`docs/CHANGELOG.md`**: User-visible changes have bullets under `## [Unreleased]`
- [ ] **`docs/DOCKER.md`**: Deployment/docker-compose/env changes documented
- [ ] **`docs/USER_GUIDE.md`**: User-facing feature changes documented

## biomuzak-Specific Language & Framework Checks

When performing a code review, verify:

### Go Idioms & Patterns
- [ ] **Concurrency Safety**: Code reviewed for race conditions; passes
  `go test -race ./...`
- [ ] **Interface Design**: Interfaces are small and focused; no bloated interfaces
- [ ] **Error Handling**: Uses explicit `if err != nil` checks; no error-ignoring
  patterns
- [ ] **Nil Handling**: Nil pointers properly checked; no nil dereferences
- [ ] **Goroutine Cleanup**: All goroutines have clear exit paths; no goroutine leaks
- [ ] **Channel Usage**: Channels properly closed by sender; receives check for closure

### PostgreSQL & pgx
- [ ] **Prepared Statements**: All queries use parameterized statements; no SQL
  injection risks
- [ ] **Connection Pooling**: Uses pgx pool, no raw connection leaks
- [ ] **Transaction Handling**: Transactions properly committed/rolled back
- [ ] **Index Usage**: New queries leverage appropriate indexes
- [ ] **EXPLAIN Plans**: Complex queries verified with EXPLAIN for performance

### Vector Search
- [ ] **VectorChord Usage**: Follows `.github/instructions/vchord.instructions.md`
- [ ] **Dimension Consistency**: Query and stored vectors are both 38-dim
- [ ] **k-NN Bounds**: All searches use `LIMIT k`, no unbounded loops
- [ ] **Distance Comparisons**: Only in retrieval layer, never in WHERE clauses
- [ ] **Index Selection**: `vchordrq` with `vector_cosine_ops`

### Uploads & Storage
- [ ] **Upload Validation**: File type and size validated; filenames sanitized
- [ ] **MinIO**: Object keys derived from safe, non-user-controlled paths
- [ ] **Cleanup**: Deleting a song removes its stored file and dependent rows

### Audio Processor (Essentia / FastAPI)
- [ ] **Follows `.github/instructions/audio-processor.instructions.md`**
- [ ] **38-dim contract**: Embedding is still 13+13+6+6 = 38; a dimension change is a
  BREAKING change with a migration + `pkg/db/similar.go` update + changelog
- [ ] **Normalization**: min-max to [0, 1] with the constant-vector guard; no silent
  switch to z-score/L2
- [ ] **Temp file hygiene**: `tempfile.mkstemp` + cleanup in `finally`; no path from
  user input
- [ ] **No real audio in tests**: `es.MonoLoader`/`es.Extractor` mocked; assert shape
  and range, not exact values

### Frontend (React / Vite / Chakra)
- [ ] **Follows `.github/instructions/frontend.instructions.md`**
- [ ] **API calls via `src/api/client.js`**: no ad-hoc `fetch`/`axios` in components
- [ ] **State via contexts**: auth from `AuthContext`, playback from `PlayerContext`;
  no `localStorage` reads in components
- [ ] **No secrets client-side**: no API keys/tokens in frontend code
- [ ] **Chakra theme**: colors/spacing from `src/theme.js`, not hardcoded
- [ ] **ESLint clean**: `npm run lint` passes

## biomuzak Build & Deployment

When performing a code review, verify:

### Build & Verification
- [ ] **`go build ./...`**: Change compiles
- [ ] **`go vet ./...`**: No vet findings
- [ ] **`go test -race ./...`**: No race conditions detected
- [ ] **`pytest`** (audio-processor): Python suite green where the change touches it
- [ ] **Docker**: Dockerfile and docker-compose.yml updated if needed

### Configuration & Runtime
- [ ] **Environment Variables**: All new config uses env vars, not hardcoded
- [ ] **Boot Failures**: Missing required env (e.g. `JWT_SECRET`) cause startup failure
  (fast-fail)
- [ ] **Observability**: Errors logged with context; no silent failures

## biomuzak Contract Rules

When performing a code review, enforce:

1. **Per-user scoping**: A library operation is scoped to the requesting user from the
   JWT; the same operation is not reachable with another user's data.
2. **Changelog**: Every user-visible change has a `docs/CHANGELOG.md` bullet under
   `## [Unreleased]`.
3. **Conventional Commits**: Commit messages follow `feat(scope):`, `fix(scope):`,
   `refactor(scope):`, `docs:` format.

## Checklist Template

When completing a review, mark each item:

```
- [x] Requirement met
- [ ] Requirement not met
- [n/a] Not applicable
```

## Examples of biomuzak-Specific Issues to Flag

### ❌ Per-user scoping bypass (Directive 1 Violation)
```go
// BAD: id from the path, no user predicate
songID := r.URL.Path
song, err := db.GetSong(ctx, songID)
```

```go
// GOOD: id from the path, scoped to the JWT user
principal, err := auth.ExtractPrincipal(r)
if err != nil {
    return nil, fmt.Errorf("extract principal: %w", err)
}
song, err := db.GetSong(ctx, principal.UserID, songID)
```

### ❌ SQL built by concatenation (Directive 2 Violation)
```go
// BAD: string concatenation
query := "SELECT * FROM songs WHERE title LIKE '%" + title + "%'"
```

```go
// GOOD: parameterized
query := "SELECT * FROM songs WHERE user_id = $1 AND title LIKE $2"
rows, err := db.Query(ctx, query, principal.UserID, "%"+title+"%")
```

### ❌ Secret in a default (Directive 3 Violation)
```go
// BAD: meaningful default for a secret
secret := os.Getenv("JWT_SECRET")
if secret == "" {
    secret = "default-secret-change-in-production"
}
```

```go
// GOOD: fail fast when the secret is missing
secret := os.Getenv("JWT_SECRET")
if secret == "" {
    return nil, errors.New("JWT_SECRET is required")
}
```

### ❌ Unbounded vector search (Arch Violation)
```sql
-- BAD: no limit
SELECT embedding <=> $1 AS distance FROM song_embeddings ORDER BY distance
```

```sql
-- GOOD: bounded k-NN
SELECT embedding <=> $1 AS distance FROM song_embeddings ORDER BY distance LIMIT $2
```

## Additional Resources

For more information on biomuzak and code review:

- **Go Instructions**: `.github/instructions/go.instructions.md` — Go idioms for this repo
- **SQL Instructions**: `.github/instructions/sql.instructions.md` — PostgreSQL patterns
- **VectorChord Guide**: `.github/instructions/vchord.instructions.md` — vector search patterns
- **Audio Processor**: `.github/instructions/audio-processor.instructions.md` — Essentia/FastAPI rules
- **Frontend**: `.github/instructions/frontend.instructions.md` — React/Vite/Chakra rules
- **Test Instructions**: `.github/instructions/tests.instructions.md` — Red/green TDD policy
- **Docs Sync**: `.github/instructions/docs.instructions.md` — which docs to update
- **Review invariants**: `REVIEW.md` — what failure looks like in this repo

## Project Context

**Tech Stack**:
- Go 1.25 with `chi` router, `pgx/v4`, `golang-jwt`, `dhowden/tag`, `minio-go`
- PostgreSQL 18 with vchord for 38-dim cosine vector search
- Python FastAPI + Essentia audio-processor microservice
- React + Vite frontend

**Architecture**: Self-hosted music organizer/player/streamer; uploads are stored in
MinIO, metadata in Postgres, 38-dim embeddings generated by the audio-processor and used
for "similar songs" via bounded k-NN.

**Key Files**:
- Entry point: `cmd/server/main.go`
- HTTP handlers: `pkg/handlers/`
- Database: `pkg/db/` (vector search in `pkg/db/similar.go`)
- Auth & middleware: `pkg/auth/`, `pkg/middleware/`
- Storage: `pkg/storage/` (MinIO)
- Subsonic API: `pkg/subsonic/`
- Migrations: `db/migrations/`
- Audio processor: `audio-processor/`

**Build Tool**: `go build ./...`, `go test -race ./...`, `go vet ./...`, `pytest`
(audio-processor), `docker compose up`

**Testing**: Table-driven tests, fixtures in `testdata/`, red/green TDD mandatory,
`go-sqlmock` + `httptest.Server` for offline unit tests

**Code Style**:
- Follows `.github/instructions/go.instructions.md`
- Conventional commits
- Explicit error handling
- Parameterized SQL only
- Red/green TDD (failing test before behavior change)
