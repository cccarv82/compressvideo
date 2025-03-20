package util

import (
	"fmt"
	"os"
	"time"

	"github.com/schollz/progressbar/v3"
)

// ProgressTracker handles displaying progress for long-running operations
type ProgressTracker struct {
	bar         *progressbar.ProgressBar
	total       int64
	description string
	startTime   time.Time
	logger      *Logger
}

// NewProgressTracker creates a new progress tracker
func NewProgressTracker(total int64, description string, logger *Logger) *ProgressTracker {
	bar := progressbar.NewOptions64(
		total,
		progressbar.OptionSetDescription(description),
		progressbar.OptionSetWriter(os.Stdout),
		progressbar.OptionShowBytes(true),
		progressbar.OptionShowCount(),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "=",
			SaucerHead:    ">",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionSetRenderBlankState(true),
	)

	return &ProgressTracker{
		bar:         bar,
		total:       total,
		description: description,
		startTime:   time.Now(),
		logger:      logger,
	}
}

// Update updates the progress bar
func (p *ProgressTracker) Update(current int64) error {
	return p.bar.Set64(current)
}

// Increment increments the progress bar by the given amount
func (p *ProgressTracker) Increment(amount int64) error {
	return p.bar.Add64(amount)
}

// Finish completes the progress bar and displays final stats
func (p *ProgressTracker) Finish() {
	// Ensure bar shows 100%
	p.bar.Finish()
	duration := time.Since(p.startTime).Round(time.Second)
	fmt.Println() // Add newline after progress bar
	p.logger.Info("Operation completed in %s", duration)
}

// GetElapsedTime returns the elapsed time since the start
func (p *ProgressTracker) GetElapsedTime() time.Duration {
	return time.Since(p.startTime)
}

// EstimateTimeRemaining estimates the remaining time based on progress
func (p *ProgressTracker) EstimateTimeRemaining(current int64) time.Duration {
	if current <= 0 {
		return 0
	}
	
	elapsed := time.Since(p.startTime)
	rate := float64(current) / elapsed.Seconds()
	
	if rate <= 0 {
		return 0
	}
	
	remaining := float64(p.total-current) / rate
	return time.Duration(remaining) * time.Second
} 