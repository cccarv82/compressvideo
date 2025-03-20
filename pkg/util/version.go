package util

// Version information
const (
	// AppName is the name of the application
	AppName = "CompressVideo"
	// Version is the current version of the application
	Version = "0.1.1"
	// BuildDate is the date the application was built
	BuildDate = "development"
)

// GetVersionInfo returns a formatted string with version information
func GetVersionInfo() string {
	return AppName + " v" + Version + " (" + BuildDate + ")"
} 