package cli

import (
	"context"
	"fmt"

	urfave "github.com/urfave/cli/v3"

	appcaps "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/capabilities"
	appdetect "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/detect"
	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
)

func capabilitiesFlags() []urfave.Flag {
	flags := make([]urfave.Flag, 0, len(detectionFlags())+5)
	for _, flag := range detectionFlags() {
		// capabilities always validates, so advertising --no-validate would
		// only lead users into an E7001 error.
		if flagName(flag) == "no-validate" {
			continue
		}
		flags = append(flags, flag)
	}
	return append(flags,
		&urfave.StringFlag{Name: "installation-id", Usage: "capabilities target installation id"},
		&urfave.StringFlag{Name: "help-topic", Value: "full", Usage: "help topic: full|decoder=<name>|encoder=<name>|filter=<name>|muxer=<name>|demuxer=<name>"},
		&urfave.StringFlag{Name: "cache-dir", Usage: "capability cache directory"},
		&urfave.BoolFlag{Name: "refresh", Usage: "ignore existing capability cache"},
		&urfave.BoolFlag{Name: "no-cache", Usage: "do not read or write capability cache"},
		&urfave.StringFlag{Name: "only", Usage: "run and output only one list: muxers|demuxers|devices|decoders|encoders|filters|pixel_formats|channel_layouts|sample_formats"},
	)
}

func flagName(flag urfave.Flag) string {
	switch typed := flag.(type) {
	case *urfave.StringFlag:
		return typed.Name
	case *urfave.BoolFlag:
		return typed.Name
	case *urfave.IntFlag:
		return typed.Name
	}
	return ""
}

func capabilitiesCommand() *urfave.Command {
	return &urfave.Command{
		Name:  "capabilities",
		Usage: "query FFmpeg capabilities from a detected installation",
		Flags: capabilitiesFlags(),
		Action: func(ctx context.Context, cmd *urfave.Command) error {
			if err := rejectExtraArgs(cmd); err != nil {
				return err
			}
			topic := cmd.String("help-topic")
			if err := appcaps.ValidateHelpTopic(topic); err != nil {
				return err
			}
			detectOptions, err := parseDetectOptions(cmd, false)
			if err != nil {
				return err
			}
			dir := cmd.String("dir")
			id := cmd.String("installation-id")
			if err := rejectDirDiscoveryConflict(cmd, dir); err != nil {
				return err
			}
			installation, err := selectInstallationForCapabilities(ctx, detectOptions, dir, id)
			if err != nil {
				return err
			}
			commandTimeout := detectOptions.ProbeTimeout
			if !cmd.IsSet("probe-timeout") {
				commandTimeout = 0
			}
			collector := appcaps.New()
			if only := cmd.String("only"); only != "" {
				list, listErr := collector.CollectList(ctx, appcaps.Options{
					Installation:   installation,
					Refresh:        cmd.Bool("refresh"),
					NoCache:        cmd.Bool("no-cache"),
					CacheDir:       cmd.String("cache-dir"),
					CommandTimeout: commandTimeout,
				}, only)
				if listErr != nil {
					return listErr
				}
				human := fmt.Sprintf("%s: %d entries\n", list.Kind, len(list.Entries))
				return writeResult(cmd, "capabilities", list, human)
			}
			set, err := collector.Collect(ctx, appcaps.Options{
				Installation:   installation,
				HelpTopic:      cmd.String("help-topic"),
				Refresh:        cmd.Bool("refresh"),
				NoCache:        cmd.Bool("no-cache"),
				CacheDir:       cmd.String("cache-dir"),
				CommandTimeout: commandTimeout,
			})
			if err != nil {
				return err
			}
			human := fmt.Sprintf("capabilities: %d command(s), %d decoder(s), cache_hit=%t\n", len(set.Commands), len(set.Decoders), set.CacheHit)
			return writeResult(cmd, "capabilities", set, human)
		},
	}
}

func selectInstallationForCapabilities(ctx context.Context, base appdetect.Options, dir, id string) (model.Installation, error) {
	if (dir == "") == (id == "") {
		return model.Installation{}, apperrors.New(apperrors.E7001, "use exactly one of --dir or --installation-id")
	}
	opts := base
	opts.NoValidate = false
	detector := appdetect.New()

	var installations []model.Installation
	if dir != "" {
		opts.Dir = dir
		opts.Recursive = false
		opts.Scan = false
		opts.Roots = nil
		opts.NoPath = true
		opts.NoEnv = true
		opts.NoCommon = true
		result, err := detector.Detect(ctx, opts)
		if err != nil {
			return model.Installation{}, err
		}
		installations = result.Installations
	} else {
		opts.Dir = ""
		result, err := detector.Detect(ctx, opts)
		if err != nil {
			return model.Installation{}, err
		}
		installations = result.Installations
	}

	if len(installations) == 0 {
		return model.Installation{}, apperrors.New(apperrors.E1002, "no FFmpeg installation found")
	}
	if id != "" {
		for _, installation := range installations {
			if installation.ID == id {
				return requireUsableInstallation(installation)
			}
		}
		return model.Installation{}, apperrors.New(apperrors.E1002, "installation not found").WithDetails(map[string]any{"installation_id": id})
	}
	return requireUsableInstallation(installations[0])
}

func requireUsableInstallation(installation model.Installation) (model.Installation, error) {
	if installation.Status == "unusable" || installation.Status == "unknown" {
		return model.Installation{}, apperrors.New(apperrors.E1004, "installation is not validated")
	}
	return installation, nil
}
