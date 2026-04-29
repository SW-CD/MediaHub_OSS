package entryhandler

import (
	"context"
	"fmt"
	"io"
	"strings"

	"mediahub_oss/internal/media"
	repo "mediahub_oss/internal/repository"
)

// getExtensionForMimeType returns the preferred file extension for a given MIME type (e.g., ".opus")
func getExtensionForMimeType(mimeType string) string {
	var mimeToExtension = map[string]string{
		// Images
		"image/jpeg": "jpg",
		"image/png":  "png",
		"image/gif":  "gif",
		"image/webp": "webp",
		"image/avif": "avif",

		// Audio
		"audio/mpeg":      "mp3", // Note: audio/mpeg is usually an .mp3 file
		"audio/wav":       "wav",
		"audio/flac":      "flac",
		"audio/x-flac":    "flac", // Browser-reported variant
		"audio/opus":      "opus",
		"audio/ogg":       "ogg",
		"application/ogg": "ogg", // Browser-reported variant for Ogg

		// Video
		"video/mp4":  "mp4",
		"video/webm": "webm",
		"video/ogg":  "ogv", // Video ogg files typically use .ogv
	}

	if ext, ok := mimeToExtension[mimeType]; ok {
		return "." + ext
	}

	// Fallback for unmapped types (e.g., "text/plain" -> ".plain")
	parts := strings.Split(mimeType, "/")
	if len(parts) == 2 {
		// Clean up any weird prefixes just in case
		ext := strings.TrimPrefix(parts[1], "x-")
		return "." + ext
	}

	return "" // No extension
}

func defaultMediaFields(contentType string) (map[string]any, error) {
	var val any

	metadataFields, err := media.GetMetadataFields(contentType)
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata fields: %w", err)
	}

	mediaFields := make(map[string]any)
	for _, field := range metadataFields {
		switch field.Type {
		case "uint8":
			val = uint8(0)
		case "uint64":
			val = uint64(0)
		case "int64":
			val = int64(-1)
		case "float64":
			val = float64(-1.0)
		case "bool":
			val = false
		default:
			return nil, fmt.Errorf("implementation missing default value for media field type %s", field.Type)
		}
		mediaFields[field.Name] = val
	}

	return mediaFields, nil
}

// generateAndStorePreview creates a preview from any io.ReadSeeker and saves it to storage. It returns the amount of bytes written.
// It deliberately does NOT update the database, allowing callers to manage their own state.
func (h *EntryHandler) generateAndStorePreview(ctx context.Context, db repo.Database, entryID int64, inputSeeker io.ReadSeeker, mimeType string) (uint64, error) {

	// Create a pipe to connect the preview generator (Writer) to the storage provider (Reader)
	pr, pw := io.Pipe()
	errChan := make(chan error, 1)

	// Run the preview generation in a background goroutine
	go func() {
		defer pw.Close() // Signal EOF to the storage reader when generation completes
		// NOTE: Updated interface method call to CreatePreviewFromStream
		err := h.MediaConverter.CreatePreviewFromStream(ctx, inputSeeker, pw, mimeType)
		errChan <- err
	}()

	// The main thread blocks here, reading from the pipe and writing to storage
	previewSize, err := h.Storage.WritePreview(ctx, db.ID, entryID, pr)
	if err != nil {
		return 0, fmt.Errorf("failed to save preview to storage: %w", err)
	}

	// Check if the background preview generation encountered any errors
	if genErr := <-errChan; genErr != nil {
		return 0, fmt.Errorf("failed to generate preview: %w", genErr)
	}

	return uint64(previewSize), nil
}
