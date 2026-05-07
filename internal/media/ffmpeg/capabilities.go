package ffmpeg

import (
	"mediahub_oss/internal/media"
	"sort"
	"strings"
)

// GetOutputMimeTypes dynamically returns target formats based on our supportedConversions map.
func (c *FfmpegConverter) GetOutputMimeTypes(contentType string) []string {
	outputs := make([]string, 0, len(c.supportedConversions)) // Micro-optimization: pre-allocate capacity
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
func (c *FfmpegConverter) CanCreatePreview(inputMimeType string) bool {
	if !c.IsFFmpegAvailable() {
		return false
	}

	normalized := media.NormalizeMimeType(inputMimeType)

	// Return the evaluation directly
	return strings.HasPrefix(normalized, "image/") ||
		strings.HasPrefix(normalized, "video/") ||
		strings.HasPrefix(normalized, "audio/")
}

// CanConvert checks if a conversion is possible based on our supportedConversions map.
func (c *FfmpegConverter) CanConvert(inputMimeType string, outputMimeType string) media.ConversionCheck {
	normInput := media.NormalizeMimeType(inputMimeType)
	normOutput := media.NormalizeMimeType(outputMimeType)

	// check if we would need conversion
	needsConversion := (normInput != normOutput)
	canConvert := false

	// check if we can convert
	if c.IsFFmpegAvailable() {
		contentType, _ := media.GetContentType(normInput)

		if contentType != "file" {
			if profile, exists := c.supportedConversions[normOutput]; exists && profile.ContentType == contentType {
				canConvert = true
			}
		}
	}

	return media.ConversionCheck{
		NeedsConversion: needsConversion,
		CanConvert:      canConvert,
	}
}
