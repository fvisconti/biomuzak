import React, { createContext, useContext, useState, useEffect, useCallback } from 'react';
import { useAuth } from './AuthContext';

const PlaylistContext = createContext();

export const usePlaylists = () => useContext(PlaylistContext);

export const PlaylistProvider = ({ children }) => {
    const [playlists, setPlaylists] = useState([]);
    const [loading, setLoading] = useState(false);
    const { token } = useAuth();

    const fetchPlaylists = useCallback(async () => {
        if (!token) {
            setPlaylists([]);
            return;
        }
        setLoading(true);
        try {
            const res = await fetch('/api/playlists', {
                headers: { 'Authorization': `Bearer ${token}` }
            });
            if (res.ok) {
                const data = await res.json();
                setPlaylists(data || []);
            }
        } catch (error) {
            console.error("Failed to fetch playlists", error);
        } finally {
            setLoading(false);
        }
    }, [token]);

    useEffect(() => {
        fetchPlaylists();
    }, [fetchPlaylists]);

    const value = {
        playlists,
        loading,
        refreshPlaylists: fetchPlaylists,
        setPlaylists
    };

    return (
        <PlaylistContext.Provider value={value}>
            {children}
        </PlaylistContext.Provider>
    );
};
