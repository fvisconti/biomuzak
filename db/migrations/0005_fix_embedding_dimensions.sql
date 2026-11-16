-- Fix embedding dimensions from 512 to 38 to match audio processor output
-- This drops and recreates the table since PostgreSQL vector extension doesn't support ALTER COLUMN for vector types

-- Drop the existing table and recreate with correct dimensions
DROP TABLE IF EXISTS song_embeddings CASCADE;

CREATE TABLE song_embeddings (
    song_id INTEGER PRIMARY KEY REFERENCES songs(id) ON DELETE CASCADE,
    embedding VECTOR(38)
);

-- Recreate the VectorChord index for similarity search
CREATE INDEX ON song_embeddings USING vchordrq (embedding vector_cosine_ops);
