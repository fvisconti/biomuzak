package tidal

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"fmt"
)

// DecryptStream decrypts a Tidal encrypted audio stream.
//
// Tidal encrypts streams with AES-128-CBC where:
//   - key = first 16 bytes of SHA1(str(trackID))
//   - IV  = first 16 bytes of the encrypted payload
//   - the remainder is the ciphertext (PKCS7-padded)
//
// The result is the raw audio bytes (FLAC for LOSSLESS/HIRES, AAC for LOW/NORMAL).
func DecryptStream(encrypted []byte, trackID int) ([]byte, error) {
	if len(encrypted) < 32 {
		return nil, fmt.Errorf("encrypted stream too short (%d bytes)", len(encrypted))
	}

	// Derive key from track ID.
	h := sha1.New()
	_, _ = h.Write([]byte(fmt.Sprintf("%d", trackID)))
	key := h.Sum(nil)[:16]

	iv := encrypted[:16]
	ciphertext := encrypted[16:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	if len(ciphertext)%block.BlockSize() != 0 {
		return nil, fmt.Errorf("ciphertext length %d is not a multiple of block size", len(ciphertext))
	}

	decrypted := make([]byte, len(ciphertext))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(decrypted, ciphertext)

	// Strip PKCS7 padding.
	decrypted = stripPKCS7(decrypted)
	if decrypted == nil {
		return nil, fmt.Errorf("invalid PKCS7 padding")
	}
	return decrypted, nil
}

// stripPKCS7 removes PKCS7 padding. Returns nil if the padding is invalid.
func stripPKCS7(data []byte) []byte {
	if len(data) == 0 {
		return nil
	}
	padLen := int(data[len(data)-1])
	if padLen == 0 || padLen > 16 || padLen > len(data) {
		return nil
	}
	for i := len(data) - padLen; i < len(data); i++ {
		if int(data[i]) != padLen {
			return nil
		}
	}
	return data[:len(data)-padLen]
}

// IsFLAC reports whether the bytes start with the FLAC magic "fLaC".
func IsFLAC(b []byte) bool {
	return len(b) >= 4 && b[0] == 'f' && b[1] == 'L' && b[2] == 'a' && b[3] == 'C'
}
