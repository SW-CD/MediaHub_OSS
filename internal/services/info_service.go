// filepath: internal/services/info_service.go
package services

import (
	"mediahub/internal/models"
	"time"
)

var _ InfoService = (*infoService)(nil)

type infoService struct {
	Version          string
	StartTime        time.Time
	FFmpegAvailable  bool
	FFprobeAvailable bool
}

// NewInfoService creates a new InfoService.
func NewInfoService(version string, startTime time.Time, ffmpegAvailable bool, ffprobeAvailable bool) *infoService {
	return &infoService{
		Version:          version,
		StartTime:        startTime,
		FFmpegAvailable:  ffmpegAvailable,
		FFprobeAvailable: ffprobeAvailable,
	}
}

// GetInfo retrieves the application information.
func (s *infoService) GetInfo() models.Info {
	return models.Info{
		ServiceName:      "SWCD MediaHub-API",
		Version:          s.Version,
		UptimeSince:      s.StartTime,
		FFmpegAvailable:  s.FFmpegAvailable,
		FFprobeAvailable: s.FFprobeAvailable,
	}
}
