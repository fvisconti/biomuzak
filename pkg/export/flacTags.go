package export

import (
	"encoding/binary"
	"fmt"
	"os"
)

// embedFLACTags inserts a Vorbis Comment metadata block into a FLAC file.
//
// FLAC structure:
//   - 4 bytes: "fLaC" magic
//   - Metadata blocks: [1 byte header][3 bytes length][data]
//     - header bit 7: last metadata block flag
//     - header bits 0-6: block type (0=STREAMINFO, 4=VORBIS_COMMENT)
//   - Audio frames
//
// We insert a VORBIS_COMMENT block (type 4) after the first metadata block
// (STREAMINFO). To keep the file valid we clear the "last" flag on every
// existing metadata block, then set it on the new final block.
func embedFLACTags(path string, title, artist, album string, year int, genre string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read FLAC file: %w", err)
	}
	if len(data) < 4 || string(data[:4]) != "fLaC" {
		return fmt.Errorf("not a FLAC file")
	}

	vcData := buildVorbisComment(title, artist, album, year, genre)

	// Walk the metadata blocks, recording their byte ranges and clearing the
	// "last" flag on each.
	type blockRange struct{ start, end int }
	var blocks []blockRange
	offset := 4
	for {
		if offset+4 > len(data) {
			return fmt.Errorf("malformed FLAC metadata")
		}
		header := data[offset]
		// Block length is a 24-bit big-endian value (3 bytes).
		blockLen := int(data[offset+1])<<16 | int(data[offset+2])<<8 | int(data[offset+3])
		blockEnd := offset + 4 + blockLen
		if blockEnd > len(data) {
			return fmt.Errorf("malformed FLAC metadata (block overruns file)")
		}
		blocks = append(blocks, blockRange{start: offset, end: blockEnd})
		data[offset] &= 0x7F // clear last flag
		if header&0x80 != 0 { // was the last block
			break
		}
		offset = blockEnd
	}
	if len(blocks) == 0 {
		return fmt.Errorf("no metadata blocks found")
	}

	// Build our VORBIS_COMMENT block. We append it after the last existing
	// block, so it becomes the new last block (flag 0x84). All existing blocks
	// already had their "last" flag cleared above.
	// Block length is a 24-bit big-endian value (3 bytes).
	ourLen := []byte{
		byte(len(vcData) >> 16),
		byte(len(vcData) >> 8),
		byte(len(vcData)),
	}
	ourBlock := append([]byte{0x84}, ourLen...)
	ourBlock = append(ourBlock, vcData...)

	// Append after the last existing metadata block.
	appendAt := blocks[len(blocks)-1].end
	result := make([]byte, 0, len(data)+len(ourBlock))
	result = append(result, data[:appendAt]...)
	result = append(result, ourBlock...)
	result = append(result, data[appendAt:]...)

	return os.WriteFile(path, result, 0o644)
}

// buildVorbisComment constructs the Vorbis Comment block payload.
func buildVorbisComment(title, artist, album string, year int, genre string) []byte {
	// Vendor string (empty is fine).
	vendor := []byte("biomuzak")
	vendorLen := make([]byte, 4)
	binary.LittleEndian.PutUint32(vendorLen, uint32(len(vendor)))

	// Build comment strings.
	var comments []string
	if artist != "" {
		comments = append(comments, "ARTIST="+artist)
	}
	if title != "" {
		comments = append(comments, "TITLE="+title)
	}
	if album != "" {
		comments = append(comments, "ALBUM="+album)
	}
	if year > 0 {
		comments = append(comments, fmt.Sprintf("DATE=%d", year))
	}
	if genre != "" {
		comments = append(comments, "GENRE="+genre)
	}

	numComments := make([]byte, 4)
	binary.LittleEndian.PutUint32(numComments, uint32(len(comments)))

	var buf []byte
	buf = append(buf, vendorLen...)
	buf = append(buf, vendor...)
	buf = append(buf, numComments...)
	for _, c := range comments {
		clen := make([]byte, 4)
		binary.LittleEndian.PutUint32(clen, uint32(len(c)))
		buf = append(buf, clen...)
		buf = append(buf, []byte(c)...)
	}
	return buf
}
