package handlers

import (
	"database/sql"
	"fmt"
	"go-postgres-example/pkg/config"
	"go-postgres-example/pkg/metadata"
	"go-postgres-example/pkg/middleware"
	"go-postgres-example/pkg/storage"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// UploadHandler holds the dependencies for the upload handlers
type UploadHandler struct {
	DB      *sql.DB
	Cfg     *config.Config
	Storage storage.StorageService
}

// NewUploadHandler creates a new UploadHandler
func NewUploadHandler(db *sql.DB, cfg *config.Config, storage storage.StorageService) *UploadHandler {
	return &UploadHandler{DB: db, Cfg: cfg, Storage: storage}
}

// Upload handles file uploads and starts the processing.
func (h *UploadHandler) Upload(w http.ResponseWriter, r *http.Request) {
	// Set a reasonable limit for the file size. Here, 500 MB.
	r.Body = http.MaxBytesReader(w, r.Body, 500<<20)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "The uploaded file is too big. Please choose a file less than 500MB.", http.StatusBadRequest)
		return
	}

	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		http.Error(w, "No files were uploaded", http.StatusBadRequest)
		return
	}

	// Parse optional playlist ID
	var playlistID int
	if playlistIDStr := r.MultipartForm.Value["playlist_id"]; len(playlistIDStr) > 0 {
		fmt.Sscanf(playlistIDStr[0], "%d", &playlistID)
	}

	// Get UserID from context (set by auth middleware)
	// We need to import the middleware package
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		// Should have been caught by middleware, but safe fallback
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	tempDir, err := os.MkdirTemp("", "music-uploads-*")
	if err != nil {
		log.Printf("Error creating temp dir: %v", err)
		http.Error(w, "Failed to create temporary directory for uploads", http.StatusInternalServerError)
		return
	}
	log.Printf("Created temporary directory: %s", tempDir)

	for _, fileHeader := range files {
		if err := saveUploadedFile(fileHeader, tempDir); err != nil {
			log.Printf("Error saving uploaded file: %v", err)
			http.Error(w, "Failed to save uploaded file", http.StatusInternalServerError)
			// Clean up the temp directory in case of an error
			os.RemoveAll(tempDir)
			return
		}
	}

	// Create a new metadata processor
	processor := metadata.NewProcessor(h.DB, h.Cfg, h.Storage)

	// Run the processing in a background goroutine
	go h.processDirectory(tempDir, processor, userID, playlistID)

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Successfully uploaded %d files. Processing has started in the background.", len(files))
}

// saveUploadedFile saves a single multipart file to the destination directory.
func saveUploadedFile(fileHeader *multipart.FileHeader, destDir string) error {
	log.Printf("Saving uploaded file: %s", fileHeader.Filename)

	file, err := fileHeader.Open()
	if err != nil {
		return fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer file.Close()

	destPath := filepath.Join(destDir, fileHeader.Filename)
	destFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, file); err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}
	return nil
}

// processDirectory walks through the given directory and processes all supported audio files.
func (h *UploadHandler) processDirectory(dir string, processor metadata.ProcessorAPI, userID int, playlistID int) {
	log.Printf("Starting to process directory: %s", dir)

	defer func() {
		log.Printf("Processing complete. Removing temporary directory: %s", dir)
		os.RemoveAll(dir)
	}()

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil // Skip directories
		}

		if isSupportedAudioFile(path) {
			log.Printf("Found supported audio file: %s", path)
			if err := processor.ProcessFile(path, userID, playlistID); err != nil {
				log.Printf("Error processing file %s: %v", path, err)
			}
		} else {
			log.Printf("Skipping unsupported file: %s", path)
		}
		return nil
	})

	if err != nil {
		log.Printf("Error walking directory %s: %v", dir, err)
	}
}

// isSupportedAudioFile checks if the file has a supported audio extension.
func isSupportedAudioFile(filePath string) bool {
	supportedExtensions := []string{".mp3", ".flac", ".wav", ".m4a", ".ogg"}
	ext := strings.ToLower(filepath.Ext(filePath))
	for _, supportedExt := range supportedExtensions {
		if ext == supportedExt {
			return true
		}
	}
	return false
}
