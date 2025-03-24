// Package transcriber provides functionality to transcribe video audio to text.
//
// This package is part of the CompressVideo CLI tool and handles the
// transcription of video audio to text using the Whisper speech-to-text model.
//
// The package offers two main components:
//
// 1. A transcription engine that extracts audio from videos and converts it to text
// 2. An installer that can automatically set up the Whisper dependencies
//
// It supports both Python-based Whisper and the C++ implementation for better performance.
//
// Added in version 1.5.7 of CompressVideo.
package transcriber 