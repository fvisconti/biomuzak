---
description: 'VectorChord engine rules for this repository: vchordrq dense index on 38-dim cosine song embeddings'
applyTo: '**/*.sql,**/*.go,db/migrations/**,docker-compose*.yml,Dockerfile'
---

# VectorChord Instructions

Dense similarity search is VectorChord's, shipped by `tensorchord/vchord-suite`. This
file is the API contract for the `vchord` extension as used here: which operator, which
index option, which GUC, and the ceiling that silently truncates results when it is
ignored. `sql.instructions.md` owns the surrounding policy (per-user scoping,
parameterization, one-helper rule); this file owns the engine.

This repository uses **dense search only** — a `vchordrq` index over the 38-dim cosine
embeddings in `song_embeddings.embedding`. There is no sparse (BM25) search, no
`pg_tokenizer`, and no RaBitQ quantization in this project.

## Grounding — the official docs are the source of truth

This file is a **summary**, not the reference. Before changing an index option, GUC,
operator, or the engine version, read the official VectorChord documentation and ground
the decision there — do not invent an option name, default, or ceiling from memory. The
options and GUCs above are version-dependent and have moved between releases; the docs
are the only thing that is always current.

- **Start here (the suite we ship):**
  https://docs.vectorchord.ai/vectorchord/getting-started/vectorchord-suite.html
- **Indexing, operator classes, and every option/GUC referenced in this file:**
  https://docs.vectorchord.ai/vectorchord/usage/indexing.html
- **Similarity filter (`sphere`, `<<->>`, `<<#>>`):**
  https://docs.vectorchord.ai/vectorchord/usage/range-query.html
- **Fallback parameters (index-stored `probes`/`epsilon`):**
  https://docs.vectorchord.ai/vectorchord/usage/fallback-parameters.html

When a decision here and the docs disagree, the docs win: update this file and
`docs/DOCKER.md` in the same commit, and note the change under `## [Unreleased]` in
`docs/CHANGELOG.md`.

## What the suite gives you (as used here)

| Extension | Schema | Provides | Used for |
| :--- | :--- | :--- | :--- |
| `vchord` | `public` | `vchordrq` (IVF) index type, `<<->>` `<<#>>` `<<=>>` operators | `song_embeddings.embedding` |
| `vector` (pgvector) | `public` | the `vector(N)` **type** | column type only — we create no pgvector index |

## Mandatory rules

1. **Write the index options out; never ship the extension's defaults.** `CREATE INDEX
   ... USING vchordrq (embedding vector_cosine_ops) WITH (options = $$ ... $$)`, where
   every value comes from a documented default. Defaults moved between releases, and an
   index whose shape is not written down cannot be rebuilt identically after a restore.
2. **`build.internal.lists` is sized against the row count.** `[]` means "no
   partitioning" and is correct below ~100k rows. For a single-level list the docs
   require `lists >= 4 * N` (N = rows): fewer partitions than that makes a probe cost
   more than it saves. Use two levels (`[a, b]`) only when a corpus outgrows one.
3. **Cosine models set `spherical_centroids = true` and `residual_quantization =
   true`.** Both default to `false`; the vendor's tuning guidance enables both for
   cosine. `residual_quantization` is unsupported on RaBitQ indexes — pick one, never
   both, or the build fails.
4. **`vchordrq.probes` must have the same shape as `lists`.** `lists = []` ⇒ probes
   empty; `lists = [11, 22]` ⇒ `2,4` or `4,8`; `3`, `7,8,9`, `5,5,5,5` are errors, not
   clamped values. Never carry a scalar probe count over to a two-level index.
5. **Query GUCs are `SET LOCAL`, inside a transaction.** `vchordrq.probes`,
   `vchordrq.epsilon` and `vchordrq.max_scan_tuples` are session settings and pooled
   connections are reused, so a plain `SET` leaks the value into somebody else's
   request. `SET LOCAL` is transaction-scoped and dies with it.
6. **One surface for search parameters: the index, or the GUC, never both.** Since
   v1.1.0 `probes` and `epsilon` can be stored on the index (`WITH (options =
   $$...$$, probes = '10')`, `ALTER INDEX ... SET (probes = '100')`). A GUC *silently
   overrides* the index value, so mixing the two produces a query plan nobody can
   explain from either source.
7. **Never put a distance comparison in the `WHERE` of a k-NN query.** `WHERE
   embedding <=> $1 < 0.15 ORDER BY ... LIMIT 10` is a `Filter`, evaluated after the
   index scan, so PostgreSQL keeps scanning until it accumulates 10 rows inside the
   radius — the pathological case, and the one that turns a fast query into a table
   scan. The sanctioned push-down is the **similarity filter**: `WHERE embedding <<=>>
   sphere($1::vector, 0.15)` (L2 `<<->>`, inner product `<<#>>`), which becomes an
   `Index Cond` and prunes inside the index. The `ORDER BY ... LIMIT k` stays either
   way: the filter bounds *which* rows qualify, the `LIMIT` bounds *how many* are
   scanned. A filter arriving from a request body is clamped in Go first — the
   push-down changes where the bound is applied, not who is allowed to set it.
8. **`vchordrq.epsilon` is `1.9`, and that is deliberately pessimistic.** Domain
   `[0.0, 4.0]`: the lower bound on distances gets tighter as it rises, so accuracy
   rises and speed falls. The docs note the default "tends to favor higher accuracy
   than needed", which is a licence to lower it *after a measurement*, never a reason
   to hardcode a smaller number.
9. **`vchordrq.max_scan_tuples` is the low-selectivity remedy; `LIMIT` is not.**
   `LIMIT` applies *after* the `WHERE`, `max_scan_tuples` *before*, so it is the only
   way to cap the work a selective predicate can cause. `-1` (default) means
   unbounded. Set it on the query that needs it.
10. **The embedding width is 38.** A vector whose length is not 38 is rejected before
    SQL is built (in `pkg/db/similar.go`); a search query and the stored rows must
    share the width or the distance is meaningless.
11. **Verify the version; do not trust the floating tag.** `pg18-latest` is a moving
    target and every option above is version-dependent. A production deployment pins a
    dated tag.

## Defect pairs

| Never write | Write instead | Because |
| :--- | :--- | :--- |
| `WHERE embedding <=> $1 < 0.15` | `WHERE embedding <<=>> sphere($1::vector, 0.15)` | the comparison is a post-scan `Filter`; `sphere` is an `Index Cond` |
| `SET vchordrq.probes = '10'` on a pooled connection | `SET LOCAL` in a transaction | the value leaks to the next borrower |
| `probes='7,8,9'` with `lists=[11,22]` | `probes='4,8'` | shape must match `lists`, not merely count |
| `CREATE INDEX ... USING hnsw` | `USING vchordrq` | pgvector ships the type; vchord is the elected engine (`sql.instructions.md`) |

## Changing a default

A `VCHORD_*` value is a corpus-dependent measurement, not a preference. Changing one
means: record the recall/latency numbers, update `docs/DOCKER.md`, and note the change
under `## [Unreleased]` in `docs/CHANGELOG.md`. Build-time options
(`build.internal.lists`, `spherical_centroids`, `residual_quantization`) only take
effect on the next index build — a `REINDEX` is part of the change, and a migration
that forgets it ships the old shape under the new number.

## Upstream

- Suite (what ships in the image): https://docs.vectorchord.ai/vectorchord/getting-started/vectorchord-suite.html
- Indexing, operator classes and every option/GUC above: https://docs.vectorchord.ai/vectorchord/usage/indexing.html
- Similarity filter (`sphere`, `<<->>`): https://docs.vectorchord.ai/vectorchord/usage/range-query.html
- Fallback parameters (index storage values): https://docs.vectorchord.ai/vectorchord/usage/fallback-parameters.html
- Dense engine: https://github.com/supervc-stack/VectorChord
- Licence: dual **AGPLv3 / Elastic License v2**. Distribution is a product decision — record it, do not assume it away.
