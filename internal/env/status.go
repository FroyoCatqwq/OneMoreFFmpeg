// Package env inspects the managed environment without writing to it.
package env

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/install"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
)

// Options controls environment inspection.
type Options struct {
	DataRoot string
	Env      []string
}

// Status reports the current pointer, PATH and key variables.
func Status(opts Options) model.EnvStatusData {
	environment := opts.Env
	if environment == nil {
		environment = os.Environ()
	}
	kind, target := install.InspectCurrent(opts.DataRoot)
	data := model.EnvStatusData{
		Schema:              model.EnvStatusSchema,
		CurrentType:         kind,
		CurrentTarget:       target,
		PathContainsCurrent: pathContains(environment, filepath.Join(opts.DataRoot, "current", "bin")),
		Variables:           variables(environment),
		Warnings:            []model.Warning{},
	}
	if kind == model.CurrentPointerFile {
		data.Warnings = append(data.Warnings, model.Warning{
			Code:    string(apperrors.W1005),
			Message: "current is a pointer file; PATH cannot follow it",
			Details: map[string]any{"pointer": filepath.Join(opts.DataRoot, "current.pointer")},
		})
	}
	data.ResolvedFFmpeg = resolvedBinary(opts.DataRoot, "ffmpeg")
	return data
}

func variables(environment []string) map[string]*string {
	values := map[string]*string{}
	for _, name := range []string{"ONE_MORE_FFMPEG_DIR", "FFMPEG_DIR", "LAE_FFMPEG_DIR"} {
		values[name] = nil
	}
	for _, pair := range environment {
		key, value, ok := strings.Cut(pair, "=")
		if !ok {
			continue
		}
		if _, tracked := values[key]; tracked {
			copied := value
			values[key] = &copied
		}
	}
	for _, name := range []string{"ONE_MORE_FFMPEG_DIR", "FFMPEG_DIR", "LAE_FFMPEG_DIR"} {
		// Keep null for unset variables; detect empty as unset.
		if pointer := values[name]; pointer != nil && *pointer == "" {
			values[name] = nil
		}
	}
	return values
}

func pathContains(environment []string, dir string) bool {
	target := filepath.Clean(dir)
	for _, pair := range environment {
		key, value, ok := strings.Cut(pair, "=")
		if !ok || !strings.EqualFold(key, "PATH") {
			continue
		}
		for _, entry := range filepath.SplitList(value) {
			if samePath(entry, target) {
				return true
			}
		}
	}
	return false
}

func samePath(left, right string) bool {
	left = filepath.Clean(strings.TrimSpace(left))
	right = filepath.Clean(strings.TrimSpace(right))
	if runtime.GOOS == "windows" {
		return strings.EqualFold(left, right)
	}
	return left == right
}

func resolvedBinary(dataRoot, name string) string {
	for _, candidate := range []string{name, name + ".exe"} {
		path := filepath.Join(dataRoot, "current", "bin", candidate)
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path
		}
	}
	return ""
}
