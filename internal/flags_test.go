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
		expected CCUFlags
	}{
		{
			name: "default values",
			args: []string{},
			expected: CCUFlags{
				Help:        false,
				Update:      false,
				Restart:     false,
				Interactive: false,
				Directory:   ".",
				Full:        false,
				Major:       false,
				Minor:       false,
				Patch:       true,
				LogLevel:    "warning",
				MaxTime:     5 * time.Second,
			},
		},
		{
			name: "update flag",
			args: []string{"-u"},
			expected: CCUFlags{
				Update:    true,
				Directory: ".",
				Patch:     true,
				LogLevel:  "warning",
				MaxTime:   5 * time.Second,
			},
		},
		{
			name: "full flag",
			args: []string{"-f"},
			expected: CCUFlags{
				Full:      true,
				Major:     true,
				Minor:     true,
				Directory: ".",
				Patch:     true,
				LogLevel:  "warning",
				MaxTime:   5 * time.Second,
			},
		},
		{
			name: "directory flag",
			args: []string{"-d", "/path/to/dir"},
			expected: CCUFlags{
				Directory: "/path/to/dir",
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
