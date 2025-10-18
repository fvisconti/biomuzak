import React, { useState, useEffect, useRef } from 'react';
import { getSimilarSongs, rateSong } from '../api';
import './PlayerBar.css';

function PlayerBar({ song, songs, onSongChange }) {
  const [isPlaying, setIsPlaying] = useState(false);
  const [volume, setVolume] = useState(70);
  const [currentTime, setCurrentTime] = useState(0);
  const [duration, setDuration] = useState(0);
  const [similarSongs, setSimilarSongs] = useState([]);
  const [showSimilar, setShowSimilar] = useState(true);
  const [rating, setRating] = useState(song?.rating || 0);
  const audioRef = useRef(null);

  useEffect(() => {
    if (song) {
      setRating(song.rating || 0);
      loadSimilarSongs(song.id);
      // Auto-play when a new song is selected
      setIsPlaying(true);
    }
  }, [song]);

  useEffect(() => {
    if (audioRef.current) {
      audioRef.current.volume = volume / 100;
    }
  }, [volume]);

  useEffect(() => {
    if (audioRef.current) {
      if (isPlaying) {
        audioRef.current.play().catch(err => {
          console.error('Playback error:', err);
          setIsPlaying(false);
        });
      } else {
        audioRef.current.pause();
      }
    }
  }, [isPlaying]);

  const loadSimilarSongs = async (songId) => {
    try {
      const response = await getSimilarSongs(songId);
      // Get top 3 similar songs
      setSimilarSongs((response.data || []).slice(0, 3));
    } catch (err) {
      console.error('Failed to load similar songs:', err);
      setSimilarSongs([]);
    }
  };

  const handlePlayPause = () => {
    setIsPlaying(!isPlaying);
  };

  const handleNextSong = () => {
    const currentIndex = songs.findIndex(s => s.id === song.id);
    const nextIndex = (currentIndex + 1) % songs.length;
    onSongChange(songs[nextIndex]);
  };

  const handlePrevSong = () => {
    const currentIndex = songs.findIndex(s => s.id === song.id);
    const prevIndex = currentIndex === 0 ? songs.length - 1 : currentIndex - 1;
    onSongChange(songs[prevIndex]);
  };

  const handleTimeUpdate = () => {
    if (audioRef.current) {
      setCurrentTime(audioRef.current.currentTime);
      setDuration(audioRef.current.duration || 0);
    }
  };

  const handleSeek = (e) => {
    const seekTime = (e.target.value / 100) * duration;
    if (audioRef.current) {
      audioRef.current.currentTime = seekTime;
      setCurrentTime(seekTime);
    }
  };

  const handleRating = async (newRating) => {
    try {
      await rateSong(song.id, newRating);
      setRating(newRating);
      song.rating = newRating; // Update the song object
    } catch (err) {
      console.error('Failed to rate song:', err);
    }
  };

  const formatTime = (seconds) => {
    if (isNaN(seconds)) return '0:00';
    const mins = Math.floor(seconds / 60);
    const secs = Math.floor(seconds % 60);
    return `${mins}:${secs.toString().padStart(2, '0')}`;
  };

  const handleSongEnd = () => {
    handleNextSong();
  };

  if (!song) return null;

  return (
    <div className="player-bar">
      <audio
        ref={audioRef}
        src={song.file_path}
        onTimeUpdate={handleTimeUpdate}
        onLoadedMetadata={handleTimeUpdate}
        onEnded={handleSongEnd}
      />

      <div className="player-main">
        <div className="player-info">
          <div className="song-details">
            <div className="song-title">{song.title || 'Unknown Title'}</div>
            <div className="song-artist">{song.artist || 'Unknown Artist'}</div>
          </div>
          <div className="song-rating">
            {[1, 2, 3, 4, 5].map(star => (
              <span
                key={star}
                className={`star ${star <= rating ? 'filled' : ''}`}
                onClick={() => handleRating(star)}
              >
                ★
              </span>
            ))}
          </div>
        </div>

        <div className="player-controls">
          <button className="control-btn" onClick={handlePrevSong}>⏮</button>
          <button className="control-btn play-pause" onClick={handlePlayPause}>
            {isPlaying ? '⏸' : '▶'}
          </button>
          <button className="control-btn" onClick={handleNextSong}>⏭</button>
        </div>

        <div className="player-progress">
          <span className="time">{formatTime(currentTime)}</span>
          <input
            type="range"
            min="0"
            max="100"
            value={(currentTime / duration) * 100 || 0}
            onChange={handleSeek}
            className="progress-bar"
          />
          <span className="time">{formatTime(duration)}</span>
        </div>

        <div className="player-volume">
          <span>🔊</span>
          <input
            type="range"
            min="0"
            max="100"
            value={volume}
            onChange={(e) => setVolume(e.target.value)}
            className="volume-slider"
          />
        </div>
      </div>

      {similarSongs.length > 0 && showSimilar && (
        <div className="similar-songs">
          <div className="similar-header">
            <span>Similar Songs</span>
            <button className="close-btn" onClick={() => setShowSimilar(false)}>✕</button>
          </div>
          <div className="similar-list">
            {similarSongs.map(similarSong => (
              <div
                key={similarSong.id}
                className="similar-item"
                onClick={() => onSongChange(similarSong)}
              >
                <div className="similar-info">
                  <div className="similar-title">{similarSong.title}</div>
                  <div className="similar-artist">{similarSong.artist}</div>
                </div>
                <div className="similarity-score">
                  {Math.round(similarSong.similarity * 100)}%
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

export default PlayerBar;
