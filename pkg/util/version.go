package util

// Version information
const (
	// AppName is the name of the application
	AppName = "CompressVideo"
	// Version is the current version of the application
	Version = "1.5.2"
	// BuildDate is the date the application was built
	BuildDate = "development"
)

// GetVersionInfo returns a formatted string with version information
func GetVersionInfo() string {
	return AppName + " v" + Version + " (" + BuildDate + ")"
} 