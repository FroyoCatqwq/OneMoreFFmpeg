package cli

import (
	"strconv"
	"strings"
	"time"

	urfave "github.com/urfave/cli/v3"

	appdetect "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/detect"
	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
)

func detectionFlags() []urfave.Flag {
	return []urfave.Flag{
		&urfave.StringFlag{Name: "dir", Usage: "explicit FFmpeg trio directory or executable path"},
		&urfave.BoolFlag{Name: "recursive", Usage: "recursively scan --dir for ffmpeg-like binaries"},
		&urfave.BoolFlag{Name: "scan", Usage: "explicitly scan configured/default roots"},
		&urfave.StringFlag{Name: "roots", Usage: "comma-separated scan roots; requires --scan"},
		&urfave.StringFlag{Name: "max-depth", Value: "8", Usage: "maximum scan depth"},
		&urfave.StringFlag{Name: "max-files", Value: "200000", Usage: "maximum files visited per scan"},
		&urfave.StringFlag{Name: "timeout", Value: "10", Usage: "discovery/scan budget in seconds (must be >= --probe-timeout)"},
		&urfave.StringFlag{Name: "probe-timeout", Value: "5", Usage: "per-process probe timeout in seconds"},
		&urfave.BoolFlag{Name: "include-removable", Usage: "include removable Windows drives in default roots (requires --scan)"},
		&urfave.BoolFlag{Name: "include-network", Usage: "include network Windows drives in default roots (requires --scan)"},
		&urfave.BoolFlag{Name: "no-path", Usage: "skip PATH discovery"},
		&urfave.BoolFlag{Name: "no-env", Usage: "skip environment variable discovery"},
		&urfave.BoolFlag{Name: "no-common", Usage: "skip common directory discovery"},
		&urfave.BoolFlag{Name: "no-validate", Usage: "skip running ffmpeg validation commands"},
	}
}

func parseDetectOptions(cmd *urfave.Command) (appdetect.Options, error) {
	opts := appdetect.DefaultOptions()
	opts.Dir = cmd.String("dir")
	opts.Recursive = cmd.Bool("recursive")
	opts.Scan = cmd.Bool("scan")
	opts.Roots = parseRoots(cmd.String("roots"))
	opts.IncludeRemovable = cmd.Bool("include-removable")
	opts.IncludeNetwork = cmd.Bool("include-network")
	opts.NoPath = cmd.Bool("no-path")
	opts.NoEnv = cmd.Bool("no-env")
	opts.NoCommon = cmd.Bool("no-common")
	opts.NoValidate = cmd.Bool("no-validate")

	maxDepth, err := parseNonNegativeInt(cmd.String("max-depth"), "max-depth")
	if err != nil {
		return opts, err
	}
	opts.MaxDepth = maxDepth
	maxFiles, err := parsePositiveInt(cmd.String("max-files"), "max-files")
	if err != nil {
		return opts, err
	}
	opts.MaxFiles = maxFiles
	timeoutSeconds, err := parseSeconds(cmd.String("timeout"), "timeout")
	if err != nil {
		return opts, err
	}
	opts.Timeout = time.Duration(timeoutSeconds) * time.Second
	probeSeconds, err := parseSeconds(cmd.String("probe-timeout"), "probe-timeout")
	if err != nil {
		return opts, err
	}
	opts.ProbeTimeout = time.Duration(probeSeconds) * time.Second
	if opts.ProbeTimeout > opts.Timeout {
		return opts, apperrors.New(apperrors.E7005, "--probe-timeout must not exceed --timeout").WithDetails(map[string]any{
			"probe_timeout_seconds": probeSeconds,
			"timeout_seconds":       timeoutSeconds,
		})
	}

	if len(opts.Roots) > 0 && !opts.Scan {
		return opts, apperrors.New(apperrors.E7001, "--roots requires --scan")
	}
	if !opts.Scan && (opts.IncludeRemovable || opts.IncludeNetwork) {
		return opts, apperrors.New(apperrors.E7001, "--include-removable/--include-network require --scan")
	}
	return opts, nil
}

// rejectExtraArgs makes positional arguments an error. Without it a typo such
// as `validate --dir D:\ffmpeg extra` would be silently ignored.
func rejectExtraArgs(cmd *urfave.Command) error {
	if cmd == nil || cmd.NArg() == 0 {
		return nil
	}
	return apperrors.New(apperrors.E7001, "unexpected positional arguments").WithDetails(map[string]any{"args": cmd.Args().Slice()})
}

// rejectDirDiscoveryConflict guards commands that resolve exactly one
// installation from --dir: discovery flags would be silently ignored otherwise.
func rejectDirDiscoveryConflict(cmd *urfave.Command, dir string) error {
	if dir == "" {
		return nil
	}
	if cmd.Bool("scan") || cmd.Bool("recursive") || strings.TrimSpace(cmd.String("roots")) != "" {
		return apperrors.New(apperrors.E7001, "--dir cannot be combined with --scan, --recursive or --roots")
	}
	return nil
}

func parseRoots(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	roots := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			roots = append(roots, part)
		}
	}
	return roots
}

func parseNonNegativeInt(raw, name string) (int, error) {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value < 0 {
		return 0, apperrors.New(apperrors.E7005, "invalid "+name)
	}
	return value, nil
}

func parsePositiveInt(raw, name string) (int, error) {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value <= 0 {
		return 0, apperrors.New(apperrors.E7005, "invalid "+name)
	}
	return value, nil
}

// maxSeconds caps durations well below the int64 nanosecond overflow point.
const maxSeconds = 86400

func parseSeconds(raw, name string) (int, error) {
	value, err := parsePositiveInt(raw, name)
	if err != nil {
		return 0, err
	}
	if value > maxSeconds {
		return 0, apperrors.New(apperrors.E7005, "invalid "+name).WithDetails(map[string]any{"max_seconds": maxSeconds})
	}
	return value, nil
}
