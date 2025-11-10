// filepath: internal/media/metadata.go
package media

import (
	"fmt"
	"image"
	"mediahub/internal/logging"
	"os"
	"strings"

	// Import decoders for common image formats
	_ "image/gif"
	_ "image/png"

	_ "golang.org/x/image/webp"

	"mediahub/internal/models"

	"github.com/dhowden/tag"
	"github.com/go-audio/wav"
)

// ExtractMetadata attempts to read metadata from a file path.
func ExtractMetadata(filePath string, contentType string) (*models.MediaMetadata, error) {
	meta := &models.MediaMetadata{}
	logging.Log.Debugf("ExtractMetadata: Starting for %s (Content-Type: %s)", filePath, contentType)

	// 1. Open the file just once
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file for metadata: %w", err)
	}
	defer file.Close()

	// 2. Try generic metadata library (good for audio/video)
	// tag.ReadFrom uses an io.ReadSeeker, which *os.File implements
	tagMeta, tagErr := tag.ReadFrom(file)
	if tagErr == nil {
		meta.Title = tagMeta.Title()
		meta.Artist = tagMeta.Artist()
		meta.Album = tagMeta.Album()
		meta.Genre = tagMeta.Genre()
	}

	// 3. Try audio duration/channels
	var durationErr error
	if strings.HasPrefix(contentType, "audio/") {
		logging.Log.Debugf("ExtractMetadata: Processing as audio.")
		// Rewind file for the next library
		if _, err := file.Seek(0, 0); err != nil {
			return nil, fmt.Errorf("failed to seek for duration: %w", err)
		}

		switch contentType {
		case "audio/wav":
			logging.Log.Debugf("ExtractMetadata: Using pure-Go WAV parser.")
			// wav.NewDecoder is smart and uses io.ReadSeeker
			decoder := wav.NewDecoder(file)
			if decoder.IsValidFile() {
				dur, err := decoder.Duration()
				if err == nil {
					meta.DurationSec = dur.Seconds()
				} else {
					durationErr = err
				}
				meta.Channels = int(decoder.NumChans)
			} else {
				durationErr = fmt.Errorf("not a valid wav file")
			}

		case "audio/flac", "audio/mpeg", "audio/ogg", "audio/opus":
			logging.Log.Debugf("ExtractMetadata: Using ffprobe for %s.", contentType)
			if !IsFFprobeAvailable() {
				durationErr = fmt.Errorf("ffprobe not available for %s metadata", contentType)
				break
			}
			// Run ffprobe on the *file path*
			ffprobeMeta, err := runFFprobe(filePath)
			if err != nil {
				durationErr = fmt.Errorf("ffprobe failed: %w", err)
			} else {
				meta.DurationSec = ffprobeMeta.DurationSec
				meta.Channels = ffprobeMeta.Channels
			}

		default:
			logging.Log.Warnf("ExtractMetadata: No audio metadata extractor for content type '%s'", contentType)
			durationErr = fmt.Errorf("unsupported audio type for duration extraction: %s", contentType)
		}
	}

	// 4. Try image-specific library
	var imgErr error
	if strings.HasPrefix(contentType, "image/") {
		logging.Log.Debugf("ExtractMetadata: Processing as image.")
		// Rewind file for the next library
		if _, err := file.Seek(0, 0); err != nil {
			return nil, fmt.Errorf("failed to seek for image config: %w", err)
		}

		imgConfig, _, err := image.DecodeConfig(file)
		if err == nil {
			meta.Width = imgConfig.Width
			meta.Height = imgConfig.Height
		} else {
			imgErr = err
		}
	}

	// 5. Error handling
	if meta.Width == 0 && meta.Height == 0 && meta.DurationSec == 0 {
		if imgErr != nil {
			return nil, fmt.Errorf("failed to extract image metadata: %w", imgErr)
		}
		if durationErr != nil {
			return nil, fmt.Errorf("failed to extract audio duration: %w", durationErr)
		}
		if tagErr != nil {
			// Don't fail if only tags are missing
		}
	}

	logging.Log.Debugf("ExtractMetadata: Finished. Duration: %f, Channels: %d, W: %d, H: %d", meta.DurationSec, meta.Channels, meta.Width, meta.Height)
	return meta, nil
}
