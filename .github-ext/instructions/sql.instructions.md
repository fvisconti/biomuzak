---
description: 'PostgreSQL 18 / pgvector / vchord / vchord_bm25 schema and query rules for this repository'
applyTo: '**/*.go,**/*.sql'
---

# SQL & Database Instructions

Applies to all SQL in `internal/pg/`, `internal/pg/migrate/`, `migrations/` and any handler package
that embeds query text.

## Schema facts

- PostgreSQL 18 with the `citext`, `vector` (the type, from pgvector), `vchord` (the elected dense
  index), `vchord_bm25` and `pg_trgm` extensions, declared in `migrations/` and applied by
  `mkgs migrate` — never at application boot.
- All SQL is hand-written and executed through `pgx` (`pgxpool`). There is no ORM.
- `sources` is the physical file (`filename` is a real column); `pages` and `chunks` are its
  derivative rows. Retrieval and provenance operate on `chunks`.
- **Identity is tenant-global**: `nodes` is unique on `(tenant_id, entity_type_id, normalized_name)`;
  `edges` on `(tenant_id, relation_type_id, source_node_id, target_node_id)` plus
  `CHECK (source_node_id <> target_node_id)`. No identity key contains a scope.
- **Scope is a label**: `source_scopes(source_id, scope_id)` and `node_scopes(node_id, scope_id)`. A
  scope never appears in a unique constraint and never in a policy.
- `attestations` links a node or an edge to the chunk that supports it
  (`CHECK (num_nonnulls(node_id, edge_id) = 1)`). Every node and edge needs at least one.

## Mandatory rules

1. **Tenant isolation is structural** — every tenant-owned table has `ENABLE` + `FORCE ROW LEVEL
   SECURITY`; policies read `tenant_id` and `owner_user_id` only. Never write a tenant table query
   that relies on an application-supplied `WHERE tenant_id = $1` alone. `owner_user_id` is a bare
   `uuid` with **no foreign key**: the principal it names lives in the issuer's directory, not in
   this database, so a user-ownership policy can never be backed by a `users` table here.
2. **The RLS policy never reads a label** — referencing `scope_id` (or joining a label table) inside
   a policy is a defect: it turns a retrieval concept into an authorization decision.
3. **Fail closed on a missing GUC** — read it as
   `current_setting('app.tenant_id', true)` wrapped in `NULLIF(..., '')`, so an unset tenant yields
   zero rows instead of an error or a full table.
4. **`set_config(..., is_local := true)` requires a transaction** — autocommit discards the setting,
   so every tenant query runs inside the tenant-transaction helper. Never acquire a connection for
   tenant SQL any other way.
5. **Scopes are a filter, not a wall** — a scope predicate narrows a search by joining the label
   table and is optional; a tenant-wide query across scopes is legitimate and stays tenant-bound by
   rule 1.
6. **Parameterized SQL only** — `$1, $2, ...`. No `fmt.Sprintf` into query text, ever (SQLi).
7. **Context timeouts** — every repository call takes a `context.Context` and runs under
   `context.WithTimeout` (5–8s for reads, longer only for bulk ingestion writes) to avoid 504s and
   connection-pool exhaustion.
8. **Bound vector scans** — vchord (the elected dense engine; pgvector is present only for the
   `vector(N)` type) accelerates `ORDER BY embedding <=> $1 LIMIT k`. A distance comparison in the
   `WHERE` clause is a post-scan filter and degrades to a full sequential scan — never write one. The
   sanctioned bound is the index-side similarity filter, `WHERE embedding <<=>> sphere($1::vector,
   $2)`, which becomes an `Index Cond`; the `ORDER BY ... LIMIT k` stays either way and an untrusted
   radius is clamped in Go first (`.github/instructions/vchord.instructions.md`). Dense search goes
   through the single helper in `internal/pg`, which also rejects any vector whose length is not
   `EMBEDDING_DIM` before the SQL is built, and applies the search GUCs with `SET LOCAL` inside the
   tenant transaction.
9. **Measure recall under the real policy** — RLS is a row filter over the index scan, so a
   selective tenant/owner predicate can return fewer than k rows. Recall is verified by the
   evaluation harness with policies active, never assumed.
10. **Idempotent upserts** — `ON CONFLICT ... DO UPDATE` plus `COALESCE(NULLIF($n, ''), col)` so
    empty values never overwrite rich data.
11. **NULL is distinct in a unique index** — a composite `UNIQUE` over a nullable column does not
    deduplicate. Use `WHERE ... IS NOT NULL` partial unique indexes instead.
12. **Schema changes are migrations** — new files in `migrations/`, ordered, immutable once merged.
    Never issue DDL from application code, never `IF NOT EXISTS` your way around a missing migration.
13. **Cascade cleanup is explicit** — deleting a source, node or suggestion must delete every
    dependent row in the same transaction (`edges` referencing a deleted node, `attestations`
    referencing a deleted chunk, label rows for principals that no longer exist). Never leave
    dangling references.
14. **Merge and split happen in SQL with constraints as the referee** — re-pointing edges must
    delete self-loops and collapse duplicates before updating, so no unique constraint collides
    mid-transaction.
15. **Vectors cross the boundary as one helper** — `[]float32` is formatted to pgvector's text
    literal in exactly one place (`internal/pg/vector.go`), never per query. That helper is also
    the single place the width is checked: Matryoshka truncation happens in `internal/llm`, and a
    search query and the stored rows must share `EMBEDDING_DIM` or the distance is meaningless.
16. **Explain what you index** — every new index is added together with the query it serves, in the
    same commit; an index without a consumer is a cost with no benefit. The dense engine is
    **elected (vchord), not measured per corpus**; what the harness measures is the tuning of
    `VCHORD_PROBES`/`VCHORD_EPSILON`/`VCHORD_BUILD_LISTS` and `BM25_LIMIT` — the engine's real
    parameters, not invented ones — and whether the layout holds recall under RLS. Never add a
    parallel index mechanism beside the elected one without a recorded decision in
    `docs/BUILD_PLAN.md` §5. Engine API, options, GUCs and ceiling behaviour:
    `.github/instructions/vchord.instructions.md`.

## Testing SQL changes

Use `make test-int` against a live database (`DATABASE_URL`); it is skipped when the DB is absent.
Offline unit tests must never require PostgreSQL. Every schema change ships with: a migration
up/down pair, an integration test exercising the new column/constraint, and a tenant-isolation test
proving the row is invisible from another tenant.
