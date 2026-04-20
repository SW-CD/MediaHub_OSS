package media

import (
	"mediahub_oss/internal/shared/customerrors"
	"slices"
	"strings"
)

// GetContentType determines the primary content category ("image", "video", "audio", "file")
// based on the provided MIME type.
func GetContentType(mimeType string) (string, error) {
	normType := NormalizeMimeType(mimeType)

	if strings.HasPrefix(normType, "image/") {
		return "image", nil
	} else if strings.HasPrefix(normType, "video/") {
		return "video", nil
	} else if strings.HasPrefix(normType, "audio/") {
		return "audio", nil
	}

	// If it doesn't match known media prefixes, it defaults to the generic "file" type
	return "file", nil
}

func GetContentTypes() []string {
	return []string{"image", "video", "audio", "file"}
}

// check if mime type belongs to contentType
func IsMimeOfType(contentType, mimeType string) (bool, error) {
	normType := NormalizeMimeType(mimeType)

	switch contentType {
	case "image":
		return slices.Contains(imageMimeTypes, normType), nil
	case "video":
		return slices.Contains(videoMimeTypes, normType), nil
	case "audio":
		return slices.Contains(audioMimeTypes, normType), nil
	case "file":
		return true, nil
	default:
		return false, customerrors.ErrNotFound
	}
}

// a list of provided metadata fields per contentType
// the MediaConverter has to return those in their MediaFields... methods
func GetMetadataFields(contentType string) ([]FieldDef, error) {
	switch contentType {
	case "image":
		return []FieldDef{
			{"width", "uint64"},
			{"height", "uint64"},
		}, nil
	case "video":
		return []FieldDef{
			{"width", "uint64"},
			{"height", "uint64"},
			{"duration", "float64"},
		}, nil
	case "audio":
		return []FieldDef{
			{"duration", "float64"},
			{"channels", "uint8"},
		}, nil
	case "file":
		return []FieldDef{}, nil
	default:
		return []FieldDef{}, customerrors.ErrNotFound
	}
}

// convert mime aliases into a common type
func NormalizeMimeType(mime string) string {
	switch mime {
	case "audio/x-flac":
		return "audio/flac"
	case "application/ogg":
		return "audio/ogg"
	case "image/jpg":
		return "image/jpeg"
	case "audio/mp4":
		return "audio/m4a"
	default:
		return mime
	}
}
