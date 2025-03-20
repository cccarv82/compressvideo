package ffmpeg

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cccarv82/compressvideo/pkg/util"
)

// FFmpeg represents a wrapper around the ffmpeg command-line tool
type FFmpeg struct {
	FFmpegPath  string
	FFprobePath string
	Logger      *util.Logger
}

// NewFFmpeg creates a new FFmpeg wrapper instance
func NewFFmpeg(logger *util.Logger) (*FFmpeg, error) {
	// Check if FFmpeg is installed
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, fmt.Errorf("ffmpeg not found: %w", err)
	}

	// Check if FFprobe is installed
	ffprobePath, err := exec.LookPath("ffprobe")
	if err != nil {
		return nil, fmt.Errorf("ffprobe not found: %w", err)
	}

	return &FFmpeg{
		FFmpegPath:  ffmpegPath,
		FFprobePath: ffprobePath,
		Logger:      logger,
	}, nil
}

// GetVideoInfo retrieves detailed information about a video file using ffprobe
func (f *FFmpeg) GetVideoInfo(filePath string) (*VideoFile, error) {
	f.Logger.Debug("Getting video info for: %s", filePath)

	// Run ffprobe to get JSON output with all stream info
	cmd := exec.Command(
		f.FFprobePath,
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		filePath,
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe failed: %w", err)
	}

	// Parse the JSON output
	var ffprobeOutput struct {
		Streams []map[string]interface{} `json:"streams"`
		Format  map[string]interface{}   `json:"format"`
	}

	err = json.Unmarshal(output, &ffprobeOutput)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	// Create a new VideoFile
	videoFile := &VideoFile{
		Path:     filePath,
		Format:   filepath.Ext(filePath)[1:], // Remove the dot
		Metadata: make(map[string]string),
		AudioInfo: []AudioStreamInfo{},
	}

	// Extract duration and size from format section
	if durationStr, ok := ffprobeOutput.Format["duration"].(string); ok {
		videoFile.Duration, _ = strconv.ParseFloat(durationStr, 64)
	}
	if sizeStr, ok := ffprobeOutput.Format["size"].(string); ok {
		size, _ := strconv.ParseInt(sizeStr, 10, 64)
		videoFile.Size = size
	}
	if bitrateStr, ok := ffprobeOutput.Format["bit_rate"].(string); ok {
		bitrate, _ := strconv.ParseInt(bitrateStr, 10, 64)
		videoFile.Bitrate = bitrate
	}

	// Process each stream
	for _, stream := range ffprobeOutput.Streams {
		streamType, ok := stream["codec_type"].(string)
		if !ok {
			continue
		}

		if streamType == "video" {
			// Extract video stream info
			videoInfo := VideoStreamInfo{}
			
			// Extract codec
			if codec, ok := stream["codec_name"].(string); ok {
				videoInfo.Codec = codec
			}
			
			// Extract dimensions
			if width, ok := stream["width"].(float64); ok {
				videoInfo.Width = int(width)
			}
			if height, ok := stream["height"].(float64); ok {
				videoInfo.Height = int(height)
			}
			
			// Extract framerate
			if fpsStr, ok := stream["r_frame_rate"].(string); ok {
				parts := strings.Split(fpsStr, "/")
				if len(parts) == 2 {
					num, _ := strconv.ParseFloat(parts[0], 64)
					den, _ := strconv.ParseFloat(parts[1], 64)
					if den != 0 {
						videoInfo.FPS = num / den
					}
				}
			}
			
			// Extract pixel format
			if pixFmt, ok := stream["pix_fmt"].(string); ok {
				videoInfo.PixelFormat = pixFmt
			}
			
			// Extract profile level
			if profile, ok := stream["profile"].(string); ok {
				videoInfo.ProfileLevel = profile
			}
			
			// Extract video bitrate
			if bitrateStr, ok := stream["bit_rate"].(string); ok {
				bitrate, _ := strconv.ParseInt(bitrateStr, 10, 64)
				videoInfo.Bitrate = bitrate
			}
			
			// Check for B-frames using profile
			if profile, ok := stream["profile"].(string); ok {
				// Most profiles with B-frames have "High" in the name
				if strings.Contains(strings.ToLower(profile), "high") {
					videoInfo.HasBFrames = true
				}
			}
			
			// Check for HDR
			if tags, ok := stream["tags"].(map[string]interface{}); ok {
				if colorTransfer, ok := tags["color_transfer"].(string); ok {
					if strings.Contains(strings.ToLower(colorTransfer), "smpte2084") || 
					   strings.Contains(strings.ToLower(colorTransfer), "arib-std-b67") {
						videoInfo.IsHDR = true
					}
				}
			}
			
			videoFile.VideoInfo = videoInfo
			
		} else if streamType == "audio" {
			// Extract audio stream info
			audioInfo := AudioStreamInfo{}
			
			// Extract index
			if index, ok := stream["index"].(float64); ok {
				audioInfo.Index = int(index)
			}
			
			// Extract codec
			if codec, ok := stream["codec_name"].(string); ok {
				audioInfo.Codec = codec
			}
			
			// Extract channels
			if channels, ok := stream["channels"].(float64); ok {
				audioInfo.Channels = int(channels)
			}
			
			// Extract sample rate
			if sampleRateStr, ok := stream["sample_rate"].(string); ok {
				sampleRate, _ := strconv.Atoi(sampleRateStr)
				audioInfo.SampleRate = sampleRate
			}
			
			// Extract bitrate
			if bitrateStr, ok := stream["bit_rate"].(string); ok {
				bitrate, _ := strconv.ParseInt(bitrateStr, 10, 64)
				audioInfo.Bitrate = bitrate
			}
			
			// Extract language
			if tags, ok := stream["tags"].(map[string]interface{}); ok {
				if language, ok := tags["language"].(string); ok {
					audioInfo.Language = language
				}
			}
			
			videoFile.AudioInfo = append(videoFile.AudioInfo, audioInfo)
		}
	}

	f.Logger.Debug("Video info extraction completed for: %s", filePath)
	return videoFile, nil
}

// ExecuteCommand runs an FFmpeg command with the given arguments
func (f *FFmpeg) ExecuteCommand(args []string) ([]byte, error) {
	f.Logger.Debug("Executing FFmpeg command: %s %s", f.FFmpegPath, strings.Join(args, " "))
	
	cmd := exec.Command(f.FFmpegPath, args...)
	return cmd.CombinedOutput()
}

// DetectSceneChanges analyzes a video to detect scene changes
func (f *FFmpeg) DetectSceneChanges(filePath string, threshold float64) ([]float64, error) {
	f.Logger.Debug("Detecting scene changes in: %s", filePath)
	
	// If threshold not specified, use a default value
	if threshold <= 0 {
		threshold = 0.4 // Default threshold
	}
	
	// Use FFmpeg's scene detection filter
	args := []string{
		"-i", filePath,
		"-vf", fmt.Sprintf("select='gt(scene,%f)',metadata=print", threshold),
		"-f", "null",
		"-",
	}
	
	output, err := f.ExecuteCommand(args)
	if err != nil {
		return nil, fmt.Errorf("scene detection failed: %w", err)
	}
	
	// Parse the output to extract scene change timecodes
	sceneChanges := []float64{}
	lines := strings.Split(string(output), "\n")
	
	for _, line := range lines {
		if strings.Contains(line, "scene:") {
			parts := strings.Split(line, "scene:")
			if len(parts) >= 2 {
				sceneValue, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
				if err == nil && sceneValue >= threshold {
					// Extract timestamp (this is an approximation, might need refinement)
					timeParts := strings.Split(line, "pts_time:")
					if len(timeParts) >= 2 {
						timeStr := strings.Split(timeParts[1], " ")[0]
						timestamp, err := strconv.ParseFloat(timeStr, 64)
						if err == nil {
							sceneChanges = append(sceneChanges, timestamp)
						}
					}
				}
			}
		}
	}
	
	f.Logger.Debug("Detected %d scene changes", len(sceneChanges))
	return sceneChanges, nil
}

// CalculateFrameComplexity estimates the complexity of video frames
func (f *FFmpeg) CalculateFrameComplexity(filePath string) (float64, error) {
	f.Logger.Debug("Calculating frame complexity for: %s", filePath)
	
	// Use FFmpeg to extract frames and calculate complexity
	// This is a simplified approach using FFmpeg filters
	args := []string{
		"-i", filePath,
		"-vf", "select='eq(pict_type,I)',signalstats=stat=variance",
		"-f", "null",
		"-",
	}
	
	output, err := f.ExecuteCommand(args)
	if err != nil {
		return 0, fmt.Errorf("frame complexity analysis failed: %w", err)
	}
	
	// Parse the output to find the average variance (complexity)
	lines := strings.Split(string(output), "\n")
	var totalVariance float64
	var count int
	
	for _, line := range lines {
		if strings.Contains(line, "variance:") {
			parts := strings.Split(line, "variance:")
			if len(parts) >= 2 {
				variance, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
				if err == nil {
					totalVariance += variance
					count++
				}
			}
		}
	}
	
	if count == 0 {
		return 0, fmt.Errorf("no frames analyzed for complexity")
	}
	
	averageComplexity := totalVariance / float64(count)
	f.Logger.Debug("Average frame complexity: %f", averageComplexity)
	
	return averageComplexity, nil
} 