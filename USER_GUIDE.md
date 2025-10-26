# biomuzak User Guide

Welcome to biomuzak! This guide will help you get started with your self-hosted music manager.

## Table of Contents

1. [Installation](#installation)
2. [Getting Started](#getting-started)
3. [Uploading Music](#uploading-music)
4. [Browsing Your Library](#browsing-your-library)
5. [Creating Playlists](#creating-playlists)
6. [Discovering Similar Songs](#discovering-similar-songs)
7. [Using Mobile Apps](#using-mobile-apps)
8. [Tips & Tricks](#tips--tricks)
9. [Troubleshooting](#troubleshooting)

## Installation

### Option 1: Docker (Recommended)

The easiest way to install biomuzak is using Docker:

1. **Install Docker and Docker Compose**:
   - Docker Desktop (Windows/Mac): https://www.docker.com/products/docker-desktop
   - Docker Engine (Linux): https://docs.docker.com/engine/install/

2. **Download biomuzak**:
   ```bash
   git clone https://github.com/fvisconti/biomuzak.git
   cd biomuzak
   ```

3. **Configure the application**:
   ```bash
   cp .env.example .env
   nano .env  # Edit with your preferred editor
   ```
   
   **Important**: Change these settings:
   - `POSTGRES_PASSWORD`: Choose a strong password
   - `JWT_SECRET`: Generate a random string (run `openssl rand -base64 32`)

4. **Start biomuzak**:
   ```bash
   docker-compose up -d
   ```

5. **Access the web interface**:
   Open your browser and go to http://localhost:3000

### Option 2: Manual Installation

See the [Development Setup](README.md#development-setup) section in the main README.

## Getting Started

### Getting Access

biomuzak supports two onboarding modes:

1. **Invite-only (default)**
   - Public registration is disabled by default.
   - An administrator creates user accounts from the Admin page.
   - Ask your administrator for your username and password.

2. **Public registration (optional)**
   - If enabled by the server (`ALLOW_REGISTRATION=true`), a Register page is available.
   - You can create your own account by providing username, email, and password.

### Logging In

1. Enter your username and password
2. Click **Login**
3. You'll be taken to your music library

### Admin: Inviting Users

If you are an administrator:

1. Login with your admin account
2. Open the Admin page (link in the Library header)
3. Create new users by entering a username and password
   - Passwords are securely hashed on the server
   - Email is optional and defaults to `username@local` if omitted

## Uploading Music

biomuzak supports common audio formats including MP3, FLAC, AAC, OGG, and more.

### Web Upload

1. **Navigate to the upload page**:
   - Click the **Upload** button in the navigation bar
   - Or drag and drop files anywhere on the library page

2. **Select your music files**:
   - Click **Choose Files** or drag files into the upload area
   - You can select multiple files at once
   - Folders are processed recursively

3. **Wait for processing**:
   - Files are uploaded to the server
   - Metadata is extracted automatically
   - Audio features are analyzed (this may take a moment)
   - Progress is shown for each file

4. **View your music**:
   - Once uploaded, songs appear in your library immediately

### Metadata

biomuzak automatically extracts metadata from your audio files:
- **Title** - from ID3 tags or filename
- **Artist** - from ID3 tags
- **Album** - from ID3 tags
- **Year** - from ID3 tags
- **Genre** - enhanced using MusicBrainz database when available
- **Duration** - calculated from file
- **Bitrate** - extracted from file

### Bulk Upload Tips

- Upload entire albums or folders at once
- Well-tagged files provide better organization
- Genre detection works best when artist names are accurate
- Large files (FLAC) take longer to process

## Browsing Your Library

### Library View

The library displays all your music in a sortable, filterable table:

- **Search**: Type in the search box to filter by any field
- **Sort**: Click column headers to sort
- **Select**: Click on a song to play it
- **Rate**: Click stars to rate songs (1-5 stars)

### Filters

Use the filter options to find music:
- **Artist**: View all songs by an artist
- **Album**: Browse complete albums
- **Genre**: Filter by musical genre
- **Year**: Find music from specific years
- **Rating**: Show only your favorite songs

### Playing Music

1. **Click on a song** to start playback
2. Use the **player controls** at the bottom:
   - Play/Pause
   - Previous/Next track
   - Volume control
   - Progress bar (click to seek)
3. The player continues playing as you browse

## Creating Playlists

Playlists help you organize your favorite music.

### Creating a New Playlist

1. Click **Playlists** in the navigation
2. Click **New Playlist**
3. Enter a name and optional description
4. Click **Create**

### Adding Songs to Playlists

**Method 1: From the song**
1. Find a song in your library
2. Click the **•••** (more) button
3. Select **Add to Playlist**
4. Choose the playlist

**Method 2: From the playlist**
1. Open the playlist
2. Click **Add Songs**
3. Select songs from your library
4. Click **Add Selected**

### Managing Playlists

- **Rename**: Click the playlist name to edit
- **Reorder songs**: Drag and drop songs in the playlist
- **Remove songs**: Click the **X** next to a song
- **Delete playlist**: Click **Delete Playlist** (songs remain in your library)

## Discovering Similar Songs

biomuzak uses AI to find songs similar to ones you love.

### How It Works

When you upload a song, biomuzak:
1. Analyzes the audio characteristics
2. Extracts 38 audio features (tempo, timbre, rhythm, etc.)
3. Stores these as a "fingerprint"
4. Compares fingerprints to find similar songs

### Finding Similar Songs

1. **Go to any song** in your library
2. Click the **•••** (more) button
3. Select **Find Similar Songs**
4. View a list of similar songs sorted by similarity score

### Tips for Better Recommendations

- Upload more music for better matches
- Similar songs work best with 50+ tracks in your library
- Works across genres (finds similar sounds, not just genre matches)

## Using Mobile Apps

biomuzak is compatible with Subsonic mobile clients, letting you stream your music on the go.

### Recommended Apps

**Android:**
- **DSub** (Google Play) - Full-featured, active development
- **Ultrasonic** (F-Droid) - Open source, privacy-focused

**iOS:**
- **play:Sub** (App Store) - Native iOS design
- **Amperfy** (App Store) - Modern interface

### Setting Up a Mobile App

1. **Install a Subsonic-compatible app** from your app store

2. **Add your server**:
   - Server Address: `http://your-server-ip:8080/rest`
     - Replace `your-server-ip` with your server's IP address
     - If using a domain: `https://music.yourdomain.com/rest`
   - Username: Your biomuzak username
   - Password: Your biomuzak password
   - Server Name: "biomuzak" (or any name you prefer)

3. **Test the connection**:
   - The app will test connectivity
   - If successful, you'll see your music library

### Remote Access

To access biomuzak outside your home network:

**Option 1: VPN (Recommended)**
- Set up a VPN (WireGuard, Tailscale, etc.)
- Connect to your VPN, then access biomuzak normally
- Most secure option

**Option 2: Reverse Proxy with HTTPS**
- Use nginx or Caddy as a reverse proxy
- Obtain an SSL certificate (Let's Encrypt)
- Configure DNS to point to your server
- Enable authentication

**Option 3: Port Forwarding**
- Forward port 8080 on your router
- Access via public IP: `http://public-ip:8080`
- ⚠️ Less secure - use strong passwords

## Tips & Tricks

### Organizing Your Library

- **Use consistent metadata**: Edit tags before uploading
- **Album folders**: Keep albums in separate folders
- **Naming convention**: `Artist - Album (Year)` folder structure
- **Cover art**: Embed album art in files for best results

### Performance

- **Large libraries**: biomuzak handles thousands of songs
- **First upload**: Initial processing is slower (audio analysis)
- **Subsequent uploads**: Faster as embeddings are generated
- **Search**: Near-instant search even with 10,000+ songs

### Backup Your Data

Your music data is stored in two places:

1. **Audio files**: In the upload directory (default: `./uploads`)
2. **Database**: PostgreSQL database

**Docker backup**:
```bash
# Backup database
docker-compose exec postgres pg_dump -U musicuser musicdb > backup.sql

# Backup audio files
cp -r ./uploads /path/to/backup/
```

### Keyboard Shortcuts

- **Space**: Play/Pause
- **Arrow Right**: Next track
- **Arrow Left**: Previous track
- **Arrow Up**: Volume up
- **Arrow Down**: Volume down
- **/** or **Ctrl+F**: Focus search

## Troubleshooting

### Cannot access biomuzak after installation

1. **Check if services are running**:
   ```bash
   docker-compose ps
   ```
   All services should be "Up"

2. **Check logs**:
   ```bash
   docker-compose logs backend
   docker-compose logs frontend
   ```

3. **Verify ports**:
   - Frontend: http://localhost:3000
   - Backend: http://localhost:8080
   - Ensure ports aren't used by other applications

### Upload fails or hangs

1. **Check file format**: Ensure it's a supported audio format
2. **File size**: Very large files (>500MB) may timeout
3. **Check audio processor**:
   ```bash
   docker-compose logs audio-processor
   ```
4. **Disk space**: Ensure sufficient space for uploads

### Mobile app cannot connect

1. **Verify server address**: Include `/rest` at the end
2. **Check firewall**: Ensure port 8080 is accessible
3. **Test in browser**: Try http://your-server-ip:8080/rest/ping.view
4. **Check credentials**: Username and password are correct

### Similar songs not working

1. **Minimum songs**: Need at least 10-20 songs for good results
2. **Wait for processing**: Embeddings generate after upload
3. **Check audio processor**: Ensure microservice is running
4. **View logs**: Check for errors in processing

### Songs not playing

1. **Check browser console**: Look for JavaScript errors
2. **Audio format**: Browser may not support the codec
3. **File exists**: Verify file wasn't moved or deleted
4. **Permissions**: Check file permissions in upload directory

### Database errors

1. **Check PostgreSQL is running**:
   ```bash
   docker-compose logs postgres
   ```

2. **Verify credentials in .env**:
   - POSTGRES_USER
   - POSTGRES_PASSWORD
   - POSTGRES_DB

3. **Reset database** (⚠️ destroys all data):
   ```bash
   docker-compose down -v
   docker-compose up -d
   ```

### Getting Help

If you encounter issues:

1. **Check the logs**: `docker-compose logs` shows all service logs
2. **GitHub Issues**: https://github.com/fvisconti/biomuzak/issues
3. **Documentation**: Review README.md for configuration details

## Support

For questions, bug reports, or feature requests:
- GitHub Issues: https://github.com/fvisconti/biomuzak/issues
- Read the [Developer Documentation](README.md)

Enjoy your music! 🎵
