import React, { useState, useRef } from 'react';
import { uploadFiles } from '../api';
import './Upload.css';

function Upload() {
  const [uploading, setUploading] = useState(false);
  const [uploadProgress, setUploadProgress] = useState(0);
  const [message, setMessage] = useState('');
  const [error, setError] = useState('');
  const [dragActive, setDragActive] = useState(false);
  const fileInputRef = useRef(null);

  const handleDrag = (e) => {
    e.preventDefault();
    e.stopPropagation();
    if (e.type === "dragenter" || e.type === "dragover") {
      setDragActive(true);
    } else if (e.type === "dragleave") {
      setDragActive(false);
    }
  };

  const handleDrop = (e) => {
    e.preventDefault();
    e.stopPropagation();
    setDragActive(false);

    if (e.dataTransfer.files && e.dataTransfer.files.length > 0) {
      handleFiles(e.dataTransfer.files);
    }
  };

  const handleFileInput = (e) => {
    if (e.target.files && e.target.files.length > 0) {
      handleFiles(e.target.files);
    }
  };

  const handleFiles = async (files) => {
    setMessage('');
    setError('');
    setUploading(true);
    setUploadProgress(0);

    const formData = new FormData();
    
    // Add all files to the form data
    Array.from(files).forEach(file => {
      formData.append('files', file);
    });

    try {
      await uploadFiles(formData, (progressEvent) => {
        const percentCompleted = Math.round(
          (progressEvent.loaded * 100) / progressEvent.total
        );
        setUploadProgress(percentCompleted);
      });

      setMessage(`Successfully uploaded ${files.length} file(s)!`);
      setUploadProgress(100);
      
      // Reset after 3 seconds
      setTimeout(() => {
        setMessage('');
        setUploadProgress(0);
      }, 3000);
    } catch (err) {
      setError(err.response?.data || 'Upload failed. Please try again.');
      console.error('Upload error:', err);
    } finally {
      setUploading(false);
    }
  };

  const handleButtonClick = () => {
    fileInputRef.current?.click();
  };

  return (
    <div className="upload-container">
      <div className="upload-header">
        <h1>Upload Music</h1>
        <button 
          className="back-btn"
          onClick={() => window.location.href = '/library'}
        >
          ← Back to Library
        </button>
      </div>

      <div
        className={`upload-area ${dragActive ? 'drag-active' : ''}`}
        onDragEnter={handleDrag}
        onDragLeave={handleDrag}
        onDragOver={handleDrag}
        onDrop={handleDrop}
        onClick={handleButtonClick}
      >
        <input
          ref={fileInputRef}
          type="file"
          multiple
          accept="audio/*,.mp3,.flac,.wav,.ogg,.m4a"
          onChange={handleFileInput}
          style={{ display: 'none' }}
        />

        <div className="upload-icon">📁</div>
        <h2>Drag & Drop Music Files</h2>
        <p>or click to browse</p>
        <div className="upload-hint">
          Supported formats: MP3, FLAC, WAV, OGG, M4A
        </div>
      </div>

      {uploading && (
        <div className="upload-progress">
          <div className="progress-label">Uploading... {uploadProgress}%</div>
          <div className="progress-bar-container">
            <div 
              className="progress-bar-fill"
              style={{ width: `${uploadProgress}%` }}
            />
          </div>
        </div>
      )}

      {message && (
        <div className="success-message">{message}</div>
      )}

      {error && (
        <div className="error-message">{error}</div>
      )}

      <div className="upload-instructions">
        <h3>Instructions</h3>
        <ul>
          <li>Select one or multiple audio files to upload</li>
          <li>Supported formats include MP3, FLAC, WAV, OGG, and M4A</li>
          <li>The system will automatically extract metadata from your files</li>
          <li>Large uploads may take a few minutes to process</li>
        </ul>
      </div>
    </div>
  );
}

export default Upload;
