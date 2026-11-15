-- Switch song embeddings from the hand-rolled 38-dim MFCC/spectral-contrast
-- vector to Essentia's MAEST v2 embedding (discogs-maest-10s-pw-2): 768 dims,
-- one per-token embedding averaged over the track's 10s windows and
-- L2-normalized. MAEST is loudness-robust by design and does its own feature
-- extraction, which the 38-dim hand-rolled pipeline could not provide.
--
-- pgvector cannot ALTER a column's vector width, so the table is dropped and
-- recreated (same pattern as migration 0005). The local database is empty at
-- the time of writing; any deployment with 38-dim rows must re-run the
-- embedding backfill (POST /api/admin/embeddings/backfill) after this
-- migration.

DROP TABLE IF EXISTS song_embeddings CASCADE;

CREATE TABLE song_embeddings (
    song_id INTEGER PRIMARY KEY REFERENCES songs(id) ON DELETE CASCADE,
    embedding VECTOR(768)
);

-- Recreate the VectorChord index for similarity search. Options are written
-- out explicitly (vchord.instructions.md rule 1 — never ship extension
-- defaults): the catalog is far below ~100k rows, so no IVF partitioning
-- (build.internal.lists = [], rule 2); cosine requires both
-- spherical_centroids and residual_quantization (rule 3). Empty lists keep
-- probes empty (rule 4). Note: 0005 shipped the bare form; this fresh build
-- is the moment the options get written down.
CREATE INDEX ON song_embeddings USING vchordrq (embedding vector_cosine_ops)
WITH (
    options = $$
    {
        "build.internal.lists": [],
        "spherical_centroids": true,
        "residual_quantization": true
    }
    $$
);
