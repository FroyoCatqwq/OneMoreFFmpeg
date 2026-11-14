package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	urfave "github.com/urfave/cli/v3"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/checksum"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/config"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/download"
	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/ffexec"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/install"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/lock"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/paths"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/provider"
)

func installCommand() *urfave.Command {
	flags := providerFlags()
	flags = append(flags,
		&urfave.StringFlag{Name: "release", Usage: "offline release JSON (skips resolve)"},
		&urfave.StringFlag{Name: "artifact", Usage: "downloaded archive path (absolute)"},
		&urfave.BoolFlag{Name: "apply", Usage: "actually install (default is dry-run)"},
		&urfave.BoolFlag{Name: "force", Usage: "replace an existing install directory"},
		&urfave.BoolFlag{Name: "no-current", Usage: "do not update the current pointer"},
		&urfave.BoolFlag{Name: "emit-trio-lock", Usage: "write ffmpeg-trio.lock inside the install directory"},
	)
	return &urfave.Command{
		Name:  "install",
		Usage: "extract a downloaded archive into a versioned directory (dry-run by default)",
		Flags: flags,
		Action: func(ctx context.Context, cmd *urfave.Command) error {
			if err := rejectExtraArgs(cmd); err != nil {
				return err
			}
			var release model.Release
			var err error
			if path := strings.TrimSpace(cmd.String("release")); path != "" {
				release, err = loadReleaseFile(path)
			} else {
				release, err = resolveRelease(ctx, cmd)
			}
			if err != nil {
				return err
			}
			return installRelease(ctx, cmd, release)
		},
	}
}

func installRelease(ctx context.Context, cmd *urfave.Command, release model.Release) error {
	if release.LicenseHint == "nonfree" || provider.IsNonfreeVariant(release.Variant) {
		if !cmd.Bool("allow-nonfree") {
			return apperrors.New(apperrors.E7001, "nonfree binaries require --allow-nonfree").WithDetails(map[string]any{
				"provider": release.Provider,
				"asset":    release.AssetName,
			})
		}
	}
	dataRoot, err := paths.DataRoot()
	if err != nil {
		return err
	}
	apply := cmd.Bool("apply")
	archivePath, err := installArchivePath(ctx, cmd, configFrom(ctx), release, apply)
	if err != nil {
		return err
	}
	if apply {
		unlock, err := lock.DataRoot(dataRoot)
		if err != nil {
			return err
		}
		defer unlock()
	}
	result, err := install.Run(ctx, install.Options{
		DataRoot:    dataRoot,
		Release:     release,
		ArchivePath: archivePath,
		Apply:       apply,
		Force:       cmd.Bool("force"),
		NoCurrent:   cmd.Bool("no-current"),
		EmitLock:    cmd.Bool("emit-trio-lock"),
		Runner:      newInstallRunner(),
	})
	if err != nil {
		return err
	}
	human := fmt.Sprintf("root: %s\ncurrent: %s\ninstallation: %s\n",
		result.RootDir, result.CurrentPointer, result.InstallationID)
	if result.DryRun {
		human = "dry-run (use --apply to install)\n" + human
	}
	return writeResult(cmd, "install", result, human)
}

func installArchivePath(ctx context.Context, cmd *urfave.Command, cfg config.Config, release model.Release, apply bool) (string, error) {
	if explicit := strings.TrimSpace(cmd.String("artifact")); explicit != "" {
		if !filepath.IsAbs(explicit) {
			return "", apperrors.New(apperrors.E7001, "--artifact must be an absolute path")
		}
		if err := verifyArchive(ctx, explicit, release.AssetSHA256); err != nil {
			return "", err
		}
		return explicit, nil
	}
	directory := strings.TrimSpace(cfg.Download.Dir)
	if directory == "" {
		var err error
		directory, err = paths.DownloadsDir()
		if err != nil {
			return "", err
		}
	}
	expected := filepath.Join(directory, release.AssetSHA256[:16], release.AssetName)
	if !apply {
		return expected, nil
	}
	maxBytes := cfg.Download.MaxBytes
	timeout, _ := time.ParseDuration(cfg.Network.Timeout)
	instance, err := download.New(download.Options{
		Dir:          directory,
		MaxBytes:     maxBytes,
		MaxRedirects: cfg.Download.MaxRedirects,
		Timeout:      timeout,
		Warn:         warnToStderr,
	})
	if err != nil {
		return "", err
	}
	artifact, err := instance.Fetch(ctx, release)
	if err != nil {
		return "", err
	}
	return artifact.LocalPath, nil
}

func newInstallRunner() ffexec.Runner {
	return ffexec.Runner{Timeout: 15 * time.Second}
}

func verifyArchive(ctx context.Context, path, expected string) error {
	file, err := os.Open(path)
	if err != nil {
		return apperrors.New(apperrors.E7007, "archive file cannot be read").WithCause(err.Error())
	}
	defer file.Close()
	sum, _, err := checksum.StreamSHA256(ctx, file)
	if err != nil {
		return err
	}
	if sum != expected {
		return apperrors.New(apperrors.E4002, "archive does not match the pinned sha256").WithDetails(map[string]any{
			"expected": expected,
			"actual":   sum,
		})
	}
	return nil
}
