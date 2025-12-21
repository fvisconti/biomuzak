# Contributing to biomuzak

Thank you for your interest in contributing to biomuzak! This document provides guidelines and instructions for contributing.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Workflow](#development-workflow)
- [Coding Standards](#coding-standards)
- [Testing Guidelines](#testing-guidelines)
- [Pull Request Process](#pull-request-process)
- [Project Structure](#project-structure)
- [Architecture Decisions](#architecture-decisions)

## Code of Conduct

- Be respectful and inclusive
- Provide constructive feedback
- Focus on what is best for the community
- Show empathy towards other contributors

## Getting Started

### Prerequisites

- Go 1.24+
- Node.js 18+
- PostgreSQL 16+ with pgvector
- Python 3.12+
- Docker and Docker Compose (for testing)

### Setting Up Development Environment

1. **Fork and clone the repository**:
   ```bash
   git clone https://github.com/YOUR_USERNAME/biomuzak.git
   cd biomuzak
   ```

2. **Set up the backend**:
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   go mod download
   ```

3. **Set up the frontend**:
   ```bash
   cd frontend
   npm install
   ```

4. **Set up the audio processor**:
   ```bash
   cd audio-processor
   pip install uv
   uv pip install ".[test]"
   ```

5. **Start development services**:
   ```bash
   # Terminal 1: Database
   docker compose up postgres
   
   # Terminal 2: Audio processor
   cd audio-processor
   uvicorn main:app --reload
   
   # Terminal 3: Backend
   go run cmd/server/main.go
   
   # Terminal 4: Frontend
   cd frontend
   npm start
   ```

## Development Workflow

### Making Changes

1. **Create a feature branch**:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes**:
   - Write clear, self-documenting code
   - Add tests for new functionality
   - Update documentation as needed

3. **Test your changes**:
   ```bash
   # Go tests
   go test ./...
   
   # Python tests
   cd audio-processor
   pytest
   
   # Frontend tests
   cd frontend
   npm test
   ```

4. **Commit your changes**:
   ```bash
   git add .
   git commit -m "feat: add new feature"
   ```

### Commit Message Format

Follow the [Conventional Commits](https://www.conventionalcommits.org/) specification:

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Maintenance tasks

**Examples:**
```
feat(api): add playlist sharing endpoint
fix(upload): handle large file uploads correctly
docs(readme): update installation instructions
test(auth): add JWT validation tests
```

## Coding Standards

### Go Code

- Follow the [Effective Go](https://golang.org/doc/effective_go.html) guidelines
- Use `gofmt` for formatting: `go fmt ./...`
- Run `go vet` to check for common mistakes: `go vet ./...`
- Keep functions small and focused (prefer < 50 lines)
- Use meaningful variable names
- Add comments for exported functions and types
- Handle errors explicitly (don't ignore them)

**Example:**
```go
// GetSongByID retrieves a song by its ID from the database.
func GetSongByID(db *sql.DB, id int) (*models.Song, error) {
    query := "SELECT id, title, artist FROM songs WHERE id = $1"
    var song models.Song
    
    err := db.QueryRow(query, id).Scan(&song.ID, &song.Title, &song.Artist)
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, fmt.Errorf("song not found: %d", id)
        }
        return nil, fmt.Errorf("database error: %w", err)
    }
    
    return &song, nil
}
```

### Python Code

- Follow [PEP 8](https://www.python.org/dev/peps/pep-0008/)
- Use type hints where possible
- Use docstrings for functions and classes
- Keep functions focused and small

**Example:**
```python
def normalize(v: np.ndarray) -> np.ndarray:
    """
    Normalize a vector to the 0-1 range.
    
    Args:
        v: Input vector to normalize
        
    Returns:
        Normalized vector with values between 0 and 1
    """
    if (np.max(v) - np.min(v)) == 0:
        return v - np.min(v)
    return (v - np.min(v)) / (np.max(v) - np.min(v))
```

### JavaScript/React Code

- Use functional components with hooks
- Follow [Airbnb JavaScript Style Guide](https://github.com/airbnb/javascript)
- Use ESLint: `npm run lint`
- Use meaningful component and variable names
- Keep components small and focused

**Example:**
```javascript
// Good: Functional component with hooks
const SongList = ({ songs, onSongClick }) => {
  const [filter, setFilter] = useState('');
  
  const filteredSongs = useMemo(() => 
    songs.filter(song => 
      song.title.toLowerCase().includes(filter.toLowerCase())
    ),
    [songs, filter]
  );
  
  return (
    <div>
      <SearchBar value={filter} onChange={setFilter} />
      {filteredSongs.map(song => (
        <SongItem key={song.id} song={song} onClick={onSongClick} />
      ))}
    </div>
  );
};
```

## Testing Guidelines

### Go Testing

- Write tests for all new functions
- Use table-driven tests for multiple test cases
- Mock external dependencies (database, HTTP calls)
- Aim for >80% code coverage

**Example:**
```go
func TestHashPassword(t *testing.T) {
    tests := []struct {
        name     string
        password string
        wantErr  bool
    }{
        {
            name:     "valid password",
            password: "secure123",
            wantErr:  false,
        },
        {
            name:     "empty password",
            password: "",
            wantErr:  true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            hash, err := HashPassword(tt.password)
            if (err != nil) != tt.wantErr {
                t.Errorf("HashPassword() error = %v, wantErr %v", err, tt.wantErr)
            }
            if !tt.wantErr && hash == "" {
                t.Error("expected non-empty hash")
            }
        })
    }
}
```

### Python Testing

- Use pytest for testing
- Mock external dependencies (file I/O, audio processing)
- Test edge cases and error conditions

**Example:**
```python
@patch('main.es.MonoLoader')
def test_process_audio_success(mock_loader_class):
    """Test successful audio processing"""
    mock_loader = Mock()
    mock_audio = np.random.rand(44100)
    mock_loader.return_value = mock_audio
    mock_loader_class.return_value = mock_loader
    
    files = {"file": ("test.mp3", io.BytesIO(b"mock"), "audio/mpeg")}
    response = client.post("/process-audio/", files=files)
    
    assert response.status_code == 200
    assert "embedding" in response.json()
```

### Integration Testing

When adding features that span multiple components:

1. Write unit tests for each component
2. Add integration tests using docker compose
3. Test the full workflow (e.g., upload → process → store → retrieve)

## Pull Request Process

### Before Submitting

1. **Update documentation**:
   - Update README.md if you changed APIs
   - Add inline code comments
   - Update USER_GUIDE.md for user-facing changes

2. **Run all tests**:
   ```bash
   # Go tests
   go test ./...
   
   # Python tests
   cd audio-processor && pytest
   
   # Frontend tests
   cd frontend && npm test
   ```

3. **Test with Docker**:
   ```bash
   docker compose build
   docker compose up -d
   # Test the full application
   docker compose down
   ```

4. **Check formatting**:
   ```bash
   go fmt ./...
   ```

### Submitting a Pull Request

1. **Push your branch**:
   ```bash
   git push origin feature/your-feature-name
   ```

2. **Open a pull request**:
   - Use a clear, descriptive title
   - Reference any related issues
   - Describe what changed and why
   - Include screenshots for UI changes
   - List any breaking changes

3. **PR template**:
   ```markdown
   ## Description
   Brief description of the changes
   
   ## Type of Change
   - [ ] Bug fix
   - [ ] New feature
   - [ ] Breaking change
   - [ ] Documentation update
   
   ## Testing
   - [ ] Unit tests pass
   - [ ] Integration tests pass
   - [ ] Tested with Docker
   
   ## Screenshots (if applicable)
   
   ## Related Issues
   Closes #123
   ```

### Review Process

- A maintainer will review your PR
- Address feedback and push updates
- Once approved, your PR will be merged
- Maintainers may request changes or additional tests

## Project Structure

```
biomuzak/
├── cmd/
│   └── server/          # Main application entry point
├── pkg/
│   ├── auth/           # Authentication (JWT, passwords)
│   ├── config/         # Configuration management
│   ├── db/             # Database operations
│   ├── handlers/       # HTTP request handlers
│   ├── middleware/     # HTTP middleware
│   ├── models/         # Data models
│   ├── metadata/       # Audio metadata extraction
│   ├── musicbrainz/    # MusicBrainz API client
│   ├── router/         # HTTP routing
│   └── subsonic/       # Subsonic API implementation
├── db/
│   └── migrations/     # Database migration files
├── frontend/
│   ├── src/
│   │   ├── components/ # React components
│   │   └── api.js      # API client
│   └── public/         # Static assets
├── audio-processor/
│   ├── main.py         # FastAPI application
│   └── test_main.py    # Tests
├── Dockerfile          # Go backend Docker image
├── docker compose.yml  # Full stack orchestration
└── README.md          # Main documentation
```

## Architecture Decisions

### Why Go?
- Fast compilation and execution
- Great for concurrent operations (file uploads, streaming)
- Strong typing and excellent standard library
- Easy deployment (single binary)

### Why PostgreSQL with pgvector?
- Reliable, mature database
- pgvector enables efficient similarity search
- Strong ACID guarantees
- Excellent Go driver support

### Why React?
- Component-based architecture
- Large ecosystem
- Good performance
- Easy to understand and maintain

### Why Python for Audio Processing?
- Essentia library (industry-standard audio analysis)
- NumPy for efficient numerical operations
- Separating audio processing allows independent scaling

### Why Microservices?
- Audio processing is CPU-intensive
- Can scale independently
- Isolated failures (audio processing won't crash main app)
- Easy to replace with alternative implementations

## Questions?

If you have questions or need help:

1. Check existing issues: https://github.com/fvisconti/biomuzak/issues
2. Open a new issue with the "question" label
3. Join discussions in pull requests

Thank you for contributing to biomuzak! 🎵
