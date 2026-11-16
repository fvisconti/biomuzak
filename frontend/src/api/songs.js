import apiClient from './client';

// Similar songs for the currently playing track. Returns an array of
// { song fields..., similarity } sorted by similarity (highest first).
export const getSimilarSongs = (songId) =>
    apiClient.get(`/songs/${songId}/similar`);
