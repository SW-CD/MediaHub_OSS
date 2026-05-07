package ffmpeg

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"mediahub_oss/internal/media"
)

// ConversionProfile defines the FFmpeg arguments required for a specific output format.
type ConversionProfile struct {
	ContentType string
	Args        []string
}

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
	formatArgs, err := c.buildConversionArgs(normTarget)
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

// ConvertStream transcodes small files in RAM, utilizing the HTTP loopback server for input
// and an optimized OS-level temporary file for seekable output.
func (c *FfmpegConverter) ConvertStream(ctx context.Context, inputData io.ReadSeeker, outputStream io.Writer, inputMimeType, targetMimeType string) error {
	ffmpegPath, err := c.GetFFmpegPath()
	if err != nil {
		return fmt.Errorf("ffmpeg is not available: %w", err)
	}

	// Register the stream with the local loopback server.
	id, fullURL, err := c.localServer.Register(inputData, 30*time.Minute)
	if err != nil {
		return fmt.Errorf("failed to register stream: %w", err)
	}
	defer c.localServer.Unregister(id)

	// Create the highly optimized temporary file to satisfy FFmpeg's need for a seekable output
	tmpPath, err := createInMemoryFile("", "ffmpeg-output-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temporary output file: %w", err)
	}
	// Guarantee the file is deleted from RAM/Disk when this function exits
	defer os.Remove(tmpPath)

	normTarget := media.NormalizeMimeType(targetMimeType)

	// -y to automatically overwrite the temp file, -i to read from the loopback server
	args := []string{"-y", "-i", fullURL}

	// Get the required codec and format arguments.
	formatArgs, err := c.buildConversionArgs(normTarget)
	if err != nil {
		return err
	}
	args = append(args, formatArgs...)

	// Point the output to our optimized temporary file
	args = append(args, tmpPath)

	// Bind the FFmpeg process to the provided context to prevent zombie processes
	cmd := exec.CommandContext(ctx, ffmpegPath, args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		c.logger.Error("FFmpeg stream conversion failed", "error", err, "stderr", stderr.String(), "target", targetMimeType)
		return fmt.Errorf("ffmpeg conversion error: %w", err)
	}

	// FFmpeg successfully wrote the file. Open it so we can copy it to the user's requested io.Writer
	generatedFile, err := os.Open(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to open generated temporary file: %w", err)
	}
	defer generatedFile.Close()

	// Stream the data from our memory-backed file into the final output destination
	if _, err := io.Copy(outputStream, generatedFile); err != nil {
		return fmt.Errorf("failed to copy converted data to output stream: %w", err)
	}

	return nil
}

// buildConversionArgs safely retrieves a copy of the pre-computed FFmpeg arguments.
func (c *FfmpegConverter) buildConversionArgs(targetMimeType string) ([]string, error) {
	profile, exists := c.supportedConversions[targetMimeType]
	if !exists {
		return nil, fmt.Errorf("unsupported conversion target format: %s", targetMimeType)
	}

	// Create a fresh copy of the slice to prevent concurrent requests from accidentally mutating the base profile
	argsCopy := make([]string, len(profile.Args))
	copy(argsCopy, profile.Args)

	return argsCopy, nil
}

// initConversions dynamically builds the supported conversions map, prioritizing hardware encoders.
func (c *FfmpegConverter) initConversions() {
	c.supportedConversions = make(map[string]ConversionProfile)

	// stop if ffmpeg not available
	_, err := c.GetFFmpegPath()
	if err != nil {
		c.logger.Warn("FFmpeg not available, skipping conversion profile setup")
		return
	}

	// 1. Add static Image and Audio profiles (these rely heavily on standard software encoding)
	// 1. Image and Audio Profiles
	c.supportedConversions["image/jpeg"] = ConversionProfile{
		ContentType: "image",
		Args:        []string{"-c:v", "mjpeg", "-vframes", "1", "-f", "image2"},
	}
	c.supportedConversions["image/webp"] = ConversionProfile{
		ContentType: "image",
		Args:        []string{"-c:v", "libwebp", "-vframes", "1", "-f", "webp"},
	}
	c.supportedConversions["image/avif"] = ConversionProfile{
		ContentType: "image",
		Args: []string{
			"-c:v", "libaom-av1",
			"-still-picture", "1", // Tells the AV1 encoder it's a static image, not a 1-frame video
			"-vframes", "1", // PREVENTS ANIMATED LOOP FLICKERING: Forces exactly 1 frame
			"-pix_fmt", "yuv420p", // Ensures cross-browser chroma subsampling compatibility
			"-cpu-used", "6", // Speed scale is 0-8 (0 is slowest, 8 is fastest). 6 is the web sweet spot!
			"-row-mt", "1", // Enables Row-Based Multithreading (Use all CPU cores)
			"-f", "avif",
		},
	}
	c.supportedConversions["audio/flac"] = ConversionProfile{
		ContentType: "audio",
		Args:        []string{"-c:a", "flac", "-f", "flac"},
	}
	c.supportedConversions["audio/opus"] = ConversionProfile{
		ContentType: "audio",
		Args:        []string{"-c:a", "libopus", "-f", "opus"},
	}

	// 2. Detect available video encoders
	available := c.getAvailableEncoders()

	// 3. Configure H.264 (MP4)
	h264Enc := c.selectBestEncoder(available, []string{
		"h264_nvenc", "h264_qsv", "h264_videotoolbox", "h264_amf",
		"h264_mf",     // Windows standard
		"h264_vulkan", // Universal Vulkan fallback
		"h264_vaapi",
		"libx264", // Software fallback
	})
	mp4Args := []string{"-c:v", h264Enc, "-c:a", "aac", "-b:a", "192k"}
	mp4Args = append(mp4Args, getEncoderSpecificFlags(h264Enc)...)    // Add quality flags
	mp4Args = append(mp4Args, "-f", "mp4", "-movflags", "+faststart") // Add file/muxer flags

	c.supportedConversions["video/mp4"] = ConversionProfile{
		ContentType: "video",
		Args:        mp4Args,
	}
	if h264Enc != "libx264" {
		c.logger.Info("Hardware acceleration enabled", "format", "video/mp4", "encoder", h264Enc)
	}

	// 4. Configure Webm (AV1)
	av1Enc := c.selectBestEncoder(available, []string{
		"av1_nvenc", "av1_qsv", "av1_amf",
		"av1_mf",     // Windows standard
		"av1_vulkan", // Universal Vulkan fallback
		"libsvtav1",  // Software fallback
	})

	av1Args := []string{"-c:v", av1Enc, "-c:a", "libopus"}
	av1Args = append(av1Args, getEncoderSpecificFlags(av1Enc)...) // Add dynamic quality flags
	av1Args = append(av1Args, "-f", "webm")                       // Add file/muxer flags

	c.supportedConversions["video/webm"] = ConversionProfile{
		ContentType: "video",
		Args:        av1Args,
	}
	if av1Enc != "libsvtav1" {
		c.logger.Info("Hardware acceleration enabled", "format", "video/webm", "encoder", av1Enc)
	}
}
