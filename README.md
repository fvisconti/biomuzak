# biomuzak
music organizer, player, streamer

A modern web-based music manager with a powerful Go backend and React frontend.

## Features

- 🎵 **Music Library Management**: Upload, organize, and browse your music collection
- 🔍 **Advanced Search & Filtering**: Find songs by title, artist, album, genre, or year
- 🎵 **Music Playback**: Built-in audio player with volume control
- ⭐ **Song Rating**: Rate your favorite tracks
- 🤖 **Similar Songs Discovery**: AI-powered recommendations based on audio features
- 📋 **Playlist Management**: Create and organize custom playlists
- 📤 **Drag & Drop Upload**: Easy file upload with progress tracking
- 🔐 **User Authentication**: Secure JWT-based authentication
- 📱 **Responsive Design**: Works on desktop, tablet, and mobile devices

## Quick Start

### Prerequisites

- Go 1.24 or higher
- PostgreSQL
- Node.js 14+ (for frontend development)

### Backend Setup

1. Clone the repository:
```bash
git clone https://github.com/fvisconti/biomuzak.git
cd biomuzak
```

2. Set up environment variables:
```bash
cp .env.example .env
# Edit .env with your database credentials and JWT secret
```

3. Run database migrations:
```bash
go run cmd/server/main.go
```

### Frontend Setup

1. Install frontend dependencies:
```bash
cd frontend
npm install
```

2. For development, start the dev server:
```bash
npm start
```

3. For production, build the static files:
```bash
npm run build
```

The Go backend automatically serves the built frontend from `frontend/build/`.

### Running the Application

1. Start the backend server:
```bash
go run cmd/server/main.go
```

2. Access the application at `http://localhost:8080`

## Project Structure

```
biomuzak/
├── cmd/server/          # Main application entry point
├── pkg/                 # Go packages
│   ├── auth/           # Authentication logic
│   ├── handlers/       # HTTP handlers
│   ├── db/             # Database operations
│   ├── models/         # Data models
│   ├── router/         # HTTP routing
│   └── ...
├── frontend/           # React frontend application
│   ├── src/
│   │   ├── components/ # React components
│   │   ├── api.js      # API client
│   │   └── ...
│   └── build/          # Production build (generated)
└── db/migrations/      # Database migrations
```

## API Documentation

The backend provides RESTful APIs for:

- User authentication (`/register`, `/login`)
- Library management (`/api/library`)
- Song operations (`/api/songs/{id}/rate`, `/api/songs/{id}/similar`)
- File uploads (`/api/upload`)
- Playlist management (`/api/playlists`)
- Subsonic API compatibility (`/rest/*`)

See `frontend/SETUP.md` for detailed API documentation.

## Testing

Run Go tests:
```bash
go test ./...
```

Run frontend tests:
```bash
cd frontend
npm test
```

## License

See LICENSE file for details.
