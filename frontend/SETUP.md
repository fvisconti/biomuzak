# Frontend Setup Guide

This is the web-based user interface for the Biomuzak music manager.

## Prerequisites

- Node.js (v14 or higher)
- npm (v6 or higher)

## Installation

1. Install dependencies:
```bash
cd frontend
npm install
```

2. Configure the API endpoint:
```bash
cp .env.example .env
```

Edit `.env` and set the `REACT_APP_API_URL` to your backend server URL:
```
REACT_APP_API_URL=http://localhost:8080
```

## Development

Run the development server:
```bash
npm start
```

This will start the app at `http://localhost:3000` and proxy API requests to your backend.

## Production Build

Build the production-ready static files:
```bash
npm run build
```

The built files will be in the `build/` directory. These files are served by the Go backend automatically.

## Features

### Authentication
- **Login**: Access your music library with your credentials
- **Register**: Create a new account

### Library Management
- **Browse**: View all songs in your library
- **Search**: Real-time search by title, artist, or album
- **Filter**: Filter by genre, artist, or year
- **Sort**: Sort by title, artist, or rating

### Music Playback
- **Player Bar**: Persistent audio player at the bottom of the screen
- **Controls**: Play/pause, skip tracks, volume control
- **Similar Songs**: Automatically displays 3 most similar songs when playing
- **Rating**: Rate songs with a 5-star rating system

### File Upload
- **Drag & Drop**: Easy drag-and-drop interface for uploading music
- **Batch Upload**: Upload multiple files at once
- **Progress Tracking**: Real-time upload progress indicator
- **Supported Formats**: MP3, FLAC, WAV, OGG, M4A

### Playlists
- **Create**: Create custom playlists
- **Manage**: Add or remove songs from playlists
- **Browse**: View all your playlists and their songs
- **Delete**: Remove playlists you no longer need

## Architecture

The frontend is built with:
- **React**: Component-based UI framework
- **React Router**: Client-side routing
- **Axios**: HTTP client for API communication
- **CSS**: Custom responsive styling

## API Integration

The app communicates with the backend through the following endpoints:

- `POST /register` - User registration
- `POST /login` - User authentication
- `GET /api/library` - Fetch user's music library
- `POST /api/upload` - Upload music files
- `POST /api/songs/{id}/rate` - Rate a song
- `GET /api/songs/{id}/similar` - Get similar songs
- `GET /api/playlists` - List all playlists
- `POST /api/playlists` - Create a new playlist
- `GET /api/playlists/{id}` - Get playlist details
- `POST /api/playlists/{id}/songs` - Add song to playlist
- `DELETE /api/playlists/{id}/songs/{songId}` - Remove song from playlist
- `DELETE /api/playlists/{id}` - Delete playlist

## Responsive Design

The application is fully responsive and works on:
- Desktop computers
- Tablets
- Mobile devices

## Browser Support

- Chrome (latest)
- Firefox (latest)
- Safari (latest)
- Edge (latest)
