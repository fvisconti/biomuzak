---
description: 'PostgreSQL 18 / pgvector / vchord schema and query rules for this repository'
applyTo: '**/*.go,**/*.sql'
---

# SQL & Database Instructions

Applies to all SQL in `pkg/db/`, `db/migrations/` and any handler package that embeds
query text.

## Schema facts

- PostgreSQL 18 (`tensorchord/vchord-suite:pg18-latest`) with the `vector` (the type,
  from pgvector), `vchord` (the dense index) and `pg_trgm` extensions, declared in
  `db/migrations/` and applied in filename order on startup — never at application
  boot from code.
- All SQL is hand-written and executed through `pgx/v4` (via `database/sql`). There is
  no ORM.
- `songs` is the core row; `song_embeddings` holds the 38-dim `vector(38)` per song.
  Similarity search operates on `song_embeddings`.
- **Scoping is per-user**: library rows carry the owning user; a query returns only the
  requesting user's data. There is no multi-tenant RLS — the user predicate is the
  boundary.

## Mandatory rules

1. **Per-user scoping is structural** — every library query carries a `user_id`
   predicate bound from the verified JWT claim. Never write a library query that relies
   on an application-supplied id alone without the user predicate. A query that can
   return another user's songs, albums or playlists is a defect.
2. **Parameterized SQL only** — `$1, $2, ...`. No `fmt.Sprintf` into query text, ever
   (SQLi).
3. **Context timeouts** — every repository call takes a `context.Context` and runs
   under `context.WithTimeout` (5–8s for reads, longer only for bulk writes) to avoid
   504s and connection-pool exhaustion.
4. **Bound vector scans** — vchord accelerates `ORDER BY embedding <=> $1 LIMIT k`. A
   distance comparison in the `WHERE` clause is a post-scan filter and degrades to a
   full sequential scan — never write one. The sanctioned bound is the index-side
   similarity filter, `WHERE embedding <<=>> sphere($1::vector, $2)`, which becomes an
   `Index Cond`; the `ORDER BY ... LIMIT k` stays either way and an untrusted radius is
   clamped in Go first (`.github/instructions/vchord.instructions.md`). Dense search
   goes through the single helper in `pkg/db/similar.go`, which also rejects any vector
   whose length is not 38 before the SQL is built.
5. **Idempotent upserts** — `ON CONFLICT ... DO UPDATE` plus `COALESCE(NULLIF($n, ''),
   col)` so empty values never overwrite rich data.
6. **NULL is distinct in a unique index** — a composite `UNIQUE` over a nullable column
   does not deduplicate. Use `WHERE ... IS NOT NULL` partial unique indexes instead.
7. **Schema changes are migrations** — new numbered files in `db/migrations/`, ordered,
   immutable once merged. Never issue DDL from application code, never `IF NOT EXISTS`
   your way around a missing migration.
8. **Cascade cleanup is explicit** — deleting a song must delete every dependent row in
   the same transaction (`song_embeddings` referencing a deleted song, playlist entries,
   stored files in MinIO). Never leave dangling references.
9. **Vectors cross the boundary as one helper** — `[]float64` is formatted to
   pgvector's text literal in exactly one place (`pkg/db/similar.go`), never per query.
   That helper is also the single place the width is checked: a search query and the
   stored rows must share the 38-dim width or the distance is meaningless.
10. **Explain what you index** — every new index is added together with the query it
    serves, in the same commit; an index without a consumer is a cost with no benefit.
    Engine options, GUCs and ceiling behaviour:
    `.github/instructions/vchord.instructions.md`.

## Testing SQL changes

Offline unit tests use `go-sqlmock` and must never require PostgreSQL. Every schema
change ships with: a migration, a unit test exercising the new column/constraint via
`go-sqlmock`, and a per-user-scoping test proving the row is invisible from another
user.
