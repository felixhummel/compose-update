package internal

import (
	"os"
	"time"

	flag "github.com/spf13/pflag"
)

type CCUFlags struct {
	Help        bool          // Show help message
	Update      bool          // Update the Docker Compose files with the new image tags
	Restart     bool          // Restart the services after updating the Docker Compose files
	Interactive bool          // Interactively choose which docker images to update
	Directory   string        // Root directory to search for Docker Compose files
	Full        bool          // Update to the latest semver version
	Major       bool          // Update to the latest major version
	Minor       bool          // Update to the latest minor version
	Patch       bool          // Update to the latest patch version
	Version     bool          // Version of ccu
	LogLevel    string        // Log level (debug, info, warning, error)
	MaxTime     time.Duration // HTTP request timeout
}

func Parse(version string) CCUFlags {
	args := CCUFlags{}

	flag.BoolVarP(&args.Help, "help", "h", false, "Show help message")
	flag.BoolVarP(&args.Update, "update", "u", false, "Update the Docker Compose files with the new image tags")
	flag.BoolVarP(&args.Restart, "restart", "r", false, "Restart the services after updating the Docker Compose files")
	flag.BoolVarP(&args.Interactive, "interactive", "i", false, "Interactively choose which docker images to update")
	flag.StringVarP(&args.Directory, "directory", "d", ".", "Root directory to search for Docker Compose files")
	flag.BoolVarP(&args.Full, "full", "f", false, "Update to the latest semver version")
	flag.BoolVar(&args.Major, "major", false, "Update to the latest major version")
	flag.BoolVar(&args.Minor, "minor", false, "Update to the latest minor version")
	flag.BoolVar(&args.Patch, "patch", true, "Update to the latest patch version")
	flag.BoolVarP(&args.Version, "version", "v", false, "Show version information")
	flag.StringVarP(&args.LogLevel, "log-level", "l", "warning", "Log level (debug, info, warning, error)")
	flag.DurationVarP(&args.MaxTime, "max-time", "m", 5*time.Second, "HTTP request timeout per registry call")

	flag.Parse()

	if args.Version {
		println("Version:", version)
		os.Exit(0)
	}

	if args.Help {
		flag.Usage()
		os.Exit(0)
	}

	if args.Full {
		args.Major = true
		args.Minor = true
		args.Patch = true
	}

	return args
}
