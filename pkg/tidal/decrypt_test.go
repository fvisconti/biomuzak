package tidal

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"fmt"
	"testing"
)

// encryptForTest builds an encrypted Tidal-style payload: IV || AES-CBC(PKCS7(plain)).
func encryptForTest(plain []byte, trackID int) []byte {
	h := sha1.New()
	_, _ = h.Write([]byte(fmt.Sprintf("%d", trackID)))
	key := h.Sum(nil)[:16]

	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}

	// PKCS7 pad.
	bs := block.BlockSize()
	pad := bs - len(plain)%bs
	padded := make([]byte, len(plain)+pad)
	copy(padded, plain)
	for i := len(plain); i < len(padded); i++ {
		padded[i] = byte(pad)
	}

	iv := make([]byte, bs)
	for i := range iv {
		iv[i] = byte(i) // deterministic IV
	}

	ciphertext := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(ciphertext, padded)

	out := make([]byte, 0, len(iv)+len(ciphertext))
	out = append(out, iv...)
	out = append(out, ciphertext...)
	return out
}

func TestDecryptStream_RoundTrip(t *testing.T) {
	// A fake "FLAC" payload: magic + some bytes.
	plain := append([]byte("fLaC"), make([]byte, 100)...)
	for i := 4; i < len(plain); i++ {
		plain[i] = byte(i % 251)
	}

	trackID := 12345678
	encrypted := encryptForTest(plain, trackID)

	got, err := DecryptStream(encrypted, trackID)
	if err != nil {
		t.Fatalf("DecryptStream returned error: %v", err)
	}
	if !bytes.Equal(got, plain) {
		t.Fatalf("decrypted bytes do not match plaintext (got %d bytes, want %d)", len(got), len(plain))
	}
	if !IsFLAC(got) {
		t.Fatalf("expected decrypted payload to start with fLaC magic")
	}
}

func TestDecryptStream_WrongTrackID(t *testing.T) {
	plain := append([]byte("fLaC"), make([]byte, 64)...)
	encrypted := encryptForTest(plain, 111)

	// Decrypting with the wrong track ID should not reproduce the plaintext.
	got, err := DecryptStream(encrypted, 222)
	if err != nil {
		// Could fail on padding; that's acceptable.
		return
	}
	if bytes.Equal(got, plain) {
		t.Fatalf("wrong track ID unexpectedly reproduced plaintext")
	}
}

func TestDecryptStream_TooShort(t *testing.T) {
	if _, err := DecryptStream([]byte("short"), 1); err == nil {
		t.Fatalf("expected error for too-short input")
	}
}

func TestStripPKCS7(t *testing.T) {
	if got := stripPKCS7([]byte{1, 2, 3, 4, 5, 2, 2}); !bytes.Equal(got, []byte{1, 2, 3, 4, 5}) {
		t.Fatalf("stripPKCS7 failed: %v", got)
	}
	if got := stripPKCS7([]byte{1, 2, 0}); got != nil {
		t.Fatalf("expected nil for zero pad, got %v", got)
	}
	if got := stripPKCS7([]byte{}); got != nil {
		t.Fatalf("expected nil for empty, got %v", got)
	}
}
