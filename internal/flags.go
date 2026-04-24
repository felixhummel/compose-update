package internal

import (
	"fmt"
	"os"
	"time"

	flag "github.com/spf13/pflag"
)

type Flags struct {
	Help      bool          // Show help message
	DryRun    bool          // Only check for updates, do not write
	Directory string        // Root directory to search for Docker Compose files
	Image     string        // Single image to check (e.g. nginx:1.25.0)
	Tags      string        // Print all tags for an image (e.g. postgres:14.5)
	Major     bool          // Include major version updates
	Minor     bool          // Include minor version updates
	Patch     bool          // Include patch version updates
	Version   bool          // Version of compose-update
	LogLevel  string        // Log level (debug, info, warning, error)
	MaxTime   time.Duration // HTTP request timeout
}

func Parse(version string) Flags {
	args := Flags{}

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: compose-update [flags] [directory]\n\n")
		fmt.Fprintf(os.Stderr, "Arguments:\n  directory\tRoot directory to scan for Docker Compose files (default: \".\")\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}

	var patchOnly, minorOnly bool

	flag.BoolVarP(&args.Help, "help", "h", false, "Show help message")
	flag.BoolVarP(&args.DryRun, "dry-run", "n", false, "Only check for updates, do not write")
	flag.StringVar(&args.Image, "image", "", "Check a single image (e.g. nginx:1.25.0)")
	flag.StringVar(&args.Tags, "tags", "", "Print all tags for an image (e.g. postgres:14.5)")
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
