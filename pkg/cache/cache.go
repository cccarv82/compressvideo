// Package cache provides functionality to store and retrieve video analysis results
package cache

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/cccarv82/compressvideo/pkg/analyzer"
	"github.com/cccarv82/compressvideo/pkg/ffmpeg"
	"github.com/cccarv82/compressvideo/pkg/util"
	_ "github.com/mattn/go-sqlite3" // Import SQLite driver
)

// VideoAnalysisCache manages caching of video analysis results
type VideoAnalysisCache struct {
	DB         *sql.DB
	CacheDir   string
	Logger     *util.Logger
	Enabled    bool
	MaxAgeHours int
}

// CacheEntry represents a cached video analysis entry
type CacheEntry struct {
	ID            string    // Hash of the video file
	VideoPath     string    // Original path of the video file
	SizeBytes     int64     // Size of the video file in bytes
	ModTime       time.Time // Last modification time of the video file
	DateCached    time.Time // When the entry was cached
	AnalysisData  []byte    // Serialized VideoAnalysis data
	VideoInfo     []byte    // Serialized VideoFile data
	Duration      float64   // Duration of the video in seconds
	Resolution    string    // Resolution of the video (e.g. "1920x1080")
	Codec         string    // Video codec
	Valid         bool      // Whether the entry is valid
}

// NewVideoAnalysisCache creates a new cache instance
func NewVideoAnalysisCache(logger *util.Logger) (*VideoAnalysisCache, error) {
	// Create cache directory if it doesn't exist
	cacheDir := getCacheDir()
	err := os.MkdirAll(cacheDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Initialize cache with default settings
	cache := &VideoAnalysisCache{
		CacheDir:     cacheDir,
		Logger:       logger,
		Enabled:      true,
		MaxAgeHours:  720, // Default: 30 days
	}

	// Open/create the database
	err = cache.initDB()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cache database: %w", err)
	}

	return cache, nil
}

// getCacheDir returns the directory for storing cache data
func getCacheDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to temporary directory if home directory can't be determined
		return filepath.Join(os.TempDir(), ".compressvideo", "cache")
	}
	return filepath.Join(homeDir, ".compressvideo", "cache")
}

// initDB initializes the SQLite database for the cache
func (vc *VideoAnalysisCache) initDB() error {
	dbPath := filepath.Join(vc.CacheDir, "analysis_cache.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}
	vc.DB = db

	// Create tables if they don't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS video_analysis (
			id TEXT PRIMARY KEY,
			video_path TEXT,
			size_bytes INTEGER,
			mod_time TIMESTAMP,
			date_cached TIMESTAMP,
			analysis_data BLOB,
			video_info BLOB,
			duration REAL,
			resolution TEXT,
			codec TEXT,
			valid BOOLEAN
		);
		
		CREATE INDEX IF NOT EXISTS idx_video_path ON video_analysis(video_path);
		CREATE INDEX IF NOT EXISTS idx_date_cached ON video_analysis(date_cached);
	`)
	if err != nil {
		db.Close()
		return err
	}

	return nil
}

// Close closes the database connection
func (vc *VideoAnalysisCache) Close() error {
	if vc.DB != nil {
		return vc.DB.Close()
	}
	return nil
}

// GetVideoFingerprint generates a unique identifier for a video file based on its path, size, and modification time
func (vc *VideoAnalysisCache) GetVideoFingerprint(videoPath string) (string, error) {
	fileInfo, err := os.Stat(videoPath)
	if err != nil {
		return "", err
	}

	// Create a hash of the file path, size, and modification time
	h := md5.New()
	io.WriteString(h, videoPath)
	io.WriteString(h, fmt.Sprintf("%d", fileInfo.Size()))
	io.WriteString(h, fileInfo.ModTime().String())
	
	return hex.EncodeToString(h.Sum(nil)), nil
}

// Get retrieves a cached analysis for a video file
func (vc *VideoAnalysisCache) Get(videoPath string) (*analyzer.VideoAnalysis, *ffmpeg.VideoFile, bool, error) {
	if !vc.Enabled {
		return nil, nil, false, nil
	}

	fingerprint, err := vc.GetVideoFingerprint(videoPath)
	if err != nil {
		return nil, nil, false, err
	}

	row := vc.DB.QueryRow(`
		SELECT analysis_data, video_info, valid, date_cached 
		FROM video_analysis 
		WHERE id = ? AND valid = TRUE
	`, fingerprint)

	var analysisData, videoInfoData []byte
	var valid bool
	var dateCached time.Time

	err = row.Scan(&analysisData, &videoInfoData, &valid, &dateCached)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, false, nil
		}
		return nil, nil, false, err
	}

	// Check if cache entry is too old
	if time.Since(dateCached).Hours() > float64(vc.MaxAgeHours) {
		// Cache entry is too old, invalidate it
		vc.Logger.Debug("Cache entry for %s is too old (cached: %s), invalidating", videoPath, dateCached)
		_, err = vc.DB.Exec("UPDATE video_analysis SET valid = FALSE WHERE id = ?", fingerprint)
		if err != nil {
			vc.Logger.Warning("Failed to invalidate old cache entry: %v", err)
		}
		return nil, nil, false, nil
	}

	// Deserialize the analysis data
	var analysis analyzer.VideoAnalysis
	err = json.Unmarshal(analysisData, &analysis)
	if err != nil {
		return nil, nil, false, err
	}

	// Deserialize the video info data
	var videoInfo ffmpeg.VideoFile
	err = json.Unmarshal(videoInfoData, &videoInfo)
	if err != nil {
		return nil, nil, false, err
	}

	vc.Logger.Debug("Found cached analysis for %s (cached: %s)", videoPath, dateCached)
	return &analysis, &videoInfo, true, nil
}

// Put stores a video analysis in the cache
func (vc *VideoAnalysisCache) Put(videoPath string, analysis *analyzer.VideoAnalysis, videoInfo *ffmpeg.VideoFile) error {
	if !vc.Enabled || analysis == nil || videoInfo == nil {
		return nil
	}

	fingerprint, err := vc.GetVideoFingerprint(videoPath)
	if err != nil {
		return err
	}

	// Get file information
	fileInfo, err := os.Stat(videoPath)
	if err != nil {
		return err
	}

	// Serialize the analysis data
	analysisData, err := json.Marshal(analysis)
	if err != nil {
		return err
	}

	// Serialize the video info data
	videoInfoData, err := json.Marshal(videoInfo)
	if err != nil {
		return err
	}

	// Resolution string
	resolution := fmt.Sprintf("%dx%d", videoInfo.VideoInfo.Width, videoInfo.VideoInfo.Height)

	// Store in database
	_, err = vc.DB.Exec(`
		INSERT OR REPLACE INTO video_analysis 
		(id, video_path, size_bytes, mod_time, date_cached, analysis_data, video_info, duration, resolution, codec, valid) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, 
		fingerprint, 
		videoPath, 
		fileInfo.Size(), 
		fileInfo.ModTime(),
		time.Now(),
		analysisData,
		videoInfoData,
		videoInfo.Duration,
		resolution,
		videoInfo.VideoInfo.Codec,
		true,
	)

	if err != nil {
		return err
	}

	vc.Logger.Debug("Cached analysis for %s", videoPath)
	return nil
}

// InvalidateByPath removes a cached analysis for a specific video path
func (vc *VideoAnalysisCache) InvalidateByPath(videoPath string) error {
	if !vc.Enabled {
		return nil
	}

	fingerprint, err := vc.GetVideoFingerprint(videoPath)
	if err != nil {
		return err
	}

	_, err = vc.DB.Exec("DELETE FROM video_analysis WHERE id = ?", fingerprint)
	if err != nil {
		return err
	}

	vc.Logger.Debug("Invalidated cache for %s", videoPath)
	return nil
}

// CleanExpiredEntries removes expired entries from the cache
func (vc *VideoAnalysisCache) CleanExpiredEntries() (int64, error) {
	if !vc.Enabled {
		return 0, nil
	}

	// Calculate cutoff time
	cutoffTime := time.Now().Add(-time.Duration(vc.MaxAgeHours) * time.Hour)

	result, err := vc.DB.Exec("DELETE FROM video_analysis WHERE date_cached < ?", cutoffTime)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	if rowsAffected > 0 {
		vc.Logger.Info("Cleaned %d expired cache entries", rowsAffected)
	}

	return rowsAffected, nil
}

// GetCacheStats returns statistics about the cache
func (vc *VideoAnalysisCache) GetCacheStats() (int64, int64, error) {
	if !vc.Enabled {
		return 0, 0, nil
	}

	var totalEntries, validEntries int64

	err := vc.DB.QueryRow("SELECT COUNT(*) FROM video_analysis").Scan(&totalEntries)
	if err != nil {
		return 0, 0, err
	}

	err = vc.DB.QueryRow("SELECT COUNT(*) FROM video_analysis WHERE valid = TRUE").Scan(&validEntries)
	if err != nil {
		return 0, 0, err
	}

	return totalEntries, validEntries, nil
}

// FindSimilarVideos finds videos in the cache that have similar characteristics
func (vc *VideoAnalysisCache) FindSimilarVideos(resolution string, duration float64, codec string, tolerance float64) ([]*CacheEntry, error) {
	if !vc.Enabled {
		return nil, nil
	}

	// Calculate duration range with tolerance (e.g., 10%)
	minDuration := duration * (1.0 - tolerance)
	maxDuration := duration * (1.0 + tolerance)

	rows, err := vc.DB.Query(`
		SELECT id, video_path, size_bytes, mod_time, date_cached, analysis_data, video_info, duration, resolution, codec, valid
		FROM video_analysis
		WHERE resolution = ? 
		AND duration BETWEEN ? AND ?
		AND codec = ?
		AND valid = TRUE
		ORDER BY date_cached DESC
		LIMIT 5
	`, resolution, minDuration, maxDuration, codec)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*CacheEntry
	for rows.Next() {
		var entry CacheEntry
		err := rows.Scan(
			&entry.ID, 
			&entry.VideoPath, 
			&entry.SizeBytes, 
			&entry.ModTime, 
			&entry.DateCached, 
			&entry.AnalysisData, 
			&entry.VideoInfo,
			&entry.Duration,
			&entry.Resolution,
			&entry.Codec,
			&entry.Valid,
		)
		if err != nil {
			return nil, err
		}
		entries = append(entries, &entry)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	vc.Logger.Debug("Found %d similar videos for resolution %s, duration %.2fs, codec %s", 
		len(entries), resolution, duration, codec)
	return entries, nil
}

// SetEnabled enables or disables the cache
func (vc *VideoAnalysisCache) SetEnabled(enabled bool) {
	vc.Enabled = enabled
	if enabled {
		vc.Logger.Info("Video analysis cache enabled")
	} else {
		vc.Logger.Info("Video analysis cache disabled")
	}
}

// SetMaxAge sets the maximum age for cache entries in hours
func (vc *VideoAnalysisCache) SetMaxAge(hours int) {
	vc.MaxAgeHours = hours
	vc.Logger.Debug("Set cache maximum age to %d hours", hours)
} 