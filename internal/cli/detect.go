package cli

import (
	"context"

	urfave "github.com/urfave/cli/v3"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
)

func detectCommand() *urfave.Command {
	return &urfave.Command{
		Name:  "detect",
		Usage: "Phase 1 read-only placeholder; real discovery arrives in Phase 2",
		Action: func(ctx context.Context, cmd *urfave.Command) error {
			data := model.DetectData{
				Schema:        model.DetectSchema,
				Placeholder:   true,
				Installations: []model.Installation{},
				Errors:        []model.DetectError{},
			}
			return writeResult(cmd, "detect", data, "detect: Phase 1 placeholder; no detection performed\n")
		},
	}
}
