package analyzer

import (
	"fmt"
	"math"
	"path/filepath"
	"strings"

	"github.com/cccarv82/compressvideo/pkg/ffmpeg"
	"github.com/cccarv82/compressvideo/pkg/util"
)

// ContentType represents different types of video content
type ContentType int

const (
	// ContentTypeUnknown is the default content type
	ContentTypeUnknown ContentType = iota
	// ContentTypeAnimation for cartoon, anime, or 3D animation
	ContentTypeAnimation
	// ContentTypeScreencast for screen recordings, presentations
	ContentTypeScreencast
	// ContentTypeGaming for video game footage
	ContentTypeGaming
	// ContentTypeLiveAction for real-world video footage
	ContentTypeLiveAction
	// ContentTypeSportsAction for high-motion sports content
	ContentTypeSportsAction
	// ContentTypeDocumentary for documentary style content
	ContentTypeDocumentary
)

// String returns the string representation of a content type
func (c ContentType) String() string {
	switch c {
	case ContentTypeAnimation:
		return "Animation"
	case ContentTypeScreencast:
		return "Screencast"
	case ContentTypeGaming:
		return "Gaming"
	case ContentTypeLiveAction:
		return "Live Action"
	case ContentTypeSportsAction:
		return "Sports Action"
	case ContentTypeDocumentary:
		return "Documentary"
	default:
		return "Unknown"
	}
}

// MotionComplexity represents the level of motion complexity in a video
type MotionComplexity int

const (
	// MotionComplexityLow for static content with minimal motion
	MotionComplexityLow MotionComplexity = iota + 1
	// MotionComplexityMedium for moderate motion
	MotionComplexityMedium
	// MotionComplexityHigh for high motion content
	MotionComplexityHigh
	// MotionComplexityVeryHigh for very dynamic content
	MotionComplexityVeryHigh
)

// String returns the string representation of motion complexity
func (m MotionComplexity) String() string {
	switch m {
	case MotionComplexityLow:
		return "Low"
	case MotionComplexityMedium:
		return "Medium"
	case MotionComplexityHigh:
		return "High"
	case MotionComplexityVeryHigh:
		return "Very High"
	default:
		return "Unknown"
	}
}

// VideoAnalysis represents the results of video content analysis
type VideoAnalysis struct {
	VideoFile       *ffmpeg.VideoFile  // Original video file metadata
	ContentType     ContentType        // Detected content type
	MotionComplexity MotionComplexity  // Motion complexity level
	SceneChanges    int                // Number of scene changes
	FrameComplexity float64            // Average frame complexity
	CompressionPotential int           // Estimated compression potential (%)
	RecommendedCodec string            // Recommended codec for compression
	OptimalBitrate  int64              // Optimal bitrate for target quality
	SpatialComplexity float64          // Spatial complexity (detail level)
	IsHDContent     bool               // Whether the content is HD (720p+)
	IsUHDContent    bool               // Whether the content is UHD (4K+)
}

// ContentAnalyzer analyzes video content to determine optimal compression settings
type ContentAnalyzer struct {
	FFmpeg *ffmpeg.FFmpeg
	Logger *util.Logger
}

// NewContentAnalyzer creates a new content analyzer
func NewContentAnalyzer(ffmpeg *ffmpeg.FFmpeg, logger *util.Logger) *ContentAnalyzer {
	return &ContentAnalyzer{
		FFmpeg: ffmpeg,
		Logger: logger,
	}
}

// AnalyzeVideo performs comprehensive analysis of a video file
func (ca *ContentAnalyzer) AnalyzeVideo(videoFile *ffmpeg.VideoFile) (*VideoAnalysis, error) {
	ca.Logger.Info("Analyzing video content: %s", filepath.Base(videoFile.Path))
	
	analysis := &VideoAnalysis{
		VideoFile: videoFile,
	}
	
	// Detect content type based on filename, format, and video properties
	analysis.ContentType = ca.detectContentType(videoFile)
	ca.Logger.Info("Detected content type: %s", analysis.ContentType)
	
	// Analyze scene changes to determine content complexity
	sceneChanges, err := ca.FFmpeg.DetectSceneChanges(videoFile.Path, 0.3)
	if err != nil {
		ca.Logger.Error("Failed to detect scene changes: %v", err)
		// Continue with analysis, as this is not critical
	} else {
		analysis.SceneChanges = len(sceneChanges)
		ca.Logger.Debug("Detected %d scene changes", analysis.SceneChanges)
	}
	
	// Calculate frame complexity
	frameComplexity, err := ca.FFmpeg.CalculateFrameComplexity(videoFile.Path)
	if err != nil {
		ca.Logger.Error("Failed to calculate frame complexity: %v", err)
		// Use a default value based on content type
		switch analysis.ContentType {
		case ContentTypeScreencast:
			frameComplexity = 100
		case ContentTypeAnimation:
			frameComplexity = 200
		default:
			frameComplexity = 500
		}
	}
	analysis.FrameComplexity = frameComplexity
	
	// Determine motion complexity
	analysis.MotionComplexity = ca.determineMotionComplexity(videoFile, analysis.SceneChanges, analysis.FrameComplexity)
	ca.Logger.Info("Determined motion complexity: %s", analysis.MotionComplexity)
	
	// Calculate spatial complexity (image detail level)
	analysis.SpatialComplexity = ca.calculateSpatialComplexity(videoFile, frameComplexity)
	
	// Check if content is HD or UHD
	analysis.IsHDContent = videoFile.VideoInfo.Height >= 720
	analysis.IsUHDContent = videoFile.VideoInfo.Height >= 2160 || videoFile.VideoInfo.Width >= 3840
	
	// Determine optimal codec
	analysis.RecommendedCodec = ca.determineOptimalCodec(videoFile, analysis.ContentType)
	ca.Logger.Info("Recommended codec: %s", analysis.RecommendedCodec)
	
	// Calculate optimal bitrate
	analysis.OptimalBitrate = ca.calculateOptimalBitrate(videoFile, analysis)
	ca.Logger.Info("Optimal bitrate: %d kbps", analysis.OptimalBitrate/1000)
	
	// Estimate compression potential
	analysis.CompressionPotential = ca.estimateCompressionPotential(videoFile, analysis)
	ca.Logger.Info("Estimated compression potential: %d%%", analysis.CompressionPotential)
	
	return analysis, nil
}

// detectContentType attempts to determine the type of content in the video
func (ca *ContentAnalyzer) detectContentType(videoFile *ffmpeg.VideoFile) ContentType {
	filename := strings.ToLower(filepath.Base(videoFile.Path))
	
	// Check for common indicators in filename
	if containsAny(filename, []string{"screencast", "screen", "capture", "tutorial", "recording", "desktop", "presentation"}) {
		return ContentTypeScreencast
	}
	
	if containsAny(filename, []string{"anime", "animation", "cartoon", "animated", "3d", "cgi"}) {
		return ContentTypeAnimation
	}
	
	if containsAny(filename, []string{"game", "gaming", "gameplay", "playthrough", "walkthrough", "let's play"}) {
		return ContentTypeGaming
	}
	
	if containsAny(filename, []string{"sports", "football", "soccer", "basketball", "hockey", "match", "race"}) {
		return ContentTypeSportsAction
	}
	
	if containsAny(filename, []string{"documentary", "nature", "wildlife", "science", "history"}) {
		return ContentTypeDocumentary
	}
	
	// If no match in filename, analyze video properties
	
	// Screencasts typically have very steady frame rates and limited color palettes
	if videoFile.VideoInfo.FPS <= 30 && 
	   (videoFile.VideoInfo.Width == 1920 || videoFile.VideoInfo.Width == 1280 || 
		videoFile.VideoInfo.Width == 1366 || videoFile.VideoInfo.Width == 1440) {
		// Common screen resolutions for screencasts
		return ContentTypeScreencast
	}
	
	// Animation often has specific aspect ratios and frame rates
	if videoFile.VideoInfo.FPS < 30 && 
	   (videoFile.VideoInfo.Width == 1920 || videoFile.VideoInfo.Width == 1280) &&
	   !videoFile.VideoInfo.IsHDR {
		return ContentTypeAnimation
	}
	
	// Gaming content often has specific frame rates and resolutions
	if (videoFile.VideoInfo.FPS == 30 || videoFile.VideoInfo.FPS == 60) && 
	   (videoFile.VideoInfo.Width == 1920 || videoFile.VideoInfo.Width == 2560 || 
		videoFile.VideoInfo.Width == 3840) {
		return ContentTypeGaming
	}
	
	// Default to live action for real-world content
	return ContentTypeLiveAction
}

// determineMotionComplexity analyzes the video to determine motion complexity
func (ca *ContentAnalyzer) determineMotionComplexity(videoFile *ffmpeg.VideoFile, sceneChanges int, frameComplexity float64) MotionComplexity {
	// Calculate scene changes per minute
	durationMinutes := videoFile.Duration / 60
	if durationMinutes <= 0 {
		durationMinutes = 1
	}
	sceneChangesPerMinute := float64(sceneChanges) / durationMinutes
	
	// Determine based on scene changes and frame complexity
	if sceneChangesPerMinute < 2 && frameComplexity < 200 {
		return MotionComplexityLow
	} else if sceneChangesPerMinute < 5 && frameComplexity < 500 {
		return MotionComplexityMedium
	} else if sceneChangesPerMinute < 10 && frameComplexity < 1000 {
		return MotionComplexityHigh
	} else {
		return MotionComplexityVeryHigh
	}
}

// calculateSpatialComplexity determines the level of detail in the video frames
func (ca *ContentAnalyzer) calculateSpatialComplexity(videoFile *ffmpeg.VideoFile, frameComplexity float64) float64 {
	// Base spatial complexity on resolution and measured frame complexity
	resolution := float64(videoFile.VideoInfo.Width * videoFile.VideoInfo.Height)
	normalizedResolution := math.Log10(resolution) / math.Log10(1920*1080) // Normalize to Full HD
	
	// Combine with frame complexity
	normalizedFrameComplexity := frameComplexity / 500 // Normalize to a reference value
	
	// Weight resolution more for screencasts, weight frame complexity more for live action
	spatialComplexity := 0.0
	switch ca.detectContentType(videoFile) {
	case ContentTypeScreencast:
		spatialComplexity = 0.7*normalizedResolution + 0.3*normalizedFrameComplexity
	case ContentTypeAnimation:
		spatialComplexity = 0.5*normalizedResolution + 0.5*normalizedFrameComplexity
	default:
		spatialComplexity = 0.3*normalizedResolution + 0.7*normalizedFrameComplexity
	}
	
	return spatialComplexity
}

// determineOptimalCodec recommends the best codec based on content type
func (ca *ContentAnalyzer) determineOptimalCodec(videoFile *ffmpeg.VideoFile, contentType ContentType) string {
	// Check if the system has hardware acceleration capability
	// For now, just assume no hardware acceleration
	
	// If content is HDR, must use a codec that supports it
	if videoFile.VideoInfo.IsHDR {
		return "hevc" // H.265 has better HDR support
	}
	
	// Select codec based on content type
	switch contentType {
	case ContentTypeAnimation:
		return "vp9" // VP9 is often better for animation
	case ContentTypeScreencast:
		return "hevc" // H.265 works well for screencast content
	case ContentTypeGaming:
		// Gaming benefits from superior texture preservation
		if videoFile.VideoInfo.Height >= 1080 {
			return "hevc"
		}
		return "h264"
	case ContentTypeSportsAction:
		// Sports needs good motion handling
		if videoFile.VideoInfo.Height >= 1080 {
			return "hevc"
		}
		return "h264"
	default:
		// For general live action, H.264 has best compatibility
		if videoFile.VideoInfo.Height >= 1440 {
			return "hevc"
		}
		return "h264"
	}
}

// calculateOptimalBitrate determines the ideal bitrate for the content
func (ca *ContentAnalyzer) calculateOptimalBitrate(videoFile *ffmpeg.VideoFile, analysis *VideoAnalysis) int64 {
	// Calculate base bitrate based on resolution and frame rate
	var baseBitrate float64
	
	// Resolution factor
	pixelCount := float64(videoFile.VideoInfo.Width * videoFile.VideoInfo.Height)
	
	// Different bitrate formulas based on content type
	switch analysis.ContentType {
	case ContentTypeScreencast:
		// Screencasts can use much lower bitrates
		baseBitrate = pixelCount * videoFile.VideoInfo.FPS * 0.0001
	case ContentTypeAnimation:
		// Animation also compresses efficiently
		baseBitrate = pixelCount * videoFile.VideoInfo.FPS * 0.00015
	case ContentTypeGaming:
		// Gaming needs more bits for texture preservation
		baseBitrate = pixelCount * videoFile.VideoInfo.FPS * 0.0003
	case ContentTypeSportsAction:
		// Sports needs higher bitrate for motion
		baseBitrate = pixelCount * videoFile.VideoInfo.FPS * 0.00035
	default:
		// Standard formula for live action
		baseBitrate = pixelCount * videoFile.VideoInfo.FPS * 0.00025
	}
	
	// Adjust for motion complexity
	motionFactor := 1.0
	switch analysis.MotionComplexity {
	case MotionComplexityLow:
		motionFactor = 0.7
	case MotionComplexityMedium:
		motionFactor = 1.0
	case MotionComplexityHigh:
		motionFactor = 1.3
	case MotionComplexityVeryHigh:
		motionFactor = 1.6
	}
	
	// Adjust for codec efficiency
	codecFactor := 1.0
	switch analysis.RecommendedCodec {
	case "h264":
		codecFactor = 1.0
	case "hevc": // H.265
		codecFactor = 0.6 // HEVC is about 40% more efficient than H.264
	case "vp9":
		codecFactor = 0.7 // VP9 is about 30% more efficient than H.264
	case "av1":
		codecFactor = 0.5 // AV1 is about 50% more efficient than H.264
	}
	
	// Calculate the adjusted bitrate
	adjustedBitrate := baseBitrate * motionFactor * codecFactor
	
	// Apply minimum/maximum boundaries to ensure reasonable quality
	minBitrate := 500000.0  // 500 Kbps minimum
	maxBitrate := 15000000.0 // 15 Mbps maximum
	
	if adjustedBitrate < minBitrate {
		adjustedBitrate = minBitrate
	} else if adjustedBitrate > maxBitrate {
		adjustedBitrate = maxBitrate
	}
	
	return int64(adjustedBitrate)
}

// estimateCompressionPotential estimates how much the video can be compressed
func (ca *ContentAnalyzer) estimateCompressionPotential(videoFile *ffmpeg.VideoFile, analysis *VideoAnalysis) int {
	// Cannot estimate if we don't have original bitrate
	if videoFile.VideoInfo.Bitrate <= 0 {
		// Make a rougher estimate based on content type
		switch analysis.ContentType {
		case ContentTypeScreencast:
			return 80 // Screencasts typically compress very well
		case ContentTypeAnimation:
			return 70 // Animation also compresses well
		case ContentTypeGaming:
			return 50 // Gaming varies widely
		case ContentTypeSportsAction:
			return 40 // Sports action is hard to compress well
		default:
			return 50 // Default to 50% for unknown types
		}
	}
	
	// Calculate compression ratio
	compressionRatio := float64(videoFile.VideoInfo.Bitrate) / float64(analysis.OptimalBitrate)
	
	// Estimate the percentage savings
	potentialSavings := (1.0 - (1.0 / compressionRatio)) * 100
	
	// Ensure the result is reasonable
	if potentialSavings < 0 {
		// If original is already well-compressed, suggest modest gain
		return 10
	} else if potentialSavings > 90 {
		// Cap at 90% to avoid over-promising
		return 90
	}
	
	return int(potentialSavings)
}

// containsAny checks if a string contains any of the given substrings
func containsAny(s string, substrings []string) bool {
	for _, sub := range substrings {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// GetCompressionSettings returns FFmpeg settings based on analysis
func (ca *ContentAnalyzer) GetCompressionSettings(analysis *VideoAnalysis, quality int) (map[string]string, error) {
	settings := make(map[string]string)
	
	// Base FFmpeg command (common for all)
	settings["input"] = analysis.VideoFile.Path
	
	// Determine codec-specific parameters
	switch analysis.RecommendedCodec {
	case "h264":
		settings["codec"] = "libx264"
		
		// Quality presets (1=max compression, 5=max quality)
		presets := []string{"veryslow", "slower", "medium", "fast", "ultrafast"}
		settings["preset"] = presets[quality-1]
		
		// CRF (Constant Rate Factor) settings adjusted by content type
		// Lower CRF = Higher quality
		baseCRF := 23 // Default for balanced quality
		
		// Adjust CRF based on quality level (higher quality = lower CRF)
		qualityAdjustment := (3 - quality) * 3
		
		// Adjust CRF based on content type
		contentAdjustment := 0
		switch analysis.ContentType {
		case ContentTypeScreencast:
			contentAdjustment = 3 // Screencasts can use higher CRF (lower quality)
		case ContentTypeAnimation:
			contentAdjustment = 1 // Animation needs slightly higher quality
		case ContentTypeGaming:
			contentAdjustment = -1 // Gaming needs higher quality
		case ContentTypeSportsAction:
			contentAdjustment = -2 // Sports needs even higher quality
		}
		
		// Final CRF calculation
		crf := baseCRF + qualityAdjustment + contentAdjustment
		
		// Ensure CRF stays within reasonable bounds
		if crf < 18 {
			crf = 18
		} else if crf > 28 {
			crf = 28
		}
		
		settings["crf"] = fmt.Sprintf("%d", crf)
		
		// Additional settings
		settings["profile"] = "high"
		settings["level"] = "4.1"
		settings["tune"] = getTuneParameter(analysis.ContentType)
		
	case "hevc":
		settings["codec"] = "libx265"
		
		// Quality presets
		presets := []string{"veryslow", "slow", "medium", "fast", "ultrafast"}
		settings["preset"] = presets[quality-1]
		
		// x265 uses a different CRF scale
		baseCRF := 28 // Default for balanced quality
		
		// Adjust CRF based on quality level
		qualityAdjustment := (3 - quality) * 3
		
		// Adjust CRF based on content type
		contentAdjustment := 0
		switch analysis.ContentType {
		case ContentTypeScreencast:
			contentAdjustment = 3
		case ContentTypeAnimation:
			contentAdjustment = 1
		case ContentTypeGaming:
			contentAdjustment = -1
		case ContentTypeSportsAction:
			contentAdjustment = -2
		}
		
		// Final CRF calculation
		crf := baseCRF + qualityAdjustment + contentAdjustment
		
		// Ensure CRF stays within reasonable bounds
		if crf < 20 {
			crf = 20
		} else if crf > 32 {
			crf = 32
		}
		
		settings["crf"] = fmt.Sprintf("%d", crf)
		
		// Additional settings
		settings["x265-params"] = "profile=main"
		if analysis.VideoFile.VideoInfo.IsHDR {
			settings["pix_fmt"] = "yuv420p10le" // 10-bit color for HDR
		}
		
	case "vp9":
		settings["codec"] = "libvpx-vp9"
		
		// For VP9, we use a different approach with target bitrate and CRF
		targetBitrate := analysis.OptimalBitrate / 1000 // Convert to kbps
		
		// Adjust based on quality setting
		if quality == 1 {
			targetBitrate = int64(float64(targetBitrate) * 0.7)
		} else if quality == 5 {
			targetBitrate = int64(float64(targetBitrate) * 1.3)
		}
		
		settings["bitrate"] = fmt.Sprintf("%d", targetBitrate)
		
		// Quality settings for VP9
		baseCRF := 31 // Default for balanced quality
		
		// Adjust CRF based on quality level
		qualityAdjustment := (3 - quality) * 3
		
		// Final CRF calculation
		crf := baseCRF + qualityAdjustment
		
		// Ensure CRF stays within reasonable bounds
		if crf < 15 {
			crf = 15
		} else if crf > 35 {
			crf = 35
		}
		
		settings["crf"] = fmt.Sprintf("%d", crf)
		settings["quality"] = "good"
		settings["speed"] = fmt.Sprintf("%d", 6-quality) // Speed 2-5 is good range
		
		// Set CPU usage based on preset
		cpuUsed := 6 - quality // 1 = best quality, 5 = fastest
		settings["cpu-used"] = fmt.Sprintf("%d", cpuUsed)
		
		if analysis.VideoFile.VideoInfo.IsHDR {
			settings["pix_fmt"] = "yuv420p10le" // 10-bit color for HDR
		}
	}
	
	// Set thread count for parallel processing
	settings["threads"] = "0" // Let FFmpeg decide based on system capabilities
	
	// Audio settings (generally keep the same quality)
	// Determine if we should transcode the audio
	if len(analysis.VideoFile.AudioInfo) > 0 {
		audioCodec := analysis.VideoFile.AudioInfo[0].Codec
		
		// Keep audio codec if it's already efficient
		if audioCodec == "aac" || audioCodec == "opus" {
			settings["audio_codec"] = "copy"
		} else {
			// Convert to AAC with reasonable quality
			settings["audio_codec"] = "aac"
			settings["audio_bitrate"] = "128k" // Good quality for most content
		}
	} else {
		// No audio detected
		settings["audio_codec"] = "copy"
	}
	
	return settings, nil
}

// getTuneParameter returns the appropriate tune parameter for x264/x265
func getTuneParameter(contentType ContentType) string {
	switch contentType {
	case ContentTypeAnimation:
		return "animation"
	case ContentTypeScreencast:
		return "stillimage"
	case ContentTypeGaming:
		return "grain" // Preserve the texture in games
	case ContentTypeSportsAction:
		return "zerolatency" // Better for high motion
	default:
		return "film" // Good default for general content
	}
} 