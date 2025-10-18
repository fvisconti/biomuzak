import axios from 'axios';

const API_BASE_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080';

const api = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Add auth token to all requests if it exists
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// Auth APIs
export const register = (username, email, password) => 
  api.post('/register', { username, email, password });

export const login = (username, password) => 
  api.post('/login', { username, password });

// Library APIs
export const getLibrary = (filters = {}, sortBy = '') => {
  const params = new URLSearchParams();
  Object.keys(filters).forEach(key => {
    if (filters[key]) params.append(key, filters[key]);
  });
  if (sortBy) params.append('sort_by', sortBy);
  return api.get(`/api/library?${params.toString()}`);
};

// Song APIs
export const rateSong = (songId, rating) => 
  api.post(`/api/songs/${songId}/rate`, { rating });

export const getSimilarSongs = (songId) => 
  api.get(`/api/songs/${songId}/similar`);

// Upload API
export const uploadFiles = (formData, onUploadProgress) => 
  api.post('/api/upload', formData, {
    headers: {
      'Content-Type': 'multipart/form-data',
    },
    onUploadProgress,
  });

// Playlist APIs
export const createPlaylist = (name) => 
  api.post('/api/playlists', { name });

export const getPlaylists = () => 
  api.get('/api/playlists');

export const getPlaylist = (playlistId) => 
  api.get(`/api/playlists/${playlistId}`);

export const updatePlaylist = (playlistId, name) => 
  api.put(`/api/playlists/${playlistId}`, { name });

export const deletePlaylist = (playlistId) => 
  api.delete(`/api/playlists/${playlistId}`);

export const addSongToPlaylist = (playlistId, songId, position = 0) => 
  api.post(`/api/playlists/${playlistId}/songs`, { song_id: songId, position });

export const removeSongFromPlaylist = (playlistId, songId) => 
  api.delete(`/api/playlists/${playlistId}/songs/${songId}`);

export default api;
