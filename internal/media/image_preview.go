// filepath: internal/media/image_preview.go
// (Moved from internal/storage/preview.go)
package media

import (
	"fmt"
	"image"
	"image/jpeg"

	// Import decoders for common formats
	_ "image/gif"
	_ "image/png"
	"io"
	"os"

	"golang.org/x/image/draw"
)

const (
	// PreviewMaxSide defines the maximum width or height for a preview entry.
	// The aspect ratio will be maintained.
	PreviewMaxSide = 200
)

// CreateImagePreview generates a JPEG preview from entry data, scaling it to fit
// within a PreviewMaxSide bounding box while maintaining aspect ratio.
// (Renamed from CreatePreview)
func CreateImagePreview(entryData io.Reader, previewPath string) error {
	// Decode the entry. This loads the entry into memory.
	img, _, err := image.Decode(entryData)
	if err != nil {
		return fmt.Errorf("could not decode entry for preview: %w", err)
	}

	// --- Calculate new dimensions ---
	origBounds := img.Bounds()
	origWidth := origBounds.Dx()
	origHeight := origBounds.Dy()

	if origWidth == 0 || origHeight == 0 {
		return fmt.Errorf("cannot create preview for zero-dimension entry")
	}

	var newWidth, newHeight int
	// Determine which side is the limiting factor
	if origWidth > origHeight {
		if origWidth > PreviewMaxSide {
			newWidth = PreviewMaxSide
			// Calculate new height based on aspect ratio
			newHeight = (origHeight * PreviewMaxSide) / origWidth
		} else {
			// Image is smaller than max size, don't scale up
			newWidth = origWidth
			newHeight = origHeight
		}
	} else {
		if origHeight > PreviewMaxSide {
			newHeight = PreviewMaxSide
			// Calculate new width based on aspect ratio
			newWidth = (origWidth * PreviewMaxSide) / origHeight
		} else {
			// Image is smaller than max size, don't scale up
			newWidth = origWidth
			newHeight = origHeight
		}
	}
	// --- End dimension calculation ---

	// Create a new RGBA entry of the *calculated* preview size.
	dst := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))

	// Resize the original entry (img) into the destination entry (dst).
	draw.ApproxBiLinear.Scale(dst, dst.Rect, img, img.Bounds(), draw.Over, nil)

	// Create the preview file
	f, err := os.Create(previewPath)
	if err != nil {
		return fmt.Errorf("could not create preview file: %w", err)
	}
	defer f.Close()

	// Encode the resized entry as JPEG with a reasonable quality.
	if err := jpeg.Encode(f, dst, &jpeg.Options{Quality: 75}); err != nil {
		os.Remove(previewPath) // Clean up failed write
		return fmt.Errorf("failed to encode preview to jpeg: %w", err)
	}

	return nil
}
