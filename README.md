# CompressVideo

A smart video compression CLI tool written in Go that reduces video file sizes by up to 70% while maintaining the highest possible visual quality.

## Features

- Smart content analysis to determine optimal compression settings
- Automatic detection of video type (screencast, animation, gaming, etc.)
- Motion complexity analysis for determining optimal encoding parameters
- Intelligent codec selection based on content type
- Real-time compression progress display
- Detailed before/after compression reports
- Parallel processing for faster compression

## Implementation Status

- ✅ Sprint 1: CLI interface and project structure
- ✅ Sprint 2: Video analysis and content detection
- ⬜ Sprint 3: Compression engine
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
- Optimal codec selection (H.264, H.265, VP9)
- Ideal bitrate for target quality

Based on this analysis, it automatically selects the optimal compression settings to maintain visual quality while maximizing file size reduction.

## Requirements

- Go 1.18 or higher
- FFmpeg installed on your system

## License

MIT 