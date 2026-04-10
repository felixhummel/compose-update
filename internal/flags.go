package internal

import (
	"fmt"
	"os"
	"time"

	flag "github.com/spf13/pflag"
)

type CCUFlags struct {
	Help      bool          // Show help message
	Directory string        // Root directory to search for Docker Compose files
	Major     bool          // Include major version updates
	Minor     bool          // Include minor version updates
	Patch     bool          // Include patch version updates
	Version   bool          // Version of ccu
	LogLevel  string        // Log level (debug, info, warning, error)
	MaxTime   time.Duration // HTTP request timeout
}

func Parse(version string) CCUFlags {
	args := CCUFlags{}

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: ccu [flags] [directory]\n\n")
		fmt.Fprintf(os.Stderr, "Arguments:\n  directory\tRoot directory to scan for Docker Compose files (default: \".\")\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}

	var patchOnly, minorOnly bool

	flag.BoolVarP(&args.Help, "help", "h", false, "Show help message")
	flag.BoolVar(&minorOnly, "minor", false, "Only update to the latest minor version")
	flag.BoolVar(&patchOnly, "patch", false, "Only update to the latest patch version")
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

	if flag.NArg() > 0 {
		args.Directory = flag.Arg(0)
	} else {
		args.Directory = "."
	}

	if patchOnly {
		args.Major = false
		args.Minor = false
		args.Patch = true
	} else if minorOnly {
		args.Major = false
		args.Minor = true
		args.Patch = true
	} else {
		args.Major = true
		args.Minor = true
		args.Patch = true
	}

	return args
}
