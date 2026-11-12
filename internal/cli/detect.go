package cli

import (
	"context"
	"fmt"

	urfave "github.com/urfave/cli/v3"

	appdetect "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/detect"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
)

func detectCommand() *urfave.Command {
	return &urfave.Command{
		Name:  "detect",
		Usage: "discover existing FFmpeg trio installations (read-only)",
		Flags: detectionFlags(),
		Action: func(ctx context.Context, cmd *urfave.Command) error {
			if err := rejectExtraArgs(cmd); err != nil {
				return err
			}
			opts, err := parseDetectOptions(cmd, true)
			if err != nil {
				return err
			}
			detector := appdetect.New()
			result, err := detector.Detect(ctx, opts)
			if err != nil {
				return err
			}
			data := model.DetectData{
				Schema:        model.DetectSchema,
				Installations: result.Installations,
				Errors:        result.Errors,
			}
			human := fmt.Sprintf("detect: %d installation(s), %d source error(s)\n", len(result.Installations), len(result.Errors))
			return writeResult(cmd, "detect", data, human)
		},
	}
}
