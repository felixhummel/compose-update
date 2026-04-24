package main

import (
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/felixhummel/compose-update/internal"
	customlogger "github.com/felixhummel/compose-update/internal/logger"
	"github.com/felixhummel/compose-update/internal/modes"
)

var version = "dev"

func parseLogLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelWarn
	}
}

func main() {
	flags := internal.Parse(version)

	level := parseLogLevel(flags.LogLevel)
	log := slog.New(customlogger.NewCustomHandler(level, os.Stdout))
	slog.SetDefault(log)

	var updateInfos []internal.UpdateInfo

	if flags.Image != "" {
		infos, err := internal.CheckImage(flags.Image, internal.NewRegistryWithTimeout(flags.MaxTime))
		if err != nil {
			slog.Error("Error checking image", "error", err)
			os.Exit(1)
		}
		updateInfos = infos
	} else {
		composeFilePaths, err := internal.GetComposeFilePaths(flags.Directory)
		if err != nil {
			slog.Error("Error getting compose file paths", "error", err)
			os.Exit(1)
		}

		var mu sync.Mutex
		var wg sync.WaitGroup

		for _, path := range composeFilePaths {
			wg.Add(1)
			go func(path string) {
				defer wg.Done()
				updateChecker := internal.NewUpdateChecker(path, internal.NewRegistryWithTimeout(flags.MaxTime))
				info, err := updateChecker.Check()
				if err != nil {
					slog.Error("Error checking for updates", "error", err)
					return
				}
				mu.Lock()
				updateInfos = append(updateInfos, info...)
				mu.Unlock()
			}(path)
		}

		wg.Wait()
	}

	modes.Default(updateInfos, flags.DryRun)
}
