// Package util fornece funções utilitárias para o CompressVideo
package util

// Version information
const (
	// Name é o nome do aplicativo
	Name = "CompressVideo"
	// Version é a versão do aplicativo
	Version = "1.6.6"
	// BuildDate é a data em que o aplicativo foi compilado
	BuildDate = "development"
)

// GetVersionInfo returns a formatted string with version information
func GetVersionInfo() string {
	return Name + " v" + Version + " (" + BuildDate + ")"
} 