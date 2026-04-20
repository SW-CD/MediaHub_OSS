package ffmpeg

import (
	"context"
	"fmt"
	"log/slog"
	"mediahub_oss/internal/shared/customerrors"
	"os"
	"os/exec"
	"strings"
)

type FfmpegConverter struct {
	ffmpegPath           string
	ffprobePath          string
	logger               *slog.Logger
	supportedConversions map[string]ConversionProfile
	localServer          *LocalStreamServer
}

// Updated signature: now returns a pointer and an error
func NewFFMPEGConverter(ffmpegConfiguredPath string, ffprobeConfiguredPath string, logger *slog.Logger) (*FfmpegConverter, error) {
	var ffmpegPath string = ""
	var ffprobePath string = ""

	// --- FFmpeg Check ---
	if ffmpegConfiguredPath != "" {
		if _, err := os.Stat(ffmpegConfiguredPath); err == nil {
			logger.Info("Using configured FFmpeg path", "path", ffmpegConfiguredPath)
			ffmpegPath = ffmpegConfiguredPath
		} else {
			logger.Warn("Configured ffmpeg_path not found, falling back to system PATH.", "config_path", ffmpegConfiguredPath)
		}
	}

	if ffmpegPath == "" { // Only check PATH if not configured
		path, err := exec.LookPath("ffmpeg")
		if err != nil {
			logger.Warn("---------------------------------------------------------")
			logger.Warn("FFmpeg executable not found in configured path or system PATH.")
			logger.Warn("Auto-conversions will be DISABLED.")
			logger.Warn("---------------------------------------------------------")
			ffmpegPath = "" // Explicitly set to empty
		} else {
			logger.Info("FFmpeg found in PATH. Media conversion enabled.", "path", path)
			ffmpegPath = path
		}
	}

	// --- FFprobe Check ---
	if ffprobeConfiguredPath != "" {
		if _, err := os.Stat(ffprobeConfiguredPath); err == nil {
			logger.Info("Using configured FFprobe path", "path", ffprobeConfiguredPath)
			ffprobePath = ffprobeConfiguredPath
		} else {
			logger.Warn("Configured ffprobe_path not found, falling back to system PATH.", "path", ffprobeConfiguredPath)
		}
	}

	if ffprobePath == "" { // Only check PATH if not found or configured
		if ffmpegPath != "" {
			probePath := strings.Replace(ffmpegPath, "ffmpeg", "ffprobe", 1)
			if _, err := os.Stat(probePath); err == nil {
				logger.Info("Found ffprobe alongside ffmpeg in PATH", "path", probePath)
				ffprobePath = probePath
			}
		}
	}

	if ffprobePath == "" { // Still not found? Check PATH explicitly.
		path, err := exec.LookPath("ffprobe")
		if err != nil {
			logger.Warn("---------------------------------------------------------")
			logger.Warn("ffprobe executable not found. Metadata extraction will be disabled.")
			logger.Warn("---------------------------------------------------------")
			ffprobePath = ""
		} else {
			logger.Info("ffprobe found in PATH. Metadata extraction enabled.", "path", path)
			ffprobePath = path
		}
	}

	// --- Initialize the Local Stream Server ---
	streamServer, err := NewLocalStreamServer(logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize internal loopback server: %w", err)
	}

	converter := &FfmpegConverter{
		ffmpegPath:  ffmpegPath,
		ffprobePath: ffprobePath,
		logger:      logger,
		localServer: streamServer,
	}

	// Probe FFmpeg and set up hardware acceleration
	converter.initConversions()

	return converter, nil
}

// Ensure you add a Shutdown method so you can cleanly stop the loopback server when your app closes
func (ffmpegc *FfmpegConverter) Shutdown(ctx context.Context) error {
	if ffmpegc.localServer != nil {
		return ffmpegc.localServer.Shutdown(ctx)
	}
	return nil
}

// IsFFmpegAvailable checks if the ffmpeg executable path was successfully found.
func (ffmpegc *FfmpegConverter) IsFFmpegAvailable() bool {
	return ffmpegc.ffmpegPath != ""
}

// GetFFmpegPath returns the determined path to the ffmpeg executable.
func (ffmpegc *FfmpegConverter) GetFFmpegPath() (string, error) {
	if ffmpegc.IsFFmpegAvailable() {
		return ffmpegc.ffmpegPath, nil
	} else {
		return "", customerrors.ErrNotFound
	}
}

// IsFFprobeAvailable checks if the ffprobe executable path was successfully found.
func (ffmpegc *FfmpegConverter) IsFFprobeAvailable() bool {
	return ffmpegc.ffprobePath != ""
}

// GetFFprobePath returns the determined path to the ffprobe executable.
func (ffmpegc *FfmpegConverter) GetFFprobePath() (string, error) {
	if ffmpegc.IsFFprobeAvailable() {
		return ffmpegc.ffprobePath, nil
	} else {
		return "", customerrors.ErrNotFound
	}
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
		CommonArgs:  []string{"-c:v", "mjpeg"},
		FileArgs:    []string{"-f", "image2"},
		StreamArgs:  []string{"-f", "image2pipe"},
	}
	c.supportedConversions["image/webp"] = ConversionProfile{
		ContentType: "image",
		CommonArgs:  []string{"-c:v", "libwebp"},
		FileArgs:    []string{"-f", "webp"},
		StreamArgs:  []string{"-f", "image2pipe"},
	}
	c.supportedConversions["image/avif"] = ConversionProfile{
		ContentType: "image",
		CommonArgs:  []string{"-c:v", "libaom-av1", "-strict", "experimental"},
		FileArgs:    []string{"-f", "avif"},
		StreamArgs:  []string{"-f", "avif", "-movflags", "empty_moov+frag_keyframe+default_base_moof"},
	}
	c.supportedConversions["audio/flac"] = ConversionProfile{
		ContentType: "audio",
		CommonArgs:  []string{"-c:a", "flac"},
		FileArgs:    []string{"-f", "flac"},
		StreamArgs:  []string{"-f", "flac"}, // FLAC doesn't need special pipe flags
	}
	c.supportedConversions["audio/opus"] = ConversionProfile{
		ContentType: "audio",
		CommonArgs:  []string{"-c:a", "libopus"},
		FileArgs:    []string{"-f", "opus"},
		StreamArgs:  []string{"-f", "opus"},
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
	c.supportedConversions["video/mp4"] = ConversionProfile{
		ContentType: "video",
		CommonArgs:  []string{"-c:v", h264Enc, "-c:a", "aac"},
		FileArgs:    []string{"-f", "mp4"},
		StreamArgs:  []string{"-f", "mp4", "-movflags", "frag_keyframe+empty_moov"},
	}
	if h264Enc != "libx264" {
		c.logger.Info("Hardware acceleration enabled", "format", "video/mp4", "encoder", h264Enc)
	}

	// 4. Configure VP9 (WebM)
	vp9Enc := c.selectBestEncoder(available, []string{
		"vp9_nvenc", "vp9_qsv", "vp9_vaapi",
		"vp9_mf", "vp9_vulkan",
		"libvpx-vp9",
	})
	c.supportedConversions["video/webm"] = ConversionProfile{
		ContentType: "video",
		CommonArgs:  []string{"-c:v", vp9Enc, "-c:a", "libopus"},
		FileArgs:    []string{"-f", "webm"},
		StreamArgs:  []string{"-f", "webm", "-live", "1"}, // Added the -live flag for safe piping
	}
	if vp9Enc != "libvpx-vp9" {
		c.logger.Info("Hardware acceleration enabled", "format", "video/webm", "encoder", vp9Enc)
	}

	// 5. Configure AV1
	av1Enc := c.selectBestEncoder(available, []string{
		"av1_nvenc", "av1_qsv", "av1_amf",
		"av1_mf",     // Windows standard
		"av1_vulkan", // Universal Vulkan fallback
		"libsvtav1",  // Software fallback
	})
	c.supportedConversions["video/av1"] = ConversionProfile{
		ContentType: "video",
		CommonArgs:  []string{"-c:v", av1Enc, "-c:a", "libopus"},
		FileArgs:    []string{"-f", "webm"},
		StreamArgs:  []string{"-f", "webm", "-live", "1"}, // Crucial for piping WebM containers safely
	}
	if av1Enc != "libsvtav1" {
		c.logger.Info("Hardware acceleration enabled", "format", "video/av1", "encoder", av1Enc)
	}
}

// getAvailableEncoders asks FFmpeg for a list of all compiled encoders.
func (c *FfmpegConverter) getAvailableEncoders() string {
	if c.ffmpegPath == "" {
		return ""
	}
	cmd := exec.Command(c.ffmpegPath, "-hide_banner", "-encoders")
	out, err := cmd.Output()
	if err != nil {
		c.logger.Warn("Failed to probe ffmpeg encoders, falling back to software defaults", "error", err)
		return ""
	}
	return string(out)
}

// selectBestEncoder checks the FFmpeg output and runs a hardware test to ensure viability.
func (c *FfmpegConverter) selectBestEncoder(available string, preferences []string) string {
	for i, enc := range preferences {
		// The last option in the array is our software fallback (e.g., libx264).
		// We usually assume the software fallback works if it is compiled.
		isSoftwareFallback := i == len(preferences)-1

		searchStr := fmt.Sprintf(" %s ", enc)
		if strings.Contains(available, searchStr) {

			if isSoftwareFallback {
				return enc // Accept software fallback immediately
			}

			// It is compiled, but does the hardware actually support it?
			c.logger.Debug("Probing hardware encoder capability...", "encoder", enc)
			if c.testEncoder(enc) {
				return enc // Hardware test passed!
			} else {
				c.logger.Debug("Encoder compiled but unsupported by system hardware/drivers", "encoder", enc)
			}
		}
	}
	// Ultimate fallback, should ideally never be reached if preferences are set up right
	return preferences[len(preferences)-1]
}

// testEncoder runs a tiny, 1-frame dummy conversion to verify if the hardware actually supports the encoder.
func (c *FfmpegConverter) testEncoder(encoder string) bool {
	// -f lavfi -i color=... : Generates a blank video stream entirely in memory
	// -vframes 1            : Only process a single frame
	// -c:v <encoder>        : Try using our specific hardware encoder
	// -f null -             : Throw the output away (don't write to disk)
	cmd := exec.Command(c.ffmpegPath,
		"-v", "error", // Only output fatal errors to keep logs clean
		"-f", "lavfi", "-i", "color=size=128x128:rate=1:duration=1",
		"-vframes", "1",
		"-c:v", encoder,
		"-f", "null", "-",
	)

	err := cmd.Run()
	// If err is nil, the hardware successfully initialized and encoded the frame!
	return err == nil
}
