package subsonic

import "encoding/xml"

// Response is the top-level response object for all Subsonic API calls
type Response struct {
	XMLName      xml.Name      `xml:"subsonic-response"`
	Status       string        `xml:"status,attr"`
	Version      string        `xml:"version,attr"`
	XMLNS        string        `xml:"xmlns,attr"`
	Error        *Error        `xml:"error,omitempty"`
	MusicFolders  *MusicFolders  `xml:"musicFolders,omitempty"`
	Indexes       *Indexes       `xml:"indexes,omitempty"`
	SearchResult3 *SearchResult3 `xml:"searchResult3,omitempty"`
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

// Artist represents a single artist
type Artist struct {
	ID   string `xml:"id,attr"`
	Name string `xml:"name,attr"`
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
	ID     string `xml:"id,attr"`
	Name   string `xml:"name,attr"`
	Artist string `xml:"artist,attr"`
}

// Song represents a single song
type Song struct {
	ID     string `xml:"id,attr"`
	Title  string `xml:"title,attr"`
	Artist string `xml:"artist,attr"`
	Album  string `xml:"album,attr"`
}
