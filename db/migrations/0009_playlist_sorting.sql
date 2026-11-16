-- E6: sortable playlists with a most-recent-first default.
--
-- playlist_songs.added_at: when the song was (re)added to the playlist.
-- Existing rows are backfilled with the migration time; position remains the
-- deterministic tiebreak.
--
-- playlists.sort_key: the per-playlist column the GUI persists. Whitelisted
-- by the API: added_at (default), title, artist, position.
ALTER TABLE playlist_songs ADD COLUMN added_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE playlists ADD COLUMN sort_key VARCHAR(20) NOT NULL DEFAULT 'added_at';
