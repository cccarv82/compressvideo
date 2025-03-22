package cmd

import (
	"fmt"
	"time"

	"github.com/cccarv82/compressvideo/pkg/cache"
	"github.com/cccarv82/compressvideo/pkg/util"
	"github.com/spf13/cobra"
)

var (
	clearAllCache   bool
	cacheClearDays  int
)

// cacheCmd represents the cache command
var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage the analysis cache",
	Long: `Manage the video analysis cache.

The cache stores results of video analysis to speed up repeated 
compressions of the same or similar videos. This command allows
you to view statistics, clean expired entries, or clear the cache
completely.`,
	Run: func(cmd *cobra.Command, args []string) {
		manageCacheCommand()
	},
}

func init() {
	rootCmd.AddCommand(cacheCmd)

	// Add flags specific to the cache command
	cacheCmd.Flags().BoolVarP(&clearAllCache, "clear-all", "a", false, "Clear all cache entries")
	cacheCmd.Flags().IntVarP(&cacheClearDays, "max-age", "m", 30, "Clear entries older than this many days")
	cacheCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show verbose output")
}

func manageCacheCommand() {
	// Configure logger
	logger = util.NewLogger(verbose)
	logger.Title("CompressVideo - Cache Manager")

	// Initialize cache
	videoCache, err := cache.NewVideoAnalysisCache(logger)
	if err != nil {
		logger.Fatal("Failed to initialize cache: %v", err)
	}
	defer videoCache.Close()

	// Get cache statistics
	total, valid, err := videoCache.GetCacheStats()
	if err != nil {
		logger.Fatal("Failed to get cache statistics: %v", err)
	}

	// Display cache statistics
	logger.Section("Cache Statistics")
	logger.Info("Cache directory: %s", videoCache.CacheDir)
	logger.Info("Total entries: %d", total)
	logger.Info("Valid entries: %d", valid)
	logger.Info("Invalid/expired entries: %d", total-valid)

	// Clear all cache if requested
	if clearAllCache {
		logger.Section("Clearing Cache")
		logger.Info("Clearing all cache entries...")
		
		_, err = videoCache.DB.Exec("DELETE FROM video_analysis")
		if err != nil {
			logger.Fatal("Failed to clear cache: %v", err)
		}
		
		logger.Success("All cache entries cleared successfully")
		return
	}

	// Clear expired entries
	if cacheClearDays > 0 {
		logger.Section("Cleaning Expired Entries")
		logger.Info("Clearing entries older than %d days...", cacheClearDays)
		
		// Set max age for cleaning
		videoCache.SetMaxAge(cacheClearDays * 24) // Convert days to hours
		
		// Clean expired entries
		cleaned, err := videoCache.CleanExpiredEntries()
		if err != nil {
			logger.Fatal("Failed to clean expired entries: %v", err)
		}
		
		if cleaned > 0 {
			logger.Success("Cleaned %d expired cache entries", cleaned)
		} else {
			logger.Info("No expired entries found")
		}
	}

	// Get updated statistics
	if clearAllCache || cacheClearDays > 0 {
		total, valid, err = videoCache.GetCacheStats()
		if err != nil {
			logger.Warning("Failed to get updated cache statistics: %v", err)
		} else {
			logger.Section("Updated Cache Statistics")
			logger.Info("Total entries: %d", total)
			logger.Info("Valid entries: %d", valid)
		}
	}

	// Show cache usage tips
	logger.Section("Cache Usage Tips")
	logger.Info("• Use '--use-cache' or '-c' flag with compressvideo to enable caching")
	logger.Info("• Cache speeds up analysis of previously processed videos")
	logger.Info("• Regular cleaning keeps the cache size manageable")
	logger.Info("• Cache entries expire automatically after 30 days by default")
	logger.Info("• Set expiration period with '--cache-max-age' or '-A' flag")
} 