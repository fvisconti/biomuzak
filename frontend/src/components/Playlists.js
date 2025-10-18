import React, { useState, useEffect } from 'react';
import { 
  getPlaylists, 
  getPlaylist, 
  createPlaylist, 
  deletePlaylist,
  addSongToPlaylist,
  removeSongFromPlaylist 
} from '../api';
import { getLibrary } from '../api';
import './Playlists.css';

function Playlists() {
  const [playlists, setPlaylists] = useState([]);
  const [selectedPlaylist, setSelectedPlaylist] = useState(null);
  const [playlistSongs, setPlaylistSongs] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [showAddSongModal, setShowAddSongModal] = useState(false);
  const [newPlaylistName, setNewPlaylistName] = useState('');
  const [allSongs, setAllSongs] = useState([]);
  const [searchTerm, setSearchTerm] = useState('');

  useEffect(() => {
    loadPlaylists();
  }, []);

  const loadPlaylists = async () => {
    setLoading(true);
    setError('');
    try {
      const response = await getPlaylists();
      setPlaylists(response.data || []);
    } catch (err) {
      setError('Failed to load playlists');
      console.error('Playlists load error:', err);
    } finally {
      setLoading(false);
    }
  };

  const loadPlaylistDetails = async (playlistId) => {
    try {
      const response = await getPlaylist(playlistId);
      setSelectedPlaylist(response.data);
      setPlaylistSongs(response.data.songs || []);
    } catch (err) {
      setError('Failed to load playlist details');
      console.error('Playlist details error:', err);
    }
  };

  const handleCreatePlaylist = async () => {
    if (!newPlaylistName.trim()) {
      setError('Please enter a playlist name');
      return;
    }

    try {
      await createPlaylist(newPlaylistName);
      setNewPlaylistName('');
      setShowCreateModal(false);
      await loadPlaylists();
    } catch (err) {
      setError('Failed to create playlist');
      console.error('Create playlist error:', err);
    }
  };

  const handleDeletePlaylist = async (playlistId) => {
    if (!window.confirm('Are you sure you want to delete this playlist?')) {
      return;
    }

    try {
      await deletePlaylist(playlistId);
      if (selectedPlaylist?.id === playlistId) {
        setSelectedPlaylist(null);
        setPlaylistSongs([]);
      }
      await loadPlaylists();
    } catch (err) {
      setError('Failed to delete playlist');
      console.error('Delete playlist error:', err);
    }
  };

  const handleAddSongToPlaylist = async (songId) => {
    if (!selectedPlaylist) return;

    try {
      await addSongToPlaylist(selectedPlaylist.id, songId);
      await loadPlaylistDetails(selectedPlaylist.id);
      setShowAddSongModal(false);
    } catch (err) {
      setError('Failed to add song to playlist');
      console.error('Add song error:', err);
    }
  };

  const handleRemoveSongFromPlaylist = async (songId) => {
    if (!selectedPlaylist) return;

    try {
      await removeSongFromPlaylist(selectedPlaylist.id, songId);
      await loadPlaylistDetails(selectedPlaylist.id);
    } catch (err) {
      setError('Failed to remove song from playlist');
      console.error('Remove song error:', err);
    }
  };

  const loadAllSongs = async () => {
    try {
      const response = await getLibrary();
      setAllSongs(response.data || []);
    } catch (err) {
      console.error('Failed to load songs:', err);
    }
  };

  const openAddSongModal = () => {
    setShowAddSongModal(true);
    loadAllSongs();
  };

  const filteredSongs = allSongs.filter(song => {
    if (!searchTerm) return true;
    const search = searchTerm.toLowerCase();
    return (
      song.title?.toLowerCase().includes(search) ||
      song.artist?.toLowerCase().includes(search)
    );
  });

  return (
    <div className="playlists-container">
      <div className="playlists-header">
        <h1>My Playlists</h1>
        <div className="header-actions">
          <button className="create-btn" onClick={() => setShowCreateModal(true)}>
            + New Playlist
          </button>
          <button 
            className="back-btn"
            onClick={() => window.location.href = '/library'}
          >
            ← Back to Library
          </button>
        </div>
      </div>

      {error && <div className="error-message">{error}</div>}

      <div className="playlists-layout">
        <div className="playlists-sidebar">
          <h3>Playlists</h3>
          {loading ? (
            <div className="loading">Loading...</div>
          ) : playlists.length === 0 ? (
            <div className="no-playlists">No playlists yet. Create one!</div>
          ) : (
            <ul className="playlist-list">
              {playlists.map(playlist => (
                <li
                  key={playlist.id}
                  className={selectedPlaylist?.id === playlist.id ? 'active' : ''}
                  onClick={() => loadPlaylistDetails(playlist.id)}
                >
                  <span className="playlist-name">{playlist.name}</span>
                  <button
                    className="delete-icon"
                    onClick={(e) => {
                      e.stopPropagation();
                      handleDeletePlaylist(playlist.id);
                    }}
                  >
                    🗑
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>

        <div className="playlist-content">
          {selectedPlaylist ? (
            <>
              <div className="playlist-header-info">
                <h2>{selectedPlaylist.name}</h2>
                <button className="add-song-btn" onClick={openAddSongModal}>
                  + Add Songs
                </button>
              </div>
              
              {playlistSongs.length === 0 ? (
                <div className="no-songs">This playlist is empty. Add some songs!</div>
              ) : (
                <div className="songs-list">
                  {playlistSongs.map((song, index) => (
                    <div key={`${song.id}-${index}`} className="song-item">
                      <div className="song-number">{index + 1}</div>
                      <div className="song-info">
                        <div className="song-title">{song.title || 'Unknown Title'}</div>
                        <div className="song-artist">{song.artist || 'Unknown Artist'}</div>
                      </div>
                      <div className="song-album">{song.album || 'Unknown Album'}</div>
                      <button
                        className="remove-btn"
                        onClick={() => handleRemoveSongFromPlaylist(song.id)}
                      >
                        Remove
                      </button>
                    </div>
                  ))}
                </div>
              )}
            </>
          ) : (
            <div className="no-selection">
              Select a playlist to view its contents
            </div>
          )}
        </div>
      </div>

      {showCreateModal && (
        <div className="modal-overlay" onClick={() => setShowCreateModal(false)}>
          <div className="modal-content" onClick={(e) => e.stopPropagation()}>
            <h3>Create New Playlist</h3>
            <input
              type="text"
              placeholder="Playlist name"
              value={newPlaylistName}
              onChange={(e) => setNewPlaylistName(e.target.value)}
              onKeyPress={(e) => e.key === 'Enter' && handleCreatePlaylist()}
              autoFocus
            />
            <div className="modal-actions">
              <button onClick={handleCreatePlaylist}>Create</button>
              <button onClick={() => setShowCreateModal(false)}>Cancel</button>
            </div>
          </div>
        </div>
      )}

      {showAddSongModal && (
        <div className="modal-overlay" onClick={() => setShowAddSongModal(false)}>
          <div className="modal-content large" onClick={(e) => e.stopPropagation()}>
            <h3>Add Songs to {selectedPlaylist?.name}</h3>
            <input
              type="text"
              placeholder="Search songs..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="search-input"
            />
            <div className="songs-modal-list">
              {filteredSongs.map(song => (
                <div
                  key={song.id}
                  className="song-modal-item"
                  onClick={() => handleAddSongToPlaylist(song.id)}
                >
                  <div className="song-info">
                    <div className="song-title">{song.title}</div>
                    <div className="song-artist">{song.artist}</div>
                  </div>
                  <button className="add-icon">+</button>
                </div>
              ))}
            </div>
            <div className="modal-actions">
              <button onClick={() => setShowAddSongModal(false)}>Close</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export default Playlists;
