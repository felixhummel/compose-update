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
				Help:      false,
				Directory: ".",
				Major:     true,
				Minor:     true,
				Patch:     true,
				LogLevel:  "warning",
				MaxTime:   5 * time.Second,
			},
		},
		{
			name: "patch flag",
			args: []string{"--patch"},
			expected: Flags{
				Directory: ".",
				Major:     false,
				Minor:     false,
				Patch:     true,
				LogLevel:  "warning",
				MaxTime:   5 * time.Second,
			},
		},
		{
			name: "minor flag",
			args: []string{"--minor"},
			expected: Flags{
				Directory: ".",
				Major:     false,
				Minor:     true,
				Patch:     true,
				LogLevel:  "warning",
				MaxTime:   5 * time.Second,
			},
		},
		{
			name: "directory as positional arg",
			args: []string{"/path/to/dir"},
			expected: Flags{
				Directory: "/path/to/dir",
				Major:     true,
				Minor:     true,
				Patch:     true,
				LogLevel:  "warning",
				MaxTime:   5 * time.Second,
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

			if result != tt.expected {
				t.Errorf("Parse() = %+v, expected %+v", result, tt.expected)
			}
		})
	}
}
