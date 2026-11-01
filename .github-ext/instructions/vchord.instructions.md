---
description: 'VectorChord engine rules: vchordrq / vchord_bm25 / pg_tokenizer operators, index options, GUCs and their ceilings'
applyTo: '**/*.sql,**/*.go,migrations/**,docker-compose*.yml,Dockerfile'
---

# VectorChord Instructions

Dense and sparse search are VectorChord's, shipped by `tensorchord/vchord-suite`. This file is the
API contract for those extensions: which operator, which index option, which GUC, and the ceiling
that silently truncates results when it is ignored. `sql.instructions.md` owns the surrounding
policy (RLS, parameterization, one-helper rule); this file owns the engine.

Reference material — full parameter tables, operator-class matrix, tuning procedure — is
`docs/VCHORD.md`. Read it before changing an index option; do not invent an option name from memory.

## What the suite gives you

| Extension | Schema | Provides | Used for |
| :--- | :--- | :--- | :--- |
| `vchord` | `public` | `vchordrq` (IVF) and `vchordg` (graph) index types, `rabitq8`/`rabitq4` types, `<<->>` `<<#>>` `<<=>>` `@#` | `chunks.embedding`, `nodes.embedding` |
| `vector` (pgvector) | `public` | the `vector(N)` **type** | column type only — we create no pgvector index |
| `vchord_bm25` | `bm25_catalog` | `bm25vector` type, `bm25` index method, `<&>`, `to_bm25query()` | sparse search over `chunks.lexical` |
| `pg_tokenizer` | `tokenizer_catalog` | `tokenize()`, `create_tokenizer()`, `create_text_analyzer()`, `add_preload_model()` | producing `bm25vector` at write time |

## Mandatory rules

1. **Engine objects resolve by schema, not by luck.** `vchord_bm25` and `pg_tokenizer` install into
   `bm25_catalog` and `tokenizer_catalog`; the operator `<&>`, the opclass `bm25_ops` and
   `to_bm25query()` are only found if that schema is on the session `search_path`. Set it in the pool
   configuration (`pgx` runtime params / DSN `options`) or schema-qualify every call — decide once,
   in one place. A migration that "works in psql" and fails in the app is almost always this.
2. **Write the index options out; never ship the extension's defaults.** `CREATE INDEX ... USING
   vchordrq (embedding vector_cosine_ops) WITH (options = $$ ... $$)`, where every value comes from
   the knob registry. Defaults moved between releases (`build.internal.lists` was `[1000]` before
   v0.3.0 and `[]` since), and an index whose shape is not written down cannot be rebuilt identically
   after a restore.
3. **`build.internal.lists` is sized against the row count.** `[]` means "no partitioning" and is
   correct below ~100k rows. For a single-level list the docs require `lists >= 4 * N` (N = rows):
   fewer partitions than that makes a probe cost more than it saves. Use two levels (`[a, b]`) only
   when a corpus outgrows one.
4. **Cosine models set `spherical_centroids = true` and `residual_quantization = true`.** Both default
   to `false`; the vendor's tuning guidance enables both for cosine. `residual_quantization` is
   unsupported on RaBitQ indexes — pick one, never both, or the build fails.
5. **`vchordrq.probes` must have the same shape as `lists`.** `lists = []` ⇒ probes empty;
   `lists = [11, 22]` ⇒ `2,4` or `4,8`; `3`, `7,8,9`, `5,5,5,5` are errors, not clamped values.
   Never carry a scalar probe count over to a two-level index.
6. **Query GUCs are `SET LOCAL`, inside the tenant transaction.** `vchordrq.probes`,
   `vchordrq.epsilon`, `vchordrq.max_scan_tuples` and `bm25_catalog.bm25_limit` are session settings
   and pooled connections are reused, so a plain `SET` leaks the value into somebody else's request —
   and into a tenant whose configuration differs. `SET LOCAL` is transaction-scoped and dies with it.
7. **One surface for search parameters: the index, or the GUC, never both.** Since v1.1.0 `probes`
   and `epsilon` can be stored on the index (`WITH (options = $$...$$, probes = '10')`,
   `ALTER INDEX ... SET (probes = '100')`). A GUC *silently overrides* the index value, so mixing the
   two produces a query plan nobody can explain from either source.
8. **Never put a distance comparison in the `WHERE` of a k-NN query.** `WHERE embedding <=> $1 < 0.15
   ORDER BY ... LIMIT 10` is a `Filter`, evaluated after the index scan, so PostgreSQL keeps scanning
   until it accumulates 10 rows inside the radius — the pathological case, and the one that turns a
   fast query into a table scan. The sanctioned push-down is the **similarity filter**:
   `WHERE embedding <<=>> sphere($1::vector, 0.15)` (L2 `<<->>`, inner product `<<#>>`), which becomes
   an `Index Cond` and prunes inside the index. The `ORDER BY ... LIMIT k` stays either way: the
   filter bounds *which* rows qualify, the `LIMIT` bounds *how many* are scanned. A filter arriving
   from a request body is clamped in Go first — the push-down changes where the bound is applied, not
   who is allowed to set it.
9. **`vchordrq.epsilon` is `1.9`, and that is deliberately pessimistic.** Domain `[0.0, 4.0]`: the
   lower bound on RaBitQ distances gets tighter as it rises, so accuracy rises and speed falls. The
   docs note the default "tends to favor higher accuracy than needed", which is a licence to lower it
   *after a measurement*, never a reason to hardcode a smaller number.
10. **`vchordrq.max_scan_tuples` is the low-selectivity remedy; `LIMIT` is not.** `LIMIT` applies
    *after* the `WHERE`, `max_scan_tuples` *before*, so it is the only way to cap the work a selective
    predicate can cause. `-1` (default) means unbounded. Set it on the query that needs it.
11. **`vchordrq.prewarm_dim` does not exist any more** (removed v0.4.0) and
    `vchordrq.enable_scan = off` is a testing switch, not a knob. Copying either from a pgvecto.rs
    note produces a setting that is ignored or an error at startup.
12. **RaBitQ is opt-in and has no implicit casts.** `rabitq8_*`/`rabitq4_*` opclasses and
    `rabitq8`/`rabitq4` columns cost ~1/4 and ~1/8 of a `vector` and are lossy; conversions are
    explicit (`quantize_to_rabitq8()`, `dequantize_to_vector()`), so a query mixing them with
    `vector` fails to plan rather than silently comparing apples to bitmaps. Adopting one is a
    measured decision and a migration (the column type is schema-bound) — never a code tweak.
13. **`vchordg` and maxsim are known, available, and not enabled.** Graph indexing and multi-vector
    (`@#`) retrieval are capabilities we deliberately do not use in v1; adding either is a recorded
    decision in `docs/BUILD_PLAN.md` §5 with the harness numbers behind it, not a per-query choice.
14. **A text column has no `bm25` index.** Sparse search runs over a `bm25vector` column that
    `pg_tokenizer` produced at write time (`tokenize(text, 'mkgs_text')`), written in the same
    statement as the row. An un-tokenized column — `NULL`, or stale after an edit — is invisible to
    sparse search exactly as an un-embedded chunk is invisible to dense search, so the freshness
    sweeper repairs both. The index is `USING bm25 (lexical bm25_ops)`, and `tokenize()` is not
    immutable, so a functional index over `chunks.text` is not an option.
15. **BM25 scores are negative on purpose.** `embedding <&> to_bm25query('idx_name', tokenize(q, 't'))`
    returns a negative score where *more negative is more relevant*, which is why plain ascending
    `ORDER BY rank` ranks correctly. Do not negate it, `abs()` it, or rescale it to a maximum: that
    inverts the ranking and looks like a bad tokenizer.
16. **`bm25_catalog.bm25_limit` (default `100`) caps results before PostgreSQL sees them.** A sparse
    query with `LIMIT > 100`, or a `WHERE` selective enough that fewer than `LIMIT` rows survive
    filtering, returns fewer rows than asked — no error, no warning. Raise it with `SET LOCAL` on the
    query that needs more (`-1` = brute force, correct and unindexed).
17. **Tokenizers are database objects with a memory cost.** `create_tokenizer()` runs in a migration
    (idempotent, versioned with the schema), never per request. A model that is not preloaded is
    loaded per connection on first use — 100–200 MB and a latency spike on the first query after a
    restart — so `add_preload_model()` plus a restart is the shipped configuration and the preloaded
    set is documented in `docs/OPERATIONS.md`.
18. **Only English is validated upstream; our corpora are not English.** An English BERT vocabulary
    over Italian legal/technical text is a recall decision, and the harness (Epic 7) is what decides
    it. Prefer a `unicode` pre-tokenizer with a custom analyzer (lowercase, `nfkd`, alphanumeric
    filter) unless a measurement says otherwise, and never treat the tokenizer as a translation
    layer — cross-lingual discoverability comes from `metadata.aliases` (§5), not from BM25.
19. **Verify the version; do not trust the floating tag.** `pg18-latest` is a moving target and every
    option above is version-dependent (`lists` default v0.3.0, fallback parameters and `rabitq*` types
    v1.1.0, `prewarm_dim` removed v0.4.0). An integration test asserts `pg_extension.extversion` for
    `vchord`, `vchord_bm25` and `pg_tokenizer`, and a production deployment pins a dated tag.

## Defect pairs

| Never write | Write instead | Because |
| :--- | :--- | :--- |
| `WHERE embedding <=> $1 < 0.15` | `WHERE embedding <<=>> sphere($1::vector, 0.15)` | the comparison is a post-scan `Filter`; `sphere` is an `Index Cond` |
| `SET vchordrq.probes = '10'` on a pooled connection | `SET LOCAL` in the tenant transaction | the value leaks to the next borrower |
| `probes='7,8,9'` with `lists=[11,22]` | `probes='4,8'` | shape must match `lists`, not merely count |
| `ORDER BY rank DESC` | `ORDER BY rank` | BM25 scores are negative: descending returns the *least* relevant first |
| `... LIMIT 500` over a `bm25` index | `SET LOCAL bm25_catalog.bm25_limit = '500'` first | the index returns at most `bm25_limit` rows |
| `CREATE INDEX ... USING hnsw` | `USING vchordrq` | pgvector ships the type; vchord is the elected engine (`sql.instructions.md` rule 16) |
| `WHERE lexical IS NOT NULL` to "fix" sparse search | tokenize at write time, repair with the sweeper | a `NULL` sparse vector is silent data loss, not a query problem |

## Changing a default

A `VCHORD_*` or `BM25_LIMIT` value is a corpus-dependent measurement, not a preference. Changing
one means: run Epic 7 with RLS active, record the recall/latency numbers in `docs/BUILD_PLAN.md` §5,
update `docs/CONFIGURATION.md`, and note the change under `## [Unreleased]`. Build-time options
(`VCHORD_BUILD_LISTS`, `spherical_centroids`, `residual_quantization`) only take effect on the next
index build — a `REINDEX` is part of the change, and a migration that forgets it ships the old shape
under the new number.

## Upstream

- Suite (what ships in the image): https://docs.vectorchord.ai/vectorchord/getting-started/vectorchord-suite.html
- Indexing, operator classes and every option/GUC above: https://docs.vectorchord.ai/vectorchord/usage/indexing.html
- Similarity filter (`sphere`, `<<->>`): https://docs.vectorchord.ai/vectorchord/usage/range-query.html
- Fallback parameters (index storage values): https://docs.vectorchord.ai/vectorchord/usage/fallback-parameters.html
- RaBitQ column types: https://docs.vectorchord.ai/vectorchord/usage/quantization-types.html
- Sparse engine: https://github.com/supervc-stack/VectorChord-bm25
- Dense engine: https://github.com/supervc-stack/VectorChord
- Licence: dual **AGPLv3 / Elastic License v2**. Distribution is a product decision — record it, do
  not assume it away.
