-- Add genre column to playlists table
ALTER TABLE playlists ADD COLUMN IF NOT EXISTS genre VARCHAR(255);
