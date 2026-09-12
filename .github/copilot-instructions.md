# biomuzak

A self-hosted music organizer, player, and streamer. Users upload audio files; the
server extracts metadata, stores files in MinIO, and serves a REST API plus a
Subsonic-compatible API for playback. A Python microservice generates 38-dim audio
feature embeddings used for "similar songs."

## Tech stack

- **Backend**: Go 1.24, `chi` router, `pgx` (Postgres), `golang-jwt`, `dhowden/tag`,
  `minio-go`. Module name is `go-postgres-example` (historical — keep it).
- **Database**: PostgreSQL with `pgvector`/`vchord` for embeddings.
- **Audio processor**: Python FastAPI + Essentia (`audio-processor/`).
- **Frontend**: React + Vite (`frontend/`).
- **Testing**: Go `testify` + `go-sqlmock`; Python `pytest`.

## Project structure

- `cmd/server/main.go` — entry point (config, DB, storage, migrations, server).
- `pkg/handlers/` — HTTP handlers (auth, upload, library, playlist, song, stream).
- `pkg/db/` — SQL queries and migrations runner.
- `pkg/metadata/` — file hashing, tag extraction, genre detection, embedding calls.
- `pkg/subsonic/` — Subsonic API layer (`/rest/*`).
- `pkg/middleware/` — JWT auth, admin check, CORS.
- `pkg/storage/` — MinIO object storage.
- `pkg/models/`, `pkg/config/`, `pkg/auth/`, `pkg/musicbrainz/`.
- `db/migrations/` — SQL migrations, applied in filename order on startup.
- `audio-processor/` — Python embedding service.

## Guidelines

- **Security**: parameterized queries only; validate/sanitize all input (filenames,
  uploads); scope data access to the requesting user; never log secrets.
- **Errors**: check and handle every error explicitly; don't leak internals to clients.
- **Tests**: co-located `*_test.go`; table-driven where sensible; must pass before done.
- **Migrations**: add a new numbered file in `db/migrations/`; never edit applied ones.
- **Changelog**: after completing a task, add an entry to `docs/CHANGELOG.md`
  (Keep a Changelog format) under `[Unreleased]`.

## Resources

- Run tests: `go test ./...` (Go), `pytest` (audio-processor).
- Build: `go build ./...`; vet: `go vet ./...`.
- Local run: configure `.env` (see `.env.example`), `docker compose up audio-processor`,
  then `go run cmd/server/main.go`.
- Full stack: `docker compose up`.
- Key env: `DATABASE_URL`, `JWT_SECRET` (required, non-default), `PORT`,
  `AUDIO_PROCESSOR_URL`, `MINIO_*`, `CORS_ALLOWED_ORIGINS`.
