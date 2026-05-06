package internal

import (
	"os"
	"testing"
	"time"

	flag "github.com/spf13/pflag"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected Flags
	}{
		{
			name: "default values",
			args: []string{},
			expected: Flags{
				Help:        false,
				Directory:   ".",
				UpdateLevel: MajorLevel,
				LogLevel:    "warning",
				MaxTime:     5 * time.Second,
			},
		},
		{
			name: "patch flag",
			args: []string{"--patch"},
			expected: Flags{
				Directory:   ".",
				UpdateLevel: PatchLevel,
				LogLevel:    "warning",
				MaxTime:     5 * time.Second,
			},
		},
		{
			name: "minor flag",
			args: []string{"--minor"},
			expected: Flags{
				Directory:   ".",
				UpdateLevel: MinorLevel,
				LogLevel:    "warning",
				MaxTime:     5 * time.Second,
			},
		},
		{
			name: "directory as positional arg",
			args: []string{"/path/to/dir"},
			expected: Flags{
				Directory:   "/path/to/dir",
				UpdateLevel: MajorLevel,
				LogLevel:    "warning",
				MaxTime:     5 * time.Second,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origArgs := os.Args
			defer func() { os.Args = origArgs }()

			os.Args = append([]string{"cmd"}, tt.args...)

			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

			result := Parse("test")

			// Compare individual fields since Exclude is not comparable
			if result.Directory != tt.expected.Directory {
				t.Errorf("Directory = %v, want %v", result.Directory, tt.expected.Directory)
			}
			if result.UpdateLevel != tt.expected.UpdateLevel {
				t.Errorf("UpdateLevel = %v, want %v", result.UpdateLevel, tt.expected.UpdateLevel)
			}
			if result.LogLevel != tt.expected.LogLevel {
				t.Errorf("LogLevel = %v, want %v", result.LogLevel, tt.expected.LogLevel)
			}
			if result.MaxTime != tt.expected.MaxTime {
				t.Errorf("MaxTime = %v, want %v", result.MaxTime, tt.expected.MaxTime)
			}
			if len(result.Exclude) != len(tt.expected.Exclude) {
				t.Errorf("Exclude = %v, want %v", result.Exclude, tt.expected.Exclude)
			}
		})
	}
}
