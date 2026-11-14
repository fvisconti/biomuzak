-- E3: index-backed trigram genre matching.
-- Serves the `%` similarity filter in FindGenreByTrigramSearch
-- (pkg/db/library.go): SELECT name FROM genres WHERE name % $1.
-- pg_trgm is already enabled in 0003; gin_trgm_ops accelerates `%` / `<->`.
CREATE INDEX idx_genres_name_trgm ON genres USING gin (name gin_trgm_ops);
