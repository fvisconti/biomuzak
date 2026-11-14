-- E3c: seed the genre catalogue so name-based matching (E3a/E3b) has rows to
-- match against out of the box. Idempotent: re-applying never duplicates.
-- Additional genres can be created at runtime via POST /api/admin/genres, or
-- implicitly when a user edits a song's genre (the update path upserts).
INSERT INTO genres (name) VALUES
    ('Rock'), ('Pop'), ('Jazz'), ('Blues'), ('Classical'), ('Country'),
    ('Hip Hop'), ('Rap'), ('Metal'), ('Punk'), ('Folk'), ('Reggae'),
    ('Soul'), ('Funk'), ('Disco'), ('Electronic'), ('House'), ('Techno'),
    ('Deep House'), ('Drum and Bass'), ('Trance'), ('Ambient'), ('Indie'),
    ('Alternative'), ('R&B'), ('Gospel'), ('Latin'), ('K-Pop'), ('Soundtrack'),
    ('Lo-Fi'), ('Synthwave')
ON CONFLICT (name) DO NOTHING;
