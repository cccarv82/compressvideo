package analyzer

import (
	"fmt"
	"math"
	"path/filepath"
	"strconv"
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

// estimateCompressionPotential estimates the potential compression percentage
func (ca *ContentAnalyzer) estimateCompressionPotential(videoFile *ffmpeg.VideoFile, analysis *VideoAnalysis) int {
	// If we can't determine bitrate, make an estimate based on other factors
	if videoFile.VideoInfo.BitRate <= 0 {
		// Estimate based on content type and resolution
		if analysis.IsUHDContent { // 4K
			return 50
		} else if analysis.IsHDContent { // HD
			return 60
		}
		return 65 // SD
	}
	
	// Calculate based on optimal bitrate vs current bitrate
	compressionRatio := float64(videoFile.VideoInfo.BitRate) / float64(analysis.OptimalBitrate)
	
	// Convert to percentage potential
	potential := int((1.0 - (1.0 / compressionRatio)) * 100)
	
	// Clamp value between 0 and 95%
	if potential < 0 {
		potential = 0
	} else if potential > 95 {
		potential = 95
	}
	
	return potential
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

// GetCompressionSettings calculates the optimal compression settings based on video analysis
func (ca *ContentAnalyzer) GetCompressionSettings(analysis *VideoAnalysis, qualityLevel int) (map[string]string, error) {
	if analysis == nil {
		return nil, fmt.Errorf("video analysis is required")
	}
	
	// Initialize settings map
	settings := make(map[string]string)
	
	// Select codec based on content type
	settings["codec"] = ca.selectCodec(analysis.ContentType)
	
	// Calculate optimal quality (CRF) value based on quality level and content type
	settings["crf"] = ca.calculateCRF(analysis.ContentType, analysis.MotionComplexity, qualityLevel)
	
	// Set preset based on quality level and motion complexity
	settings["preset"] = ca.selectPreset(qualityLevel, analysis.MotionComplexity)
	
	// Add codec-specific settings
	ca.addCodecSpecificSettings(settings, analysis)
	
	// Calculate optimal bitrate if needed
	optimalBitrateStr := ca.calculateOptimalBitrateString(analysis, qualityLevel)
	if optimalBitrateStr != "" {
		settings["bitrate"] = optimalBitrateStr
	}
	
	// Audio settings
	ca.setAudioSettings(settings, analysis)
	
	return settings, nil
}

// selectCodec chooses the most appropriate codec for the content type
func (ca *ContentAnalyzer) selectCodec(contentType ContentType) string {
	switch contentType {
	case ContentTypeScreencast, ContentTypeAnimation:
		// Screencasts and animations tend to have large flat areas and sharp edges
		// HEVC/H.265 is more efficient for this type of content
		return "libx265"
	case ContentTypeGaming:
		// Gaming has a mix of complex motion and UI elements
		// H.264 offers a good balance of compatibility and efficiency
		return "libx264"
	default:
		// For most other content types, H.264 provides good compatibility
		return "libx264"
	}
}

// calculateCRF determines the Constant Rate Factor for quality control
func (ca *ContentAnalyzer) calculateCRF(contentType ContentType, motionComplexity MotionComplexity, qualityLevel int) string {
	// Base CRF values per codec and content type
	// Lower CRF = Higher quality
	baseCRF := 23 // Default for H.264
	
	// Adjust for content type
	switch contentType {
	case ContentTypeScreencast:
		baseCRF = 28 // Screencasts can use higher CRF (lower quality) as they're simpler
	case ContentTypeAnimation:
		baseCRF = 26 // Animations can use higher CRF while maintaining quality
	case ContentTypeGaming:
		baseCRF = 23 // Gaming needs more quality to preserve details
	case ContentTypeLiveAction, ContentTypeDocumentary:
		baseCRF = 20 // Natural content needs lower CRF (higher quality)
	}
	
	// Adjust for motion complexity
	if motionComplexity == MotionComplexityHigh {
		baseCRF -= 2 // Higher motion needs better quality
	} else if motionComplexity == MotionComplexityLow {
		baseCRF += 2 // Lower motion can use higher CRF
	}
	
	// Adjust for quality level (1-5)
	// Quality 1 = max compression (higher CRF)
	// Quality 5 = max quality (lower CRF)
	qualityAdjustment := (3 - qualityLevel) * 2
	
	finalCRF := baseCRF + qualityAdjustment
	
	// Ensure CRF is within valid range
	if finalCRF < 18 {
		finalCRF = 18
	} else if finalCRF > 32 {
		finalCRF = 32
	}
	
	return strconv.Itoa(finalCRF)
}

// selectPreset chooses the FFmpeg preset based on quality level and motion complexity
func (ca *ContentAnalyzer) selectPreset(qualityLevel int, motionComplexity MotionComplexity) string {
	// Presets from fastest to slowest: ultrafast, superfast, veryfast, faster, fast, medium, slow, slower, veryslow
	
	// Base preset based on quality level
	// Higher quality levels use slower presets for better compression
	switch qualityLevel {
	case 1: // Maximum compression
		if motionComplexity == MotionComplexityHigh {
			return "veryslow"
		}
		return "slower"
	case 2:
		return "slow"
	case 3: // Balanced
		return "medium"
	case 4:
		return "fast"
	case 5: // Maximum quality
		return "ultrafast"
	default:
		return "medium"
	}
}

// addCodecSpecificSettings adds settings specific to the selected codec
func (ca *ContentAnalyzer) addCodecSpecificSettings(settings map[string]string, analysis *VideoAnalysis) {
	codec := settings["codec"]
	
	// Add x265 specific settings
	if codec == "libx265" {
		// Add appropriate profile
		// For 8-bit content, main profile is sufficient
		// For 10-bit content, main10 profile is needed
		settings["profile"] = "main"
		
		// For screencasts, tune for zero-latency can help
		if analysis.ContentType == ContentTypeScreencast {
			settings["tune"] = "zerolatency"
			// x265 specific parameters
			settings["x265-params"] = "bframes=0"
		}
	}
	
	// Add x264 specific settings
	if codec == "libx264" {
		// Set appropriate profile
		if analysis.ContentType == ContentTypeGaming || analysis.ContentType == ContentTypeLiveAction {
			settings["profile"] = "high"
			settings["level"] = "4.1"
		} else {
			settings["profile"] = "main"
			settings["level"] = "3.1"
		}
		
		// For film content, use film tuning
		if analysis.ContentType == ContentTypeLiveAction || analysis.ContentType == ContentTypeDocumentary {
			settings["tune"] = "film"
		}
	}
	
	// Set appropriate pixel format
	settings["pix_fmt"] = "yuv420p" // Most compatible format
}

// calculateOptimalBitrateString calculates the optimal bitrate for the video and returns as string
func (ca *ContentAnalyzer) calculateOptimalBitrateString(analysis *VideoAnalysis, qualityLevel int) string {
	// Get video dimensions
	width := analysis.VideoFile.VideoInfo.Width
	height := analysis.VideoFile.VideoInfo.Height
	
	// Calculate pixels per frame (in millions)
	pixelsPerFrame := float64(width * height) / 1000000.0
	
	// Base bitrate factors by content type (in Kbps per million pixels)
	var baseBitrateFactor float64
	
	switch analysis.ContentType {
	case ContentTypeScreencast:
		baseBitrateFactor = 1000 // Lower for screencasts
	case ContentTypeAnimation:
		baseBitrateFactor = 1200
	case ContentTypeGaming:
		baseBitrateFactor = 1800
	case ContentTypeLiveAction, ContentTypeDocumentary:
		baseBitrateFactor = 2000 // Higher for natural content
	default:
		baseBitrateFactor = 1500
	}
	
	// Adjust for motion complexity
	if analysis.MotionComplexity == MotionComplexityHigh {
		baseBitrateFactor *= 1.5 // Higher motion needs more bitrate
	} else if analysis.MotionComplexity == MotionComplexityLow {
		baseBitrateFactor *= 0.7 // Lower motion needs less bitrate
	}
	
	// Adjust for quality level (1-5)
	qualityMultiplier := 0.6 + float64(qualityLevel)*0.2 // Ranges from 0.8 to 1.6
	
	// Calculate final bitrate in Kbps
	bitrate := int(pixelsPerFrame * baseBitrateFactor * qualityMultiplier)
	
	// Set minimum bitrate based on resolution
	minBitrate := 500 // 500 Kbps
	if width >= 1920 || height >= 1080 {
		minBitrate = 2000 // 2 Mbps for 1080p
	} else if width >= 1280 || height >= 720 {
		minBitrate = 1000 // 1 Mbps for 720p
	}
	
	// Ensure bitrate is not below minimum
	if bitrate < minBitrate {
		bitrate = minBitrate
	}
	
	// Return bitrate in Kbps
	return fmt.Sprintf("%dk", bitrate)
}

// setAudioSettings adds audio encoding settings
func (ca *ContentAnalyzer) setAudioSettings(settings map[string]string, analysis *VideoAnalysis) {
	// Check if input has audio
	if len(analysis.VideoFile.AudioInfo) == 0 {
		return // No audio stream
	}
	
	// For most cases, copying the audio is fine
	settings["audio_codec"] = "copy"
	
	// If the audio bitrate is very high, we might want to re-encode
	if len(analysis.VideoFile.AudioInfo) > 0 && analysis.VideoFile.AudioInfo[0].BitRate > 128000 && 
	   (analysis.ContentType == ContentTypeScreencast || analysis.ContentType == ContentTypeAnimation) {
		// Use AAC with a reasonable bitrate for screencast/animation
		settings["audio_codec"] = "aac"
		settings["audio_bitrate"] = "128k"
	} else if len(analysis.VideoFile.AudioInfo) > 0 && analysis.VideoFile.AudioInfo[0].BitRate > 192000 && 
			  (analysis.ContentType == ContentTypeLiveAction || analysis.ContentType == ContentTypeDocumentary) {
		// Use a higher bitrate for content where audio quality is more important
		settings["audio_codec"] = "aac"
		settings["audio_bitrate"] = "192k"
	}
} 