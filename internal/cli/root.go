package cli

import (
	"context"
	"fmt"
	"io"
	"os"
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
type stderrContextKey struct{}

func withConfig(ctx context.Context, cfg config.Config) context.Context {
	return context.WithValue(ctx, configContextKey{}, cfg)
}

func withStderr(ctx context.Context, stderr io.Writer) context.Context {
	return context.WithValue(ctx, stderrContextKey{}, stderr)
}

// stderrFrom returns the caller-supplied stderr so non-fatal warnings from
// download/install obey the same output contract as the rest of the CLI.
func stderrFrom(ctx context.Context) io.Writer {
	if w, ok := ctx.Value(stderrContextKey{}).(io.Writer); ok && w != nil {
		return w
	}
	return os.Stderr
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
	// urfave registers a built-in `help` subcommand that writes plain text to
	// root.Writer (stdout), bypassing the flag-based help checks and the
	// per-command rejectPlainTextUnderJSON guard. Treat it as a plain-text
	// request too, so `--json help [cmd]` cannot emit an envelope-less text
	// dump on stdout.
	plainTextRequest := hasInDetailUsage(opts.Args) || hasHelpFlag(opts.Args) || hasHelpSubcommand(opts.Args)
	if plainTextRequest {
		if wantsJSON(opts.Args) {
			// stdout is JSON-only by contract. Help/usage text has no JSON
			// envelope, so the combination is refused with a valid error
			// envelope rather than dumping plain text onto stdout.
			err := apperrors.New(apperrors.E1001,
				"help/usage writes plain text; drop --json (stdout stays JSON-only)")
			writeFailure(opts.Stdout, opts.Stderr, true, err)
			return apperrors.ExitCode(err)
		}
		if hasInDetailUsage(opts.Args) {
			_, _ = fmt.Fprint(opts.Stdout, detailedUsage)
			return 0
		}
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			// The raw panic value can contain URLs, headers or tokens. Keep the
			// stable code/message on the wire and put only a short, sanitized
			// type hint in cause. Full detail belongs to controlled logs, never
			// to the external JSON/error contract.
			writeFailure(opts.Stdout, opts.Stderr, wantsJSON(opts.Args),
				apperrors.New(apperrors.E9002, "panic recovered").WithCause(sanitizePanic(recovered)))
			code = 1
		}
	}()

	cmd, err := New(opts)
	if err != nil {
		writeFailure(opts.Stdout, opts.Stderr, wantsJSON(opts.Args), err)
		return apperrors.ExitCode(err)
	}
	ctx = withStderr(ctx, opts.Stderr)
	if err := cmd.Run(ctx, opts.Args); err != nil {
		writeFailure(opts.Stdout, opts.Stderr, wantsJSON(opts.Args), err)
		return apperrors.ExitCode(err)
	}
	return 0
}

// v0.6.x: the first-use banner is gone. Start-up prints nothing beyond the
// command's own output; the same guidance lives in --in-detail-usage and
// docs/HOWTO.md, where it is read on purpose instead of on every run.

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
			updateCommand(opts.Version),
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
			ctx = withConfig(ctx, cfg)
			return logging.WithLogger(ctx, logger), nil
		},
		Action: func(ctx context.Context, cmd *urfave.Command) error {
			if cmd.NArg() > 0 {
				return apperrors.New(apperrors.E7001, "unknown command").WithDetails(map[string]any{"args": cmd.Args().Slice()})
			}
			if err := rejectPlainTextUnderJSON(cmd); err != nil {
				return err
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

// takesValue is the single source of truth for "which flag eats the next
// argument". Both wantsJSON (json.go) and hasInDetailUsage must agree, or a
// flag value equal to --in-detail-usage is misread as the usage request.
var takesValue = map[string]bool{
	"--config": true, "-c": true, "--log-level": true,
	"--dir": true, "--roots": true, "--max-depth": true, "--max-files": true,
	"--timeout": true, "--probe-timeout": true,
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
		if arg == "--" {
			break
		}
		if arg == "--in-detail-usage" || arg == "-in-detail-usage" {
			return true
		}
		name, _, hasEquals := strings.Cut(arg, "=")
		if takesValue[name] && !hasEquals {
			index++
		}
	}
	return false
}

// hasHelpFlag detects --help/-h requested as a flag (not as a flag value).
// Combined with --json it must be refused so stdout stays JSON-only.
func hasHelpFlag(args []string) bool {
	for index := 0; index < len(args); index++ {
		arg := args[index]
		if arg == "--" {
			break
		}
		if arg == "--help" || arg == "-h" || arg == "-help" {
			return true
		}
		name, _, hasEquals := strings.Cut(arg, "=")
		if takesValue[name] && !hasEquals {
			index++
		}
	}
	return false
}

// hasHelpSubcommand detects urfave's built-in `help` command at any command
// level. urfave registers a `help` subcommand on EVERY command that has
// children, so `--json provider help`, `--json env help`, etc. all dump plain
// text to stdout. Scan the whole argv (skipping flag values and anything after
// `--`) for a bare `help` token; the parent command's Action never runs for
// the built-in help command, so this check must be global.
func hasHelpSubcommand(args []string) bool {
	for index := 1; index < len(args); index++ { // skip argv[0] (program name)
		arg := args[index]
		if arg == "--" {
			return false // everything after is a positional arg, not a command
		}
		if arg == "help" {
			return true
		}
		if strings.HasPrefix(arg, "-") {
			name, _, hasEquals := strings.Cut(arg, "=")
			if takesValue[name] && !hasEquals {
				index++ // this flag consumes the next token as its value
			}
		}
		// Keep scanning: `help` may follow a parent command (provider help,
		// env set help, ...). Returning early here used to miss those.
	}
	return false
}

// rejectPlainTextUnderJSON guards parent commands whose action only prints
// human-readable help. Under --json such output has no envelope.
func rejectPlainTextUnderJSON(cmd *urfave.Command) error {
	if jsonEnabled(cmd) {
		return apperrors.New(apperrors.E1001,
			"help/usage writes plain text; drop --json (stdout stays JSON-only)")
	}
	return nil
}

// rejectOfflineResolveFlags rejects provider flags that are silently ignored
// when an offline --release file is supplied. Failing fast avoids download and
// install acting on a source the caller did not intend.
func rejectOfflineResolveFlags(cmd *urfave.Command) error {
	if strings.TrimSpace(cmd.String("release")) == "" {
		return nil
	}
	for _, name := range []string{"provider", "channel", "target", "variant", "linkage", "release-line", "release-tag"} {
		if cmd.IsSet(name) {
			return apperrors.New(apperrors.E1001, "--release cannot be combined with provider resolve flags").
				WithDetails(map[string]any{"flag": "--" + name})
		}
	}
	return nil
}

func sanitizePanic(recovered any) string {
	// Only a bounded type label is safe on the wire. Do not include the value:
	// panic payloads are arbitrary and may carry credentials.
	return fmt.Sprintf("panic of type %T", recovered)
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
