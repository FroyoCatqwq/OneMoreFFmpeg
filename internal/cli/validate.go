package cli

import (
	"context"
	"fmt"

	urfave "github.com/urfave/cli/v3"

	appdetect "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/detect"
	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
)

func validateCommand() *urfave.Command {
	flags := append(detectionFlags(), &urfave.StringFlag{
		Name:  "installation-id",
		Usage: "validate a previously detected installation id",
	})
	return &urfave.Command{
		Name:  "validate",
		Usage: "validate one FFmpeg installation (read-only)",
		Flags: flags,
		Action: func(ctx context.Context, cmd *urfave.Command) error {
			opts, err := parseDetectOptions(cmd)
			if err != nil {
				return err
			}
			dir := cmd.String("dir")
			id := cmd.String("installation-id")
			if (dir == "") == (id == "") {
				return apperrors.New(apperrors.E7001, "use exactly one of --dir or --installation-id")
			}

			detector := appdetect.New()
			var reportData any
			var status string
			if dir != "" {
				report, err := detector.ValidateDir(ctx, dir, opts)
				if err != nil {
					return err
				}
				reportData = report
				status = report.Status
				id = report.InstallationID
			} else {
				report, err := detector.ValidateID(ctx, id, opts)
				if err != nil {
					return err
				}
				reportData = report
				status = report.Status
			}
			human := fmt.Sprintf("validate: %s status=%s\n", id, status)
			return writeResult(cmd, "validate", reportData, human)
		},
	}
}
