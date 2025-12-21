# biomuzak

A modern, self-hosted music organizer, player, and streamer with AI-powered music discovery.

## 🎵 Features

- **Music Library Management**: Upload, organize, and browse your music collection
- **Advanced Search & Filtering**: Find songs by title, artist, album, genre, or year
- **Music Playback**: Built-in audio player with volume control
- **Song Rating**: Rate your favorite tracks
- **AI-Powered Similar Songs**: Discover music similar to your favorites using audio feature embeddings
- **Playlist Management**: Create and organize custom playlists
- **Drag & Drop Upload**: Easy file upload with progress tracking
- **User Authentication**: Secure JWT-based authentication
- **Subsonic API**: Compatible with mobile Subsonic clients (DSub, Ultrasonic, etc.)
- **Responsive Design**: Works on desktop, tablet, and mobile devices
- **Containerized Deployment**: Easy deployment with Docker and Docker Compose

## 📋 Table of Contents

- [Architecture](#architecture)
- [Prerequisites](#prerequisites)
- [Quick Start with Docker](#quick-start-with-docker)
- [Development Setup](#development-setup)
- [API Documentation](#api-documentation)
- [Testing](#testing)
- [Configuration](#configuration)
- [Contributing](#contributing)
- [License](#license)

## 🏗️ Architecture

biomuzak consists of four main components:

```
┌─────────────────────────────────────────────────────────────────┐
│                         User / Client                            │
│          (Web Browser / Subsonic Mobile Apps)                    │
└─────────────────┬───────────────────────────────┬────────────────┘
                  │                               │
                  ▼                               ▼
      ┌────────────────────┐          ┌────────────────────┐
      │  Frontend (React)  │          │  Subsonic Clients  │
      │   Nginx (Port 80)  │          │  (DSub, etc.)      │
      └──────────┬─────────┘          └─────────┬──────────┘
                 │                              │
                 │          HTTP/REST           │
                 └──────────────┬───────────────┘
                                │
                                ▼
                  ┌──────────────────────────┐
                  │   Go Backend Server      │
                  │   (Port 8080)            │
                  │                          │
                  │  - RESTful API           │
                  │  - Subsonic API          │
                  │  - Authentication        │
                  │  - File Management       │
                  └─────┬─────────────┬──────┘
                        │             │
            ┌───────────┘             └──────────┐
            │                                    │
            ▼                                    ▼
  ┌──────────────────┐              ┌──────────────────────┐
  │   PostgreSQL     │              │  Audio Processor     │
  │   with pgvector  │              │  (Python/Essentia)   │
  │   (Port 5432)    │              │  (Port 8000)         │
  │                  │              │                      │
  │  - Song metadata │              │  - Feature extraction│
  │  - User data     │              │  - 38D embeddings    │
  │  - Playlists     │              │  - Audio analysis    │
  │  - Embeddings    │              └──────────────────────┘
  └──────────────────┘
```

### Component Details

1. **Frontend (React)**: User interface for browsing, playing, and managing music
2. **Go Backend**: RESTful API server with Subsonic API compatibility
3. **PostgreSQL with pgvector**: Stores metadata and vector embeddings for similarity search
4. **Audio Processor (Python)**: Microservice that generates audio feature embeddings using Essentia

## 📦 Prerequisites

### For Docker Deployment (Recommended)
- Docker 20.10+
- Docker Compose 2.0+

### For Development
- Go 1.24+
- Node.js 18+
- PostgreSQL 16+ with pgvector extension
- Python 3.12+

## 🚀 Quick Start with Docker

The easiest way to run biomuzak is with Docker Compose:

1. **Clone the repository**:
```bash
git clone https://github.com/fvisconti/biomuzak.git
cd biomuzak
```

2. **Configure environment variables**:
```bash
cp .env.example .env
# Edit .env and set secure passwords and JWT secret
nano .env  # or use your favorite editor
```

3. **Start all services**:
```bash
docker compose up -d
```

4. **Access the application**:
- Frontend: http://localhost:3000
- Backend API: http://localhost:8080
- Audio Processor: http://localhost:8000

5. **Bootstrap the initial admin user**:

Edit `.env` and set these variables before the first start (when the database is empty):

```
ALLOW_REGISTRATION=false
ADMIN_USERNAME=admin
ADMIN_PASSWORD=your-secure-password
ADMIN_EMAIL=admin@example.com
```

On first startup, the backend will create the admin account and disable public registration. Log in via the web UI with the admin credentials.

6. **Stop the services**:
```bash
docker compose down
```

To remove all data including the database:
```bash
docker compose down -v
```

## 💻 Development Setup

## 💻 Development Setup

### Backend (Go)

1. **Clone and navigate to the repository**:
```bash
git clone https://github.com/fvisconti/biomuzak.git
cd biomuzak
```

2. **Install PostgreSQL with pgvector**:
```bash
# macOS with Homebrew
brew install postgresql pgvector

# Ubuntu/Debian
sudo apt-get install postgresql postgresql-contrib
# Install pgvector from https://github.com/pgvector/pgvector

# Or use Docker
docker run -d \
  --name biomuzak-postgres \
  -e POSTGRES_PASSWORD=musicpass \
  -e POSTGRES_USER=musicuser \
  -e POSTGRES_DB=musicdb \
  -p 5432:5432 \
  ghcr.io/tensorchord/vchord-postgres:pg17-v0.4.2
```

3. **Set up environment variables**:
```bash
cp .env.example .env
# Edit .env with your database credentials
```

4. **Install Go dependencies**:
```bash
go mod download
```

5. **Run database migrations** (automatic on first run):
```bash
go run cmd/server/main.go
```

6. **Start the backend server**:
```bash
go run cmd/server/main.go
# Server will start on http://localhost:8080
```

### Frontend (React)

1. **Navigate to frontend directory**:
```bash
cd frontend
```

2. **Install dependencies**:
```bash
npm install
```

3. **Start development server**:
```bash
npm start
# Opens browser at http://localhost:3000
```

4. **Build for production**:
```bash
npm run build
# Creates optimized build in frontend/build/
```

### Audio Processor (Python)

1. **Navigate to audio-processor directory**:
```bash
cd audio-processor
```

2. **Install dependencies**:
```bash
pip install uv
uv pip install .
```

3. **Start the service**:
```bash
uvicorn main:app --host 0.0.0.0 --port 8000
# Service will start on http://localhost:8000
```

## 📚 API Documentation

### Authentication Endpoints

#### Register User (may be disabled)
```http
POST /register
Content-Type: application/json

{
  "username": "user",
  "email": "user@example.com",
  "password": "secure-password"
}

Notes:
- Returns 403 Forbidden when `ALLOW_REGISTRATION=false` (default). Use the admin endpoint to invite users instead.
```

#### Login
```http
POST /login
Content-Type: application/json

{
  "username": "user",
  "password": "secure-password"
}

Response:
{
  "token": "jwt-token-here"
}
```

#### Current User
```http
GET /api/me
Authorization: Bearer <token>

Response:
{
  "id": 1,
  "username": "admin",
  "is_admin": true
}
```

#### Admin: Create User
```http
POST /api/admin/users
Authorization: Bearer <token> (admin only)
Content-Type: application/json

{
  "username": "newuser",
  "password": "secure-password",
  "email": "newuser@example.com"  // optional; defaults to username@local
}

Response: 201 Created
{
  "message": "User created successfully"
}
```

### Music Library Endpoints

All library endpoints require authentication via `Authorization: Bearer <token>` header.

#### Get Library
```http
GET /api/library
Authorization: Bearer <token>

Response:
{
  "songs": [
    {
      "id": 1,
      "title": "Song Title",
      "artist": "Artist Name",
      "album": "Album Name",
      "year": 2023,
      "genre": "Rock",
      "duration": 240,
      "bitrate": 320,
      "file_size": 8000000,
      "rating": 5
    }
  ]
}
```

#### Upload Song
```http
POST /api/upload
Authorization: Bearer <token>
Content-Type: multipart/form-data

Form Data:
- file: audio file (MP3, FLAC, etc.)
```

#### Rate Song
```http
POST /api/songs/{songID}/rate
Authorization: Bearer <token>
Content-Type: application/json

{
  "rating": 5
}
```

#### Get Similar Songs
```http
GET /api/songs/{songID}/similar
Authorization: Bearer <token>

Response:
{
  "similar_songs": [
    {
      "id": 2,
      "title": "Similar Song",
      "artist": "Artist",
      "similarity": 0.95
    }
  ]
}
```

### Playlist Endpoints

#### Create Playlist
```http
POST /api/playlists
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "My Playlist",
  "description": "My favorite songs"
}
```

#### Get User Playlists
```http
GET /api/playlists
Authorization: Bearer <token>
```

#### Get Playlist
```http
GET /api/playlists/{playlistID}
Authorization: Bearer <token>
```

#### Add Song to Playlist
```http
POST /api/playlists/{playlistID}/songs
Authorization: Bearer <token>
Content-Type: application/json

{
  "song_id": 1
}
```

#### Remove Song from Playlist
```http
DELETE /api/playlists/{playlistID}/songs/{songID}
Authorization: Bearer <token>
```

### Subsonic API

biomuzak implements the Subsonic API for compatibility with mobile clients. All Subsonic endpoints are available under `/rest/`.

Example endpoints:
- `/rest/ping.view` - Check server status
- `/rest/getMusicFolders.view` - Get music folders
- `/rest/getIndexes.view` - Get artist index
- `/rest/search3.view` - Search for music
- `/rest/stream.view` - Stream audio

For full Subsonic API documentation, visit: http://www.subsonic.org/pages/api.jsp

## 🧪 Testing

### Go Backend Tests
```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with verbose output
go test -v ./...

# Run specific package tests
go test ./pkg/auth -v
go test ./pkg/handlers -v
go test ./pkg/db -v
```

### Python Audio Processor Tests
```bash
cd audio-processor
pip install ".[test]"
pytest test_main.py -v
```

### Frontend Tests
```bash
cd frontend
npm test
```

### Integration Tests

Run the full integration test suite to verify all components work together:

```bash
# Start all services first
docker compose up -d

# Wait for services to be ready
sleep 10

# Run integration tests
./test-integration.sh

# Stop services
docker compose down
```

The integration test script tests:
- Backend health endpoint
- Audio processor health endpoint
- Frontend accessibility
- User registration and login
- Protected API endpoint access with JWT
- Subsonic API compatibility

## ⚙️ Configuration

Configuration is managed through environment variables. See `.env.example` for all available options.

### Required Variables

- `DATABASE_URL`: PostgreSQL connection string
- `JWT_SECRET`: Secret key for JWT token signing (use a strong random string in production)
- `PORT`: Port for the backend server (default: 8080)

### Optional Variables

- `UPLOAD_DIR`: Directory for uploaded audio files (default: ./uploads)
- `AUDIO_PROCESSOR_URL`: URL of the audio processor service (default: http://localhost:8000)
- `ALLOW_REGISTRATION`: Enables public registration endpoint when `true` (default: `false`)
- `ADMIN_USERNAME`, `ADMIN_PASSWORD`, `ADMIN_EMAIL`: Used once to bootstrap the first admin user if the users table is empty
- `POSTGRES_USER`: PostgreSQL username (for Docker)
- `POSTGRES_PASSWORD`: PostgreSQL password (for Docker)
- `POSTGRES_DB`: PostgreSQL database name (for Docker)

### Production Configuration

For production deployments:

1. **Use strong, unique passwords** for all services
2. **Generate a secure JWT secret**: `openssl rand -base64 32`
3. **Enable HTTPS** with a reverse proxy (nginx, Caddy)
4. **Set up regular backups** of the PostgreSQL database
5. **Configure volume persistence** in docker compose.yml
6. **Use Docker secrets** or environment file management for sensitive data

## 📱 Using with Mobile Clients

biomuzak is compatible with Subsonic-compatible mobile apps:

### Android
- **DSub**: https://play.google.com/store/apps/details?id=github.daneren2005.dsub
- **Ultrasonic**: https://f-droid.org/packages/org.moire.ultrasonic/

### iOS
- **play:Sub**: https://apps.apple.com/app/play-sub/id955329386
- **Amperfy**: https://apps.apple.com/app/amperfy/id1530723492

### Configuration
1. Server: `http://your-server:8080/rest`
2. Username: Your biomuzak username
3. Password: Your biomuzak password

## 🤝 Contributing

Contributions are welcome! Please follow these steps:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Guidelines

- Write tests for new features
- Follow existing code style
- Update documentation for API changes
- Test with Docker before submitting PR

## 📄 License

See LICENSE file for details.

## 🙏 Acknowledgments

- [Essentia](https://essentia.upf.edu/) for audio analysis
- [pgvector](https://github.com/pgvector/pgvector) for vector similarity search
- [Subsonic API](http://www.subsonic.org/) for mobile client compatibility
