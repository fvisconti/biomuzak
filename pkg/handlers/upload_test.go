package handlers

import (
	"os"
	"path/filepath"
	"testing"

	"go-postgres-example/pkg/metadata"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockProcessor is a mock implementation of the ProcessorAPI for testing.
type mockProcessor struct {
	ProcessFileCalls int
	LastFilePath     string
}

func (m *mockProcessor) ProcessFile(filePath string) error {
	m.ProcessFileCalls++
	m.LastFilePath = filePath
	return nil
}

func TestProcessDirectory(t *testing.T) {
	// 1. Setup
	handler := &UploadHandler{}
	mockProc := &mockProcessor{}
	var processor metadata.ProcessorAPI = mockProc

	tempDir := t.TempDir()

	// Create a subdirectory to ensure recursion is handled
	err := os.Mkdir(filepath.Join(tempDir, "subdir"), 0755)
	require.NoError(t, err)

	// Create a supported audio file
	supportedFile, err := os.Create(filepath.Join(tempDir, "song.mp3"))
	require.NoError(t, err)
	supportedFile.Close()

	// Create an unsupported file
	unsupportedFile, err := os.Create(filepath.Join(tempDir, "document.txt"))
	require.NoError(t, err)
	unsupportedFile.Close()

	// Create another supported file in the subdirectory
	supportedFile2, err := os.Create(filepath.Join(tempDir, "subdir", "song2.flac"))
	require.NoError(t, err)
	supportedFile2.Close()

	// 2. Execute
	handler.processDirectory(tempDir, processor)

	// 3. Assert
	assert.Equal(t, 2, mockProc.ProcessFileCalls, "ProcessFile should be called for two supported files")
}
