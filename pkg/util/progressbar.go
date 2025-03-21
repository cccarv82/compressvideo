package util

import (
	"fmt"
	"os"
	"time"

	"github.com/schollz/progressbar/v3"
)

// ProgressTracker handles displaying progress for long-running operations
type ProgressTracker struct {
	bar            *progressbar.ProgressBar
	total          int64
	description    string
	startTime      time.Time
	logger         *Logger
	processingRate float64 // processing rate in units per second
	showSpeed      bool
	lastUpdate     time.Time
	lastProgress   int64
	statusCallback func(progress int64, timeRemaining time.Duration, rate float64)
}

// NewProgressTrackerOptions configures a new progress tracker
type ProgressTrackerOptions struct {
	Total          int64
	Description    string
	Logger         *Logger
	ShowBytes      bool
	ShowPercentage bool
	ShowSpeed      bool
	StatusCallback func(progress int64, timeRemaining time.Duration, rate float64)
}

// NewProgressTracker creates a new progress tracker
func NewProgressTracker(total int64, description string, logger *Logger) *ProgressTracker {
	return NewProgressTrackerWithOptions(ProgressTrackerOptions{
		Total:          total,
		Description:    description, 
		Logger:         logger,
		ShowBytes:      false,
		ShowPercentage: true,
		ShowSpeed:      false,
	})
}

// NewProgressTrackerWithOptions creates a new progress tracker with advanced options
func NewProgressTrackerWithOptions(options ProgressTrackerOptions) *ProgressTracker {
	// Set up progress bar options
	barOptions := []progressbar.Option{
		progressbar.OptionSetDescription(options.Description),
		progressbar.OptionSetWriter(os.Stdout),
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
		progressbar.OptionClearOnFinish(),
		progressbar.OptionUseANSICodes(true),
		progressbar.OptionOnCompletion(func() {
			fmt.Println()
		}),
	}

	// Add optional settings
	if options.ShowBytes {
		barOptions = append(barOptions, progressbar.OptionShowBytes(true))
	}
	
	// Create the progress bar
	bar := progressbar.NewOptions64(
		options.Total,
		barOptions...,
	)
	
	return &ProgressTracker{
		bar:            bar,
		total:          options.Total,
		description:    options.Description,
		startTime:      time.Now(),
		logger:         options.Logger,
		showSpeed:      options.ShowSpeed,
		lastUpdate:     time.Now(),
		lastProgress:   0,
		statusCallback: options.StatusCallback,
	}
}

// Update updates the progress bar
func (p *ProgressTracker) Update(current int64) error {
	// Calculate processing rate
	now := time.Now()
	timeDiff := now.Sub(p.lastUpdate).Seconds()
	
	if timeDiff >= 1.0 && p.lastProgress > 0 {
		progressDiff := current - p.lastProgress
		rate := float64(progressDiff) / timeDiff
		p.processingRate = rate
		
		// Update the description with the rate if enabled
		if p.showSpeed {
			remaining := p.EstimateTimeRemaining(current)
			remainingStr := formatDuration(remaining)
			
			// Format rate based on whether we're showing bytes or not
			rateStr := fmt.Sprintf("%.1f/s", rate)
			
			description := fmt.Sprintf("%s [%s remain, %s]", 
				p.description, remainingStr, rateStr)
				
			p.bar.Describe(description)
		}
		
		p.lastUpdate = now
		p.lastProgress = current
		
		// Call status callback if set
		if p.statusCallback != nil {
			p.statusCallback(current, p.EstimateTimeRemaining(current), p.processingRate)
		}
	} else if p.lastProgress == 0 {
		p.lastProgress = current
	}
	
	return p.bar.Set64(current)
}

// Increment increments the progress bar by the given amount
func (p *ProgressTracker) Increment(amount int64) error {
	current := p.lastProgress + amount
	return p.Update(current)
}

// Finish completes the progress bar and displays final stats
func (p *ProgressTracker) Finish() {
	// Ensure bar shows 100%
	p.bar.Finish()
	duration := time.Since(p.startTime).Round(time.Second)
	
	// Show final stats
	if p.total > 0 {
		rate := float64(p.total) / duration.Seconds()
		if p.showSpeed {
			p.logger.Info("Operation completed in %s (%.1f units/s)", 
				formatDuration(duration), rate)
		} else {
			p.logger.Info("Operation completed in %s", formatDuration(duration))
		}
	} else {
		p.logger.Info("Operation completed in %s", formatDuration(duration))
	}
}

// GetElapsedTime returns the elapsed time since the start
func (p *ProgressTracker) GetElapsedTime() time.Duration {
	return time.Since(p.startTime)
}

// GetProcessingRate returns the current processing rate in units per second
func (p *ProgressTracker) GetProcessingRate() float64 {
	return p.processingRate
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

// SetStatusCallback sets a callback function to receive progress updates
func (p *ProgressTracker) SetStatusCallback(callback func(progress int64, timeRemaining time.Duration, rate float64)) {
	p.statusCallback = callback
}

// formatDuration formats a duration in a human-readable format
func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	
	h := d / time.Hour
	d -= h * time.Hour
	
	m := d / time.Minute
	d -= m * time.Minute
	
	s := d / time.Second
	
	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	} else if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
} 