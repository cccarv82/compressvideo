package ffmpeg

import (
	"os"
	"path/filepath"
)

// VideoFile represents a video file with its metadata
type VideoFile struct {
	Path      string            // Full path to the video file
	Size      int64             // Size in bytes
	Format    string            // Container format (mp4, mkv, etc.)
	Duration  float64           // Duration in seconds
	BitRate   int64             // Overall bitrate in bits/s
	VideoInfo VideoStreamInfo   // Information about the video stream
	AudioInfo []AudioStreamInfo // Information about audio streams
	Metadata  map[string]string // Additional metadata
}

// VideoStreamInfo contains information about a video stream
type VideoStreamInfo struct {
	Codec         string  // Video codec (h264, h265, etc.)
	Width         int     // Width in pixels
	Height        int     // Height in pixels
	FPS           float64 // Frames per second
	BitRate       int64   // Video bitrate in bits/s
	PixelFormat   string  // Pixel format (yuv420p, etc.)
	ColorSpace    string  // Color space
	IsHDR         bool    // Whether the video uses HDR
	HasBFrames    bool    // Whether the video uses B-frames
	ProfileLevel  string  // Codec profile level
}

// AudioStreamInfo contains information about an audio stream
type AudioStreamInfo struct {
	Index         int    // Stream index
	Codec         string // Audio codec (aac, opus, etc.)
	Channels      int    // Number of channels
	SampleRate    int    // Sample rate in Hz
	BitRate       int64  // Audio bitrate in bits/s
	Language      string // Language code
}

// GetVideoInfo extracts information about a video file
// This is a placeholder for now and will be implemented later
func GetVideoInfo(filePath string) (*VideoFile, error) {
	// For now, just return basic file information
	stat, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	return &VideoFile{
		Path:     filePath,
		Size:     stat.Size(),
		Format:   filepath.Ext(filePath)[1:], // Remove the dot
		Metadata: make(map[string]string),
	}, nil
} 