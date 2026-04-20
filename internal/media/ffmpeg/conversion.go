package ffmpeg

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"time"

	"mediahub_oss/internal/media"
)

// ConvertFile transcodes a large file using pure disk-to-disk direct I/O.
func (c *FfmpegConverter) ConvertFile(ctx context.Context, inputPath string, outputPath string, inputMimeType, targetMimeType string) error {
	ffmpegPath, err := c.GetFFmpegPath()
	if err != nil {
		return fmt.Errorf("ffmpeg is not available: %w", err)
	}

	normTarget := media.NormalizeMimeType(targetMimeType)

	// -y to overwrite existing output files automatically, -i to read direct from disk
	args := []string{"-y", "-i", inputPath}

	// Get the required codec and format arguments (isStream = false)
	formatArgs, err := c.buildConversionArgs(normTarget, false)
	if err != nil {
		return err
	}
	args = append(args, formatArgs...)

	// Specify the final output path
	args = append(args, outputPath)

	// Bind the FFmpeg process to the provided context to prevent zombie processes
	cmd := exec.CommandContext(ctx, ffmpegPath, args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		c.logger.Error("FFmpeg file conversion failed", "error", err, "stderr", stderr.String(), "target", targetMimeType)
		return fmt.Errorf("ffmpeg conversion error: %w", err)
	}

	return nil
}

// ConvertStream transcodes small files in RAM, utilizing the HTTP loopback server for input and piping to output.
func (c *FfmpegConverter) ConvertStream(ctx context.Context, inputData io.ReadSeeker, outputStream io.Writer, inputMimeType, targetMimeType string) error {
	ffmpegPath, err := c.GetFFmpegPath()
	if err != nil {
		return fmt.Errorf("ffmpeg is not available: %w", err)
	}

	// Register the stream with the local loopback server.
	// We give it a generous TTL (e.g., 30 minutes) to ensure FFmpeg has enough time to read and convert the stream.
	id, fullURL, err := c.localServer.Register(inputData, 30*time.Minute)
	if err != nil {
		return fmt.Errorf("failed to register stream: %w", err)
	}
	defer c.localServer.Unregister(id)

	normTarget := media.NormalizeMimeType(targetMimeType)

	// Tell FFmpeg to read the input from the local HTTP loopback server
	args := []string{"-i", fullURL}

	// Get the required codec and format arguments (isStream = true)
	formatArgs, err := c.buildConversionArgs(normTarget, true)
	if err != nil {
		return err
	}
	args = append(args, formatArgs...)

	// Pipe the output directly to the provided io.Writer
	args = append(args, "pipe:1")

	// Bind the FFmpeg process to the provided context to prevent zombie processes
	cmd := exec.CommandContext(ctx, ffmpegPath, args...)
	cmd.Stdout = outputStream

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		c.logger.Error("FFmpeg stream conversion failed", "error", err, "stderr", stderr.String(), "target", targetMimeType)
		return fmt.Errorf("ffmpeg conversion error: %w", err)
	}

	return nil
}

// buildConversionArgs maps a MIME type to the specific FFmpeg flags dynamically.
func (c *FfmpegConverter) buildConversionArgs(targetMimeType string, isStream bool) ([]string, error) {
	profile, exists := c.supportedConversions[targetMimeType] // Uses struct's map
	if !exists {
		return nil, fmt.Errorf("unsupported conversion target format: %s", targetMimeType)
	}

	// Pre-allocate capacity to avoid reallocation overhead
	capacity := len(profile.CommonArgs)
	if isStream {
		capacity += len(profile.StreamArgs)
	} else {
		capacity += len(profile.FileArgs)
	}

	args := make([]string, 0, capacity)
	args = append(args, profile.CommonArgs...)

	if isStream {
		args = append(args, profile.StreamArgs...)
	} else {
		args = append(args, profile.FileArgs...)
	}

	return args, nil
}
