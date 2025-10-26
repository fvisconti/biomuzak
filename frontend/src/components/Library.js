import React, { useState, useEffect } from 'react';
import { getLibrary, getMe } from '../api';
import PlayerBar from './PlayerBar';
import './Library.css';

function Library() {
  const [songs, setSongs] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [currentSong, setCurrentSong] = useState(null);
  const [isAdmin, setIsAdmin] = useState(false);
  
  // Filters and sort
  const [searchTerm, setSearchTerm] = useState('');
  const [genreFilter, setGenreFilter] = useState('');
  const [artistFilter, setArtistFilter] = useState('');
  const [yearFilter, setYearFilter] = useState('');
  const [sortBy, setSortBy] = useState('');

  useEffect(() => {
    loadLibrary();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [genreFilter, artistFilter, yearFilter, sortBy]);

  useEffect(() => {
    // Fetch current user role once to conditionally show Admin link
    getMe()
      .then(({ data }) => setIsAdmin(!!data?.is_admin))
      .catch(() => setIsAdmin(false));
  }, []);

  const loadLibrary = async () => {
    setLoading(true);
    setError('');
    try {
      const filters = {};
      if (genreFilter) filters.genre = genreFilter;
      if (artistFilter) filters.artist = artistFilter;
      if (yearFilter) filters.year = yearFilter;
      
      const response = await getLibrary(filters, sortBy);
      setSongs(response.data || []);
    } catch (err) {
      setError('Failed to load library. Please try again.');
      console.error('Library load error:', err);
    } finally {
      setLoading(false);
    }
  };

  const handlePlaySong = (song) => {
    setCurrentSong(song);
  };

  const filteredSongs = songs.filter(song => {
    if (!searchTerm) return true;
    const search = searchTerm.toLowerCase();
    return (
      song.title?.toLowerCase().includes(search) ||
      song.artist?.toLowerCase().includes(search) ||
      song.album?.toLowerCase().includes(search)
    );
  });

  // Extract unique values for filter dropdowns
  const genres = [...new Set(songs.map(s => s.genre).filter(Boolean))];
  const artists = [...new Set(songs.map(s => s.artist).filter(Boolean))];

  return (
    <div className="library-container">
      <div className="library-header">
        <h1>My Library</h1>
        <div className="header-actions">
          <button className="nav-btn" onClick={() => window.location.href = '/upload'}>
            📤 Upload
          </button>
          <button className="nav-btn" onClick={() => window.location.href = '/playlists'}>
            📋 Playlists
          </button>
          {isAdmin && (
            <button className="nav-btn" onClick={() => window.location.href = '/admin'}>
              🛠️ Admin
            </button>
          )}
          <button className="logout-btn" onClick={() => {
            localStorage.removeItem('token');
            window.location.href = '/login';
          }}>Logout</button>
        </div>
      </div>

      <div className="library-controls">
        <input
          type="text"
          placeholder="Search songs..."
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          className="search-input"
        />

        <select value={genreFilter} onChange={(e) => setGenreFilter(e.target.value)} className="filter-select">
          <option value="">All Genres</option>
          {genres.map(genre => (
            <option key={genre} value={genre}>{genre}</option>
          ))}
        </select>

        <select value={artistFilter} onChange={(e) => setArtistFilter(e.target.value)} className="filter-select">
          <option value="">All Artists</option>
          {artists.map(artist => (
            <option key={artist} value={artist}>{artist}</option>
          ))}
        </select>

        <input
          type="text"
          placeholder="Year"
          value={yearFilter}
          onChange={(e) => setYearFilter(e.target.value)}
          className="year-input"
        />

        <select value={sortBy} onChange={(e) => setSortBy(e.target.value)} className="sort-select">
          <option value="">Sort by...</option>
          <option value="title">Title</option>
          <option value="artist">Artist</option>
          <option value="rating">Rating</option>
        </select>
      </div>

      {loading && <div className="loading">Loading library...</div>}
      {error && <div className="error-message">{error}</div>}

      {!loading && !error && (
        <div className="songs-grid">
          {filteredSongs.length === 0 ? (
            <div className="no-songs">No songs found. Upload some music to get started!</div>
          ) : (
            filteredSongs.map(song => (
              <div key={song.id} className="song-card">
                <div className="song-info">
                  <h3>{song.title || 'Unknown Title'}</h3>
                  <p className="artist">{song.artist || 'Unknown Artist'}</p>
                  <p className="album">{song.album || 'Unknown Album'}</p>
                  {song.year && <p className="year">{song.year}</p>}
                  {song.genre && <p className="genre">{song.genre}</p>}
                  {song.rating > 0 && (
                    <div className="rating">
                      {'★'.repeat(song.rating)}{'☆'.repeat(5 - song.rating)}
                    </div>
                  )}
                </div>
                <button 
                  className="play-btn"
                  onClick={() => handlePlaySong(song)}
                >
                  ▶ Play
                </button>
              </div>
            ))
          )}
        </div>
      )}

      {currentSong && <PlayerBar song={currentSong} songs={filteredSongs} onSongChange={setCurrentSong} />}
    </div>
  );
}

export default Library;
