# REVIEW.md — what failure looks like in biomuzak

The short, repo-specific statement of the hard invariants. Reviewers
(`.github/agents/`) read this first; it is the single source of truth for what
matters in this repo. The generic phrasing rules live in
`.github/instructions/code-review-generic.instructions.md`.

## Hard invariants

1. **Per-user scoping.** Every library query is scoped to the requesting user,
   taken from the verified JWT claim. A query that can return another user's
   songs, albums, or playlists is CRITICAL. Admin-only routes require the
   `is_admin` check in `pkg/middleware`.
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
