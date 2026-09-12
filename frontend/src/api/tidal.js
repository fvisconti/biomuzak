import apiClient from './client';

// Tidal pairing
export const pairStart = () => apiClient.post('/tidal/pair/start');
export const pairStatus = (pairId) =>
    apiClient.get('/tidal/pair/status', { params: { pairId } });

// Connection status + config
export const getStatus = () => apiClient.get('/tidal/status');
export const saveSettings = (config) => apiClient.put('/tidal/settings', config);
export const disconnect = () => apiClient.delete('/tidal/connection');

// Browse
export const getLibrary = (kind, offset = 0, limit = 50) =>
    apiClient.get(`/tidal/library/${kind}`, { params: { offset, limit } });
export const getAlbum = (albumId) => apiClient.get(`/tidal/albums/${albumId}`);
export const getPlaylist = (playlistId) => apiClient.get(`/tidal/playlists/${playlistId}`);

// Import
export const startImport = (payload) => apiClient.post('/tidal/import', payload);
export const importStatus = (jobId) =>
    apiClient.get('/tidal/import/status', { params: { jobId } });

// Export to local folder
export const startExport = (songIds) => apiClient.post('/export', { song_ids: songIds });
export const exportStatus = (jobId) =>
    apiClient.get('/export/status', { params: { jobId } });
