package util

import (
	"os"
	"runtime"
)

// OSType representa um sistema operacional suportado
type OSType string

const (
	// Linux representa o sistema operacional Linux
	Linux OSType = "linux"
	// Windows representa o sistema operacional Windows
	Windows OSType = "windows"
	// MacOS representa o sistema operacional macOS
	MacOS OSType = "darwin"
	// Unknown representa um sistema operacional desconhecido
	Unknown OSType = "unknown"
)

// GetCurrentOS retorna o sistema operacional atual
func GetCurrentOS() OSType {
	switch runtime.GOOS {
	case "linux":
		return Linux
	case "windows":
		return Windows
	case "darwin":
		return MacOS
	default:
		return Unknown
	}
}

// IsUnixLike retorna true se o sistema operacional for Unix-like (Linux ou macOS)
func IsUnixLike() bool {
	os := GetCurrentOS()
	return os == Linux || os == MacOS
}

// IsWindows retorna true se o sistema operacional for Windows
func IsWindows() bool {
	return GetCurrentOS() == Windows
}

// GetPathSeparator retorna o separador de caminho para o sistema operacional atual
func GetPathSeparator() string {
	if IsWindows() {
		return "\\"
	}
	return "/"
}

// GetExecutableExtension retorna a extensão para arquivos executáveis no sistema operacional atual
func GetExecutableExtension() string {
	if IsWindows() {
		return ".exe"
	}
	return ""
}

// GetTempDir retorna o diretório temporário para o sistema operacional atual
func GetTempDir() string {
	if temp := os.Getenv("TEMP"); temp != "" && IsWindows() {
		return temp
	}
	
	if temp := os.Getenv("TMP"); temp != "" && IsWindows() {
		return temp
	}
	
	if temp := os.Getenv("TMPDIR"); temp != "" && IsUnixLike() {
		return temp
	}
	
	if IsWindows() {
		return "C:\\Windows\\Temp"
	}
	
	return "/tmp"
} 