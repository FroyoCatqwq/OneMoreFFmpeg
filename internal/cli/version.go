package cli

import (
	"context"
	"fmt"

	urfave "github.com/urfave/cli/v3"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/buildinfo"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
)

func versionCommand(info buildinfo.Info) *urfave.Command {
	return &urfave.Command{
		Name:  "version",
		Usage: "show build and runtime information",
		Action: func(ctx context.Context, cmd *urfave.Command) error {
			if err := rejectExtraArgs(cmd); err != nil {
				return err
			}
			data := model.VersionData{
				Schema:    model.VersionSchema,
				Name:      "OneMoreFFmpeg",
				Version:   info.Version,
				GoVersion: info.GoVersion,
				GitCommit: info.Commit,
				BuildTime: info.BuildTime,
				OS:        info.OS,
				Arch:      info.Arch,
			}
			human := fmt.Sprintf(
				"OneMoreFFmpeg %s (%s) %s/%s Go %s\n",
				info.Version, info.Commit, info.OS, info.Arch, info.GoVersion,
			)
			return writeResult(cmd, "version", data, human)
		},
	}
}
