package entryhandler

import (
	"mediahub_oss/internal/media"
	repo "mediahub_oss/internal/repository"
	"mediahub_oss/internal/shared/customerrors"
	"mime/multipart"
	"path/filepath"
	"strings"
)

// DetermineConversionPlan evaluates if a file needs conversion based on the database configuration.
func determineConversionPlan(mc media.MediaConverter, db repo.Database, header *multipart.FileHeader, userFileName string) (ProcessingPlan, error) {

	// validate Mime type
	originalMimeType := media.NormalizeMimeType(header.Header.Get("Content-Type"))

	isValid, err := media.IsMimeOfType(db.ContentType, originalMimeType)
	if !isValid {
		return ProcessingPlan{InitMimeType: originalMimeType}, customerrors.ErrBadMimeType
	}
	if err != nil {
		return ProcessingPlan{InitMimeType: originalMimeType}, err
	}

	wantsConversion := (db.Config.AutoConversion != "")
	targetMimeType := originalMimeType
	resultMimeType := originalMimeType

	var convCheck media.ConversionCheck
	if wantsConversion {
		targetMimeType = db.Config.AutoConversion

		// check capabilities
		convCheck = mc.CanConvert(originalMimeType, db.Config.AutoConversion)
		if convCheck.CanConvert {
			resultMimeType = targetMimeType
		}
	}

	canGenPreview := mc.CanCreatePreview(originalMimeType)

	//derive file name
	finalFileName := header.Filename
	if userFileName != "" {
		finalFileName = userFileName
		if filepath.Ext(finalFileName) == "" {
			originalExt := filepath.Ext(header.Filename)
			finalFileName = finalFileName + originalExt
		}
	}
	if convCheck.NeedsConversion && convCheck.CanConvert {
		newExtension := getExtensionForMimeType(db.Config.AutoConversion)
		finalFileName = replaceExtension(finalFileName, newExtension)
	}

	return ProcessingPlan{
		WantsConversion: wantsConversion,
		NeedsConversion: convCheck.NeedsConversion,
		CanConvert:      convCheck.CanConvert,
		WantsPreview:    db.Config.CreatePreview,
		CanGenPreview:   canGenPreview,
		InitMimeType:    originalMimeType,
		TargetMimeType:  targetMimeType,
		ResultMimeType:  resultMimeType,
		FinalFileName:   finalFileName,
	}, nil
}

// replaceExtension replaces the extension of a filename with a new one.
// e.g., ("song.mp3", ".opus") -> "song.opus"
// e.g., ("file_no_ext", ".jpg") -> "file_no_ext.jpg"
func replaceExtension(filename string, newExt string) string {
	if filename == "" {
		return "" // Don't create a filename if one didn't exist
	}
	if newExt == "" {
		return filename // No new extension provided
	}
	ext := filepath.Ext(filename)
	base := strings.TrimSuffix(filename, ext)
	return base + newExt
}
