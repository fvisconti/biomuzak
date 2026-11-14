package subsonic

import "encoding/xml"

// Response is the top-level response object for all Subsonic API calls
type Response struct {
	XMLName          xml.Name            `xml:"subsonic-response"`
	Status           string              `xml:"status,attr"`
	Version          string              `xml:"version,attr"`
	XMLNS            string              `xml:"xmlns,attr"`
	Error            *Error              `xml:"error,omitempty"`
	MusicFolders     *MusicFolders       `xml:"musicFolders,omitempty"`
	Indexes          *Indexes            `xml:"indexes,omitempty"`
	SearchResult3    *SearchResult3      `xml:"searchResult3,omitempty"`
	AlbumList        *AlbumList          `xml:"albumList,omitempty"`
	Album            *Album              `xml:"album,omitempty"`
	ArtistList       *ArtistList         `xml:"artists,omitempty"`
	Artist           *Artist             `xml:"artist,omitempty"`
	SimilarSongs     *SimilarSongsResult `xml:"similarSongs,omitempty"`
	RandomSongs      *RandomSongsResult  `xml:"randomSongs,omitempty"`
	PlaylistList     *PlaylistList       `xml:"playlists,omitempty"`
	SubsonicPlaylist *SubsonicPlaylist   `xml:"playlist,omitempty"`
}

// Error represents an error returned by the Subsonic API
type Error struct {
	Code    int    `xml:"code,attr"`
	Message string `xml:"message,attr"`
}

// MusicFolders is a container for MusicFolder elements
type MusicFolders struct {
	XMLName      xml.Name      `xml:"musicFolders"`
	MusicFolders []MusicFolder `xml:"musicFolder"`
}

// MusicFolder represents a single music folder
type MusicFolder struct {
	ID   int    `xml:"id,attr"`
	Name string `xml:"name,attr"`
}

// Indexes is a container for Index elements
type Indexes struct {
	XMLName xml.Name `xml:"indexes"`
	Indexes []Index  `xml:"index"`
}

// Index represents a single index (e.g., "A", "B", "C")
type Index struct {
	Name    string   `xml:"name,attr"`
	Artists []Artist `xml:"artist"`
}

// Artist represents a single artist. For getArtist it carries its albums.
type Artist struct {
	ID         string  `xml:"id,attr"`
	Name       string  `xml:"name,attr"`
	AlbumCount int     `xml:"albumCount,attr,omitempty"`
	Albums     []Album `xml:"album,omitempty"`
}

// SearchResult3 is a container for search results
type SearchResult3 struct {
	XMLName xml.Name `xml:"searchResult3"`
	Artists []Artist `xml:"artist"`
	Albums  []Album  `xml:"album"`
	Songs   []Song   `xml:"song"`
}

// Album represents a single album
type Album struct {
	ID            string `xml:"id,attr"`
	Name          string `xml:"name,attr"`
	Artist        string `xml:"artist,attr"`
	ArtistID      string `xml:"artistId,attr,omitempty"`
	SongCount     int    `xml:"songCount,attr,omitempty"`
	Duration      int    `xml:"duration,attr,omitempty"`
	AlbumCoverArt string `xml:"albumCoverArt,attr,omitempty"`
	Songs         []Song `xml:"song,omitempty"`
}

// Song represents a single song (Subsonic Song model).
type Song struct {
	ID        string `xml:"id,attr"`
	Parent    string `xml:"parent,attr,omitempty"`
	IsDir     bool   `xml:"isDir,attr"`
	Title     string `xml:"title,attr"`
	Album     string `xml:"album,attr"`
	Artist    string `xml:"artist,attr"`
	AlbumID   string `xml:"albumId,attr,omitempty"`
	Year      int    `xml:"year,attr,omitempty"`
	Genre     string `xml:"genre,attr,omitempty"`
	Duration  int    `xml:"duration,attr"`
	Bitrate   int    `xml:"bitrate,attr,omitempty"`
	Size      int64  `xml:"size,attr,omitempty"`
	Created   string `xml:"created,attr,omitempty"`
	StreamURL string `xml:"streamUrl,attr,omitempty"`
	IsVideo   bool   `xml:"isVideo,attr"`
	CoverArt  string `xml:"coverArt,attr,omitempty"`
}

// --- Response containers for the browse / playlist endpoints ---

// AlbumList is the getAlbums result.
type AlbumList struct {
	XMLName xml.Name `xml:"albumList"`
	Albums  []Album  `xml:"album"`
}

// ArtistList is the getArtists result.
type ArtistList struct {
	XMLName xml.Name `xml:"artists"`
	Artists []Artist `xml:"artist"`
}

// SimilarSongsResult is the getSimilarSongs result.
type SimilarSongsResult struct {
	XMLName xml.Name `xml:"similarSongs"`
	Songs   []Song   `xml:"song"`
}

// RandomSongsResult is the getRandomSongs result.
type RandomSongsResult struct {
	XMLName xml.Name `xml:"randomSongs"`
	Songs   []Song   `xml:"song"`
}

// SubsonicPlaylist is one playlist entry (Subsonic playlist model).
type SubsonicPlaylist struct {
	ID        string `xml:"id,attr"`
	Name      string `xml:"name,attr"`
	Public    bool   `xml:"public,attr"`
	Comment   string `xml:"comment,attr,omitempty"`
	SongCount int    `xml:"songCount,attr"`
	Duration  int    `xml:"duration,attr,omitempty"`
	Created   string `xml:"created,attr,omitempty"`
	Songs     []Song `xml:"song"`
}

// PlaylistList is the getPlaylists result.
type PlaylistList struct {
	XMLName   xml.Name           `xml:"playlists"`
	Playlists []SubsonicPlaylist `xml:"playlist"`
}
