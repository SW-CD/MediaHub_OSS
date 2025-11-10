// filepath: internal/media/conversion.go
package media

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"io"
	"mediahub/internal/logging"
	"mediahub/internal/models"
	"strconv"

	// Register decoders for PNG and GIF
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"os"
	"os/exec"
	"strings"
	"sync"

	// We can use this as it's a pure-Go implementation
	_ "golang.org/x/image/webp"
)

var (
	// ffmpegPath holds the validated path to the executable.
	ffmpegPath string
	// ffprobePath holds the validated path to the ffprobe executable.
	ffprobePath string
	// ffmpegCheckOnce ensures we only look for ffmpeg once.
	ffmpegCheckOnce sync.Once
)

// Initialize sets up the path for the ffmpeg executable.
// It should be called once at startup.
func Initialize(ffmpegConfiguredPath string, ffprobeConfiguredPath string) {
	ffmpegCheckOnce.Do(func() {
		// --- FFmpeg Check ---
		if ffmpegConfiguredPath != "" {
			if _, err := os.Stat(ffmpegConfiguredPath); err == nil {
				logging.Log.Infof("Using configured FFmpeg path: %s", ffmpegConfiguredPath)
				ffmpegPath = ffmpegConfiguredPath
			} else {
				logging.Log.Warnf("Configured ffmpeg_path '%s' not found, falling back to system PATH.", ffmpegConfiguredPath)
			}
		}

		if ffmpegPath == "" { // Only check PATH if not configured
			path, err := exec.LookPath("ffmpeg")
			if err != nil {
				logging.Log.Warn("---------------------------------------------------------")
				logging.Log.Warn("FFmpeg executable not found in configured path or system PATH.")
				logging.Log.Warn("Audio auto-conversion will be DISABLED.")
				logging.Log.Warn("---------------------------------------------------------")
				ffmpegPath = "" // Explicitly set to empty
			} else {
				logging.Log.Infof("FFmpeg found in PATH: %s. Audio and advanced image conversion enabled.", path)
				ffmpegPath = path
			}
		}

		// --- FFprobe Check ---
		if ffprobeConfiguredPath != "" {
			if _, err := os.Stat(ffprobeConfiguredPath); err == nil {
				logging.Log.Infof("Using configured FFprobe path: %s", ffprobeConfiguredPath)
				ffprobePath = ffprobeConfiguredPath
			} else {
				logging.Log.Warnf("Configured ffprobe_path '%s' not found, falling back to system PATH.", ffprobeConfiguredPath)
			}
		}

		if ffprobePath == "" { // Only check PATH if not found or configured
			if ffmpegPath != "" {
				probePath := strings.Replace(ffmpegPath, "ffmpeg", "ffprobe", 1)
				if _, err := os.Stat(probePath); err == nil {
					logging.Log.Infof("Found ffprobe alongside ffmpeg in PATH: %s", probePath)
					ffprobePath = probePath
				}
			}
		}

		if ffprobePath == "" { // Still not found? Check PATH explicitly.
			path, err := exec.LookPath("ffprobe")
			if err != nil {
				logging.Log.Warn("---------------------------------------------------------")
				logging.Log.Warn("ffprobe executable not found. Audio metadata extraction (for MP3, FLAC, etc.) will be disabled.")
				logging.Log.Warn("---------------------------------------------------------")
				ffprobePath = ""
			} else {
				logging.Log.Infof("ffprobe found in PATH: %s. Audio metadata extraction enabled.", path)
				ffprobePath = path
			}
		}
	})
}

// IsFFmpegAvailable checks if the ffmpeg executable path was successfully found.
func IsFFmpegAvailable() bool {
	Initialize("", "")
	return ffmpegPath != ""
}

// GetFFmpegPath returns the determined path to the ffmpeg executable.
func GetFFmpegPath() string {
	Initialize("", "")
	return ffmpegPath
}

// IsFFprobeAvailable checks if the ffprobe executable path was successfully found.
func IsFFprobeAvailable() bool {
	Initialize("", "")
	return ffprobePath != ""
}

// GetFFprobePath returns the determined path to the ffprobe executable.
func GetFFprobePath() string {
	Initialize("", "")
	return ffprobePath
}

type ffprobeStream struct {
	CodecType string `json:"codec_type"`
	Duration  string `json:"duration"` // Can be string, needs parsing
	Channels  int    `json:"channels"`
}

type ffprobeFormat struct {
	Duration string `json:"duration"`
}

type ffprobeOutput struct {
	Streams []ffprobeStream `json:"streams"`
	Format  ffprobeFormat   `json:"format"`
}

// runFFprobe executes ffprobe on a file path and returns the parsed metadata.
func runFFprobe(filePath string) (*models.MediaMetadata, error) {
	if !IsFFprobeAvailable() {
		return nil, fmt.Errorf("ffprobe is not available")
	}

	cmdArgs := []string{
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		"-i", filePath, // Read from filePath
	}

	cmd := exec.Command(GetFFprobePath(), cmdArgs...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	logging.Log.Debugf("Starting ffprobe metadata extraction (from file): %s %s", GetFFprobePath(), strings.Join(cmdArgs, " "))

	if err := cmd.Run(); err != nil {
		logging.Log.Errorf("ffprobe execution failed: %v\nffprobe output:\n%s", err, stderr.String())
		return nil, fmt.Errorf("ffprobe error: %s", stderr.String())
	}

	var output ffprobeOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		logging.Log.Errorf("Failed to parse ffprobe JSON output: %v\nOutput: %s", err, stdout.String())
		return nil, fmt.Errorf("failed to parse ffprobe JSON: %w", err)
	}

	meta := &models.MediaMetadata{}

	// Find the first audio stream to get channels
	for _, stream := range output.Streams {
		if stream.CodecType == "audio" {
			meta.Channels = stream.Channels
			logging.Log.Debugf("ffprobe: Found audio stream. Channels: %d, Stream Duration: '%s'", stream.Channels, stream.Duration)
			// Use stream duration if available
			if d, err := parseDurationString(stream.Duration); err == nil {
				meta.DurationSec = d
			}
			break // Found audio stream
		}
	}

	// If no stream duration was found, use the format duration (often more accurate)
	if meta.DurationSec == 0 {
		logging.Log.Debugf("ffprobe: No stream duration found. Trying format duration: '%s'", output.Format.Duration)
		if d, err := parseDurationString(output.Format.Duration); err == nil {
			meta.DurationSec = d
		}
	}

	logging.Log.Debugf("ffprobe: Final extracted metadata. Duration: %f, Channels: %d", meta.DurationSec, meta.Channels)

	return meta, nil
}

// Helper to parse duration strings (e.g., "180.500000") from ffprobe
func parseDurationString(d string) (float64, error) {
	if d == "" {
		return 0, fmt.Errorf("empty duration string")
	}
	f, err := strconv.ParseFloat(d, 64)
	if err != nil {
		logging.Log.Warnf("ffprobe: Could not parse duration string '%s': %v", d, err)
		return 0, err
	}
	return f, nil
}

// RunFFmpegToFile executes an FFmpeg command, piping from an input reader
// and writing directly to a seekable output file path.
func RunFFmpegToFile(inputReader io.Reader, outputPath string, format string, args ...string) error {
	cmdArgs := []string{
		"-y",      // Overwrite output file
		"-i", "-", // Read from stdin
	}
	cmdArgs = append(cmdArgs, args...)
	cmdArgs = append(cmdArgs,
		"-f", format, // Set the output format
		outputPath, // Write to the final file path
	)

	cmd := exec.Command(GetFFmpegPath(), cmdArgs...)

	cmd.Stdin = inputReader

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	logging.Log.Debugf("Starting FFmpeg conversion (to file): %s %s", GetFFmpegPath(), strings.Join(cmdArgs, " "))

	if err := cmd.Run(); err != nil {
		logging.Log.Errorf("FFmpeg execution failed: %v\nFFmpeg output:\n%s", err, stderr.String())
		return fmt.Errorf("ffmpeg error: %s", stderr.String())
	}

	logging.Log.Debugf("Finished FFmpeg conversion (to file). Output: %s", outputPath)
	return nil
}

// ConvertImagePureGoToFile attempts to decode an image and save it as a JPEG
// using only pure Go libraries. This is a fallback for when FFmpeg is not available.
func ConvertImagePureGoToFile(inputReader io.Reader, outputPath string) error {
	img, originalFormat, err := image.Decode(inputReader)
	if err != nil {
		return fmt.Errorf("failed to decode image for conversion: %w", err)
	}
	logging.Log.Debugf("PureGo converter: decoded image format %s", originalFormat)

	// Create the output file
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file for jpeg: %w", err)
	}
	defer outputFile.Close()

	// Encode as JPEG
	if err := jpeg.Encode(outputFile, img, &jpeg.Options{Quality: 85}); err != nil {
		// Clean up failed write
		os.Remove(outputPath)
		return fmt.Errorf("failed to encode jpeg: %w", err)
	}

	return nil
}
