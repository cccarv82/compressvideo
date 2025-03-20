# CompressVideo

A smart video compression CLI tool written in Go that reduces video file sizes by up to 70% while maintaining the highest possible visual quality.

## Features

- Smart content analysis to determine optimal compression settings
- Support for multiple compression presets
- Real-time compression progress display
- Detailed before/after compression reports
- Parallel processing for faster compression

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

## Requirements

- Go 1.18 or higher
- FFmpeg installed on your system

## License

MIT 