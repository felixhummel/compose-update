package modes

import (
	"fmt"
	"log/slog"

	"github.com/felixhummel/compose-update/internal"
)

func Default(updateInfos []internal.UpdateInfo, dryRun bool) {
	for _, i := range updateInfos {
		if !i.HasNewVersion() {
			continue
		}
		if dryRun || i.FilePath == "" {
			if i.FilePath != "" {
				fmt.Printf("%s: %s -> %s\n", i.FilePath, i.ImageName+":"+i.CurrentTag, i.ImageName+":"+i.LatestTag)
			} else {
				fmt.Printf("%s:%s\n", i.ImageName, i.LatestTag)
			}
			continue
		}
		if err := i.Update(); err != nil {
			slog.Error("error updating file", "error", err)
			continue
		}
		slog.Info("updated image", "file", i.FilePath, "image", i.ImageName, "version", i.LatestTag)
	}
}
