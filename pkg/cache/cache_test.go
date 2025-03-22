package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cccarv82/compressvideo/pkg/analyzer"
	"github.com/cccarv82/compressvideo/pkg/ffmpeg"
	"github.com/cccarv82/compressvideo/pkg/util"
	"github.com/stretchr/testify/assert"
)

func createTempVideoFile(t *testing.T) string {
	tmpDir, err := os.MkdirTemp("", "cache-test-")
	assert.NoError(t, err)

	// Create a dummy video file
	videoPath := filepath.Join(tmpDir, "test-video.mp4")
	f, err := os.Create(videoPath)
	assert.NoError(t, err)
	
	// Write some dummy content so the file has a size
	_, err = f.WriteString("This is a test video file content")
	assert.NoError(t, err)
	f.Close()

	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})

	return videoPath
}

func createTestCache(t *testing.T) (*VideoAnalysisCache, string) {
	// Set up a temporary directory for the cache
	tmpDir, err := os.MkdirTemp("", "compressvideo-cache-test-")
	assert.NoError(t, err)

	// Override the getCacheDir function for testing
	originalGetCacheDir := getCacheDir
	getCacheDir = func() string {
		return tmpDir
	}

	// Restore the original function after the test
	t.Cleanup(func() {
		getCacheDir = originalGetCacheDir
		os.RemoveAll(tmpDir)
	})

	// Create a silent logger for testing
	logger := util.NewLogger(false)
	// Disable logging for tests
	logger.SetLevel(util.LogLevelError)

	// Create the cache
	cache, err := NewVideoAnalysisCache(logger)
	assert.NoError(t, err)
	assert.NotNil(t, cache)

	// Clean up the cache after the test
	t.Cleanup(func() {
		cache.Close()
	})

	return cache, tmpDir
}

func createTestVideoAnalysis() (*analyzer.VideoAnalysis, *ffmpeg.VideoFile) {
	// Create a dummy VideoFile
	videoFile := &ffmpeg.VideoFile{
		Path:     "/test/path/video.mp4",
		Size:     1024 * 1024, // 1MB
		Format:   "mp4",
		Duration: 60.0, // 1 minute
		BitRate:  1500000, // 1.5 Mbps
		VideoInfo: ffmpeg.VideoStreamInfo{
			Codec:       "h264",
			Width:       1920,
			Height:      1080,
			FPS:         30.0,
			BitRate:     1000000, // 1 Mbps
			PixelFormat: "yuv420p",
			ColorSpace:  "bt709",
			IsHDR:       false,
			HasBFrames:  true,
			ProfileLevel: "High@4.1",
		},
		AudioInfo: []ffmpeg.AudioStreamInfo{
			{
				Index:      1,
				Codec:      "aac",
				Channels:   2,
				SampleRate: 48000,
				BitRate:    128000, // 128 kbps
				Language:   "eng",
			},
		},
		Metadata: map[string]string{
			"encoder": "test",
		},
	}

	// Create a dummy VideoAnalysis
	videoAnalysis := &analyzer.VideoAnalysis{
		VideoFile:        videoFile,
		ContentType:      analyzer.ContentTypeAnimation,
		MotionComplexity: analyzer.MotionComplexityLow,
		SceneChanges:     5,
		FrameComplexity:  0.3,
		CompressionPotential: 70,
		RecommendedCodec: "h264",
		OptimalBitrate:   800000, // 800 kbps
		SpatialComplexity: 0.5,
		IsHDContent:      true,
		IsUHDContent:     false,
	}

	return videoAnalysis, videoFile
}

func TestNewVideoAnalysisCache(t *testing.T) {
	// This test verifies that the cache is created correctly
	cache, _ := createTestCache(t)
	assert.NotNil(t, cache)
	assert.NotNil(t, cache.DB)
	assert.True(t, cache.Enabled)
}

func TestCachePutAndGet(t *testing.T) {
	// This test verifies that we can store and retrieve items from the cache
	cache, _ := createTestCache(t)
	videoPath := createTempVideoFile(t)
	
	// Create test data
	analysis, videoFile := createTestVideoAnalysis()
	
	// Store in cache
	err := cache.Put(videoPath, analysis, videoFile)
	assert.NoError(t, err)
	
	// Retrieve from cache
	retrievedAnalysis, retrievedVideoFile, found, err := cache.Get(videoPath)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.NotNil(t, retrievedAnalysis)
	assert.NotNil(t, retrievedVideoFile)
	
	// Check that the data is correct
	assert.Equal(t, analysis.ContentType, retrievedAnalysis.ContentType)
	assert.Equal(t, analysis.MotionComplexity, retrievedAnalysis.MotionComplexity)
	assert.Equal(t, analysis.CompressionPotential, retrievedAnalysis.CompressionPotential)
	assert.Equal(t, videoFile.Format, retrievedVideoFile.Format)
	assert.Equal(t, videoFile.Duration, retrievedVideoFile.Duration)
	assert.Equal(t, videoFile.VideoInfo.Codec, retrievedVideoFile.VideoInfo.Codec)
}

func TestCacheInvalidation(t *testing.T) {
	// This test verifies that cache entries can be invalidated
	cache, _ := createTestCache(t)
	videoPath := createTempVideoFile(t)
	
	// Create test data
	analysis, videoFile := createTestVideoAnalysis()
	
	// Store in cache
	err := cache.Put(videoPath, analysis, videoFile)
	assert.NoError(t, err)
	
	// Invalidate the entry
	err = cache.InvalidateByPath(videoPath)
	assert.NoError(t, err)
	
	// Try to retrieve it - should not be found
	_, _, found, err := cache.Get(videoPath)
	assert.NoError(t, err)
	assert.False(t, found)
}

func TestCacheExpiration(t *testing.T) {
	// This test verifies that expired cache entries are recognized
	cache, _ := createTestCache(t)
	videoPath := createTempVideoFile(t)
	
	// Set a very short expiration time for testing
	cache.MaxAgeHours = 0 // Immediate expiration
	
	// Create test data
	analysis, videoFile := createTestVideoAnalysis()
	
	// Store in cache
	err := cache.Put(videoPath, analysis, videoFile)
	assert.NoError(t, err)
	
	// Wait a moment to ensure it's expired
	time.Sleep(100 * time.Millisecond)
	
	// Try to retrieve it - should be expired and not found
	_, _, found, err := cache.Get(videoPath)
	assert.NoError(t, err)
	assert.False(t, found)
}

func TestFindSimilarVideos(t *testing.T) {
	// This test verifies that we can find similar videos in the cache
	cache, _ := createTestCache(t)
	videoPath1 := createTempVideoFile(t)
	videoPath2 := createTempVideoFile(t)
	
	// Create test data
	analysis1, videoFile1 := createTestVideoAnalysis()
	analysis2, videoFile2 := createTestVideoAnalysis()
	
	// Second video has same resolution but different duration
	videoFile2.Duration = 65.0 // 5 seconds longer
	
	// Store in cache
	err := cache.Put(videoPath1, analysis1, videoFile1)
	assert.NoError(t, err)
	err = cache.Put(videoPath2, analysis2, videoFile2)
	assert.NoError(t, err)
	
	// Find similar videos with 10% tolerance
	resolution := "1920x1080"
	duration := 60.0
	codec := "h264"
	tolerance := 0.1
	
	similar, err := cache.FindSimilarVideos(resolution, duration, codec, tolerance)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(similar)) // Should find both videos
	
	// With a smaller tolerance, should only find the first video
	tolerance = 0.01
	similar, err = cache.FindSimilarVideos(resolution, duration, codec, tolerance)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(similar))
}

func TestCleanExpiredEntries(t *testing.T) {
	// This test verifies that we can clean expired cache entries
	cache, _ := createTestCache(t)
	videoPath := createTempVideoFile(t)
	
	// Set a very short expiration time for testing
	cache.MaxAgeHours = 0 // Immediate expiration
	
	// Create test data
	analysis, videoFile := createTestVideoAnalysis()
	
	// Store in cache
	err := cache.Put(videoPath, analysis, videoFile)
	assert.NoError(t, err)
	
	// Wait a moment to ensure it's expired
	time.Sleep(100 * time.Millisecond)
	
	// Clean expired entries
	cleaned, err := cache.CleanExpiredEntries()
	assert.NoError(t, err)
	assert.Equal(t, int64(1), cleaned) // Should have cleaned one entry
	
	// Verify the entry is gone
	_, _, found, err := cache.Get(videoPath)
	assert.NoError(t, err)
	assert.False(t, found)
}

func TestCacheStats(t *testing.T) {
	// This test verifies that we can get cache statistics
	cache, _ := createTestCache(t)
	videoPath := createTempVideoFile(t)
	
	// Create test data
	analysis, videoFile := createTestVideoAnalysis()
	
	// Store in cache
	err := cache.Put(videoPath, analysis, videoFile)
	assert.NoError(t, err)
	
	// Get cache stats
	total, valid, err := cache.GetCacheStats()
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, int64(1), valid)
	
	// Invalidate the entry
	_, err = cache.DB.Exec("UPDATE video_analysis SET valid = FALSE WHERE video_path = ?", videoPath)
	assert.NoError(t, err)
	
	// Get cache stats again
	total, valid, err = cache.GetCacheStats()
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, int64(0), valid)
} 