// Package detect implements read-only FFmpeg discovery and validation.
package detect

import (
	"context"
	stderrors "errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/ffexec"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
)

type Options struct {
	Dir              string
	Recursive        bool
	Scan             bool
	Roots            []string
	MaxDepth         int
	MaxFiles         int
	Timeout          time.Duration
	ProbeTimeout     time.Duration
	IncludeRemovable bool
	IncludeNetwork   bool
	NoPath           bool
	NoEnv            bool
	NoCommon         bool
	NoValidate       bool
}

func DefaultOptions() Options {
	return Options{
		MaxDepth:     8,
		MaxFiles:     200000,
		Timeout:      10 * time.Second,
		ProbeTimeout: 5 * time.Second,
	}
}

func normalizeOptions(opts Options) Options {
	if opts.MaxDepth < 0 {
		opts.MaxDepth = 8
	}
	if opts.MaxFiles <= 0 {
		opts.MaxFiles = 200000
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 10 * time.Second
	}
	if opts.ProbeTimeout <= 0 {
		opts.ProbeTimeout = 5 * time.Second
	}
	return opts
}

type Candidate struct {
	Root     string
	Sources  map[string]bool
	Priority int
}

type Result struct {
	Installations []model.Installation
	Errors        []model.DetectError
	Reports       map[string]model.ValidateData
}

type Detector struct {
	Runner ffexec.Runner
}

func New() *Detector {
	return &Detector{
		Runner: ffexec.Runner{
			Timeout:   5 * time.Second,
			WaitDelay: 500 * time.Millisecond,
		},
	}
}

type session struct {
	candidates map[string]*Candidate
	errors     []model.DetectError
}

func newSession() *session {
	return &session{candidates: map[string]*Candidate{}}
}

func (s *session) addDir(dir, source string) {
	root := canonicalRoot(dir)
	key := rootKey(root)
	priority := sourcePriority[source]
	if existing, ok := s.candidates[key]; ok {
		existing.Sources[source] = true
		if priority < existing.Priority {
			existing.Priority = priority
		}
		return
	}
	s.candidates[key] = &Candidate{
		Root:     root,
		Sources:  map[string]bool{source: true},
		Priority: priority,
	}
}

func (s *session) addPath(raw, source string, fatal bool, details map[string]any) error {
	dirs, err := resolveDirs(raw)
	if err != nil {
		info := map[string]any{"path": raw, "source": source}
		for key, value := range details {
			info[key] = value
		}
		code, message := pathError(err)
		if fatal {
			return apperrors.New(code, "explicit directory is not usable").WithCause(err.Error()).WithDetails(info)
		}
		info["message"] = message
		s.errors = append(s.errors, newDetectError(code, message, info))
		return nil
	}
	for _, dir := range dirs {
		s.addDir(dir, source)
	}
	return nil
}

func (d *Detector) Detect(ctx context.Context, opts Options) (Result, error) {
	opts = normalizeOptions(opts)
	if ctx == nil {
		ctx = context.Background()
	}
	scanCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	runner := d.Runner
	runner.Timeout = opts.ProbeTimeout
	if runner.WaitDelay <= 0 {
		runner.WaitDelay = 500 * time.Millisecond
	}

	s := newSession()

	if opts.Dir != "" {
		if opts.Recursive {
			scanOpts := opts
			scanOpts.Roots = []string{opts.Dir}
			scanOpts.Scan = true
			scanRoots(scanCtx, scanOpts, func(dir, _ string) { s.addDir(dir, sourceExplicitDir) }, &s.errors)
		} else if err := s.addPath(opts.Dir, sourceExplicitDir, true, nil); err != nil {
			return Result{}, err
		}
	}

	if !opts.NoEnv {
		for _, name := range []string{"ONE_MORE_FFMPEG_DIR", "FFMPEG_DIR", "LAE_FFMPEG_DIR"} {
			if value := strings.TrimSpace(os.Getenv(name)); value != "" {
				_ = s.addPath(value, sourceEnv, false, map[string]any{"env_var": name})
			}
		}
	}

	if !opts.NoPath {
		for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
			if strings.TrimSpace(dir) == "" {
				continue
			}
			if findBinary(dir, "ffmpeg") != "" {
				s.addDir(dir, sourcePATH)
			}
		}
	}

	if !opts.NoCommon {
		for _, dir := range commonDirs() {
			if findBinary(dir, "ffmpeg") != "" {
				s.addDir(dir, sourceCommonDir)
			}
		}
	}

	if opts.Scan {
		scanRoots(scanCtx, opts, s.addDir, &s.errors)
	}

	if stderrors.Is(ctx.Err(), context.Canceled) {
		return Result{}, apperrors.New(apperrors.E9004, "context canceled")
	}
	if stderrors.Is(ctx.Err(), context.DeadlineExceeded) {
		return Result{}, apperrors.New(apperrors.E1006, "detection timed out").WithCause(ctx.Err().Error())
	}
	if stderrors.Is(scanCtx.Err(), context.DeadlineExceeded) {
		s.errors = append(s.errors, newDetectError(apperrors.E1008, "detection timeout", nil))
	}

	keys := make([]string, 0, len(s.candidates))
	for key := range s.candidates {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	result := Result{
		Installations: []model.Installation{},
		Errors:        []model.DetectError{},
		Reports:       map[string]model.ValidateData{},
	}
	type ordered struct {
		installation model.Installation
		report       model.ValidateData
		priority     int
	}
	items := make([]ordered, 0, len(keys))
	for _, key := range keys {
		candidate := s.candidates[key]
		// Probing has its own per-process timeout, so it must not inherit the
		// scanning budget: an exhausted scan budget used to mark healthy
		// installations as unusable.
		installation, report := d.buildInstallation(ctx, candidate, opts, runner)
		if installation.FFmpeg == "" && installation.FFprobe == "" && installation.FFplay == "" {
			// Directories from env vars or --dir that contain no FFmpeg binary
			// are not installations; reporting them as "unusable" is noise.
			continue
		}
		items = append(items, ordered{installation: installation, report: report, priority: candidate.Priority})
	}
	if err := ctx.Err(); err != nil {
		if stderrors.Is(err, context.Canceled) {
			return Result{}, apperrors.New(apperrors.E9004, "context canceled")
		}
		return Result{}, apperrors.New(apperrors.E1006, "detection timed out").WithCause(err.Error())
	}
	sort.Slice(items, func(i, j int) bool {
		iHasFFmpeg := items[i].installation.FFmpeg != ""
		jHasFFmpeg := items[j].installation.FFmpeg != ""
		if iHasFFmpeg != jHasFFmpeg {
			return iHasFFmpeg
		}
		if items[i].priority != items[j].priority {
			return items[i].priority < items[j].priority
		}
		return items[i].installation.RootDir < items[j].installation.RootDir
	})
	for _, item := range items {
		result.Installations = append(result.Installations, item.installation)
		result.Reports[item.installation.ID] = item.report
	}
	result.Errors = dedupeErrors(s.errors)
	return result, nil
}

func (d *Detector) ValidateDir(ctx context.Context, dir string, base Options) (model.ValidateData, error) {
	opts := normalizeOptions(base)
	opts.Dir = dir
	opts.Recursive = false
	opts.Scan = false
	opts.NoPath = true
	opts.NoEnv = true
	opts.NoCommon = true
	opts.NoValidate = false
	result, err := d.Detect(ctx, opts)
	if err != nil {
		return model.ValidateData{}, err
	}
	for _, installation := range result.Installations {
		if installation.FFmpeg == "" {
			continue
		}
		report, ok := result.Reports[installation.ID]
		if !ok {
			return model.ValidateData{}, apperrors.New(apperrors.E9001, "validation report missing")
		}
		return report, nil
	}
	return model.ValidateData{}, apperrors.New(apperrors.E1002, "no FFmpeg binaries found").WithDetails(map[string]any{"dir": dir})
}

func (d *Detector) ValidateID(ctx context.Context, id string, base Options) (model.ValidateData, error) {
	opts := normalizeOptions(base)
	opts.Dir = ""
	opts.Scan = false
	opts.NoValidate = false
	result, err := d.Detect(ctx, opts)
	if err != nil {
		return model.ValidateData{}, err
	}
	report, ok := result.Reports[id]
	if !ok {
		return model.ValidateData{}, apperrors.New(apperrors.E1002, "installation not found").WithDetails(map[string]any{"installation_id": id})
	}
	return report, nil
}

func dedupeErrors(errs []model.DetectError) []model.DetectError {
	seen := map[string]bool{}
	out := make([]model.DetectError, 0, len(errs))
	for _, item := range errs {
		key := item.Code + "|" + item.Message + "|" + joinDetails(item.Details)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, item)
	}
	return out
}
