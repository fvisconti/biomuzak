# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- `docs/PLAN.md`: a living development plan for the Tidal-first feature set
  (global catalog visibility, similar-track suggestions, genre catalogue +
  playlist-name matching, per-genre MinIO buckets + object tagging, Tidal
  link-paste download, sortable playlists, and Subsonic API completeness).
  Developer-process only; no runtime behaviour change.
- **Tidal integration** (`pkg/tidal`, `pkg/handlers/tidal.go`, `frontend/src/pages/Tidal.jsx`):
  per-user Tidal account pairing via the OAuth2 device flow, library browsing
  (tracks / albums / playlists), and batch import of selected tracks into the
  biomuzak library. Downloads are parallelized (configurable concurrency, 1–8)
  with a random human-like pause between each download. Audio is fetched from
  Tidal's stream endpoint, AES-128-CBC decrypted, deduplicated by SHA-256, and
  stored in MinIO. Import jobs run in the background with live per-track status.
- **Export to local folder** (`pkg/export`, `pkg/handlers/export.go`): copy any
  selected library songs out of MinIO to a per-user, GUI-configured absolute
  path on the server, using a tiddl-style `{artist}/{album}/{title}` template.
  Existing files are skipped; FLAC exports get Vorbis Comment tags embedded.
  Path-traversal is guarded so output always stays under the export path.
- `TIDAL_CLIENT_ID` / `TIDAL_CLIENT_SECRET` config (optional; the Tidal section
  is disabled and shows a "not configured" notice when unset).
- `tidal_credentials` table (migration `0006`) storing per-user Tidal tokens and
  import/export config.
- Frontend: new **Tidal** page (pairing card, settings panel, browse/import with
  live progress) and an **Export selected** action on the Library page.

- `CORS_ALLOWED_ORIGINS` config: CORS now honors an origin allow-list (reflects a
  matching `Origin`) instead of always sending `Access-Control-Allow-Origin: *`.
  Falls back to `*` only when unset (dev convenience).
- `StorageService.DeleteObject` for removing objects from MinIO.
- Server timeouts (`ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout`, `IdleTimeout`)
  to mitigate Slowloris / connection-exhaustion.
- Bounded concurrency for background upload processing (semaphore, max 4).
- HTTP client timeout (5 min) for calls to the audio processor.
- **Similar tracks** (E2): the player bar now shows a "Similar" panel with the
  top similar tracks for the currently playing song (click-to-play). A song with
  no embedding (or no matches) shows a graceful empty state, not an error.
- **Embedding backfill** (E2): admin endpoints `POST /api/admin/embeddings/backfill`
  and `GET /api/admin/embeddings/backfill/status` generate embeddings for every
  song that lacks one (pre-existing library, Tidal imports, past processor
  failures) as a background job, so "similar songs" works across the whole catalog.
  The job is single-flight (a second start while one runs returns `409`), is
  cancellable (a shutdown signal stops the worker goroutine), and finished jobs
  are evicted from the in-memory store after a 1-hour retention TTL.
- New offline test suites (E2 review): `pkg/config` (env resolution,
  table-driven), `pkg/metadata` (`GenerateEmbeddingForSong` happy/failure paths
  via `httptest` + `go-sqlmock`), and `pkg/handlers` backfill tests for the
  admin gate (`AdminOnly` 403), status happy path, per-song Done/Failed
  accounting, and the single-flight 409.
- Security-review test suites: upload playlist-ownership rejection
  (`pkg/handlers/upload_playlist_test.go`), migration renumbering
  (`pkg/db/db_test.go`), export-job owner scoping
  (`pkg/export/export_owner_test.go`, `pkg/handlers/export_status_test.go`),
  MinIO credential defaults (`pkg/config/config_test.go`), and internal-field
  JSON redaction (`pkg/models/models_test.go`).

### Changed
- **Migration renumbering repair**: `db.MigrateAll` rewrites legacy
  `schema_migrations` rows whose filenames were renumbered when
  `0002_add_playlist_genre` was inserted (old `0002`-`0004` → `0003`-`0005`),
  so an applied database never re-executes a renamed file — notably the
  destructive `0005_fix_embedding_dimensions` (`DROP TABLE song_embeddings`).
  Migration files themselves remain immutable.
- **MinIO secrets have no defaults**: `MINIO_ACCESS_KEY` / `MINIO_SECRET_KEY`
  no longer fall back to `minioadmin`; the server fails fast at boot when they
  are unset (directive 3). `.env.example` and `README.md` document them as
  required. The compose bootstrap no longer marks the MinIO bucket
  world-readable (`mc anonymous set public` removed) — audio is served through
  the authenticated stream/download endpoints.
- **Embedding dimension guard**: `db.SaveSongEmbedding` and `db.FindSimilarSongs`
  now reject vectors whose width is not exactly 38 in Go, before building any
  SQL (`db.EmbeddingDim` is the single source of the contract).
- `metadata.NewProcessor` returns an error instead of calling `log.Fatalf`, so
  a MusicBrainz client failure on an upload or backfill request returns a
  sanitized 500 rather than killing the server process.
- Finished export and Tidal-import jobs are evicted from their in-memory
  stores after a 1-hour retention TTL (previously they leaked for the process
  lifetime).
- The server now shuts down gracefully on SIGINT/SIGTERM: in-flight requests
  drain within a 30-second timeout and any running backfill job is cancelled.
- `config.DefaultSimilarSongsLimit` is the single definition of the default
  similar-songs count (used by `New()` and the handler clamp).
- `docker-compose.yml` passes `SIMILAR_SONGS_LIMIT` (default 3) and
  `EXPORT_CONCURRENCY` (default 4) through to the backend container.
- **Similar songs limit** (E2): the number of similar tracks returned by
  `GET /api/songs/{id}/similar` is now configurable via `SIMILAR_SONGS_LIMIT`
  (default 3) instead of a hardcoded value.
- **Shared catalog** (E1): the song catalog is now global — any authenticated user
  can list, stream, and download any track. Library listing, streaming, and
  download are no longer gated on per-user `user_songs` membership; a user's
  rating is still shown via a `LEFT JOIN`. Playlists and ratings remain per-user.
  Deleting a user no longer garbage-collects shared songs (only their ratings
  and playlists are removed).
- **Security hardening** (E1 follow-up): genre editing (`PUT /api/songs/{id}/genre`)
  is now admin-only to prevent a compromised account from corrupting shared
  metadata. Added `DELETE /api/admin/songs/{songID}` for admins to purge a song
  from the catalog (DB row + MinIO object), addressing monotonic storage growth.
- **Export performance** (E1 follow-up): song resolution in the export path is
  now a single batched `WHERE id = ANY($1)` query instead of one query per song
  (N+1 eliminated). The export worker uses a bounded worker pool
  (`EXPORT_CONCURRENCY`, default 4) so independent MinIO reads + disk writes run
  in parallel. `DELETE /api/admin/songs/{id}` now deletes the DB row and the
  storage object concurrently.
- Rewrote the `.github` Copilot configuration (agents, instructions, skills,
  PR template, CI) to a modernized layout, and added a repo-root `REVIEW.md`
  capturing the project's hard invariants. Added dedicated instruction files
  and scoped review skills (`/audio-processor-review`, `/frontend-review`) for
  the audio-processor (Essentia/FastAPI) and the frontend (React/Vite/Chakra),
  plus a VectorChord docs-grounding section. Developer-process only; no
  runtime behaviour change.
- Subsonic password auth now verifies the password against the stored bcrypt hash.
- Subsonic token auth fails closed (not supported with one-way bcrypt storage).
- `GetAllArtists` and `Search` aggregate from the `songs` table (the `artists`/
  `albums` tables do not exist in the schema).
- Audio processor (`audio-processor/audio_processor/main.py`) rewritten to use
  `es.Extractor`, producing the 38-dim embedding the DB column expects; rejects
  non-audio uploads with 400; uses a unique temp file per request.
- Auth middleware requires the `Bearer ` prefix when an `Authorization` header is
  present (query-param `?token=` fallback retained for audio streaming).

### Fixed
- **Security:** IDOR on upload — `POST /api/upload` accepted an arbitrary
  `playlist_id` from the form body without an ownership check, letting a user
  add songs to (and learn the existence of) another user's playlist. The ID is
  now parsed strictly and verified against the JWT user
  (`WHERE id = $1 AND user_id = $2`): foreign or unknown ids → 404, malformed
  → 400, DB failure → sanitized 500.
- **Security:** Export job status (`GET /api/export/status?jobId=...`) returned
  any user's job snapshot to any authenticated user who knew or guessed the
  id. Jobs now record their owner (from the JWT at start) and the status route
  answers 403 for non-owners, mirroring the Tidal import-status route.
- **Security:** Song responses leaked the absolute MinIO object path
  (`file_path`) and the content-derived `fingerprint_hash` to every client;
  both are server-internal and now serialize as `json:"-"` on `models.Song`
  and `models.LibrarySong` (no frontend or Subsonic consumer uses them).
- Docs: `README.md` documented the similar-songs response as
  `{"similar_songs": [...]}`; the endpoint actually returns a bare JSON array.
  `docs/USER_GUIDE.md` still described a "••• → Find Similar Songs" flow that no
  longer exists (the feature lives in the player bar). Both corrected, and the
  three review docs/agents citations of the deleted
  `code-review-generic.instructions.md` now point at
  `code-review-biomuzak.instructions.md`.
- Frontend `npm run lint` crashed on load: `eslint.config.js` referenced  `reactHooks.configs.flat.recommended`, which does not exist in the installed
  `eslint-plugin-react-hooks` v5. Now uses the flat-compatible
  `configs['recommended-latest']`.
- **Security:** Subsonic auth bypass — any non-empty password previously
  authenticated successfully.
- **Security:** Path traversal in upload temp-file writer (now sanitized via
  `filepath.Base`).
- **Security:** IDOR on stream/download — any authenticated user could access any
  other user's uploads (now scoped to the requesting user).
- **Security:** Server refuses to start with the default/empty `JWT_SECRET`.
- `GetIndexes` panic on empty artist names.
- Race conditions in genre creation (`findOrCreateGenre`, `UpdateSongGenre`) and
  user registration (unique-violation now returns 409).
- Embedding dimension mismatch (processor now emits 38 dims, matching `VECTOR(38)`).
- Orphaned songs and their MinIO objects are garbage-collected on user deletion.
