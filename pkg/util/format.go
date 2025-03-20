package util

import (
	"fmt"
)

// FormatSize formata um tamanho em bytes para uma representação legível
// Exemplo: 1024 -> "1.00 KB", 1048576 -> "1.00 MB"
func FormatSize(sizeBytes int64) string {
	const unit = 1024
	if sizeBytes < unit {
		return fmt.Sprintf("%d B", sizeBytes)
	}
	
	div, exp := int64(unit), 0
	for n := sizeBytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	
	switch exp {
	case 0:
		return fmt.Sprintf("%.2f KB", float64(sizeBytes)/float64(div*unit/unit))
	case 1:
		return fmt.Sprintf("%.2f MB", float64(sizeBytes)/float64(div*unit/unit))
	case 2:
		return fmt.Sprintf("%.2f GB", float64(sizeBytes)/float64(div*unit/unit))
	}
	
	return fmt.Sprintf("%.2f TB", float64(sizeBytes)/float64(div*unit/unit))
}

// FormatBitrate formata uma taxa de bits para uma representação legível
// Exemplo: 1000000 -> "1.00 Mbps", 500000 -> "500.00 kbps"
func FormatBitrate(bitrate int64) string {
	if bitrate >= 1000000 {
		return fmt.Sprintf("%.2f Mbps", float64(bitrate)/1000000)
	} else {
		return fmt.Sprintf("%.2f kbps", float64(bitrate)/1000)
	}
}

// FormatDuration formata uma duração em segundos para uma representação legível
// Exemplo: 65 -> "1m 5s", 3600 -> "1h 0m 0s"
func FormatDuration(seconds int) string {
	h := seconds / 3600
	seconds -= h * 3600
	
	m := seconds / 60
	seconds -= m * 60
	
	s := seconds
	
	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	} else if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	
	return fmt.Sprintf("%ds", s)
} 