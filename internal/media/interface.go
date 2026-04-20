package media

import (
	"context"
	"io"
)

type MediaConverter interface {
	// --- Capabilities ---
	GetOutputMimeTypes(contentType string) []string
	CanCreatePreview(inputMimeType string) bool
	CanConvert(inputMimeType string, outputMimeType string) ConversionCheck

	// --- File Conversion ---
	// ConvertStream: For small files in RAM. Uses HTTP loopback for input, pipes to output.
	ConvertStream(ctx context.Context, inputData io.ReadSeeker, outputStream io.Writer, inputMimeType, targetMimeType string) error

	// ConvertFile: For large files or videos. Pure disk-to-disk direct I/O.
	ConvertFile(ctx context.Context, inputPath string, outputPath string, inputMimeType, targetMimeType string) error

	// --- Metadata Extraction ---
	// ReadMediaFieldsFromStream: Uses HTTP loopback to extract metadata from RAM.
	ReadMediaFieldsFromStream(ctx context.Context, inputData io.ReadSeeker, contentType string) (map[string]any, error)

	// ReadMediaFieldsFromFile: Direct disk read. Extremely fast for large files.
	ReadMediaFieldsFromFile(ctx context.Context, filepath string, contentType string) (map[string]any, error)

	// --- Preview Generation ---
	// CreatePreviewFromStream: Uses HTTP loopback. Pipes WEBP bytes to output.
	CreatePreviewFromStream(ctx context.Context, inputData io.ReadSeeker, outputWriter io.Writer, inputMimeType string) error

	// CreatePreviewFromFile: Reads direct from disk. Pipes WEBP bytes to output.
	CreatePreviewFromFile(ctx context.Context, filepath string, outputWriter io.Writer, inputMimeType string) error
}
