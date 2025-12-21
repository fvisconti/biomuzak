# Docker Deployment Guide

This guide covers deploying biomuzak using Docker and Docker Compose.

## Quick Start

```bash
# Clone the repository
git clone https://github.com/fvisconti/biomuzak.git
cd biomuzak

# Configure environment
cp .env.example .env
nano .env  # Edit configuration

# Start all services
docker compose up -d

# View logs
docker compose logs -f

# Access the application
# Frontend: http://localhost:3000
# Backend API: http://localhost:8080
```

## Architecture

The Docker setup consists of four services:

1. **postgres** - PostgreSQL database with pgvector extension
2. **audio-processor** - Python microservice for audio analysis
3. **backend** - Go API server
4. **frontend** - Nginx serving React application

## Configuration

### Environment Variables

Edit the `.env` file before starting:

```bash
# Database
POSTGRES_USER=musicuser
POSTGRES_PASSWORD=your-secure-password-here
POSTGRES_DB=musicdb

# Backend
JWT_SECRET=generate-with-openssl-rand-base64-32
PORT=8080
UPLOAD_DIR=/root/uploads
AUDIO_PROCESSOR_URL=http://audio-processor:8000

# Database URL (uses above variables)
DATABASE_URL=postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/${POSTGRES_DB}?sslmode=disable
```

### Generate Secure JWT Secret

```bash
openssl rand -base64 32
```

## Services

### PostgreSQL (Database)

**Image**: `ghcr.io/tensorchord/vchord-postgres:pg17-v0.4.2`
**Port**: 5432
**Volume**: `postgres_data`

This image includes PostgreSQL 17 with the pgvector extension pre-installed for vector similarity search.

**Health Check**:
```bash
docker compose exec postgres pg_isready -U musicuser
```

### Audio Processor (Python)

**Build**: `./audio-processor/Dockerfile`
**Port**: 8000
**Dependencies**: Essentia, FastAPI, NumPy

Processes audio files and generates 38-dimensional feature embeddings.

**Test Endpoint**:
```bash
curl http://localhost:8000/
# Should return: {"message":"Audio processing service is running"}
```

### Backend (Go)

**Build**: `./Dockerfile` (multi-stage)
**Port**: 8080
**Volume**: `upload_data` (mounted at `/root/uploads`)

Handles API requests, authentication, database operations, and serves the Subsonic API.

**Health Check**:
```bash
curl http://localhost:8080/api/health
# Should return: {"status":"ok"}
```

### Frontend (React/Nginx)

**Build**: `./frontend/Dockerfile`
**Port**: 3000 (mapped to 80 in container)

Serves the React single-page application via Nginx.

## Volume Management

### Persistent Data

Two volumes store persistent data:

1. **postgres_data**: Database files
2. **upload_data**: Uploaded audio files

**List volumes**:
```bash
docker volume ls | grep biomuzak
```

**Inspect volume**:
```bash
docker volume inspect biomuzak_postgres_data
docker volume inspect biomuzak_upload_data
```

### Backup

**Backup database**:
```bash
# Create backup
docker compose exec postgres pg_dump -U musicuser musicdb > backup_$(date +%Y%m%d).sql

# Restore backup
docker compose exec -T postgres psql -U musicuser musicdb < backup_20240101.sql
```

**Backup audio files**:
```bash
# Find volume location
docker volume inspect biomuzak_upload_data | grep Mountpoint

# Copy files (requires sudo)
sudo cp -r /var/lib/docker/volumes/biomuzak_upload_data/_data ./audio_backup

# Or use docker cp
docker run --rm -v biomuzak_upload_data:/data -v $(pwd):/backup alpine tar czf /backup/audio_backup.tar.gz /data
```

### Clean Up

**Remove containers and networks** (keeps volumes):
```bash
docker compose down
```

**Remove everything including volumes** (⚠️ destroys all data):
```bash
docker compose down -v
```

## Building Images

### Build All Services

```bash
docker compose build
```

### Build Individual Services

```bash
docker compose build backend
docker compose build frontend
docker compose build audio-processor
```

### Build with No Cache

```bash
docker compose build --no-cache
```

## Running Services

### Start All Services

```bash
docker compose up -d
```

### Start Specific Services

```bash
docker compose up -d postgres audio-processor backend
```

### View Logs

```bash
# All services
docker compose logs -f

# Specific service
docker compose logs -f backend

# Last 100 lines
docker compose logs --tail=100 backend
```

### Restart Services

```bash
# Restart all
docker compose restart

# Restart specific service
docker compose restart backend
```

### Stop Services

```bash
docker compose stop
```

## Health Checks

All services include health checks. View status:

```bash
docker compose ps
```

Healthy output:
```
NAME                      STATUS
biomuzak_backend          Up (healthy)
biomuzak_frontend         Up (healthy)
biomuzak_audio_processor  Up (healthy)
music_db                  Up (healthy)
```

## Networking

Services communicate on the `biomuzak-network` bridge network.

**Internal service addresses**:
- `postgres:5432` - Database
- `audio-processor:8000` - Audio processor
- `backend:8080` - Backend API
- `frontend:80` - Frontend

**External access** (from host):
- `localhost:3000` - Frontend
- `localhost:8080` - Backend API
- `localhost:8000` - Audio processor
- `localhost:5432` - Database

## Troubleshooting

### Services Won't Start

**Check logs**:
```bash
docker compose logs
```

**Check individual service**:
```bash
docker compose logs postgres
docker compose logs backend
docker compose logs audio-processor
docker compose logs frontend
```

### Port Conflicts

If ports are already in use, edit `docker compose.yml`:

```yaml
services:
  frontend:
    ports:
      - "8081:80"  # Change 3000 to 8081
```

### Database Connection Issues

**Verify database is running**:
```bash
docker compose ps postgres
```

**Test connection**:
```bash
docker compose exec postgres psql -U musicuser -d musicdb -c "SELECT 1;"
```

**Check environment variables**:
```bash
docker compose exec backend env | grep DATABASE_URL
```

### Backend Can't Connect to Audio Processor

**Verify service is running**:
```bash
docker compose ps audio-processor
```

**Test connection from backend**:
```bash
docker compose exec backend wget -O- http://audio-processor:8000/
```

### Out of Disk Space

**Check disk usage**:
```bash
docker system df
```

**Clean up unused data**:
```bash
# Remove stopped containers, unused networks, dangling images
docker system prune

# Remove unused volumes (⚠️ check before running)
docker volume prune
```

### Reset Everything

**Complete reset** (⚠️ destroys all data):
```bash
docker compose down -v
docker system prune -a
docker compose up -d
```

## Production Deployment

### Security Considerations

1. **Change default passwords**:
   - Set strong `POSTGRES_PASSWORD`
   - Generate secure `JWT_SECRET`

2. **Use secrets management**:
   ```yaml
   # docker compose.yml
   services:
     backend:
       secrets:
         - jwt_secret
   
   secrets:
     jwt_secret:
       file: ./secrets/jwt_secret.txt
   ```

3. **Enable HTTPS**:
   - Use a reverse proxy (nginx, Caddy, Traefik)
   - Obtain SSL certificates (Let's Encrypt)

4. **Restrict network access**:
   - Don't expose database port externally
   - Use firewall rules
   - Consider using a VPN

### Reverse Proxy Setup (Nginx)

Example nginx configuration:

```nginx
server {
    listen 80;
    server_name music.yourdomain.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name music.yourdomain.com;
    
    ssl_certificate /etc/letsencrypt/live/music.yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/music.yourdomain.com/privkey.pem;
    
    # Frontend
    location / {
        proxy_pass http://localhost:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
    
    # Backend API
    location /api/ {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
    
    # Subsonic API
    location /rest/ {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

### Resource Limits

Add resource constraints in `docker compose.yml`:

```yaml
services:
  backend:
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 2G
        reservations:
          cpus: '1'
          memory: 1G
```

### Monitoring

**Check resource usage**:
```bash
docker stats
```

**Export metrics** (Prometheus):
```yaml
services:
  backend:
    labels:
      - "prometheus.scrape=true"
      - "prometheus.port=8080"
```

### Automatic Restarts

Services are configured to restart automatically:

```yaml
services:
  backend:
    restart: unless-stopped
```

### Updates

**Update images**:
```bash
docker compose pull
docker compose up -d
```

**Rebuild after code changes**:
```bash
git pull
docker compose build
docker compose up -d
```

## Performance Tuning

### PostgreSQL

**Increase shared buffers** (1/4 of RAM):
```yaml
services:
  postgres:
    command: postgres -c shared_buffers=2GB -c max_connections=100
```

### Audio Processor

**Scale workers**:
```yaml
services:
  audio-processor:
    command: uvicorn main:app --host 0.0.0.0 --port 8000 --workers 4
```

### Backend

**Run multiple instances** (with load balancer):
```yaml
services:
  backend:
    deploy:
      replicas: 3
```

## Useful Commands

```bash
# View running containers
docker compose ps

# Execute command in container
docker compose exec backend sh

# View resource usage
docker stats

# Inspect service configuration
docker compose config

# Validate docker compose.yml
docker compose config --quiet

# Pull latest images
docker compose pull

# Remove stopped containers
docker compose rm

# Follow logs for all services
docker compose logs -f

# Export logs to file
docker compose logs > logs.txt
```

## Getting Help

If you encounter issues:

1. Check service logs: `docker compose logs [service]`
2. Verify health: `docker compose ps`
3. Test connectivity between services
4. Review this documentation
5. Open an issue: https://github.com/fvisconti/biomuzak/issues
