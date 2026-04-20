package ffmpeg

import (
	"mediahub_oss/internal/media"
	"sort"
	"strings"
)

// ConversionProfile defines the FFmpeg arguments required for a specific output format.
type ConversionProfile struct {
	ContentType string
	CommonArgs  []string // Codecs and settings shared by both outputs (e.g., "-c:v", "libwebp")
	FileArgs    []string // Flags strictly for disk files (e.g., "-f", "webp")
	StreamArgs  []string // Flags strictly for streams/pipes (e.g., "-f", "image2pipe")
}

// GetOutputMimeTypes dynamically returns target formats based on our supportedConversions map.
func (c FfmpegConverter) GetOutputMimeTypes(contentType string) []string {
	outputs := make([]string, 0)
	for mime, profile := range c.supportedConversions {
		if profile.ContentType == contentType {
			outputs = append(outputs, mime)
		}
	}

	// Sort to ensure the API response is consistent, as Go map iteration is random
	sort.Strings(outputs)
	return outputs
}

// CanCreatePreview determines if FFmpeg can generate a visual preview for this file.
func (c FfmpegConverter) CanCreatePreview(inputMimeType string) bool {
	if !c.IsFFmpegAvailable() {
		return false
	}

	normalized := media.NormalizeMimeType(inputMimeType)

	if strings.HasPrefix(normalized, "image/") ||
		strings.HasPrefix(normalized, "video/") ||
		strings.HasPrefix(normalized, "audio/") {
		return true
	}
	return false
}

// CanConvert checks if a conversion is possible based on our supportedConversions map.
func (c FfmpegConverter) CanConvert(inputMimeType string, outputMimeType string) media.ConversionCheck {

	normInput := media.NormalizeMimeType(inputMimeType)
	normOutput := media.NormalizeMimeType(outputMimeType)

	var needsConversion, canConvert bool

	// check if we would need conversion
	needsConversion = (normInput != normOutput)

	// check if we can convert
	if c.IsFFmpegAvailable() {

		contentType, _ := media.GetContentType(normInput)
		if contentType == "file" {
			canConvert = false
		} else if profile, exists := c.supportedConversions[normOutput]; exists && profile.ContentType == contentType {
			canConvert = true
		}

	} else {
		canConvert = false
	}

	return media.ConversionCheck{NeedsConversion: needsConversion, CanConvert: canConvert}
}
