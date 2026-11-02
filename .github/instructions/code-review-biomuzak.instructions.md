---
description: 'biomuzak-specific code review instructions for the self-hosted music organizer, player, and streamer'
applyTo: '**'
excludeAgent: ["coding-agent"]
---

# biomuzak Code Review Instructions

Review language: **English**. `REVIEW.md` (repo root) is the single source of truth
for what failure looks like here. Detailed per-area rules live in the sibling
instruction files — this file is the review **checklist** that ties them together.
Do not restate a sibling file's rules in a finding; cite it.

## Priorities

| Tier | Action | Examples |
| :--- | :--- | :--- |
| 🔴 CRITICAL | Block merge | Security hole, data loss/corruption, race, breaking API change, prime-directive violation |
| 🟡 IMPORTANT | Requires discussion | Missing tests for a new path, N+1 query, memory leak, architecture drift, docs out of sync |
| 🟢 SUGGESTION | Non-blocking | Naming, readability, minor convention drift, missing comments |

## Prime directives (a violation is always 🔴)

1. **Shared catalog, per-user playlists.** The song catalog is global: any
   authenticated user can list/stream/download any track; genre editing and song
   deletion are admin-only (`is_admin` check in `pkg/middleware`). Every *playlist*
   and *rating* query is scoped to the requesting user from the verified JWT claim —
   never from a body field, path segment, or query parameter.
2. **Parameterized SQL only.** `$1, $2, ...` bindings; no `fmt.Sprintf` into query
   text, ever.
3. **No magic literals.** Every tunable is an env var with a documented default.
   Secrets (`JWT_SECRET`, `MINIO_*`, `TIDAL_*`) have no meaningful defaults and are
   never logged or returned to a client. Missing required env fails fast at boot.
4. **Migrations immutable.** Schema change = new numbered file in `db/migrations/`.
   No DDL from application code; never edit an applied migration.
5. **Explicit errors.** Check every error; handlers return sanitized messages and
   never leak internals.

## Checklist — check the diff against each file, flag violations with file:line

- [ ] **Go** — `go.instructions.md`: idiomatic, `fmt.Errorf(...%w, err)` wrapping,
  no file past ~800 lines, goroutines have clear exit paths.
- [ ] **SQL** — `sql.instructions.md`: parameterized, playlist/rating queries carry
  a `user_id` predicate, transactions commit/rollback, cascade cleanup on delete.
- [ ] **Vector search** — `vchord.instructions.md`: bounded k-NN only
  (`ORDER BY embedding <=> $1 LIMIT k`) through `pkg/db/similar.go`; no distance
  comparison in `WHERE`; 38-dim vectors rejected before SQL is built; limits from
  env vars, never hardcoded.
- [ ] **Audio processor** — `audio-processor.instructions.md`: 38-dim contract
  (13+13+6+6), min-max normalization, `tempfile.mkstemp` + `finally` cleanup, no
  real audio in tests.
- [ ] **Frontend** — `frontend.instructions.md`: API calls through
  `src/api/client.js` (no ad-hoc fetch/axios in components), auth/playback state
  via contexts, theme values not hardcoded, `npm run lint` passes.
- [ ] **Tests** — `tests.instructions.md`: red/green (failing test first),
  table-driven, offline (`go-sqlmock` + `httptest.Server`), fixtures in
  `testdata/`, order-independent, `go test -race ./...` green.
- [ ] **Docs sync** — `docs.instructions.md`: `docs/CHANGELOG.md` bullet under
  `## [Unreleased]` for every user-visible change; new env var in `.env.example`
  **and** `docs/DOCKER.md`; `docs/USER_GUIDE.md` for user-facing changes.
- [ ] **Config & secrets** — `JWT_SECRET` / `MINIO_*` / `TIDAL_*` never logged,
  never in a response body, never defaulted to something usable.

## Build gate — green before any review is valid

```bash
go build ./... && go vet ./... && go test -race ./...
pytest                        # in audio-processor/, when touched
npm run lint && npm run build # in frontend/, when touched
```

## Defect pairs (the repo's five recurring failure modes)

| Never write | Write instead | Directive |
| :--- | :--- | :--- |
| `db.GetPlaylist(ctx, playlistID)` | `db.GetPlaylist(ctx, principal.UserID, playlistID)` | 1 — scoping from JWT |
| `"… WHERE name LIKE '%" + name + "%'"` | `"… WHERE user_id = $1 AND name LIKE $2"` | 2 — parameterize |
| empty `JWT_SECRET` ⇒ fallback to a default | fail fast: `errors.New("JWT_SECRET is required")` | 3 — no secret defaults |
| `ORDER BY embedding <=> $1` (no limit) | `ORDER BY embedding <=> $1 LIMIT $2` | bounded k-NN |
| embedding of any length passed to SQL | reject length ≠ 38 in `pkg/db/similar.go` first | 38-dim contract |

## Contract rules

1. Per-user boundary: a playlist or rating of user A is unreachable as user B, even
   by guessing ids. Catalog (songs/albums) access is intentionally global.
2. Every user-visible change has a `docs/CHANGELOG.md` bullet under
   `## [Unreleased]`.
3. Conventional Commits: `feat(scope):`, `fix(scope):`, `refactor(scope):`, `docs:`.

## Verdict format

Mark each checklist item `- [x]` met, `- [ ]` not met (with finding), or
`- [n/a]`. Findings are ordered 🔴 → 🟡 → 🟢, each with file:line and a one-line
fix. Do not edit code during review — disposition belongs to the author.
