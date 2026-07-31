package modes

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/felixhummel/compose-update/internal"
)

type jsonlUpdate struct {
	Path string `json:"path"`
	Old  string `json:"old"`
	New  string `json:"new"`
}

func Default(updateInfos []internal.UpdateInfo, dryRun bool, output string) {
	enc := json.NewEncoder(os.Stdout)
	for _, i := range updateInfos {
		if !i.HasNewVersion() {
			continue
		}
		oldImage := i.ImageName + ":" + i.CurrentTag
		newImage := i.ImageName + ":" + i.LatestTag

		write := !dryRun && i.FilePath != ""
		if write {
			if err := i.Update(); err != nil {
				slog.Error("error updating file", "error", err)
				continue
			}
			slog.Info("updated image", "file", i.FilePath, "image", i.ImageName, "version", i.LatestTag)
		}

		if output == internal.OutputJSONL {
			if err := enc.Encode(jsonlUpdate{Path: i.FilePath, Old: oldImage, New: newImage}); err != nil {
				slog.Error("error writing output", "error", err)
			}
			continue
		}

		if !write {
			if i.FilePath != "" {
				fmt.Printf("%s: %s %s -> %s\n", i.FilePath, i.ImageName, i.CurrentTag, i.LatestTag)
			} else {
				fmt.Printf("%s\n", newImage)
			}
		}
	}
}
