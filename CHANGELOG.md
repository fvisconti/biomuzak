# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
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

### Changed
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
