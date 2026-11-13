package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	urfave "github.com/urfave/cli/v3"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/buildinfo"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/config"
	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/logging"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/paths"
)

type RunOptions struct {
	Args    []string
	Stdout  io.Writer
	Stderr  io.Writer
	Version buildinfo.Info
}

type configContextKey struct{}

func withConfig(ctx context.Context, cfg config.Config) context.Context {
	return context.WithValue(ctx, configContextKey{}, cfg)
}

func configFrom(ctx context.Context) config.Config {
	if cfg, ok := ctx.Value(configContextKey{}).(config.Config); ok {
		return cfg
	}
	return config.Default()
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
	if hasInDetailUsage(opts.Args) {
		_, _ = fmt.Fprint(opts.Stdout, detailedUsage)
		return 0
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

func showFirstUseNotice(stderr io.Writer) {
	root, err := paths.DataRoot()
	if err == nil {
		marker := filepath.Join(root, ".first-use")
		if _, statErr := os.Stat(marker); statErr == nil {
			return
		}
		_ = os.MkdirAll(root, 0o755)
		_ = os.WriteFile(marker, []byte("seen\n"), 0o644)
	}
	_, _ = fmt.Fprint(stderr, firstUseNotice)
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
			&urfave.BoolFlag{Name: "in-detail-usage", Usage: "print the full built-in usage guide"},
		},
		Commands: []*urfave.Command{
			versionCommand(opts.Version),
			detectCommand(),
			validateCommand(),
			capabilitiesCommand(),
			providerCommand(),
			resolveCommand(),
			downloadCommand(),
			installCommand(),
			envCommand(),
			doctorCommand(),
			uninstallCommand(),
			journalCommand(),
			releaseCommand(),
		},
		Before: func(ctx context.Context, cmd *urfave.Command) (context.Context, error) {
			cfg, configPath, err := loadConfig(cmd)
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
				"config", configPath,
				"log_level", cfg.Log.Level,
				"log_json", cfg.Log.JSON,
			)
			if configPath == "" {
				showFirstUseNotice(opts.Stderr)
			}
			ctx = withConfig(ctx, cfg)
			return logging.WithLogger(ctx, logger), nil
		},
		Action: func(ctx context.Context, cmd *urfave.Command) error {
			if cmd.NArg() > 0 {
				return apperrors.New(apperrors.E7001, "unknown command").WithDetails(map[string]any{"args": cmd.Args().Slice()})
			}
			return urfave.ShowRootCommandHelp(cmd)
		},
		OnUsageError: func(ctx context.Context, cmd *urfave.Command, err error, isSubcommand bool) error {
			return apperrors.New(apperrors.E7001, "invalid command usage").WithCause(err.Error())
		},
		ExitErrHandler: func(ctx context.Context, cmd *urfave.Command, err error) {
			// Errors are handled by Run so tests and hosts never call os.Exit.
		},
	}
	var configure func(*urfave.Command)
	configure = func(command *urfave.Command) {
		command.Writer = root.Writer
		command.ErrWriter = root.ErrWriter
		command.OnUsageError = root.OnUsageError
		command.ExitErrHandler = root.ExitErrHandler
		for _, sub := range command.Commands {
			configure(sub)
		}
	}
	for _, sub := range root.Commands {
		configure(sub)
	}
	return root, nil
}

var globalValueFlags = map[string]bool{
	"--config": true, "-c": true, "--log-level": true,
	"--provider": true, "--channel": true, "--target": true, "--variant": true,
	"--linkage": true, "--release-line": true, "--release-tag": true,
	"--release": true, "--artifact": true, "--root": true, "--journal": true,
	"--id": true, "--url": true, "--sha256": true, "--checksums-url": true,
	"--license-hint": true, "--max-bytes": true, "--days": true, "--count": true,
	"--releases-url": true, "--version": true, "--snapshot": true, "--only": true,
	"--installation-id": true, "--help-topic": true, "--cache-dir": true,
}

// hasInDetailUsage avoids treating a flag value such as
// `--config --in-detail-usage` as the usage flag itself.
func hasInDetailUsage(args []string) bool {
	for index := 0; index < len(args); index++ {
		arg := args[index]
		if arg == "--in-detail-usage" || arg == "-in-detail-usage" {
			return true
		}
		name, _, hasEquals := strings.Cut(arg, "=")
		if globalValueFlags[name] && !hasEquals {
			index++
		}
	}
	return false
}

// loadConfig resolves the config source in this order:
//  1. an explicit --config path (must be readable);
//  2. <data-root>/config.toml when it exists (missing file is not an error);
//  3. built-in defaults.
//
// An unresolvable data root is ignored here: read-only commands such as
// `capabilities --no-cache` must work even without HOME/LOCALAPPDATA.
func loadConfig(cmd *urfave.Command) (config.Config, string, error) {
	if cmd.IsSet("config") {
		path := strings.TrimSpace(cmd.String("config"))
		if path == "" {
			return config.Default(), "", nil
		}
		cfg, err := config.Load(path)
		return cfg, path, err
	}
	path, err := paths.ConfigFile()
	if err != nil {
		typed, ok := apperrors.As(err)
		if ok && typed.Code == apperrors.E9003 {
			return config.Default(), "", nil
		}
		return config.Config{}, "", err
	}
	if _, err := os.Stat(path); err != nil && os.IsNotExist(err) {
		return config.Default(), "", nil
	}
	cfg, err := config.LoadOptional(path)
	if err != nil {
		return config.Config{}, path, err
	}
	return cfg, path, nil
}
