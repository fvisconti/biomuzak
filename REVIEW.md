# REVIEW.md — what failure looks like in biomuzak

The short, repo-specific statement of the hard invariants. Reviewers
(`.github/agents/`) read this first; it is the single source of truth for what
matters in this repo. The review checklist, priorities, and verdict format
live in `.github/instructions/code-review-biomuzak.instructions.md`.

## Hard invariants

1. **Shared catalog, per-user playlists.** The song catalog is **global**: any
   authenticated user can list, stream, and download any track. Genre editing
   and song deletion are admin-only. The
   per-user boundary is **playlists and ratings** — a query that can return
   another user's *playlists* or *ratings* is CRITICAL. Admin-only routes require
   the `is_admin` check in `pkg/middleware`. (Deliberate change from the old
   per-user scoping; see `docs/PLAN.md` E1.)
2. **Parameterized SQL only.** `$1, $2, ...` bindings. `fmt.Sprintf` into query
   text is CRITICAL (SQLi).
3. **Auth on every route.** No route reachable without the JWT auth middleware
   except the public ones (login, register, health, Subsonic with its own auth).
   A missing auth check is CRITICAL.
4. **Upload validation.** Uploaded files are validated for type and size;
   filenames are sanitized. An unvalidated upload path is IMPORTANT.
5. **Migrations are immutable.** New numbered file in `db/migrations/`; never
   edit an applied one; never DDL from application code.
6. **Bounded vector search.** k-NN only (`ORDER BY embedding <=> $1 LIMIT k`)
   through `pkg/db/similar.go`; a distance comparison in `WHERE` is a defect.
   Embedding width is 38.
7. **No secrets in logs or responses.** `JWT_SECRET`, `MINIO_*`, `TIDAL_*` are
   never logged or returned to a client.
8. **Explicit errors.** Every error is checked; handlers return sanitized
   messages and never leak internals.

## Nit cap and paths to skip

- At most three 🟢 SUGGESTION findings per review; if there are more, keep the
  three most useful and say how many were dropped.
- Skip: generated files, `frontend/node_modules`, lockfiles, `minio_data/`.

## Convergence

Review must converge. Round cap: 3. After the first pass, review only the new
diff. Never re-post a finding already reported. Flag the class, not the line.
Tag pre-existing issues with `Pre-existing`.
