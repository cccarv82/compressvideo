# CompressVideo

A smart video compression CLI tool written in Go that reduces video file sizes by up to 70% while maintaining the highest possible visual quality.

## Features

- Smart content analysis to determine optimal compression settings
- Automatic detection of video type (screencast, animation, gaming, etc.)
- Motion complexity analysis for determining optimal encoding parameters
- Intelligent codec selection based on content type
- Parallel processing for faster compression
- Advanced compression algorithms with quality-size balancing
- Automatic segmentation for efficient multi-core processing
- Real-time compression progress display
- Detailed before/after compression reports
- Support for H.264 and H.265 codecs

## Implementation Status

- ✅ Sprint 1: CLI interface and project structure
- ✅ Sprint 2: Video analysis and content detection
- ✅ Sprint 3: Compression engine
- ⬜ Sprint 4: User interface improvements
- ⬜ Sprint 5: Testing and finalization

## Installation

```bash
go install github.com/cccarv82/compressvideo@latest
```

## Usage

Basic usage:

```bash
compressvideo -i input.mp4
```

With options:

```bash
compressvideo -i input.mp4 -o output.mp4 -q 4 -p thorough
```

### Available Options

- `-i, --input`: Path to the video file to compress (required)
- `-o, --output`: Path to save the compressed file (optional, uses same name with suffix if omitted)
- `-q, --quality`: Quality level from 1-5 (1=maximum compression, 5=maximum quality, default=3)
- `-p, --preset`: Compression preset ("fast", "balanced", "thorough", default="balanced")
- `-f, --force`: Overwrite output file if it exists
- `-v, --verbose`: Show detailed information during the process
- `-h, --help`: Show detailed help

## Content Analysis

CompressVideo analyzes your video to determine:

- Content type (animation, screencast, gaming, live action, sports, etc.)
- Motion complexity (low, medium, high, very high)
- Scene changes frequency
- Frame complexity
- Spatial detail level
- Optimal codec selection (H.264, H.265)
- Ideal bitrate for target quality

Based on this analysis, it automatically selects the optimal compression settings to maintain visual quality while maximizing file size reduction.

## Compression Engine

The compression engine provides:

- Parallel processing using goroutines for faster compression
- Video segmentation for multi-core utilization
- Adaptive quality settings based on content type
- Dynamic bitrate adjustment based on complexity
- Intelligent codec selection (H.264 for compatibility, H.265 for efficiency)
- Quality-optimized audio compression
- Real-time progress tracking

The presets offer different tradeoffs:

- **Quality levels (1-5)**:
  - 1: Maximum compression, lower quality
  - 3: Balanced compression and quality
  - 5: Maximum quality, less compression

- **Speed presets**:
  - fast: Quicker compression, slightly larger file size
  - balanced: Good balance between speed and compression
  - thorough: Slower but more efficient compression

## Test Videos

For development and testing purposes, you can use the included script to download sample videos:

```bash
# Download all test videos (requires Python 3)
./scripts/download_test_videos.py

# Or use the make command
make download-test-videos
```

This will download several test videos to the `data/` directory, including:
- Car detection video (768x432, similar to animation)
- People detection footage (768x432, similar to gaming)
- Face demographics walking video (768x432, documentary-like)
- Classroom/screencast video (1920x1080, high-resolution)

All test videos are sourced from the [Intel IoT DevKit sample videos](https://github.com/intel-iot-devkit/sample-videos) repository, which provides free sample videos for testing.

### Other Free Testing Resources

These free resources provide excellent test videos for compression applications:

1. **Xiph.org Video Test Media**: [https://media.xiph.org/video/derf/](https://media.xiph.org/video/derf/)
   - High-quality, uncompressed video sequences specifically designed for testing

2. **Pexels Free Stock Videos**: [https://www.pexels.com/videos/](https://www.pexels.com/videos/)
   - Royalty-free videos for testing different content types

3. **Coverr**: [https://coverr.co/](https://coverr.co/)
   - Free motion videos, useful for testing compression of cinematic content

4. **Sample-Videos.com**: [https://sample-videos.com/](https://sample-videos.com/)
   - Various sample videos in different formats and resolutions

**Note**: The `data/` directory is excluded from version control in `.gitignore`.

## Requirements

- Go 1.18 or higher
- FFmpeg installed on your system

## License

MIT 