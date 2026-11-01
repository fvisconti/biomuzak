# biomuzak — Feature Plan (Tidal-first music library)

> **Status**: Planning / in progress. This is a **living document** — update the
> status of each epic (⬜ not started · 🟨 in progress · ✅ done · ⛔ blocked) and
> the decision log as development proceeds.
>
> **Last updated**: 2026-11-01

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
| D2 | Storage layout | One **bucket per genre** (dynamic selection) + **MinIO object tags** for artist/label. | Matches the explicit "bucket per genre, label for artist/label" wording. Accepts a one-time re-key migration. |
| D3 | "Most recent" | = most recently **added to the playlist** (new `added_at` column). | Most intuitive for a playlist. |
| D4 | Subsonic scope | **Full** target (browse + playlists + token auth), delivered in 3 phases. | "Easy to connect any third-party client." |
| D5 | Global visibility | All tracks visible to all; **keep `user_songs` for ratings / "in my library" only**. | Preserves the existing ratings feature. |
| D6 | Security boundary | Global visibility **deliberately overrides** Prime Directive 1 for *catalog browsing*. New boundary: **catalog is global; playlists + ratings are per-user.** | Required by bullet 4. Docs updated in E1. |

## Epic map

| Epic | Bullet | One-line goal | Depends on | Status |
|---|---|---|---|---|
| **E1** Global catalog visibility | 4 | ANY user sees ANY track; playlists + ratings stay per-user | — | ✅ |
| **E2** Similar tracks for the playing song | 7 | Show 3 similar tracks w.r.t. the playing song | E1 | ⬜ |
| **E3** Genre catalogue + playlist-name matching | 3 | Match each downloaded playlist to a genre by name (cheap) | — | ⬜ |
| **E4** MinIO per-genre buckets + object tagging | 2 | One bucket per genre; object tags for artist/label | E3 | ⬜ |
| **E5** Tidal link-paste download | 1 | Paste a Tidal URL → download | E3 | ⬜ |
| **E6** Sortable playlists, default most-recent | 5 | GUI-sortable playlists; default most-recent-first | — | ⬜ |
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

**Status**: ⬜

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

**Status**: ⬜

---

## E4 — MinIO per-genre buckets + object tagging

**Goal**: one bucket per genre (highly populated); object tags for artist/label.

**Current state**: single flat bucket (`MINIO_BUCKET`, default `music`), key =
`sha256+ext`, no tagging, bucket fixed at construction.

**Changes**:
- `pkg/storage/storage.go`: dynamic bucket selection (bucket name derived from
  genre, e.g. `music-<slug>`); create-on-demand; keep hash-based key for dedup.
- Add MinIO object tagging via `PutObjectOptions.UserTags` (genre/artist/label).
- Move the bucket/key decision to **after** genre resolution (genre is known only
  post-processing / post-match).
- **Migration**: re-key/relocate existing objects into their genre buckets + tag
  them (one-time job).
- Handle **genre change**: moving a song's genre moves its object between buckets
  (copy + delete) — document the cost.
- Update `.env.example` + `docs/DOCKER.md` for the new bucket-naming scheme.

**Verification**: import into two genres → objects land in two buckets with tags;
streaming still resolves the file; existing library re-keyed without data loss.

**Risk**: highest of the set (storage layout + migration). Keep behind E3.

**Status**: ⬜

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

**Status**: ⬜

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

**Status**: ⬜

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

**Verification**: connect a real Subsonic client (e.g. a mobile app) with token
auth; browse albums/artists, stream, see similar songs, list/create playlists.

**Status**: ⬜

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
  known — so E4 (per-genre buckets) must come after E3 (genre resolution).
- **`user_songs`** is already a many-to-many "library" view, not true ownership — so
  E1 is mostly dropping the membership gate, not a schema rewrite.
