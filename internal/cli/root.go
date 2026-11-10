package cli

import (
	"context"
	"fmt"
	"io"
	"os"

	urfave "github.com/urfave/cli/v3"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/buildinfo"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/config"
	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/logging"
)

type RunOptions struct {
	Args    []string
	Stdout  io.Writer
	Stderr  io.Writer
	Version buildinfo.Info
}

func Run(ctx context.Context, opts RunOptions) (code int) {
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	if opts.Stderr == nil {
		opts.Stderr = os.Stderr
	}
	if len(opts.Args) == 0 {
		opts.Args = []string{"one-more-ffmpeg"}
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			writeFailure(opts.Stdout, opts.Stderr, wantsJSON(opts.Args),
				apperrors.New(apperrors.E9002, "panic recovered").WithCause(fmt.Sprint(recovered)))
			code = 1
		}
	}()

	cmd, err := New(opts)
	if err != nil {
		writeFailure(opts.Stdout, opts.Stderr, wantsJSON(opts.Args), err)
		return apperrors.ExitCode(err)
	}
	if err := cmd.Run(ctx, opts.Args); err != nil {
		writeFailure(opts.Stdout, opts.Stderr, wantsJSON(opts.Args), err)
		return apperrors.ExitCode(err)
	}
	return 0
}

func New(opts RunOptions) (*urfave.Command, error) {
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	if opts.Stderr == nil {
		opts.Stderr = os.Stderr
	}
	root := &urfave.Command{
		Name:        "one-more-ffmpeg",
		Usage:       "Manage FFmpeg discovery, download, installation and environment",
		HideVersion: true,
		Writer:      opts.Stdout,
		ErrWriter:   opts.Stderr,
		Flags: []urfave.Flag{
			&urfave.BoolFlag{Name: "json", Usage: "write machine-readable JSON to stdout"},
			&urfave.StringFlag{Name: "config", Aliases: []string{"c"}, Usage: "TOML config path"},
			&urfave.StringFlag{Name: "log-level", Usage: "error|warn|info|debug|trace"},
			&urfave.BoolFlag{Name: "quiet", Usage: "only log errors"},
		},
		Commands: []*urfave.Command{
			versionCommand(opts.Version),
			detectCommand(),
		},
		Before: func(ctx context.Context, cmd *urfave.Command) (context.Context, error) {
			cfg, err := config.Load(cmd.String("config"))
			if err != nil {
				return ctx, err
			}
			if level := cmd.String("log-level"); level != "" {
				cfg.Log.Level = level
			}
			if cmd.Bool("quiet") {
				cfg.Log.Level = "error"
			}
			logger, err := logging.New(opts.Stderr, cfg.Log)
			if err != nil {
				return ctx, err
			}
			logger.Debug("configuration loaded",
				"config", cmd.String("config"),
				"log_level", cfg.Log.Level,
				"log_json", cfg.Log.JSON,
			)
			return logging.WithLogger(ctx, logger), nil
		},
		Action: func(ctx context.Context, cmd *urfave.Command) error {
			return urfave.ShowRootCommandHelp(cmd)
		},
		OnUsageError: func(ctx context.Context, cmd *urfave.Command, err error, isSubcommand bool) error {
			return apperrors.New(apperrors.E7001, "invalid command usage").WithCause(err.Error())
		},
		ExitErrHandler: func(ctx context.Context, cmd *urfave.Command, err error) {
			// Errors are handled by Run so tests and hosts never call os.Exit.
		},
	}
	for _, sub := range root.Commands {
		sub.Writer = root.Writer
		sub.ErrWriter = root.ErrWriter
		sub.OnUsageError = root.OnUsageError
		sub.ExitErrHandler = root.ExitErrHandler
	}
	return root, nil
}
