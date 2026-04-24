package internal

import (
	"fmt"
	"os"
	"time"

	flag "github.com/spf13/pflag"
)

// UpdateLevel specifies which semver component updates to include.
type UpdateLevel int

const (
	PatchLevel UpdateLevel = iota
	MinorLevel
	MajorLevel
)

func (l UpdateLevel) IncludePatch() bool { return true }

func (l UpdateLevel) IncludeMinor() bool { return l >= MinorLevel }

func (l UpdateLevel) IncludeMajor() bool { return l >= MajorLevel }

type Flags struct {
	Help        bool          // Show help message
	DryRun      bool          // Only check for updates, do not write
	Directory   string        // Root directory to search for Docker Compose files
	Image       string        // Single image to check (e.g. nginx:1.25.0)
	Tags        string        // Print all tags for an image (e.g. postgres:14.5)
	UpdateLevel UpdateLevel   // Level of updates to include (major, minor, patch)
	Version     bool          // Version of compose-update
	LogLevel    string        // Log level (debug, info, warning, error)
	MaxTime     time.Duration // HTTP request timeout
}

func Parse(version string) Flags {
	args := Flags{}

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: compose-update [flags] [directory]\n\n")
		fmt.Fprintf(os.Stderr, "Arguments:\n  directory\tRoot directory to scan for Docker Compose files (default: \".\")\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}

	var major, minor, patch bool

	flag.BoolVarP(&args.Help, "help", "h", false, "Show help message")
	flag.BoolVarP(&args.DryRun, "dry-run", "n", false, "Only check for updates, do not write")
	flag.StringVar(&args.Image, "image", "", "Check a single image (e.g. nginx:1.25.0)")
	flag.StringVar(&args.Tags, "tags", "", "Print all tags for an image (e.g. postgres:14.5)")
	flag.BoolVar(&major, "major", false, "Include major version updates")
	flag.BoolVar(&minor, "minor", false, "Only update to the latest minor version")
	flag.BoolVar(&patch, "patch", false, "Only update to the latest patch version")
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

	if patch {
		args.UpdateLevel = PatchLevel
	} else if minor {
		args.UpdateLevel = MinorLevel
	} else {
		args.UpdateLevel = MajorLevel
	}

	return args
}
