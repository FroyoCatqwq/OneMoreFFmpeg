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
		&urfave.StringFlag{Name: "timeout", Value: "10", Usage: "overall detection timeout in seconds"},
		&urfave.StringFlag{Name: "probe-timeout", Value: "5", Usage: "per-process probe timeout in seconds"},
		&urfave.BoolFlag{Name: "include-removable", Usage: "include removable Windows drives in default roots"},
		&urfave.BoolFlag{Name: "include-network", Usage: "include network Windows drives in default roots"},
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
	timeoutSeconds, err := parsePositiveInt(cmd.String("timeout"), "timeout")
	if err != nil {
		return opts, err
	}
	opts.Timeout = time.Duration(timeoutSeconds) * time.Second
	probeSeconds, err := parsePositiveInt(cmd.String("probe-timeout"), "probe-timeout")
	if err != nil {
		return opts, err
	}
	opts.ProbeTimeout = time.Duration(probeSeconds) * time.Second

	if len(opts.Roots) > 0 && !opts.Scan {
		return opts, apperrors.New(apperrors.E7001, "--roots requires --scan")
	}
	return opts, nil
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
