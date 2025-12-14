import React, { createContext, useContext, useState, useRef, useEffect } from 'react';

const PlayerContext = createContext();

export const usePlayer = () => useContext(PlayerContext);

export const PlayerProvider = ({ children }) => {
    const [currentSong, setCurrentSong] = useState(null);
    const [isPlaying, setIsPlaying] = useState(false);
    const [queue, setQueue] = useState([]);
    const [currentIndex, setCurrentIndex] = useState(-1);
    const [progress, setProgress] = useState(0);
    const [duration, setDuration] = useState(0);
    const [loading, setLoading] = useState(false);

    const audioRef = useRef(new Audio());
    const currentBlobUrl = useRef(null);

    useEffect(() => {
        const audio = audioRef.current;

        const handleTimeUpdate = () => setProgress(audio.currentTime);
        const handleDurationChange = () => setDuration(audio.duration);
        const handleEnded = () => playNext();

        audio.addEventListener('timeupdate', handleTimeUpdate);
        audio.addEventListener('durationchange', handleDurationChange);
        audio.addEventListener('ended', handleEnded);

        return () => {
            audio.removeEventListener('timeupdate', handleTimeUpdate);
            audio.removeEventListener('durationchange', handleDurationChange);
            audio.removeEventListener('ended', handleEnded);
            // Cleanup blob URL on unmount
            if (currentBlobUrl.current) {
                URL.revokeObjectURL(currentBlobUrl.current);
            }
        };
    }, [queue, currentIndex]);

    const playSong = (song) => {
        setQueue([song]);
        setCurrentIndex(0);
        loadAndPlay(song);
    };

    const playPlaylist = (songs, startIndex = 0) => {
        setQueue(songs);
        setCurrentIndex(startIndex);
        loadAndPlay(songs[startIndex]);
    };

    const loadAndPlay = async (song) => {
        if (!song) return;
        console.log('[PlayerContext] Loading song:', song);
        setCurrentSong(song);
        setLoading(true);

        try {
            // Fetch the audio file with authentication
            const token = localStorage.getItem('token');
            const streamUrl = `/api/songs/${song.id}/stream`;
            console.log('[PlayerContext] Fetching stream URL:', streamUrl);

            const response = await fetch(streamUrl, {
                headers: {
                    'Authorization': `Bearer ${token}`
                }
            });

            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }

            // Create a blob from the response
            const blob = await response.blob();
            console.log('[PlayerContext] Blob created, size:', blob.size, 'type:', blob.type);

            // Revoke previous blob URL if exists
            if (currentBlobUrl.current) {
                URL.revokeObjectURL(currentBlobUrl.current);
            }

            // Create a blob URL
            const blobUrl = URL.createObjectURL(blob);
            currentBlobUrl.current = blobUrl;
            console.log('[PlayerContext] Blob URL created:', blobUrl);

            // Set the audio source to the blob URL
            audioRef.current.src = blobUrl;

            // Play the audio
            await audioRef.current.play();
            console.log('[PlayerContext] Playback started successfully');
            setIsPlaying(true);
        } catch (e) {
            console.error('[PlayerContext] Playback failed:', e);
            console.error('[PlayerContext] Error details:', e.message, e.name);
        } finally {
            setLoading(false);
        }
    };

    const togglePlay = () => {
        if (isPlaying) {
            audioRef.current.pause();
            setIsPlaying(false);
        } else {
            audioRef.current.play()
                .then(() => setIsPlaying(true))
                .catch(e => console.error('Play failed:', e));
        }
    };

    const playNext = () => {
        if (currentIndex < queue.length - 1) {
            const nextIndex = currentIndex + 1;
            setCurrentIndex(nextIndex);
            loadAndPlay(queue[nextIndex]);
        } else {
            setIsPlaying(false);
            setCurrentSong(null);
        }
    };

    const playPrev = () => {
        if (currentIndex > 0) {
            const prevIndex = currentIndex - 1;
            setCurrentIndex(prevIndex);
            loadAndPlay(queue[prevIndex]);
        }
    };

    const seek = (time) => {
        audioRef.current.currentTime = time;
        setProgress(time);
    };

    const value = {
        currentSong,
        isPlaying,
        progress,
        duration,
        loading,
        togglePlay,
        playNext,
        playPrev,
        playSong,
        playPlaylist,
        seek
    };

    return (
        <PlayerContext.Provider value={value}>
            {children}
        </PlayerContext.Provider>
    );
};
