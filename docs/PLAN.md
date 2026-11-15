# biomuzak — Feature Plan (Tidal-first music library)

> **Status**: Planning / in progress. This is a **living document** — update the
> status of each epic (⬜ not started · 🟨 in progress · ✅ done · ⛔ blocked) and
> the decision log as development proceeds.
>
> **Last updated**: 2026-11-14

## User story

As a user, I want this software to handle my music library. The focus is to easily
integrate Tidal downloads into playlist creation and handling. Specifically:

1. Paste a Tidal playlist/album/track link and hit a button to download.
2. Storage is a MinIO instance: a **bucket per genre** (highly populated); for
   albums/artists, explore cheaper methods than buckets (e.g. **labeling**) — a
   single track can be labelled by genre, artist, label.
3. A **catalogue of genres**: each downloaded playlist is matched to it by name,
   via fuzzy logic or substring methods (easy and computationally cheap).
4. **No track is owned** (in the sense of privateness): ANY user can see ANY track,
   but each user creates its own playlists.
5. Playlists must be **sortable via the web GUI**; default sort = **most recent
   tracks first**.
6. The backend must expose the **Subsonic API** (already in the README): it must be
   easy to connect any third-party client supporting Subsonic.
7. Music streaming works fine, but **suggestions do not**: show **3 similar tracks**
   with respect to the one playing.

## TL;DR

The codebase already has ~70% of the pieces: Tidal import + AES decrypt, a global
hash-deduped `songs` catalog, per-user playlists, a genre catalogue with a trigram
matcher, a similar-songs endpoint, and a partial Subsonic API. The work is mostly
**wiring, unscoping, and gap-filling**, not rewrites.

Eight epics, each independently shippable, ordered so later ones build on earlier
ones.

## Decision log

Decisions made during planning. **Override any of these** — each is flagged so it
can be revisited.

| # | Decision | Choice | Rationale |
|---|---|---|---|
| D1 | tiddl | Keep the working hand-rolled download; add link-paste on top. `tiddl` adoption is a separate optional epic (E8). | "Keep almost everything"; the decrypt path works. |
| D2 | Storage layout | ~~One bucket per genre~~ **Overridden in E4 (2026-11-14, user)**: single flat bucket + **MinIO object tags** for genre/artist. No re-key migration. | Simpler: dedup keys stay stable, no migration; tags are informational and cheap to refresh on genre change. |
| D3 | "Most recent" | = most recently **added to the playlist** (new `added_at` column). | Most intuitive for a playlist. |
| D4 | Subsonic scope | **Full** target (browse + playlists + token auth), delivered in 3 phases. | "Easy to connect any third-party client." |
| D5 | Global visibility | All tracks visible to all; **keep `user_songs` for ratings / "in my library" only**. | Preserves the existing ratings feature. |
| D6 | Security boundary | Global visibility **deliberately overrides** Prime Directive 1 for *catalog browsing*. New boundary: **catalog is global; playlists + ratings are per-user.** | Required by bullet 4. Docs updated in E1. |

## Epic map

| Epic | Bullet | One-line goal | Depends on | Status |
|---|---|---|---|---|
| **E1** Global catalog visibility | 4 | ANY user sees ANY track; playlists + ratings stay per-user | — | ✅ |
| **E2** Similar tracks for the playing song | 7 | Show 3 similar tracks w.r.t. the playing song | E1 | ✅ |
| **E3** Genre catalogue + playlist-name matching | 3 | Match each downloaded playlist to a genre by name (cheap) | — | ✅ |
| **E4** MinIO object tagging (single bucket) | 2 | Object tags for genre/artist; genre changes re-tag | E3 | ✅ |
| **E5** Tidal link-paste download | 1 | Paste a Tidal URL → download | E3 | ✅ |
| **E6** Sortable playlists, default most-recent | 5 | GUI-sortable playlists; default most-recent-first | — | ✅ |
| **E7** Subsonic API completeness | 6 | Any third-party Subsonic client can connect and use it | E1, E2 | ⬜ |
| **E8** (optional) Adopt tiddl | 1 | Replace hand-rolled fetch/decrypt with `tiddl` | only if D1 overridden | ⬜ |

**Suggested order**: E1 → E2 → E3 (quick, high value, low risk) → E6 → E5 → E4
(riskiest: storage layout + migration) → E7 → E8 last.

---

## E1 — Global catalog visibility

**Goal**: ANY user sees ANY track; playlists + ratings stay per-user.

**Current state**: `songs` is already a global, hash-deduped catalog (no owner
column). `user_songs` is a many-to-many "library" view. Library listing, streaming,
download, and genre-edit are gated on `user_songs` membership.

**Changes**:
- `pkg/db/library.go`: `GetSongsByUserID`, `GetAlbums`, `GetAlbumSongs` → query
  `songs` directly; keep `user_songs` as a `LEFT JOIN` so a user's rating still
  shows when present. `UpdateSongGenre` → drop the membership check.
- `pkg/handlers/stream.go`: use unscoped `GetSongFilePath` / `GetSongByID` (both
  already exist) instead of `...ForUser`.
- Reconsider `GetOrphanedSongsForUser` GC + user-deletion cleanup: deleting a user
  must NOT delete shared songs (only their `user_songs`/playlists rows).
- **Docs-sync (D6)**: update `REVIEW.md`, `.github/copilot-instructions.md`
  (Prime Directive 1), and `.github/instructions/code-review-biomuzak.instructions.md`
  to state the new boundary: global catalog, per-user playlists + ratings.

**Verification**: as user A, list/stream a song uploaded by user B; rating still
per-user; deleting user B leaves songs intact. `go test -race ./...` green.

**Status**: ✅ (2026-11-01) — unscoped `GetSongsByUserID`/`GetAlbums`/`GetAlbumSongs`
(LEFT JOIN for rating), `UpdateSongGenre` (no ownership check), stream/download +
Subsonic stream (unscoped), export (unscoped), removed song GC from `DeleteUser`,
dropped dead `GetSongFilePathForUser`/`GetSongByIDForUser`/`GetOrphanedSongsForUser`/
`DeleteSongs`. Docs updated (REVIEW.md, copilot-instructions.md, code-review
instructions). New tests: `pkg/db/library_test.go`, `pkg/handlers/stream_test.go`.

---

## E2 — Similar tracks for the playing song

**Goal**: show 3 similar tracks w.r.t. the currently playing song.

**Current state**: backend endpoint exists (`GET /api/songs/{id}/similar`, top-5,
unscoped, count hardcoded). Embedding flow works (upload → audio-processor →
`SaveSongEmbedding`). **The frontend is entirely missing.** No Subsonic endpoint.

**Changes**:
- `pkg/handlers/song.go` + `pkg/db/similar.go`: make top-N an env var
  (`SIMILAR_SONGS_LIMIT`, default 3) — removes the magic literal. (Unscoped is now
  *correct* under the global model.)
- Frontend: add `getSimilarSongs(songID)` to `src/api/client.js`; hook into
  `PlayerContext.jsx` on `loadAndPlay`; render a "Similar" panel (3 tracks,
  click-to-play) in `PlayerBar.jsx` or a side panel.
- **Cold-start**: add a backfill path (job or admin endpoint) to generate
  embeddings for songs that lack them (pre-existing library, Tidal imports,
  processor failures). Without this, many songs 404.
- Optional (pairs with E7): Subsonic `getSimilarSongs.view`.

**Verification**: play a song → 3 similar appear; click one → plays; a song with no
embedding shows a graceful empty state, not an error.

**Status**: ✅ (2026-11-01) — `SIMILAR_SONGS_LIMIT` env var (default 3) wired into
`GetSimilarSongsHandler`; frontend `getSimilarSongs` API + `PlayerContext` hook +
"Similar" panel in `PlayerBar` (click-to-play, graceful empty state); cold-start
backfill via admin job (`POST /api/admin/embeddings/backfill` + status) that
generates embeddings for all songs lacking them. Subsonic `getSimilarSongs.view`
deferred to E7.

---

## E3 — Genre catalogue + playlist-name matching

**Goal**: a genre catalogue; each downloaded playlist matched to a genre by name
(cheap fuzzy/substring).

**Current state**: `genres` table exists; `FindGenreByTrigramSearch` (pg_trgm,
threshold 0.3) exists but is only fed the upload *folder* name. No GIN index on
`genres.name`. Tidal import sets **no** genre.

**Changes**:
- New migration: `CREATE INDEX ... USING gin (genres.name gin_trgm_ops)` for cheap
  matching.
- Wire **playlist names** into `FindGenreByTrigramSearch` (new call site in the
  Tidal import / playlist flow); set the matched genre on the playlist and on
  imported songs.
- Give Tidal-imported songs a genre (currently none) — from the matched playlist
  genre, falling back to Tidal's own genre metadata if available.
- Provide a way to seed/manage the catalogue (admin endpoint + minimal UI, or a
  seed migration with a starter list).

**Verification**: import a playlist named e.g. "Deep House Mix" → matched to
"House"/"Deep House"; songs inherit the genre; matching is a single indexed query.

**Status**: ✅ (2026-11-14) — migration `0007` GIN `gin_trgm_ops` index on
`genres.name`; `FindGenreByTrigramSearch` switched to the index-backed `%`
operator (threshold = `pg_trgm.similarity_threshold` GUC, default 0.3,
inclusive — the old hardcoded predicate was strict `> 0.3`). Tidal import: `Importer.ApplySourceGenre` matches the
source (playlist/album) name once per import and stamps `TrackMeta.Genre`;
`saveTidalSong` resolves the matched genre by lookup and binds `genre_id`
(lookup only — a deleted genre is never re-created by an import); `AddToPlaylist`
sets the playlist genre via `db.SetPlaylistGenreIfEmpty` (never overwrites an
explicit genre). Client-supplied `genre` fields in the import body are
ignored (`TrackMeta.Genre` has no JSON tag). Catalogue management: seed migration `0008` (30 starter
genres, idempotent) + admin routes `GET/POST/DELETE /api/admin/genres`
(`middleware.AdminOnly` group; delete maps 0 rows → 404, duplicate create →
409). The Tidal API exposes no genre metadata on tracks, so the
"Tidal's own metadata" fallback does not apply. New tests: `pkg/db/genre_test.go`,
trigram cases in `pkg/db/library_test.go`, `pkg/tidal/import_test.go`,
`pkg/handlers/tidal_test.go`, `pkg/handlers/genre_test.go`.

---

## E4 — MinIO object tagging (single bucket)

**Goal**: object tags for genre/artist, keeping the single flat bucket.

**Design decision (2026-11-14, user)**: single flat bucket, **genre as a
MinIO object tag** — no per-genre buckets, no re-key migration. The hash key
is unchanged (dedup intact); tags are informational metadata for browsing/
filtering outside the app. Cost of a genre change: one `PutObjectTagging`
call (cheap, best-effort, never fails the genre update).

**Changes**:
- `pkg/storage/storage.go`: `StorageService` gains `TagObject` /
  `GetObjectTags` (MinIO `PutObjectTagging`/`GetObjectTagging`, v7.2.1
  `tags` package).
- Upload path (`pkg/metadata` `ProcessFile`) and Tidal import
  (`pkg/tidal/import.go` `ImportTrack`) tag the object with `genre` +
  `artist` right after the upload — genre is already resolved by then.
  Tag failures are logged, never fatal.
- Genre change (`PUT /api/songs/{id}/genre`, `LibraryHandler`) refreshes the
  `genre` tag, preserving existing tags (e.g. `artist`).

**Verification**: unit tests only (no Docker on the dev host — integration
verified in the final integration stage): `TestTagSongObject_*` (metadata),
`TestImporter_TagObject_*` (tidal), `TestUpdateSongGenreHandler_RetagsObject`
(handlers). Streaming still resolves by unchanged hash key.

**Risk**: low-medium (storage metadata only; no layout change, no migration).

**Status**: ✅ (2026-11-14)

---

## E5 — Tidal link-paste download

**Goal**: paste a Tidal playlist/album/track URL → hit a button → download.

**Current state**: import exists but is browse-based (library/album/playlist tabs).
Download is hand-rolled (stream fetch + AES-128-CBC decrypt) — works.

**Changes**:
- URL parser: recognize Tidal playlist/album/track URLs → resolve to the existing
  import target (reuse `Client` lookups).
- New endpoint (e.g. `POST /api/tidal/import/link`) + a "paste link" UI on
  `frontend/src/pages/Tidal.jsx` that calls it and reuses the existing
  `importStatus` polling + job progress.
- On import, assign genre via E3's playlist-name matching.

**Verification**: paste a playlist URL → tracks download, dedup by hash, appear in
library with the matched genre; album/track URLs also work.

**Status**: ✅ (2026-11-14) — `pkg/tidal/url.go` parses tidal.com / play.tidal.h5
track/album/playlist URLs (numeric IDs; playlist IDs are `pl`+16–32 hex) and
rejects everything else. `POST /api/tidal/import/link` resolves the link via
new `Client` lookups (`GetTrack`; string-ID playlist variants `GetPlaylistByURLID` /
`GetPlaylistTracksByURLID` — the v1 API accepts URL playlist IDs in path form,
worth an integration check) plus the existing album lookups, paginates lists
(capped at 2000 tracks), skips video items, and starts the same batch import
as the browse flow via the shared `startImportJob` (E3b genre matching applies
to album/playlist links). `frontend/src/pages/Tidal.jsx` gains a paste-link
box reusing `importStatus` polling. Tests: `pkg/tidal/url_test.go`,
`TestLinkImportHandler_*` in `pkg/handlers/tidal_test.go` (fake Tidal API +
sqlmock).

---

## E6 — Sortable playlists, default most-recent

**Goal**: playlists sortable in the GUI; default = most recent first.

**Current state**: tracks ordered by integer `position`; GUI sort is client-only
and non-persistent; no `added_at` on `playlist_songs`.

**Changes**:
- New migration: `added_at TIMESTAMPTZ` on `playlist_songs`; set it in
  `AddSongToPlaylist`.
- `GetPlaylistSongs`: default `ORDER BY added_at DESC` (D3); support an optional
  server-side `sortBy` (added_at / title / artist / position) like the library.
- GUI: make column sort **persist** (rewrite `position` / set a stored sort key)
  rather than display-only; keep the existing drag-reorder.

**Verification**: add tracks → newest first by default; sort by title persists
across reloads; drag-reorder still works.

**Status**: ✅ (2026-11-14) — migration `0009`: `playlist_songs.added_at
TIMESTAMPTZ` (backfilled with migration time) and `playlists.sort_key
VARCHAR(20) DEFAULT 'added_at'`. `GetPlaylistSongs` defaults to `ORDER BY
ps.added_at DESC, ps.position ASC` and accepts a whitelisted sort key
(added_at / title / artist / position — only literal whitelist values reach
query text, same pattern as the library's `sortBy`); unknown keys fall back to
the default instead of erroring. `AddSongToPlaylist` inserts with
`added_at = NOW()`. API: `GET /api/playlists/{id}?sort_by=...` and
`PUT /api/playlists/{id}` with `sort_key` (400 on a key outside the
whitelist, 404 on non-owner). `GET` without `?sort_by=` uses the persisted
key; `PUT /api/playlists/{id}/order` persists a full displayed order (bulk
position rewrite, validated against the playlist's contents). GUI
(PlaylistDetails): default most-recent first, column headers persist the sort
via `sort_key`; drag-reorder persists the full on-screen order and switches
the playlist to `position`; new Added column.
Tests: `pkg/db/playlist_test.go`, `pkg/handlers/playlist_test.go`.

---

## E7 — Subsonic API completeness

**Goal**: any third-party Subsonic client can connect and use the library.

**Current state**: `ping`, `getMusicFolders` (hardcoded), `getIndexes`, `search3`,
`stream`. No playlists/albums/artists/artwork/random/similar. **Token auth is
rejected** (bcrypt passwords). `stream` is the only scoped endpoint (inconsistent
with the global model).

**Phases**:
- **A (connectable)**: enrich the Subsonic `Song` XML model (duration, bitrate,
  size, `streamUrl`, `isVideo`, etc.); support **token auth** (`t`+`s`) — requires
  storing a per-user Subsonic token or deriving one; add `getSimilarSongs.view`
  (reuses E2). Align `stream` scoping with the global model (E1).
- **B (browse)**: `getAlbums`, `getAlbum`, `getArtist`, `getArtists`,
  `getChildren`, `getArtwork`/`getCoverArt`, `getRandomSongs`.
- **C (playlists)**: `getPlaylists`, `getPlaylist`, `createPlaylist`,
  `updatePlaylist`, `deletePlaylist`.

**Design decisions**:
- **Token auth**: per-user token derived as `HMAC-SHA256(JWT_SECRET,
  "biomuzak-subsonic-token:"+username)` — no new table/column, revocable by
  rotating `JWT_SECRET`; exposed via `GET /api/subsonic/token` (JWT-protected).
- **Artist/album IDs**: the schema has no artists/albums tables, so IDs are
  synthesized, reversible strings (`a.<base64>`, `al.<base64>`).

**Verification**: unit tests cover token auth (valid/wrong), all new endpoints,
and per-user playlist scoping. Connecting a real Subsonic mobile client is
deferred to the final integration stage (no Docker on the dev host).

**Status**: ✅

---

## E8 — (optional) Adopt tiddl

**Goal**: replace the hand-rolled fetch/decrypt with `github.com/oskvr37/tiddl`.

**Only if D1 is overridden.** Add the dependency; rewrite the fetch/decrypt steps
in `pkg/tidal/import.go`; keep the rest of the import pipeline (dedup, storage,
genre). Higher risk (new dependency + rewrite of a working path).

**Status**: ⬜

---

## Cross-cutting notes

- **Embedding dim**: `0001` says `VECTOR(512)` but `0005` fixes it to 38 — already
  handled; don't touch unless changing the 38-dim contract.
- **TDD**: every epic starts with a failing test (red/green). Offline tests via
  `go-sqlmock` / `httptest`; no live MinIO/Postgres in unit tests.
- **Changelog**: each user-visible epic gets a bullet under `## [Unreleased]` in
  `docs/CHANGELOG.md`.
- **Conventional commits** per epic (`feat(scope):`, `fix(scope):`, ...).
- **Review gate**: run the advisory reviews (`/pre-pr-review`, `/security-review`,
  `/performance-review`, scoped `/audio-processor-review` / `/frontend-review`) and
  `/risk-tier` before committing each epic.

## Notable findings (from codebase analysis)

- **Similar songs**: the backend endpoint already exists and works — the **entire
  frontend is missing** (no UI, no API call, no hook in `PlayerContext`). That's why
  "suggestions don't work."
- **Subsonic token auth is currently hard-rejected** (bcrypt passwords can't do
  `md5(password+salt)`) — a real blocker for most third-party clients; E7 Phase A
  fixes it.
- **Tidal import sets no genre** and storage keys are decided *before* genre is
  known — so E4 (object tagging) must come after E3 (genre resolution).
- **`user_songs`** is already a many-to-many "library" view, not true ownership — so
  E1 is mostly dropping the membership gate, not a schema rewrite.
