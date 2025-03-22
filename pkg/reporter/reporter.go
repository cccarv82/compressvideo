package reporter

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cccarv82/compressvideo/pkg/analyzer"
	"github.com/cccarv82/compressvideo/pkg/compressor"
	"github.com/cccarv82/compressvideo/pkg/ffmpeg"
	"github.com/cccarv82/compressvideo/pkg/util"
)

// Report contains all the information about a compression operation
type Report struct {
	InputFile        string
	OutputFile       string
	OriginalVideo    *ffmpeg.VideoFile
	Analysis         *analyzer.VideoAnalysis
	Result           *compressor.CompressionResult
	StartTime        time.Time
	CompletionTime   time.Time
	CompressionTips  []string
	QualityEstimate  string
	TimeSaved        float64  // Estimated time saved in transfer or playback
	StorageSaved     float64  // Amount of storage space saved
	PerformanceScore float64  // Score from 0-100 on the compression
}

// ReportGenerator creates and manages compression reports
type ReportGenerator struct {
	Logger *util.Logger
	FFmpeg *ffmpeg.FFmpeg
}

// NewReportGenerator creates a new report generator
func NewReportGenerator(logger *util.Logger, ffmpeg *ffmpeg.FFmpeg) *ReportGenerator {
	return &ReportGenerator{
		Logger: logger,
		FFmpeg: ffmpeg,
	}
}

// CreateReport initializes a new report with basic information
func (rg *ReportGenerator) CreateReport(inputFile, outputFile string, 
	originalVideo *ffmpeg.VideoFile, analysis *analyzer.VideoAnalysis) *Report {
	
	return &Report{
		InputFile:      inputFile,
		OutputFile:     outputFile,
		OriginalVideo:  originalVideo,
		Analysis:       analysis,
		StartTime:      time.Now(),
		CompressionTips: []string{},
	}
}

// FinalizeReport completes the report with results and calculates scores
func (rg *ReportGenerator) FinalizeReport(report *Report, result *compressor.CompressionResult) *Report {
	report.Result = result
	report.CompletionTime = time.Now()
	
	// Generate performance score
	report.PerformanceScore = rg.calculatePerformanceScore(report)
	
	// Generate quality estimate
	report.QualityEstimate = rg.getQualityEstimate(result.AverageFrameQuality)
	
	// Calculate estimated time savings
	// Using video duration and size difference to estimate saved download time on a 10 Mbps connection
	sizeDiffMB := float64(result.SavedSpaceBytes) / (1024 * 1024)
	// Assuming 10 Mbps download speed
	report.TimeSaved = (sizeDiffMB * 8) / 10  // Time saved in seconds
	
	// Storage space saved in MB
	report.StorageSaved = sizeDiffMB
	
	// Generate compression tips
	report.CompressionTips = rg.generateCompressionTips(report)
	
	return report
}

// generateCompressionTips provides tips based on compression result
func (rg *ReportGenerator) generateCompressionTips(report *Report) []string {
	tips := []string{}
	
	// Add tips based on compression ratio
	if report.Result.SavedSpacePercent < 10 {
		tips = append(tips, "This video was already well optimized or contains content that doesn't compress well.")
		
		// Suggest different codec if using h264
		if report.Analysis.RecommendedCodec == "libx264" {
			tips = append(tips, "Try using HEVC (H.265) codec for potentially better compression, although it may reduce compatibility.")
		}
	}
	
	// Add tips based on content type
	switch report.Analysis.ContentType {
	case analyzer.ContentTypeScreencast:
		tips = append(tips, "Screencasts often benefit from higher CRF values. Consider using CRF 28-32 if quality is acceptable.")
	case analyzer.ContentTypeAnimation:
		tips = append(tips, "For animation, consistent quality encoding (CRF) typically works better than targeting a specific bitrate.")
	case analyzer.ContentTypeLiveAction:
		if report.Analysis.MotionComplexity == analyzer.MotionComplexityHigh || 
		   report.Analysis.MotionComplexity == analyzer.MotionComplexityVeryHigh {
			tips = append(tips, "High-motion content requires higher bitrates to maintain quality. Consider a higher quality setting for critical content.")
		}
	}
	
	// Add tip about audio if original has high audio bitrate
	if len(report.OriginalVideo.AudioInfo) > 0 && report.OriginalVideo.AudioInfo[0].BitRate > 192000 {
		tips = append(tips, "Audio is using a high bitrate. Consider using 128kbps AAC for most content, or 192kbps for music videos.")
	}
	
	// Add tip about resolution
	if report.OriginalVideo.VideoInfo.Height >= 1080 && report.Result.SavedSpacePercent < 30 {
		tips = append(tips, "Consider downscaling to 720p if this video doesn't require full HD resolution.")
	}
	
	// Add tip for short videos with poor compression
	if report.OriginalVideo.Duration < 60 && report.Result.SavedSpacePercent < 20 {
		tips = append(tips, "Short videos often have less compression potential due to fewer redundant frames.")
	}
	
	return tips
}

// calculatePerformanceScore rates the compression from 0-100
func (rg *ReportGenerator) calculatePerformanceScore(report *Report) float64 {
	// Start with a base score
	score := 50.0
	
	// Adjust based on space saved percentage (max +40 points)
	spaceSavedFactor := report.Result.SavedSpacePercent / 100
	score += spaceSavedFactor * 40
	
	// Adjust based on quality (max +30 points)
	qualityFactor := report.Result.AverageFrameQuality / 100
	score += qualityFactor * 30
	
	// Adjust for processing time relative to video duration
	// Faster compression gets a small bonus (max +10 points)
	processingTimeRatio := report.OriginalVideo.Duration / report.Result.ProcessingTime.Seconds()
	if processingTimeRatio > 1 {
		// Processed faster than video duration
		timeFactor := (processingTimeRatio - 1) / 9  // Cap at 10x real-time for max bonus
		if timeFactor > 1 {
			timeFactor = 1
		}
		score += timeFactor * 10
	}
	
	// Cap score at 0-100
	if score < 0 {
		score = 0
	} else if score > 100 {
		score = 100
	}
	
	return score
}

// getQualityEstimate provides a text description of the visual quality
func (rg *ReportGenerator) getQualityEstimate(qualityScore float64) string {
	if qualityScore >= 90 {
		return "Excellent - Visually identical to the original"
	} else if qualityScore >= 80 {
		return "Very Good - Differences barely noticeable"
	} else if qualityScore >= 70 {
		return "Good - Minor visible differences"
	} else if qualityScore >= 60 {
		return "Acceptable - Visible differences but good for most purposes"
	} else if qualityScore >= 50 {
		return "Medium - Noticeable quality loss"
	} else if qualityScore >= 40 {
		return "Low - Significant quality loss"
	} else {
		return "Poor - Heavy compression artifacts"
	}
}

// DisplayReportToConsole shows a comprehensive report in the terminal
func (rg *ReportGenerator) DisplayReportToConsole(report *Report) {
	logger := rg.Logger
	
	// Header
	logger.Info("═════════════════════════════════════════════")
	logger.Info("        COMPRESSION OPERATION REPORT         ")
	logger.Info("═════════════════════════════════════════════")
	
	// Input/Output Information
	logger.Info("📁 FILES:")
	logger.Info("  Input:  %s", report.InputFile)
	logger.Info("  Output: %s", report.OutputFile)
	
	// Video Information
	logger.Info("\n🎬 VIDEO DETAILS:")
	logger.Info("  Resolution: %dx%d", report.OriginalVideo.VideoInfo.Width, report.OriginalVideo.VideoInfo.Height)
	logger.Info("  Duration:   %.2f seconds", report.OriginalVideo.Duration)
	logger.Info("  Content:    %s, %s motion", report.Analysis.ContentType, report.Analysis.MotionComplexity)
	
	// Compression Results
	logger.Info("\n📊 COMPRESSION RESULTS:")
	logger.Info("  Original Size:    %.2f MB", float64(report.Result.OriginalSize)/(1024*1024))
	logger.Info("  Compressed Size:  %.2f MB", float64(report.Result.CompressedSize)/(1024*1024))
	logger.Info("  Space Saved:      %.2f MB (%.1f%%)", float64(report.Result.SavedSpaceBytes)/(1024*1024), report.Result.SavedSpacePercent)
	logger.Info("  Compression Ratio: %.2f:1", report.Result.CompressionRatio)
	
	// Performance
	logger.Info("\n⏱️ PERFORMANCE:")
	logger.Info("  Processing Time:  %s", report.Result.ProcessingTime.Round(time.Second))
	logger.Info("  Quality Estimate: %s (%.1f/100)", report.QualityEstimate, report.Result.AverageFrameQuality)
	logger.Info("  Overall Score:    %.1f/100", report.PerformanceScore)
	
	if report.TimeSaved > 0 {
		if report.TimeSaved > 60 {
			minutes := int(report.TimeSaved) / 60
			seconds := int(report.TimeSaved) % 60
			logger.Info("  Est. Transfer Time Saved: %d min %d sec at 10 Mbps", minutes, seconds)
		} else {
			logger.Info("  Est. Transfer Time Saved: %.1f seconds at 10 Mbps", report.TimeSaved)
		}
	}
	
	// Codec & Settings
	logger.Info("\n⚙️ ENCODING SETTINGS:")
	logger.Info("  Video Codec: %s", report.Result.Settings["codec"])
	if crf, ok := report.Result.Settings["crf"]; ok {
		logger.Info("  Quality (CRF): %s", crf)
	}
	if preset, ok := report.Result.Settings["preset"]; ok {
		logger.Info("  Preset: %s", preset)
	}
	
	// Display tips
	if len(report.CompressionTips) > 0 {
		logger.Info("\n💡 OPTIMIZATION TIPS:")
		for _, tip := range report.CompressionTips {
			logger.Info("  • %s", tip)
		}
	}
	
	logger.Info("\n═════════════════════════════════════════════")
}

// SaveReportToFile saves the report as a text file
func (rg *ReportGenerator) SaveReportToFile(report *Report) (string, error) {
	// Create report name based on output filename
	baseName := filepath.Base(report.OutputFile)
	ext := filepath.Ext(baseName)
	reportName := strings.TrimSuffix(baseName, ext) + "_report.txt"
	reportPath := filepath.Join(filepath.Dir(report.OutputFile), reportName)
	
	// Open file for writing
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	
	// Write report content
	fmt.Fprintf(file, "COMPRESSION REPORT\n")
	fmt.Fprintf(file, "=======================================\n\n")
	
	fmt.Fprintf(file, "FILES:\n")
	fmt.Fprintf(file, "  Input:  %s\n", report.InputFile)
	fmt.Fprintf(file, "  Output: %s\n\n", report.OutputFile)
	
	fmt.Fprintf(file, "VIDEO DETAILS:\n")
	fmt.Fprintf(file, "  Resolution: %dx%d\n", report.OriginalVideo.VideoInfo.Width, report.OriginalVideo.VideoInfo.Height)
	fmt.Fprintf(file, "  Duration:   %.2f seconds\n", report.OriginalVideo.Duration)
	fmt.Fprintf(file, "  Content:    %s, %s motion\n\n", report.Analysis.ContentType, report.Analysis.MotionComplexity)
	
	fmt.Fprintf(file, "COMPRESSION RESULTS:\n")
	fmt.Fprintf(file, "  Original Size:    %.2f MB\n", float64(report.Result.OriginalSize)/(1024*1024))
	fmt.Fprintf(file, "  Compressed Size:  %.2f MB\n", float64(report.Result.CompressedSize)/(1024*1024))
	fmt.Fprintf(file, "  Space Saved:      %.2f MB (%.1f%%)\n", float64(report.Result.SavedSpaceBytes)/(1024*1024), report.Result.SavedSpacePercent)
	fmt.Fprintf(file, "  Compression Ratio: %.2f:1\n\n", report.Result.CompressionRatio)
	
	fmt.Fprintf(file, "PERFORMANCE:\n")
	fmt.Fprintf(file, "  Processing Time:  %s\n", report.Result.ProcessingTime.Round(time.Second))
	fmt.Fprintf(file, "  Quality Estimate: %s (%.1f/100)\n", report.QualityEstimate, report.Result.AverageFrameQuality)
	fmt.Fprintf(file, "  Overall Score:    %.1f/100\n\n", report.PerformanceScore)
	
	fmt.Fprintf(file, "ENCODING SETTINGS:\n")
	for key, value := range report.Result.Settings {
		fmt.Fprintf(file, "  %s: %s\n", key, value)
	}
	
	if len(report.CompressionTips) > 0 {
		fmt.Fprintf(file, "\nOPTIMIZATION TIPS:\n")
		for _, tip := range report.CompressionTips {
			fmt.Fprintf(file, "  • %s\n", tip)
		}
	}
	
	fmt.Fprintf(file, "\nReport generated on %s\n", time.Now().Format("2006-01-02 15:04:05"))
	
	return reportPath, nil
} 