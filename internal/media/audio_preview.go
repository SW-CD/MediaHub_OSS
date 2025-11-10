// filepath: internal/media/audio_preview.go
package media

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"io"
	"mediahub/internal/logging"
	"os"
	"os/exec"

	"github.com/go-audio/wav"
)

const (
	previewWidth  = 200
	previewHeight = 120
)

// CreateAudioPreview generates a JPEG waveform preview for an audio file.
// It uses FFmpeg if available, otherwise it falls back to a pure-Go WAV parser.
func CreateAudioPreview(audioData io.ReadSeeker, previewPath string) error {
	// Check if ffmpeg is available
	if !IsFFmpegAvailable() {
		// --- Fallback to existing WAV-only parser ---
		logging.Log.Warn("FFmpeg not found. Attempting to generate audio preview using pure-Go WAV parser.")
		// Rewind the reader in case it was used before
		if _, err := audioData.Seek(0, io.SeekStart); err != nil {
			return fmt.Errorf("failed to seek audio data for WAV fallback: %w", err)
		}
		return createAudioPreviewWAV(audioData, previewPath)
	}

	// --- Primary Method: Use FFmpeg ---

	// Rewind the input reader ---
	// The caller (entry_service) might have already read this file.
	if _, err := audioData.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek audio data for preview: %w", err)
	}

	// Create the *output* file ---
	outputFile, err := os.Create(previewPath)
	if err != nil {
		return fmt.Errorf("failed to create preview output file: %w", err)
	}
	defer outputFile.Close()

	// Run ffmpeg command to generate the waveform
	cmd := exec.Command(
		GetFFmpegPath(), // Use the discovered path
		"-y",            // Add -y to auto-overwrite
		"-i", "-",       // Read from stdin
		"-filter_complex", "showwavespic=s=200x120:colors=#464646", // 200x120, dark grey
		"-frames:v", "1",
		"-f", "image2",
		"-c:v", "mjpeg",
		"-") // Write to stdout

	// Pipe stdin/stdout ---
	cmd.Stdin = audioData
	cmd.Stdout = outputFile

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		logging.Log.Errorf("FFmpeg waveform generation failed: %v\nOutput: %s", err, stderr.String())
		// Clean up the failed preview file
		os.Remove(previewPath)
		return fmt.Errorf("failed to generate audio preview: %s", stderr.String())
	}

	return nil
}

// createAudioPreviewWAV is the old pure-Go implementation, kept as a fallback.
// Note: This is a simple implementation that only supports WAV files.
func createAudioPreviewWAV(r io.ReadSeeker, previewPath string) error {
	// For this pure-Go example, we'll only support WAV.
	// A more robust solution would convert to WAV first (using ffmpeg)
	// or decode other formats (e.g., go-mp3).
	decoder := wav.NewDecoder(r)
	if !decoder.IsValidFile() {
		return fmt.Errorf("file is not a valid WAV file, skipping audio preview")
	}

	// Read all audio frames
	buf, err := decoder.FullPCMBuffer()
	if err != nil {
		return fmt.Errorf("could not read PCM buffer: %w", err)
	}

	if buf.NumFrames() == 0 {
		return fmt.Errorf("no audio frames found")
	}

	// Create a new image
	img := image.NewRGBA(image.Rect(0, 0, previewWidth, previewHeight))
	bgColor := color.RGBA{R: 255, G: 255, B: 255, A: 255} // White background
	lineColor := color.RGBA{R: 70, G: 70, B: 70, A: 255}  // Dark grey line
	draw.Draw(img, img.Bounds(), &image.Uniform{C: bgColor}, image.Point{}, draw.Src)

	// Calculate samples per pixel
	samplesPerPx := buf.NumFrames() / previewWidth
	if samplesPerPx == 0 {
		samplesPerPx = 1
	}

	amplitude := 1 << (decoder.BitDepth - 1)
	maxAmplitude := float64(amplitude) // e.g., 32768 for 16-bit
	midY := float64(previewHeight / 2)

	// Draw the waveform
	for x := 0; x < previewWidth; x++ {
		startSample := x * samplesPerPx
		endSample := (x + 1) * samplesPerPx
		if endSample > buf.NumFrames() {
			endSample = buf.NumFrames()
		}

		var min, max float64
		for i := startSample; i < endSample; i++ {
			// Get sample from the first channel
			sample := float64(buf.Data[i*int(decoder.NumChans)])
			if sample < min {
				min = sample
			}
			if sample > max {
				max = sample
			}
		}

		// Normalize to 0-1 range, then scale to image height
		yMin := (min / maxAmplitude) * midY
		yMax := (max / maxAmplitude) * midY

		// Draw a vertical line from min to max
		for y := int(midY + yMin); y <= int(midY+yMax); y++ {
			img.Set(x, y, lineColor)
		}
	}

	// Create the preview file
	f, err := os.Create(previewPath)
	if err != nil {
		return fmt.Errorf("could not create preview file: %w", err)
	}
	defer f.Close()

	// Encode as JPEG
	if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 75}); err != nil {
		os.Remove(previewPath) // Clean up failed write
		return fmt.Errorf("failed to encode preview to jpeg: %w", err)
	}

	return nil
}
