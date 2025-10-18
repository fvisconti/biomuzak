# Copilot Instructions for biomuzak

## Project Overview

biomuzak is a music organizer, player, and streamer application consisting of:
- **Go backend server**: RESTful API with Subsonic API compatibility
- **PostgreSQL database**: For storing music library metadata, playlists, and user data
- **Python audio processor microservice**: Generates feature embeddings using Essentia library

## Architecture

### Backend (Go)
- **Entry point**: `cmd/server/main.go`
- **Package structure**:
  - `pkg/auth`: JWT-based authentication
  - `pkg/handlers`: HTTP request handlers (auth, library, playlist, song, upload)
  - `pkg/db`: Database connection and migrations
  - `pkg/models`: Data models
  - `pkg/metadata`: Audio file metadata extraction and genre detection
  - `pkg/musicbrainz`: MusicBrainz API integration
  - `pkg/router`: HTTP routing setup
  - `pkg/subsonic`: Subsonic API compatibility layer
  - `pkg/middleware`: HTTP middleware components
  - `pkg/config`: Configuration management

### Audio Processor (Python)
- **Location**: `audio-processor/`
- **Purpose**: Generates 38-dimensional feature embedding vectors from audio files
- **Technology**: FastAPI with Essentia library
- **Deployment**: Dockerized microservice

## Development Guidelines

### Go Code Style
1. **Testing**: Write tests for new handlers and business logic in `*_test.go` files
2. **Error Handling**: Always check and handle errors explicitly
3. **Database Operations**: Use the pgx library for PostgreSQL interactions
4. **Logging**: Use the standard `log` package for logging
5. **Module Name**: The module is named `go-postgres-example` in `go.mod` and import paths (historical artifact from project template - maintained for backward compatibility)

### Code Organization
- Place HTTP handlers in `pkg/handlers/`
- Put data models in `pkg/models/`
- Keep database-related code in `pkg/db/`
- Middleware goes in `pkg/middleware/`

### Testing
- Run tests with: `go test ./...`
- Test files should be co-located with the code they test
- Use table-driven tests where appropriate
- Mock database interactions using appropriate testing patterns

### Database
- Migrations are stored in `db/migrations/`
- The application runs migrations on startup
- Use parameterized queries to prevent SQL injection

### API Design
- RESTful endpoints for the main API
- Subsonic API compatibility through `/rest/` endpoints
- JWT authentication for protected endpoints
- Support for multipart file uploads for audio files

### Configuration
- Environment variables are loaded from `.env` file (see `.env.example`)
- Required variables:
  - `DATABASE_URL`: PostgreSQL connection string
  - `PORT`: Server port (default: 8080)
  - `JWT_SECRET`: Secret key for JWT signing
  - `UPLOAD_DIR`: Directory for uploaded audio files
  - `AUDIO_PROCESSOR_URL`: URL of the audio processor microservice

### Audio Processing
- Supported formats: MP3, FLAC, and other common audio formats
- Metadata extraction using the `github.com/dhowden/tag` library
- Genre detection via MusicBrainz API with fallback to file metadata
- File hashing to prevent duplicates

## Common Tasks

### Adding a new API endpoint
1. Create or update handler in `pkg/handlers/`
2. Add route in `pkg/router/`
3. Add authentication middleware if needed
4. Write tests for the handler
5. Update API documentation if it exists

### Adding a new database model
1. Define struct in `pkg/models/`
2. Create migration in `db/migrations/`
3. Add CRUD operations in appropriate handler
4. Write tests for database operations

### Working with metadata
- Use `pkg/metadata` package for audio file processing
- Integrate with MusicBrainz for enhanced metadata
- Handle missing or incorrect metadata gracefully

## Dependencies

### Go Dependencies
- `github.com/go-chi/chi/v5`: HTTP router
- `github.com/jackc/pgx/v4`: PostgreSQL driver
- `github.com/golang-jwt/jwt/v5`: JWT authentication
- `github.com/dhowden/tag`: Audio metadata extraction
- `github.com/michiwend/gomusicbrainz`: MusicBrainz API client
- `github.com/stretchr/testify`: Testing utilities

### Python Dependencies (Audio Processor)
- FastAPI: Web framework
- Essentia: Audio analysis library
- Management via `pyproject.toml`

## Running the Application

### Local Development
1. Set up PostgreSQL database
2. Copy `.env.example` to `.env` and configure
3. Build and run audio processor: `docker-compose up audio-processor`
4. Run the Go server: `go run cmd/server/main.go`

### Docker Deployment
- Use `docker-compose.yml` for complete stack deployment
- Includes PostgreSQL, Go server, and audio processor

## Important Notes

- Always validate user input before processing
- Handle file uploads securely (validate file types, limit sizes)
- Use prepared statements for all database queries
- Log errors appropriately but don't expose sensitive information to clients
- The project uses Go 1.24.3
- Database migrations run automatically on server startup
- Audio files are stored on disk, with metadata in PostgreSQL
