package modes

import (
	"log/slog"

	"github.com/padi2312/compose-check-updates/internal"
)

func Default(updateInfos []internal.UpdateInfo) {
	for _, i := range updateInfos {
		if !i.HasNewVersion() {
			continue
		}
		if err := i.Update(); err != nil {
			slog.Error("error updating file", "error", err)
			continue
		}
		slog.Info("updated image", "file", i.FilePath, "image", i.ImageName, "version", i.LatestTag)
	}
}
