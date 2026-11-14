package cli

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf16"

	urfave "github.com/urfave/cli/v3"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/contracts"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/download"
	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/paths"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/provider"
)

// warnToStderr surfaces a non-fatal degradation (a release with no independent
// checksums file, say) without changing the exit code or the JSON payload.
func warnToStderr(msg string) {
	fmt.Fprintf(os.Stderr, "warning: %s\n", msg)
}

func downloadCommand() *urfave.Command {
	flags := providerFlags()
	flags = append(flags,
		&urfave.StringFlag{Name: "release", Usage: "offline release JSON (skips resolve)"},
		&urfave.StringFlag{Name: "dir", Usage: "download directory (absolute)"},
		&urfave.Int64Flag{Name: "max-bytes", Usage: "per-asset size limit"},
	)
	return &urfave.Command{
		Name:  "download",
		Usage: "download and verify one pinned asset",
		Flags: flags,
		Action: func(ctx context.Context, cmd *urfave.Command) error {
			if err := rejectExtraArgs(cmd); err != nil {
				return err
			}
			cfg := configFrom(ctx)
			maxBytes := cfg.Download.MaxBytes
			if cmd.IsSet("max-bytes") {
				maxBytes = cmd.Int64("max-bytes")
				if maxBytes <= 0 {
					return apperrors.New(apperrors.E7005, "--max-bytes must be positive")
				}
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
			if release.LicenseHint == "nonfree" || provider.IsNonfreeVariant(release.Variant) {
				if !cmd.Bool("allow-nonfree") {
					return apperrors.New(apperrors.E7001, "nonfree binaries require --allow-nonfree").WithDetails(map[string]any{
						"provider": release.Provider,
						"asset":    release.AssetName,
					})
				}
				if !hasWarning(release.Warnings, apperrors.W1006) {
					release.Warnings = append(release.Warnings, model.Warning{
						Code:    string(apperrors.W1006),
						Message: "nonfree binary: local use only, must not be redistributed",
						Details: map[string]any{"provider": release.Provider, "variant": release.Variant},
					})
				}
			}

			directory := strings.TrimSpace(cmd.String("dir"))
			if directory == "" {
				directory = strings.TrimSpace(cfg.Download.Dir)
			}
			if directory == "" {
				directory, err = paths.DownloadsDir()
				if err != nil {
					return err
				}
			}
			if !filepath.IsAbs(directory) {
				return apperrors.New(apperrors.E7001, "--dir must be an absolute path").WithDetails(map[string]any{"dir": directory})
			}
			timeout, _ := time.ParseDuration(cfg.Network.Timeout)
			instance, err := download.New(download.Options{
				Dir:          directory,
				MaxBytes:     maxBytes,
				MaxRedirects: cfg.Download.MaxRedirects,
				Timeout:      timeout,
				Warn:         warnToStderr,
			})
			if err != nil {
				return err
			}
			artifact, err := instance.Fetch(ctx, release)
			if err != nil {
				return err
			}
			human := fmt.Sprintf("downloaded %s\nsha256: %s\n", artifact.LocalPath, artifact.AssetSHA256)
			human += formatWarnings(release)
			return writeResult(cmd, "download", artifact, human)
		},
	}
}

func loadReleaseFile(path string) (model.Release, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return model.Release{}, apperrors.New(apperrors.E7007, "release file cannot be read").WithCause(err.Error())
	}
	raw = normalizeJSONInput(raw)
	// Accept either a raw release object or a `resolve --json` result wrapper.
	var envelope struct {
		Schema string          `json:"schema"`
		Data   json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err == nil && envelope.Schema == model.ResultSchema && len(envelope.Data) > 0 {
		raw = envelope.Data
	}
	var release model.Release
	if err := json.Unmarshal(raw, &release); err != nil {
		return model.Release{}, apperrors.New(apperrors.E7003, "release file is not valid JSON").WithCause(err.Error())
	}
	if release.Schema != model.ReleaseSchema {
		return model.Release{}, apperrors.New(apperrors.E7003, "release file schema mismatch").WithDetails(map[string]any{"schema": release.Schema})
	}
	if release.ReleaseTag == "latest" && (release.ReleaseID == "" || release.PublishedAt == "") {
		return model.Release{}, apperrors.New(apperrors.E3010, "release file is not pinned").WithDetails(map[string]any{
			"release_tag": release.ReleaseTag,
			"release_id":  release.ReleaseID,
			"published":   release.PublishedAt,
		})
	}
	if err := contracts.ValidateDataFields(release); err != nil {
		return model.Release{}, err
	}
	return release, nil
}

// normalizeJSONInput accepts UTF-8 (with or without BOM) and UTF-16LE/BE,
// which Windows PowerShell 5 `>` redirection commonly produces.
func normalizeJSONInput(raw []byte) []byte {
	if len(raw) >= 3 && raw[0] == 0xEF && raw[1] == 0xBB && raw[2] == 0xBF {
		return raw[3:]
	}
	little := false
	if len(raw) >= 2 && raw[0] == 0xFF && raw[1] == 0xFE {
		little, raw = true, raw[2:]
	} else if len(raw) >= 2 && raw[0] == 0xFE && raw[1] == 0xFF {
		raw = raw[2:]
	} else if len(raw) > 3 && raw[1] == 0 && (raw[0] == '{' || raw[0] == '[') {
		little = true
	} else {
		return raw
	}
	if len(raw) < 2 {
		return raw
	}
	units := make([]uint16, len(raw)/2)
	for index := range units {
		if little {
			units[index] = binary.LittleEndian.Uint16(raw[index*2:])
		} else {
			units[index] = binary.BigEndian.Uint16(raw[index*2:])
		}
	}
	return []byte(string(utf16.Decode(units)))
}

func hasWarning(warnings []model.Warning, code apperrors.Code) bool {
	for _, warning := range warnings {
		if warning.Code == string(code) {
			return true
		}
	}
	return false
}
